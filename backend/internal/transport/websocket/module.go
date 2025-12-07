package websocket

import "go.uber.org/fx"

var Module = fx.Module(
	"websocket",
	fx.Provide(
		// websocket
		NewHub,
	),
)