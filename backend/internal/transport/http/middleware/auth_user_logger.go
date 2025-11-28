package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

// AuthUserContextLogger は 認証済みユーザーIDをロガーに統合するミドルウェア。
func AuthUserContextLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			request := c.Request()
			requestCtx := request.Context()

			// user_id がある場合 logger に付与
			if value := requestCtx.Value(CtxUserID); value != nil {
				if userID, ok := value.(string); ok && userID != ""	{
					logger := zerolog.Ctx(requestCtx).With().
						Str("user_id", userID).
						Logger()
					requestCtxWithLogger := logger.WithContext(requestCtx)
					c.SetRequest(request.WithContext(requestCtxWithLogger))
				}
			}

			return next(c)
		}
	}
}