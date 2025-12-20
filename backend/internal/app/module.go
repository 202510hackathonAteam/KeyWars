package app

import "go.uber.org/fx"

var Module = fx.Module(
	"app",
	fx.Provide(
		// logger
		NewLogger,

		// echo
		NewEcho,

		NewServer,
	),
	fx.Invoke(
		SetupRouter,
		StartServer,
		ConnectForceFinishServices,
	),
)