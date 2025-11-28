// internal/infra/redis/ping_test.go
package redisx

import (
	"context"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisPing_Miniredis(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis start failed: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(), // 例: "127.0.0.1:49213"
		DB:   0,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("redis ping failed: %v", err)
	}
}
