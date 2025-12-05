package redis

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(
		// Redis
		NewMatchQueueRepositoryRedis,
		NewPresenceRepositoryRedis,
		NewRoundStateRepositoryRedis,
	),
)