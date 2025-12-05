package app

import (
	"go.uber.org/fx"

	appRouter "keywars/backend/internal/transport/http/router"
)

// RouterParams は、ルーター初期化時に Fx から注入される依存関係の集合。
type RouterParams struct {
	fx.In
	Handlers appRouter.Handlers
}

// SetupRouter は、ルーター設定。
func SetupRouter(server *Server, routerParams RouterParams) {
	e := server.Echo
	appRouter.SetupRouter(e, routerParams.Handlers, server.AuthMiddleware, server.WebSocketHandler)
}