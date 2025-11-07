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

func newClient(address string) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: address})
}

func TestMatchQueue_Enqueue_DequeuePairAndInit(t *testing.T) {
	miniRedis, _ := miniredis.Run()
	defer miniRedis.Close()

	redisClient := newClient(miniRedis.Addr())
	repository := redisrepo.NewMatchQueueRepositoryRedis(redisClient)

	contextObject := context.Background()
	currentTimeMs := time.Now().UnixMilli()

	// enqueue 2人
	if err := repository.Enqueue(contextObject, "u1", currentTimeMs); err != nil {
		t.Fatal(err)
	}
	if err := repository.Enqueue(contextObject, "u2", currentTimeMs+1); err != nil {
		t.Fatal(err)
	}

	user1ID, user2ID, matchID, lockToken, err := repository.DequeuePairAndInitMatch(contextObject)
	if err != nil {
		t.Fatalf("dequeue err: %v", err)
	}
	if user1ID == "" || user2ID == "" || matchID == "" || lockToken == "" {
		t.Fatalf("unexpected empty values user1ID=%q user2ID=%q matchID=%q lockToken=%q", user1ID, user2ID, matchID, lockToken)
	}

	// 二重取り出しは人不足で空戻り
	user1ID2, user2ID2, matchID2, lockToken2, err := repository.DequeuePairAndInitMatch(contextObject)
	if err != nil {
		t.Fatal(err)
	}
	if user1ID2 != "" || user2ID2 != "" || matchID2 != "" || lockToken2 != "" {
		t.Fatalf("expected no pair, got %q %q %q %q", user1ID2, user2ID2, matchID2, lockToken2)
	}
}

func TestMatchQueue_Cancel(t *testing.T) {
	miniRedis, _ := miniredis.Run()
	defer miniRedis.Close()

	redisClient := newClient(miniRedis.Addr())
	repository := redisrepo.NewMatchQueueRepositoryRedis(redisClient)

	contextObject := context.Background()
	currentTimeMs := time.Now().UnixMilli()

	if err := repository.Enqueue(contextObject, "u1", currentTimeMs); err != nil {
		t.Fatal(err)
	}
	if err := repository.Cancel(contextObject, "u1"); err != nil {
		t.Fatal(err)
	}

	if _, err := repository.Score(contextObject, "u1"); err == nil {
		t.Fatalf("expected no score for canceled user")
	}
}
