package router

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"keywars/backend/internal/transport/http/handler"
	httpmiddleware "keywars/backend/internal/transport/http/middleware"
	"keywars/backend/internal/transport/websocket"
)

// SetupRouter は、アプリケーションのルーティング定義。
func SetupRouter(e *echo.Echo, api *handler.API, authMiddleware echo.MiddlewareFunc, wsHandler *websocket.Handler) *echo.Echo {
	// 公開：ヘルスチェック
	e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// ──────────────────────────────
	// 認証不要のAPI
	// ──────────────────────────────
	e.POST("/auth/signin", api.Auth.Signin)
	e.POST("/auth/signup", api.Auth.Signup)
	e.POST("/auth/refresh", api.Auth.Refresh)

	// 開発用: 認証なしWS（token=... はWS側で検証）
	e.GET("/ws", echo.WrapHandler(wsHandler))

	// ──────────────────────────────
	// 認証必須のAPIグループ (/api/v1)
	// ──────────────────────────────
	v1 := e.Group("/api/v1", authMiddleware)
	v1.Use(httpmiddleware.AuthUserContextLogger())
	v1.POST("/auth/signout", api.Auth.Signout)

	// --- マッチングAPI ---
	// 認証付きws
	// v1.GET("/ws", func(c echo.Context) error {
	// 	wsHandler.ServeHTTP(c.Response(), c.Request())
	// 	return nil
	// })

	return e
}
