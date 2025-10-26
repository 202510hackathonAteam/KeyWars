package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"keywars/backend/internal/transport/http/handler"
)

// SetupRouter は、アプリケーションのルーティング定義。
func SetupRouter(e *echo.Echo, api *handler.API, authMiddleware echo.MiddlewareFunc) *echo.Echo {
	// 公開：ヘルスチェック
	e.GET("/healthz", func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	// ──────────────────────────────
	// 認証不要のAPI
	// ──────────────────────────────
	// デバッグ用（handler.Auth.Hello が削除されたら、このエンドポイントも削除する）
	e.GET("/hello", api.Auth.Hello)

	// ──────────────────────────────
	// 認証必須のAPIグループ (/api/v1)
	// ──────────────────────────────
	v1 := e.Group("/api/v1", authMiddleware)
	// デバッグ用（handler.Auth.Hello が削除されたら、このエンドポイントも削除する）
	v1.GET("/hello", api.Auth.Hello)

	return e
}
