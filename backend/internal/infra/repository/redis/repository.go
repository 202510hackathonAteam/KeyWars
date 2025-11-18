package redisrepo

import (
	"keywars/backend/internal/domain/repository"
	"time"

	"github.com/redis/go-redis/v9"
)

// Repos は domain 層のリポジトリIFを Redis で実装した構造体をまとめた集約。
type Repos struct {
	Queue    repository.MatchQueueRepository
	Round    repository.RoundStateRepository
	Presence repository.PresenceRepository
	// 他のRedis系リポをここに追加
}

// New は *redis.Client を受け取り、Redis実装のReposを生成。
func New(rdb *redis.Client) *Repos {
	return &Repos{
		Queue:    NewMatchQueueRepositoryRedis(rdb),
		Round:    NewRoundStateRepositoryRedis(rdb),
		Presence: NewPresenceRepositoryRedis(rdb, 30*time.Second),
	}
}
