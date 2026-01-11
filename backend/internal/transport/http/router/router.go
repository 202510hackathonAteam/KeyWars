package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"

	"keywars/backend/internal/transport/http/handler"
	httpmiddleware "keywars/backend/internal/transport/http/middleware"
	"keywars/backend/internal/transport/websocket"
)

// Handlers は、ルーターで利用する HTTP ハンドラー群を Fx から受け取るための依存セット。
type Handlers struct {
	fx.In
	Auth *handler.AuthHandler
	Match *handler.MatchHandler
}

// SetupRouter は、アプリケーションのルーティング定義。
func SetupRouter(e *echo.Echo, handlers Handlers, authMiddleware echo.MiddlewareFunc, wsHandler *websocket.Handler) *echo.Echo {
	// 公開：ヘルスチェック
	e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// ──────────────────────────────
	// 認証不要のAPI
	// ──────────────────────────────
	e.POST("/auth/signin", handlers.Auth.Signin)
	e.POST("/auth/signup", handlers.Auth.Signup)
	e.POST("/auth/refresh", handlers.Auth.Refresh)
	e.POST("/auth/check", handlers.Auth.Check)

	// 開発用: 認証なしWS（token=... はWS側で検証）
	// e.GET("/ws", echo.WrapHandler(wsHandler))

	// ──────────────────────────────
	// 認証必須のAPIグループ (/api/v1)
	// ──────────────────────────────
	v1 := e.Group("/api/v1", authMiddleware)
	v1.Use(httpmiddleware.AuthUserContextLogger())
	v1.POST("/auth/signout", handlers.Auth.Signout)
	v1.GET("/match/state", handlers.Match.FrontendState)

	// --- マッチングAPI ---
	// 認証付きws
	e.GET("/api/v1/ws", echo.WrapHandler(wsHandler))

	return e
}
