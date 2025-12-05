package redisx

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(
		// config
		ExtractRedisConfig,

		// clients
		NewRedis,
	),
)