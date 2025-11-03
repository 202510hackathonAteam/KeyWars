// internal/infra/repository/redis/round_state_repository_redis_test.go
package redisrepo_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	drepo "keywars/backend/internal/domain/repository"
	redisrepo "keywars/backend/internal/infra/repository/redis"
)

func TestRoundState_ApplyAnswer(t *testing.T) {
	miniRedis, _ := miniredis.Run()
	defer miniRedis.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	repository := redisrepo.NewRoundStateRepositoryRedis(redisClient)

	contextObject := context.Background()
	matchID := "m123"
	user1ID, user2ID := "u1", "u2"
	currentTimeMs := time.Now().UnixMilli()

	// メタ & 開始
	if err := repository.CreateMeta(contextObject, matchID, user1ID, user2ID, currentTimeMs); err != nil {
		t.Fatal(err)
	}
	if err := repository.Start(contextObject, matchID); err != nil {
		t.Fatal(err)
	}

	// デッキ保存（1件だけでもOK）
	if err := repository.SaveDeckOnce(contextObject, matchID, []string{`{"pid":1,"surface":"A","reading":"a","diff":1,"char_count":1,"limit_ms":3000}`}); err != nil {
		t.Fatal(err)
	}

	// 回答適用（user1ID が user2ID へダメージ）
	eventID, turn, err := repository.ApplyAnswer(contextObject, drepo.AnswerApplyArg{
		MatchID:              matchID,
		OpponentUserID:       user2ID,
		NewOpponentLifePoint: 7, // 例: 10→7
		NextDeckIndex:        1, // 次へ進める
		CurrentServerTimeMs:  currentTimeMs + 100,
		EventFields: map[string]string{
			"uid":        user1ID,
			"deck_idx":   "1",
			"correct":    "1",
			"dmg":        "3",
			"elapsed_ms": "1200",
		},
	})
	if err != nil {
		t.Fatalf("ApplyAnswer err: %v", err)
	}
	if eventID == "" || turn != 1 {
		t.Fatalf("unexpected result eventID=%q turn=%d", eventID, turn)
	}

	// state が更新されているか（turn, deck_idx, last_event_id）
	stateHash, err := redisClient.HGetAll(contextObject, "match:"+matchID+":state").Result()
	if err != nil {
		t.Fatal(err)
	}
	if stateHash["turn"] != "1" {
		t.Fatalf("turn want=1 got=%s", stateHash["turn"])
	}
	if stateHash["deck_idx"] != "1" {
		t.Fatalf("deck_idx want=1 got=%s", stateHash["deck_idx"])
	}
	if stateHash["last_event_id"] == "" {
		t.Fatalf("last_event_id empty")
	}
	if lifePoint := stateHash["p"+user2ID+":lp"]; lifePoint != "7" {
		t.Fatalf("opponent lp want=7 got=%s", lifePoint)
	}

	// events に XADD されているか
	streamEntries, err := redisClient.XRangeN(contextObject, "match:"+matchID+":events", "-", "+", 10).Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(streamEntries) != 1 {
		t.Fatalf("events want=1 got=%d", len(streamEntries))
	}
	if streamEntries[0].Values["turn"] != strconv.FormatInt(turn, 10) {
		t.Fatalf("event turn mismatch")
	}
}
