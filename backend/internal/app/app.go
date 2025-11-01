package app

import (
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"

	"keywars/backend/internal/config"
	"keywars/backend/internal/infra/auth"
	"keywars/backend/internal/infra/db"
	redisx "keywars/backend/internal/infra/redis"
	redisrepository "keywars/backend/internal/infra/repository/redis"
	sqlrepository "keywars/backend/internal/infra/repository/sql"
	"keywars/backend/internal/service"
	"keywars/backend/internal/transport/http/handler"
	httpmiddleware "keywars/backend/internal/transport/http/middleware"
	"keywars/backend/internal/transport/http/router"
)

// Server は、アプリケーション全体の依存関係と Echo インスタンスを保持する構造体の定義。
type Server struct {
	Echo *echo.Echo
}

// New は、アプリケーションサーバーを初期化して Server を生成。
// DB 接続、リポジトリ・サービス・ハンドラの依存注入、ルータ設定をまとめて実施。
func New(config *config.Config) (*Server, error) {
	// Echo 本体の初期化と共通ミドルウェア設定
	e := echo.New()
	e.HideBanner = true
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.RequestID())

	// DB接続の初期化
	gormDB, err := db.New(config.DB)
	if err != nil {
		return nil, err
	}

	// Redisの初期化
	rdb, err := redisx.NewRedis(config.Redis.Addr, config.Redis.Password, config.Redis.DB) // 例: "redis:6379", "", 0
	if err != nil {
		return nil, err
	}

	// Repository 層の初期化
	sqlrepos := sqlrepository.Repos{
		User: sqlrepository.NewUserRepo(gormDB),
		// 下に追加していく
	}

	// Redis 側（待機キュー / ラウンド状態）
	redisrepos := redisrepository.Repos{
		Queue: redisrepository.NewMatchQueueRepositoryRedis(rdb),
		Round: redisrepository.NewRoundStateRepositoryRedis(rdb),
	}

	// JWT 認証ハンドラの初期化
	jwtHandler := auth.NewJWTHandler(auth.JWTConfig{
		IssuerName:     "keywars",
		HMACSecretKey:  []byte(os.Getenv("JWT_SECRET")),
		AccessTokenTTL: 24 * time.Hour,
	})

	// 認証ミドルウェアの設定
	authMiddleware := httpmiddleware.NewAuthenticationMiddleware(jwtHandler)

	// CORS 設定（環境変数で許可オリジンを指定可能）
	if origin := os.Getenv("CORS_ALLOWED_ORIGIN"); origin != "" {
		e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
			AllowOrigins:     []string{origin},
			AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
			AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
			AllowCredentials: true,
		}))
	}

	// Service 層の初期化
	services := service.Services{
		Auth:  service.NewAuthService(sqlrepos.User),
		Match: service.NewMatchService(redisrepos.Queue, redisrepos.Round),
		Round: service.NewRoundService(redisrepos.Round),
		// 下に追加していく
	}

	// ハンドラ群とルータの設定
	api := handler.New(services)
	router.SetupRouter(e, api, authMiddleware)

	return &Server{Echo: e}, nil
}
