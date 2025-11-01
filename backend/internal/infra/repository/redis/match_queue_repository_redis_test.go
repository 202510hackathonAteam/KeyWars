// internal/infra/repository/redis/match_queue_repository_redis_test.go
package redisrepo_test

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	redisrepo "keywars/backend/internal/infra/repository/redis"
)

func newClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: addr})
}

func TestMatchQueue_Enqueue_DequeuePairAndInit(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	rdb := newClient(mr.Addr())
	repo := redisrepo.NewMatchQueueRepositoryRedis(rdb)

	ctx := context.Background()
	now := time.Now().UnixMilli()

	// enqueue 2人
	if err := repo.Enqueue(ctx, "u1", now); err != nil {
		t.Fatal(err)
	}
	if err := repo.Enqueue(ctx, "u2", now+1); err != nil {
		t.Fatal(err)
	}

	u1, u2, mid, token, err := repo.DequeuePairAndInitMatch(ctx)
	if err != nil {
		t.Fatalf("dequeue err: %v", err)
	}
	if u1 == "" || u2 == "" || mid == "" || token == "" {
		t.Fatalf("unexpected empty values u1=%q u2=%q mid=%q token=%q", u1, u2, mid, token)
	}

	// 二重取り出しは人不足で空戻り
	uu1, uu2, mmid, tok, err := repo.DequeuePairAndInitMatch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if uu1 != "" || uu2 != "" || mmid != "" || tok != "" {
		t.Fatalf("expected no pair, got %q %q %q %q", uu1, uu2, mmid, tok)
	}
}

func TestMatchQueue_Cancel(t *testing.T) {
	mr, _ := miniredis.Run()
	defer mr.Close()

	rdb := newClient(mr.Addr())
	repo := redisrepo.NewMatchQueueRepositoryRedis(rdb)

	ctx := context.Background()
	now := time.Now().UnixMilli()

	if err := repo.Enqueue(ctx, "u1", now); err != nil {
		t.Fatal(err)
	}
	if err := repo.Cancel(ctx, "u1"); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.Score(ctx, "u1"); err == nil {
		t.Fatalf("expected no score for canceled user")
	}
}
