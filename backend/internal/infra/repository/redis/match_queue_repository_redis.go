package redisrepo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyQueue = "mq:queue"
	keyLock  = "lock:mq"
	lockTTL  = 3 * time.Second
)

type MatchQueueRepositoryRedis struct{ rdb *redis.Client }

func NewMatchQueueRepositoryRedis(r *redis.Client) *MatchQueueRepositoryRedis {
	return &MatchQueueRepositoryRedis{rdb: r}
}

func (r *MatchQueueRepositoryRedis) Enqueue(ctx context.Context, userID string, ms int64) error {
	return r.rdb.ZAddNX(ctx, keyQueue, redis.Z{Score: float64(ms), Member: userID}).Err()
}
func (r *MatchQueueRepositoryRedis) Cancel(ctx context.Context, userID string) error {
	return r.rdb.ZRem(ctx, keyQueue, userID).Err()
}
func (r *MatchQueueRepositoryRedis) Score(ctx context.Context, userID string) (float64, error) {
	return r.rdb.ZScore(ctx, keyQueue, userID).Result()
}

func (r *MatchQueueRepositoryRedis) DequeuePairAndInitMatch(ctx context.Context) (u1, u2, mid, token string, err error) {
	token, err = randToken()
	if err != nil {
		return
	}
	ok, err := r.rdb.SetNX(ctx, keyLock, token, lockTTL).Result()
	if err != nil || !ok {
		if err == nil {
			err = errors.New("lock busy")
		}
		return
	}
	defer func() { _ = r.ReleaseLock(ctx, token) }()

	zs, err := r.rdb.ZPopMin(ctx, keyQueue, 2).Result()
	if err != nil {
		return
	}
	if len(zs) < 2 {
		for _, z := range zs {
			_ = r.rdb.ZAdd(ctx, keyQueue, z).Err()
		}
		return "", "", "", "", nil
	}
	u1 = zs[0].Member.(string)
	u2 = zs[1].Member.(string)

	mid, err = randToken()
	if err != nil {
		_ = r.rdb.ZAdd(ctx, keyQueue, zs...).Err()
		return
	}
	nowMs := time.Now().UnixMilli()
	pipe := r.rdb.TxPipeline()
	pipe.HSet(ctx, fmt.Sprintf("match:%s", mid), "status", "waiting", "created_at", nowMs, "p1", u1, "p2", u2)
	pipe.HSet(ctx, fmt.Sprintf("match:%s:state", mid), "deck_idx", 0, "q_started_at_ms", nowMs, "turn", 0)
	if _, err = pipe.Exec(ctx); err != nil {
		_ = r.rdb.ZAdd(ctx, keyQueue, zs...).Err()
		return
	}
	return
}

func (r *MatchQueueRepositoryRedis) ReleaseLock(ctx context.Context, token string) error {
	return r.rdb.Watch(ctx, func(tx *redis.Tx) error {
		v, err := tx.Get(ctx, keyLock).Result()
		if err == redis.Nil {
			return nil
		}
		if err != nil {
			return err
		}
		if v != token {
			return nil
		}
		_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error { p.Del(ctx, keyLock); return nil })
		return err
	}, keyLock)
}

func randToken() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}
