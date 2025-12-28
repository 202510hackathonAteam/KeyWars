package app

import (
	"os"
	"net/http"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"keywars/backend/internal/config"
	httpmiddleware "keywars/backend/internal/transport/http/middleware"
)

// NewEcho は、Echo インスタンスを作成し、共通ミドルウェア・CSRF・CORS などの
// アプリケーション共通設定を適用して返す関数。
func NewEcho(logger *zerolog.Logger) *echo.Echo {
	cookieConfig := config.LoadCookieConfig()
	// Echo 本体の初期化と共通ミドルウェア設定
	e := echo.New()
	e.HideBanner = true
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.RequestID())
	e.Use(echomiddleware.CSRFWithConfig(echomiddleware.CSRFConfig{
		CookieName: "csrf_token",
		CookiePath: "/",
		CookieDomain:   cookieConfig.Domain,        
		CookieSameSite: cookieConfig.SameSite,      
		CookieSecure:   cookieConfig.Secure,   
		CookieHTTPOnly: false,
		TokenLookup: "header:X-CSRF-Token",
	}))

	e.Use(httpmiddleware.RequestLogger(logger))

	// Cookieの初期化
	config.LoadCookieConfig()

	// CORS 設定（環境変数で許可オリジンを指定可能）
	if origin := os.Getenv("CORS_ALLOWED_ORIGIN"); origin != "" {
		e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
			AllowOrigins:     []string{origin},
			AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
			AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
			AllowCredentials: true,
		}))
	}

	return e
}