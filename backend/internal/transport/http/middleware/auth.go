package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"keywars/backend/internal/infra/auth"
)

// NewAuthenticationMiddleware は、JWT ベースの認証を行うミドルウェアの生成。
// Authorization ヘッダ内の Bearer トークンを検証し、有効な場合は userID をコンテキストへ設定。
func NewAuthenticationMiddleware(jwtHandler *auth.JWTHandler) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(echoContext echo.Context) error {
			// Authorization ヘッダの取得と形式チェック
			authorizationHeader := echoContext.Request().Header.Get(echo.HeaderAuthorization)
			if authorizationHeader == "" || !strings.HasPrefix(authorizationHeader, "Bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing or malformed Authorization header")
			}

			// Bearer トークン部分の抽出
			tokenString := strings.TrimPrefix(authorizationHeader, "Bearer ")

			// JWT の検証処理
			userID, verifyErr := jwtHandler.VerifyAccessToken(tokenString)
			if verifyErr != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			// 検証成功時：userID をコンテキストへ設定
			echoContext.Set("userID", userID)

			// 次のハンドラへ制御を移譲
			return next(echoContext)
		}
	}
}
