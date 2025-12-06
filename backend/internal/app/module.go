package app

import "go.uber.org/fx"

var Module = fx.Module(
	"app",
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