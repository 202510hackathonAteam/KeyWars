package cookie

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"keywars/backend/internal/config"
)

// newCookie は、共通設定を用いた Cookie 構造体を生成
func newCookie(name, value string, maxAge time.Duration) *http.Cookie {
	cookieConfig := config.LoadCookieConfig()
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Domain:   cookieConfig.Domain,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   cookieConfig.Secure,
	}
}

// SetAccessToken は、JWTアクセストークンをCookieに設定
func SetAccessToken(
	c echo.Context,
	accessToken string,
	accessTokenExpiry time.Duration,
) {
	accessCookie := newCookie(
		"access_token",
		string(accessToken),
		accessTokenExpiry,
	)
	c.SetCookie(accessCookie)
}

// SetRefreshToken は、JWTリフレッシュトークンをCookieに設定
func SetRefreshToken(
	c echo.Context,
	refreshToken string,
	refreshTokenExpiry time.Duration,
) {
	refreshCookie := newCookie(
		"refresh_token",
		string(refreshToken),
		refreshTokenExpiry,
	)
	c.SetCookie(refreshCookie)
}

// SetTokens は、JWTアクセストークンとリフレッシュトークンをCookieに設定
func SetTokens(
	c echo.Context,
	accessToken,
	refreshToken string,
	accessTokenExpiry,
	refreshTokenExpiry time.Duration,
) {
	SetAccessToken(c, accessToken, accessTokenExpiry)
	SetRefreshToken(c, refreshToken, refreshTokenExpiry)
}

// ClearAccessToken は、アクセストークンのCookie削除
func ClearAccessToken(c echo.Context) {
	cookieConfig := config.LoadCookieConfig()
	c.SetCookie(&http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		Domain:   cookieConfig.Domain,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cookieConfig.Secure,
	})
}

// ClearRefreshToken は、リフレッシュトークンのCookie削除
func ClearRefreshToken(c echo.Context) {
	cookieConfig := config.LoadCookieConfig()
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		Domain:   cookieConfig.Domain,
		SameSite: http.SameSiteNoneMode,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cookieConfig.Secure,
	})
}

// ClearTokens は、アクセストークンとリフレッシュトークンのCookie削除
func ClearTokens(c echo.Context) {
	ClearAccessToken(c)
	ClearRefreshToken(c)
}
