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
	mr, _ := miniredis.Run()
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	repo := redisrepo.NewRoundStateRepositoryRedis(rdb)

	ctx := context.Background()
	mid := "m123"
	u1, u2 := "u1", "u2"
	now := time.Now().UnixMilli()

	// メタ & 開始
	if err := repo.CreateMeta(ctx, mid, u1, u2, now); err != nil {
		t.Fatal(err)
	}
	if err := repo.Start(ctx, mid); err != nil {
		t.Fatal(err)
	}

	// デッキ保存（1件だけでもOK）
	if err := repo.SaveDeckOnce(ctx, mid, []string{`{"pid":1,"surface":"A","reading":"a","diff":1,"char_count":1,"limit_ms":3000}`}); err != nil {
		t.Fatal(err)
	}

	// 回答適用（u1がu2へダメージ）
	evID, turn, err := repo.ApplyAnswer(ctx, drepo.AnswerApplyArg{
		MID:         mid,
		OppUID:      u2,
		NewOppLP:    7, // 例: 10→7
		NextDeckIdx: 1, // 次へ進める
		NowMs:       now + 100,
		EventFields: map[string]string{
			"uid":        u1,
			"deck_idx":   "1",
			"correct":    "1",
			"dmg":        "3",
			"elapsed_ms": "1200",
		},
	})
	if err != nil {
		t.Fatalf("ApplyAnswer err: %v", err)
	}
	if evID == "" || turn != 1 {
		t.Fatalf("unexpected result evID=%q turn=%d", evID, turn)
	}

	// state が更新されているか（turn, deck_idx, last_event_id）
	state, err := rdb.HGetAll(ctx, "match:"+mid+":state").Result()
	if err != nil {
		t.Fatal(err)
	}
	if state["turn"] != "1" {
		t.Fatalf("turn want=1 got=%s", state["turn"])
	}
	if state["deck_idx"] != "1" {
		t.Fatalf("deck_idx want=1 got=%s", state["deck_idx"])
	}
	if state["last_event_id"] == "" {
		t.Fatalf("last_event_id empty")
	}
	if lp := state["p"+u2+":lp"]; lp != "7" {
		t.Fatalf("opponent lp want=7 got=%s", lp)
	}

	// events に XADD されているか
	entries, err := rdb.XRangeN(ctx, "match:"+mid+":events", "-", "+", 10).Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("events want=1 got=%d", len(entries))
	}
	if entries[0].Values["turn"] != strconv.FormatInt(turn, 10) {
		t.Fatalf("event turn mismatch")
	}
}
