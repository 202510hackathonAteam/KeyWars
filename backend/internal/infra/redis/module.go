package redisx

import "go.uber.org/fx"

var Module = fx.Module(
	"redisx",
	fx.Provide(
		// config
		ExtractRedisConfig,

		// clients
		NewRedis,
	),
)