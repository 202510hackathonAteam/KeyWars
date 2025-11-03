package router

import (
	"net/http"

	"keywars/backend/internal/transport/http/handler"
	"keywars/backend/internal/transport/websocket"

	"github.com/labstack/echo/v4"
)

// SetupRouter は、アプリケーションのルーティング定義。
func SetupRouter(e *echo.Echo, api *handler.API, authMiddleware echo.MiddlewareFunc, wsHandler *websocket.Handler) *echo.Echo {
	// 公開：ヘルスチェック
	e.GET("/healthz", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	// ──────────────────────────────
	// 認証不要のAPI
	// ──────────────────────────────
	// デバッグ用（handler.Auth.Hello が削除されたら、このエンドポイントも削除する）
	e.GET("/hello", api.Auth.Hello)

	// 開発用: 認証なしWS（token=... はWS側で検証）
	e.GET("/ws", func(c echo.Context) error {
		wsHandler.ServeHTTP(c.Response(), c.Request())
		return nil
	})

	// ──────────────────────────────
	// 認証必須のAPIグループ (/api/v1)
	// ──────────────────────────────
	v1 := e.Group("/api/v1", authMiddleware)
	// デバッグ用（handler.Auth.Hello が削除されたら、このエンドポイントも削除する）
	v1.GET("/hello", api.Auth.Hello)

	// 認証付きws
	// v1.GET("/ws", func(c echo.Context) error {
	// 	wsHandler.ServeHTTP(c.Response(), c.Request())
	// 	return nil
	// })

	return e
}
