package redisrepo

import (
	"context"
	"fmt"
	"strconv"
	"time"

	drepo "keywars/backend/internal/domain/repository"

	"github.com/redis/go-redis/v9"
)

type RoundStateRepositoryRedis struct{ rdb *redis.Client }

func NewRoundStateRepositoryRedis(r *redis.Client) *RoundStateRepositoryRedis {
	return &RoundStateRepositoryRedis{rdb: r}
}

func (m *RoundStateRepositoryRedis) CreateMeta(ctx context.Context, mid, u1, u2 string, nowMs int64) error {
	key := fmt.Sprintf("match:%s", mid)
	return m.rdb.HSet(ctx, key, "status", "waiting", "created_at", nowMs, "p1", u1, "p2", u2).Err()
}

func (m *RoundStateRepositoryRedis) Start(ctx context.Context, mid string) error {
	k := fmt.Sprintf("match:%s", mid)
	pipe := m.rdb.TxPipeline()
	pipe.HSet(ctx, k, "status", "playing")
	ttl := time.Hour
	for _, s := range []string{"", ":state", ":events", ":deck"} {
		pipe.Expire(ctx, k+s, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (m *RoundStateRepositoryRedis) Finish(ctx context.Context, mid, winnerUID string) error {
	k := fmt.Sprintf("match:%s", mid)
	pipe := m.rdb.TxPipeline()
	pipe.HSet(ctx, k, "status", "finished", "winner_user_id", winnerUID)
	ttl := 10 * time.Minute
	for _, s := range []string{"", ":state", ":events", ":deck"} {
		pipe.PExpire(ctx, k+s, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (m *RoundStateRepositoryRedis) SaveDeckOnce(ctx context.Context, mid string, deckJSON []string) error {
	key := fmt.Sprintf("match:%s:deck", mid)
	return m.rdb.Watch(ctx, func(tx *redis.Tx) error {
		exists, err := tx.Exists(ctx, key).Result()
		if err != nil {
			return err
		}
		if exists == 1 {
			return nil
		}
		_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
			for _, v := range deckJSON {
				p.RPush(ctx, key, v)
			}
			return nil
		})
		return err
	}, key)
}

func (m *RoundStateRepositoryRedis) GetDeckItem(ctx context.Context, mid string, deckIdx int64) (string, error) {
	return m.rdb.LIndex(ctx, fmt.Sprintf("match:%s:deck", mid), deckIdx).Result()
}

func (m *RoundStateRepositoryRedis) ApplyAnswer(ctx context.Context, a drepo.AnswerApplyArg) (eventID string, turn int64, err error) {
	keyState := fmt.Sprintf("match:%s:state", a.MID)
	keyEvents := fmt.Sprintf("match:%s:events", a.MID)

	// Phase A: state更新（WATCH）
	err = m.rdb.Watch(ctx, func(tx *redis.Tx) error {
		vals, err := tx.HMGet(ctx, keyState, "turn").Result()
		if err != nil {
			return err
		}
		var curTurn int64
		if vals[0] != nil {
			curTurn, _ = strconv.ParseInt(vals[0].(string), 10, 64)
		}
		_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
			p.HSet(ctx, keyState, fmt.Sprintf("p%s:lp", a.OppUID), a.NewOppLP)
			p.HIncrBy(ctx, keyState, "turn", 1)
			if a.NextDeckIdx >= 0 {
				p.HSet(ctx, keyState, "deck_idx", a.NextDeckIdx, "q_started_at_ms", a.NowMs)
			}
			return nil
		})
		if err == nil {
			turn = curTurn + 1
		}
		return err
	}, keyState)
	if err != nil {
		return "", 0, err
	}

	// Phase B: events追加 → last_event_id（ベストエフォート）
	values := []interface{}{"type", "answer", "turn", strconv.FormatInt(turn, 10), "server_ts", strconv.FormatInt(a.NowMs, 10)}
	for k, v := range a.EventFields {
		if k == "type" || k == "turn" || k == "server_ts" {
			continue
		}
		values = append(values, k, v)
	}
	eventID, err = m.rdb.XAdd(ctx, &redis.XAddArgs{Stream: keyEvents, Values: values}).Result()
	if err == nil {
		_ = m.rdb.HSet(ctx, keyState, "last_event_id", eventID).Err()
	}
	return
}
