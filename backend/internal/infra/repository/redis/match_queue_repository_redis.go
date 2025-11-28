package redisrepo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"keywars/backend/internal/config"
)

const (
	// keyQueue はマッチング待機キュー（ZSET）の Redis キー。
	// - member: userID（文字列）
	// - score : enqueue 時刻（UNIX ミリ秒; 昇順＝先着）
	keyQueue = "mq:queue"

	// keyLock は待機キューのペア取り出し処理に対する排他ロック用キー。
	// DequeuePairAndInitMatch の同時実行を抑止する。
	keyLock = "lock:mq"

	// lockTTL はロック自動解放までの有効期限。
	// フェイル時のスタック状態を防ぐ（best-effort）。
	lockTTL = 3 * time.Second
)

// MatchQueueRepositoryRedis は、Redis を利用した待機キュー操作（ZSET）と
// マッチ初期化（match:{matchID} / match:{matchID}:state の初期 HSET）を提供する実装。
type MatchQueueRepositoryRedis struct {
	// redisClient は接続済み go-redis v9 クライアント。
	// 本構造体内の全操作で共有する。
	redisClient *redis.Client
}

// NewMatchQueueRepositoryRedis は、外部で生成された Redis クライアントを受け取り
// リポジトリアダプタを返すコンストラクタ。
func NewMatchQueueRepositoryRedis(redisClient *redis.Client) *MatchQueueRepositoryRedis {
	return &MatchQueueRepositoryRedis{redisClient: redisClient}
}

// Enqueue は、ユーザーを待機キューに追加する。
// 既にメンバーが存在する場合は追加されない（NX）。
//
// Redis: ZADD NX mq:queue <score=enqueueAtMs> <member=userID>
func (repository *MatchQueueRepositoryRedis) Enqueue(contextObject context.Context, userID string, enqueueAtMs int64) error {
	return repository.redisClient.ZAddNX(contextObject, keyQueue, redis.Z{
		Score:  float64(enqueueAtMs),
		Member: userID,
	}).Err()
}

// Cancel は、指定ユーザーを待機キューから削除する。
// ユーザーが待機中でない場合は no-op（エラーではない）。
//
// Redis: ZREM mq:queue <userID>
func (repository *MatchQueueRepositoryRedis) Cancel(contextObject context.Context, userID string) error {
	return repository.redisClient.ZRem(contextObject, keyQueue, userID).Err()
}

// Score は、指定ユーザーの「待機スコア（＝投入時刻 ms）」を取得する。
// メンバーが存在しない場合は redis.Nil が返る点に注意（呼び出し側で扱う）。
//
// Redis: ZSCORE mq:queue <userID>
func (repository *MatchQueueRepositoryRedis) Score(contextObject context.Context, userID string) (float64, error) {
	return repository.redisClient.ZScore(contextObject, keyQueue, userID).Result()
}

// DequeuePairAndInitMatch は、待機キューから 2 名を先着順に取り出し、
// 新しいマッチのメタ情報／進行状態を初期化する。
// この処理はロック（keyLock）を使って「1プロセスのみ」実行される。
// 2 名未満の場合は安全に取得分を戻し、空値を返して終了する。
//
// 処理フロー：
//  1. NX ロック取得（SetNX lock:mq）
//  2. 2名取り出し（ZPOPMIN mq:queue 2）
//  3. 2名未満なら ZADD で戻して終了
//  4. match:{matchID} / match:{matchID}:state を TxPipeline で初期化
//  5. 初期化失敗時は 2 名を ZADD で再投入してロールバック
func (repository *MatchQueueRepositoryRedis) DequeuePairAndInitMatch(contextObject context.Context) (user1ID, user2ID, matchID, lockToken string, err error) {

	// 1) ロック取得（トークン発行→NX セット）
	lockToken, err = generateRandomToken()
	if err != nil {
		return
	}
	lockAcquired, err := repository.redisClient.SetNX(contextObject, keyLock, lockToken, lockTTL).Result()
	if err != nil || !lockAcquired {
		if err == nil {
			err = errors.New("lock busy") // 競合により取得できなかった
		}
		return
	}
	defer func() {
		_ = repository.ReleaseLock(contextObject, lockToken)
	}()

	// 2) キューから 2 名取り出し（atomic pop）
	zsetResults, err := repository.redisClient.ZPopMin(contextObject, keyQueue, 2).Result()
	if err != nil {
		return
	}

	// 3) 2 名揃わない場合は戻して終了
	if len(zsetResults) < 2 {
		for _, zsetEntry := range zsetResults {
			_ = repository.redisClient.ZAdd(contextObject, keyQueue, zsetEntry).Err()
		}
		return "", "", "", "", nil
	}
	user1ID = zsetResults[0].Member.(string)
	user2ID = zsetResults[1].Member.(string)

	// マッチID（matchID）を生成
	matchID, err = generateRandomToken()
	if err != nil {
		// 失敗時は取得済みの2名を再投入
		_ = repository.redisClient.ZAdd(contextObject, keyQueue, zsetResults...).Err()
		return
	}

	// 4) マッチのメタ／進行状態を初期化（TxPipeline＝同時確定）
	currentTimeMs := time.Now().UnixMilli()
	pipeline := repository.redisClient.TxPipeline()
	matchKey := fmt.Sprintf("match:%s", matchID)
	matchStateKey := fmt.Sprintf("match:%s:state", matchID)

	// match:{matchID} : メタ情報
	pipeline.HSet(contextObject,
		fmt.Sprintf("match:%s", matchID),
		"status", "waiting",
		"created_at", currentTimeMs,
		"player1", user1ID,
		"player2", user2ID,
	)

	// match:{matchID}:state : 進行状態（初期値）
	pipeline.HSet(contextObject,
		fmt.Sprintf("match:%s:state", matchID),
		"deck_index", config.InitialDeckIndex,
		"round", config.InitialRound,
		"player1_lifepoint", config.InitialLifePoint,
		"player2_lifepoint", config.InitialLifePoint,
		"player1_total_miss_count", 0,
		"player2_total_miss_count", 0,
	)

	// ---  試合開始時点で TTL を設定 ---
	pipeline.Expire(contextObject, matchKey, config.MatchExpiryOnStart)
	pipeline.Expire(contextObject, matchStateKey, config.MatchExpiryOnStart)

	// 5) 実行。失敗時は 2 名をキューへ戻して整合性を保つ
	if _, err = pipeline.Exec(contextObject); err != nil {
		_ = repository.redisClient.ZAdd(contextObject, keyQueue, zsetResults...).Err()
		return
	}

	return
}

// ReleaseLock は、与えられたトークンが現行のロックと一致する場合に限って
// ロックキーを削除する。Watch を用いてトークンの整合性を保証する。
// トークン不一致・キー欠損は no-op（安全側）。
//
// Redis: GET lock:mq → (一致時) Tx(DEL lock:mq)
func (repository *MatchQueueRepositoryRedis) ReleaseLock(contextObject context.Context, lockToken string) error {
	return repository.redisClient.Watch(contextObject, func(transaction *redis.Tx) error {
		storedToken, err := transaction.Get(contextObject, keyLock).Result()
		if err == redis.Nil {
			return nil // 既に削除済み
		}
		if err != nil {
			return err
		}
		if storedToken != lockToken {
			return nil // トークン不一致：他プロセスのロックなので触らない
		}
		_, err = transaction.TxPipelined(contextObject, func(pipeline redis.Pipeliner) error {
			pipeline.Del(contextObject, keyLock)
			return nil
		})
		return err
	}, keyLock)
}

// generateRandomToken は、16 バイト乱数を16進文字列へ変換して返す。
// 用途：ロックトークン、マッチID などの識別子。
func generateRandomToken() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	return hex.EncodeToString(bytes), err
}
