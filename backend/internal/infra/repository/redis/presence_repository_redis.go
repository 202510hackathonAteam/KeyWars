package redis

import (
	"context"
	"fmt"

	"keywars/backend/internal/config"
	"keywars/backend/internal/domain/repository"

	"github.com/redis/go-redis/v9"
)

// 仕様： user:{uid}:presence（HASH + TTL 30s）
// fields: updated_at(ms), socket_count
// ops:
//  - 接続時:     HINCRBY socket_count 1 ; EXPIRE 30
//  - 心拍:       HSET updated_at now ; PEXPIRE 30000
//  - 切断:       HINCRBY socket_count -1 ; PEXPIRE 30000
// TTLは常に35秒（心拍で延長）


var _ repository.PresenceRepository = (*PresenceRepositoryRedis)(nil)

// PresenceRepositoryRedis は、ユーザーのプレゼンスを Redis に保持・更新する実装。
type PresenceRepositoryRedis struct {
	redisClient *redis.Client
}

// NewPresenceRepositoryRedis は、Redis を用いた PresenceRepository の生成。
func NewPresenceRepositoryRedis(redisClient *redis.Client) repository.PresenceRepository {
	return &PresenceRepositoryRedis{
		redisClient: redisClient,
	}
}

// presenceKey は user:{userID}:presence を返す。
func presenceKey(userID string) string {
	return fmt.Sprintf("user:%s:presence", userID)
}

// Connect は WebSocket 接続確立時に呼び出され、
// 対象ユーザーの接続数（socket_count）を増加させ、
// presence を一定時間保持するため TTL を延長する。
func (r *PresenceRepositoryRedis) Connect(
	ctx context.Context,
	userID string,
	nowUnixMilli int64,
) error {
	key := presenceKey(userID)

	if _, err := r.redisClient.HIncrBy(ctx, key, "socket_count", 1).Result(); err != nil {
		return err
	}
	if err := r.redisClient.HSet(ctx, key, "updated_at", nowUnixMilli).Err(); err != nil {
		return err
	}

	return r.redisClient.PExpire(ctx, key, config.PresenceTTL).Err()
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
	pipe.PExpire(ctx, key, config.PresenceTTL)

	_, err := pipe.Exec(ctx)
	return err
}

// Disconnect は WebSocket 切断時に呼び出され、対象ユーザーの socket_count を減算。
// 切断による接続状態の変化のみを反映する。
// presence キーは再接続を考慮して一定時間保持するため、常に TTL（PEXPIRE）を延長する。
func (r *PresenceRepositoryRedis) Disconnect(
	ctx context.Context,
	userID string,
	nowUnixMilli int64,
) error {
	key := presenceKey(userID)

	_, err := r.redisClient.HIncrBy(ctx, key, "socket_count", -1).Result()
	if err != nil {
		return err
	}

	// 接続が残っていても、誰もいなくても、presence は一定時間保持する
	err = r.redisClient.PExpire(ctx, key, config.PresenceTTL).Err()
	if err != nil {
		return err
	}

	return nil
}
