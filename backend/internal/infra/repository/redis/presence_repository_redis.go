package redisrepo

import (
	"context"
	"fmt"
	"time"

	"keywars/backend/internal/domain/repository"

	"github.com/redis/go-redis/v9"
)

// 仕様： user:{uid}:presence（HASH + TTL 30s）
// fields: status (offline|online|ingame|reconnecting), match_id, updated_at(ms), socket_count
// ops:
//  - 接続時:     HSET status online ... ; HINCRBY socket_count 1 ; EXPIRE 30
//  - 心拍:       HSET updated_at now ; PEXPIRE 30000
//  - 参戦:       HSET status ingame match_id {mid} ; PEXPIRE 30000
//  - 切断:       HINCRBY socket_count -1 ; PEXPIRE 30000
// TTLは常に30秒（心拍で延長）

const (
	presenceTTL = 30 * time.Second // 30秒（PEXPIREで延長）
)

// PresenceRepositoryRedis は、ユーザーのプレゼンスを Redis に保持・更新する実装。
type PresenceRepositoryRedis struct {
	redisClient *redis.Client
}

func NewPresenceRepositoryRedis(redisClient *redis.Client) *PresenceRepositoryRedis {
	return &PresenceRepositoryRedis{
		redisClient: redisClient,
	}
}

// presenceKey は user:{userID}:presence を返す。
func presenceKey(userID string) string {
	return fmt.Sprintf("user:%s:presence", userID)
}

// Heartbeat はクライアント側からの定期心拍で呼び出す。
// - updated_at を now に更新
// - TTL を 30s に延長（PEXPIRE）
func (r *PresenceRepositoryRedis) Heartbeat(
	ctx context.Context,
	userID string,
	nowUnixMilli int64,
) error {
	key := presenceKey(userID)
	pipe := r.redisClient.TxPipeline()

	pipe.HSet(ctx, key, "updated_at", nowUnixMilli)
	pipe.PExpire(ctx, key, presenceTTL)

	_, err := pipe.Exec(ctx)
	return err
}

// SetIngame は対戦開始時に呼び出す。
// - status = ingame
// - match_id = {matchID}
// - updated_at を now に
// - TTL を 30s に延長（PEXPIRE）
func (r *PresenceRepositoryRedis) SetIngame(
	ctx context.Context,
	userID string,
	matchID string,
	nowUnixMilli int64,
) error {
	key := presenceKey(userID)
	pipe := r.redisClient.TxPipeline()

	pipe.HSet(ctx, key,
		"status", "ingame",
		"match_id", matchID,
		"updated_at", nowUnixMilli,
	)
	pipe.PExpire(ctx, key, presenceTTL)

	_, err := pipe.Exec(ctx)
	return err
}

// SetReconnecting は回線復帰中などを表現したい場合に利用（任意）。
// - status = reconnecting
// - updated_at を now に
// - TTL を 30s に延長（PEXPIRE）
func (r *PresenceRepositoryRedis) SetReconnecting(
	ctx context.Context,
	userID string,
	nowUnixMilli int64,
) error {
	key := presenceKey(userID)
	pipe := r.redisClient.TxPipeline()

	pipe.HSet(ctx, key,
		"status", "reconnecting",
		"updated_at", nowUnixMilli,
	)
	pipe.PExpire(ctx, key, presenceTTL)

	_, err := pipe.Exec(ctx)
	return err
}

// OnDisconnect は WebSocket 切断時に呼び出し、socket_count を減算する。
// - socket_count を -1
// - 0 以下になった場合は offline へ落とし、socket_count=0 に補正、match_id は残す/消すは要件次第
//   - ここでは match_id は「残す」とし、presence は TTLで自然消滅させる。
//   - TTL は常に 30s に延長（PEXPIRE）
//     （ブラウザがすぐ再接続する前提でも presence を短時間保持したい）
func (r *PresenceRepositoryRedis) Disconnect(
	ctx context.Context,
	userID string,
	nowUnixMilli int64,
) error {
	key := presenceKey(userID)
	// HINCRBY の結果を見て分岐したいので、TxPipelineで値を拾う
	pipe := r.redisClient.TxPipeline()

	socketCountCmd := pipe.HIncrBy(ctx, key, "socket_count", -1)
	pipe.HSet(ctx, key, "updated_at", nowUnixMilli)
	pipe.PExpire(ctx, key, presenceTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	// 0 未満になった場合の補正と offline 落とし
	if socketCountCmd.Val() <= 0 {
		// 補正と状態更新（別Tx）
		repair := r.redisClient.TxPipeline()
		repair.HSet(ctx, key,
			"socket_count", 0,
			"status", "offline",
			"updated_at", nowUnixMilli,
		)
		repair.PExpire(ctx, key, presenceTTL)
		_, _ = repair.Exec(ctx)
	}

	return nil
}

// Get は現在のプレゼンスをそのまま返す。
// （キーが無ければ空マップ / redis.Nil 対応は go-redis の Result() 仕様に準ずる）
func (r *PresenceRepositoryRedis) Get(
	ctx context.Context,
	userID string,
) (map[string]string, error) {
	key := presenceKey(userID)
	return r.redisClient.HGetAll(ctx, key).Result()
}

var _ repository.PresenceRepository = (*PresenceRepositoryRedis)(nil)
