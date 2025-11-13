package middleware

import (
	"net/http"
	"context"

	"github.com/labstack/echo/v4"
	"keywars/backend/internal/infra/auth"
)

// NewAuthenticationMiddleware は、JWT ベースの認証を行うミドルウェアの生成。
// Cookie 内の access_token（JWT）を検証し、有効な場合は userID をコンテキストへ設定する。
func NewAuthenticationMiddleware(jwtHandler *auth.JWTHandler) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Cookie からアクセストークンを取得
			accessTokenCookie, err := c.Cookie("access_token")
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing access token")
			}
			accessToken := accessTokenCookie.Value
			if accessToken == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid access token")
			}

			// JWT の検証処理
			userID, err := jwtHandler.VerifyAccessToken(accessToken)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			// 検証成功時：userID をコンテキストへ設定
			requestCtx := context.WithValue(c.Request().Context(), CtxUserID, userID)
			c.SetRequest(c.Request().WithContext(requestCtx))
			c.Set("userID", userID)

			return next(c)
		}
	}
}
