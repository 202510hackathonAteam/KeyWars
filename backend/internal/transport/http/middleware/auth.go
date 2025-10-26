package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"keywars/backend/internal/infra/auth"
)

// NewAuthenticationMiddleware は Authorization ヘッダに含まれる Bearer トークンを検証し、
// 有効な場合は userID を Echo コンテキストに設定して次のハンドラに制御を渡すミドルウェアを生成する。
func NewAuthenticationMiddleware(jwtHandler *auth.JWTHandler) echo.MiddlewareFunc {
	// ミドルウェア生成関数（Echo に登録するための関数）を返す
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		// 実際のリクエスト処理時に呼び出されるハンドラを返す
		return func(echoContext echo.Context) error {
			authorizationHeader := echoContext.Request().Header.Get(echo.HeaderAuthorization)
			if authorizationHeader == "" || !strings.HasPrefix(authorizationHeader, "Bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing or malformed Authorization header")
			}

			tokenString := strings.TrimPrefix(authorizationHeader, "Bearer ")
			userID, verifyErr := jwtHandler.VerifyAccessToken(tokenString)
			if verifyErr != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			echoContext.Set("userID", userID)

			return next(echoContext)
		}
	}
}
