package redisrepo

import (
	"context"
	"fmt"
	"strconv"
	"time"

	drepo "keywars/backend/internal/domain/repository"

	"github.com/redis/go-redis/v9"
)

// RoundStateRepositoryRedis は、対戦進行中の「メタ情報・状態・イベント・デッキ」を
// Redis 上の複数キーに分割して管理するリポジトリ実装。
// キー構成：
//   - match:{mid}         ... メタ情報（status, created_at, p1, p2, winner_user_id）
//   - match:{mid}:state   ... 進行状態（deck_idx, q_started_at_ms, turn, p{uid}:lp, last_event_id）
//   - match:{mid}:events  ... イベント Streams（answer などの出来事）
//   - match:{mid}:deck    ... 出題デッキ（LIST; 要素はJSON文字列）
type RoundStateRepositoryRedis struct {
	// redisClient は go-redis v9 のクライアント。
	// 1インスタンスを本構造体で共有して各操作に使用する。
	redisClient *redis.Client
}

// NewRoundStateRepositoryRedis は、外部から提供された Redis クライアントで実装を初期化する。
// DI しやすいコンストラクタ。
func NewRoundStateRepositoryRedis(redisClient *redis.Client) *RoundStateRepositoryRedis {
	return &RoundStateRepositoryRedis{redisClient: redisClient}
}

// CreateMeta は、マッチのメタ情報を新規作成する。
// 役割：参加者IDや作成時刻を保存し、初期状態を "waiting" に設定する。
// ここでは state/events/deck には触れず、責務を分離している。
func (repository *RoundStateRepositoryRedis) CreateMeta(contextObject context.Context, matchID, user1ID, user2ID string, currentTimeMs int64) error {
	metaKey := fmt.Sprintf("match:%s", matchID)
	return repository.redisClient.HSet(contextObject, metaKey,
		"status", "waiting",
		"created_at", currentTimeMs,
		"p1", user1ID,
		"p2", user2ID,
	).Err()
}

// Start は、マッチを playing 状態へ遷移し、関連キーへ TTL を設定する。
// 意図：進行中の試合データが放置されても自動的に回収されるよう GC を効かせる。
// TTL は運用方針に応じて調整可。
func (repository *RoundStateRepositoryRedis) Start(contextObject context.Context, matchID string) error {
	matchKey := fmt.Sprintf("match:%s", matchID)
	pipeline := repository.redisClient.TxPipeline()

	// ステータスを playing に更新
	pipeline.HSet(contextObject, matchKey, "status", "playing")

	// 主要キーに 1 時間の TTL を付与（ハング・リーク対策）
	expiration := time.Hour
	for _, suffix := range []string{"", ":state", ":events", ":deck"} {
		pipeline.Expire(contextObject, matchKey+suffix, expiration)
	}

	_, err := pipeline.Exec(contextObject)
	return err
}

// Finish は、マッチを finished 状態に更新し、勝者を記録したうえで短い TTL に切り替える。
// 意図：終了後しばらくは参照できるが、不要に残り続けないようにする。
func (repository *RoundStateRepositoryRedis) Finish(contextObject context.Context, matchID, winnerUserID string) error {
	matchKey := fmt.Sprintf("match:%s", matchID)
	pipeline := repository.redisClient.TxPipeline()

	// ステータスと勝者IDを保存
	pipeline.HSet(contextObject, matchKey,
		"status", "finished",
		"winner_user_id", winnerUserID,
	)

	// 終了後は 10 分で掃除（ミリ秒精度で設定）
	expiration := 10 * time.Minute
	for _, suffix := range []string{"", ":state", ":events", ":deck"} {
		pipeline.PExpire(contextObject, matchKey+suffix, expiration)
	}

	_, err := pipeline.Exec(contextObject)
	return err
}

// SaveDeckOnce は、出題デッキ(LIST)を「未作成のときだけ」保存する（冪等）。
// 競合時は WATCH により存在チェックと追加を疑似原子的に実施する。
func (repository *RoundStateRepositoryRedis) SaveDeckOnce(contextObject context.Context, matchID string, deckItems []string) error {
	deckKey := fmt.Sprintf("match:%s:deck", matchID)

	return repository.redisClient.Watch(contextObject, func(transaction *redis.Tx) error {
		// 既存チェック（存在する場合はスキップ）
		// 既存チェック（存在する場合はスキップ）
		exists, err := transaction.Exists(contextObject, deckKey).Result()
		if err != nil {
			return err
		}
		if exists == 1 {
			return nil
		}

		// 変更競合を検知しつつ、一括で LIST 末尾に積む
		_, err = transaction.TxPipelined(contextObject, func(pipeliner redis.Pipeliner) error {
			for _, item := range deckItems {
				pipeliner.RPush(contextObject, deckKey, item)
			}
			return nil
		})
		return err
	}, deckKey)
}

// GetDeckItem は、出題デッキから指定インデックス（0-based）の要素（JSON文字列）を取得する。
// 未存在の場合は redis.Nil が返る点に注意。
func (repository *RoundStateRepositoryRedis) GetDeckItem(contextObject context.Context, matchID string, deckIndex int64) (string, error) {
	deckKey := fmt.Sprintf("match:%s:deck", matchID)
	return repository.redisClient.LIndex(contextObject, deckKey, deckIndex).Result()
}

// ApplyAnswer は、回答結果を state に反映し（LP, deck_idx, q_started_at_ms, turn）、
// 続けて events Streams に answer イベントを追加する 2 フェーズ処理。
// Phase A: WATCH + TxPipelined で turn 競合を回避しつつ state 更新
// Phase B: XADD でイベントを記録し、last_event_id を best-effort で更新
func (repository *RoundStateRepositoryRedis) ApplyAnswer(contextObject context.Context, answerArgs drepo.AnswerApplyArg) (eventID string, turn int64, err error) {
	stateKey := fmt.Sprintf("match:%s:state", answerArgs.MatchID)
	eventsKey := fmt.Sprintf("match:%s:events", answerArgs.MatchID)

	// --- Phase A: 状態更新（turn の整合性確保のため WATCH を使用）---
	err = repository.redisClient.Watch(contextObject, func(transaction *redis.Tx) error {
		// 現在の turn を取得。未設定なら 0 とみなす。
		values, err := transaction.HMGet(contextObject, stateKey, "turn").Result()
		if err != nil {
			return err
		}

		var currentTurn int64
		if len(values) > 0 && values[0] != nil {
			// HMGet は interface{} を返すため、string にキャストしてから parse
			currentTurn, _ = strconv.ParseInt(values[0].(string), 10, 64)
		}

		// TxPipelined: WATCH 中の原子的更新
		_, err = transaction.TxPipelined(contextObject, func(pipeliner redis.Pipeliner) error {
			// 対戦相手の LP を更新。キーは p{uid}:lp（例: p123:lp）
			pipeliner.HSet(contextObject, stateKey, fmt.Sprintf("p%s:lp", answerArgs.OpponentUserID), answerArgs.NewOpponentLifePoint)

			// ターンを +1（競合があれば WATCH により失敗→再実行 or エラー）
			pipeliner.HIncrBy(contextObject, stateKey, "turn", 1)

			// 次の問題へ進む場合のみ、deck_idx と q_started_at_ms を更新
			if answerArgs.NextDeckIndex >= 0 {
				pipeliner.HSet(contextObject, stateKey,
					"deck_idx", answerArgs.NextDeckIndex,
					"q_started_at_ms", answerArgs.CurrentServerTimeMs,
				)
			}
			return nil
		})
		if err == nil {
			// 呼び出し側に「更新後の turn 」を返す
			turn = currentTurn + 1
		}
		return err
	}, stateKey)
	if err != nil {
		return "", 0, err
	}

	// --- Phase B: イベント追加（予約済みフィールドを除外して積む）---
	eventValues := []interface{}{
		"type", "answer",
		"turn", strconv.FormatInt(turn, 10),
		"server_ts", strconv.FormatInt(answerArgs.CurrentServerTimeMs, 10),
	}
	for key, value := range answerArgs.EventFields {
		if key == "type" || key == "turn" || key == "server_ts" {
			continue
		}
		eventValues = append(eventValues, key, value)
	}

	// Streams にイベントを追加。成功時のみ last_event_id を best-effort で更新。
	eventID, err = repository.redisClient.XAdd(contextObject, &redis.XAddArgs{
		Stream: eventsKey,
		Values: eventValues,
	}).Result()
	if err == nil {
		_ = repository.redisClient.HSet(contextObject, stateKey, "last_event_id", eventID).Err()
	}
	return
}
