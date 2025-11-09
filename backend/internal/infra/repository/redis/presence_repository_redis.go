package redisrepo

import (
	"context"
	"fmt"
	"time"

	"keywars/backend/internal/domain/repository"

	"github.com/redis/go-redis/v9"
)

// PresenceRepositoryRedis は、ユーザーのプレゼンス（オンライン／対戦中／オフライン状態）を
// Redis 上で管理するリポジトリ実装。
//
// Redis のキー構成：
//
//	presence:{userID} （HASH）
//	  - status        : "online" | "ingame" | "offline"
//	  - last_seen_ms  : 最終アクティブ時刻（UNIXミリ秒）
//	  - match_id      : 対戦中のマッチID（"ingame" のときのみ）
//
// TTL（有効期限）を設定することで、心拍更新が止まった場合に自然消滅する仕組み。
// これにより「自動ログアウト」や「強制切断後のリーク防止」を実現する。
type PresenceRepositoryRedis struct {
	redisClient        *redis.Client // Redisクライアントインスタンス
	heartbeatTTL       time.Duration // 通常時（オンラインまたは対戦中）のTTL
	disconnectGraceTTL time.Duration // 明示切断後に情報を保持するTTL
}

// NewPresenceRepositoryRedis は PresenceRepositoryRedis のコンストラクタ。
// Redis クライアントと TTL 設定を受け取り、リポジトリを初期化する。
func NewPresenceRepositoryRedis(
	redisClient *redis.Client,
	heartbeatTTL time.Duration,
	disconnectGraceTTL time.Duration,
) *PresenceRepositoryRedis {
	return &PresenceRepositoryRedis{
		redisClient:        redisClient,
		heartbeatTTL:       heartbeatTTL,
		disconnectGraceTTL: disconnectGraceTTL,
	}
}

// presenceKey は、指定したユーザーIDに対応する Redis キー名を生成する。
// 例: presence:u001
func presenceKey(userID string) string {
	return fmt.Sprintf("presence:%s", userID)
}

// SetOnline は、ユーザーを「オンライン状態」として登録し、TTL を設定する。
//
// 引数:
//
//	contextObject - コンテキスト（タイムアウト／キャンセル制御）
//	userID        - 対象ユーザーID
//	currentTimeMs - 現在時刻（UNIXミリ秒）
//
// 動作:
//
//	Redis の presence:{userID} HASH に以下を保存：
//	  status="online", last_seen_ms=currentTimeMs, match_id=""
//	さらに TTL を heartbeatTTL に設定する。
func (repository *PresenceRepositoryRedis) SetOnline(
	contextObject context.Context,
	userID string,
	currentTimeMs int64,
) error {
	presenceKeyName := presenceKey(userID)
	pipeline := repository.redisClient.TxPipeline()

	pipeline.HSet(contextObject, presenceKeyName,
		"status", "online",
		"last_seen_ms", currentTimeMs,
		"match_id", "",
	)
	pipeline.Expire(contextObject, presenceKeyName, repository.heartbeatTTL)

	_, err := pipeline.Exec(contextObject)
	return err
}

// Heartbeat は、ユーザーが継続的にオンラインであることを示す心拍。
// last_seen_ms を更新し、TTL を延長する。
//
// 引数:
//
//	contextObject - コンテキスト
//	userID        - 対象ユーザーID
//	currentTimeMs - 現在時刻（UNIXミリ秒）
//
// 動作:
//
//	HASH の last_seen_ms を更新し、TTL を再設定。
func (repository *PresenceRepositoryRedis) Heartbeat(
	contextObject context.Context,
	userID string,
	currentTimeMs int64,
) error {
	presenceKeyName := presenceKey(userID)
	pipeline := repository.redisClient.TxPipeline()

	pipeline.HSet(contextObject, presenceKeyName, "last_seen_ms", currentTimeMs)
	pipeline.Expire(contextObject, presenceKeyName, repository.heartbeatTTL)

	_, err := pipeline.Exec(contextObject)
	return err
}

// SetIngame は、ユーザーを「対戦中（ingame）」状態として登録し、
// 対戦中のマッチIDを記録する。
//
// 引数:
//
//	contextObject - コンテキスト
//	userID        - 対象ユーザーID
//	matchID       - 対戦中のマッチID
//	currentTimeMs - 現在時刻（UNIXミリ秒）
//
// 動作:
//
//	Redis の presence:{userID} HASH に以下を保存：
//	  status="ingame", match_id=matchID, last_seen_ms=currentTimeMs
//	TTL は heartbeatTTL に設定。
func (repository *PresenceRepositoryRedis) SetIngame(
	contextObject context.Context,
	userID string,
	matchID string,
	currentTimeMs int64,
) error {
	presenceKeyName := presenceKey(userID)
	pipeline := repository.redisClient.TxPipeline()

	pipeline.HSet(contextObject, presenceKeyName,
		"status", "ingame",
		"last_seen_ms", currentTimeMs,
		"match_id", matchID,
	)
	pipeline.Expire(contextObject, presenceKeyName, repository.heartbeatTTL)

	_, err := pipeline.Exec(contextObject)
	return err
}

// Disconnect は、ユーザーを「オフライン（offline）」として登録し、
// 直近状態を短期間だけ保持する。
//
// 引数:
//
//	contextObject - コンテキスト
//	userID        - 対象ユーザーID
//	currentTimeMs - 現在時刻（UNIXミリ秒）
//
// 動作:
//
//	Redis の presence:{userID} HASH に以下を保存：
//	  status="offline", match_id="", last_seen_ms=currentTimeMs
//	TTL は disconnectGraceTTL に設定（例: 30秒など）。
func (repository *PresenceRepositoryRedis) Disconnect(
	contextObject context.Context,
	userID string,
	currentTimeMs int64,
) error {
	presenceKeyName := presenceKey(userID)
	pipeline := repository.redisClient.TxPipeline()

	pipeline.HSet(contextObject, presenceKeyName,
		"status", "offline",
		"last_seen_ms", currentTimeMs,
		"match_id", "",
	)
	pipeline.Expire(contextObject, presenceKeyName, repository.disconnectGraceTTL)

	_, err := pipeline.Exec(contextObject)
	return err
}

// Get は、指定したユーザーの現在のプレゼンス状態を取得する。
//
// 引数:
//
//	contextObject - コンテキスト
//	userID        - 対象ユーザーID
//
// 戻り値:
//
//	map[string]string - 現在の状態を格納したマップ（status, last_seen_ms, match_id）
//	error             - 取得エラー（キーが存在しない場合は空マップが返る）
func (repository *PresenceRepositoryRedis) Get(
	contextObject context.Context,
	userID string,
) (map[string]string, error) {
	presenceKeyName := presenceKey(userID)
	return repository.redisClient.HGetAll(contextObject, presenceKeyName).Result()
}

// コンパイル時にインターフェース適合を保証。
var _ repository.PresenceRepository = (*PresenceRepositoryRedis)(nil)
