package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"keywars/backend/internal/transport/http/handler"
)

func SetupRouter(e *echo.Echo, api *handler.API, authMiddleware echo.MiddlewareFunc) *echo.Echo {
	// 公開：ヘルスチェック
	e.GET("/healthz", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	// デバッグ用：認証なしの疎通確認
	e.GET("/hello", api.Auth.Hello)

	// 本番: 認証が必要な /api/v1 配下
	v1 := e.Group("/api/v1", authMiddleware)
	v1.GET("/hello", api.Auth.Hello)

	return e
}
