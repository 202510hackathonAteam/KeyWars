package app

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(
		// logger
		NewLogger,

		// echo
		NewEcho,

		// websocket
		NewWebSocketHandler,

		NewServer,
	),
	fx.Invoke(
		SetupRouter,
		StartServer,
		ConnectForceFinishServices,
	),
)