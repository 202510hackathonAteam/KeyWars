package app

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"

	"keywars/backend/internal/config"
	"keywars/backend/internal/infra/auth"
	"keywars/backend/internal/infra/db"
	redisx "keywars/backend/internal/infra/redis"
	redisrepository "keywars/backend/internal/infra/repository/redis"
	sqlrepository "keywars/backend/internal/infra/repository/sql"
	"keywars/backend/internal/service"
	"keywars/backend/internal/service/realtime"
	"keywars/backend/internal/service/round"
	"keywars/backend/internal/transport/http/handler"
	httpmiddleware "keywars/backend/internal/transport/http/middleware"
	ws "keywars/backend/internal/transport/websocket"
	"keywars/backend/internal/util/validator"
)

// Server は、アプリケーション全体の依存関係と Echo インスタンスを保持する構造体の定義。
type Server struct {
	Echo             *echo.Echo
	API              *handler.API
	AuthMiddleware   echo.MiddlewareFunc
	WebSocketHandler *ws.Handler
	RealtimeCancel   context.CancelFunc
}

// New は、アプリケーションサーバーを初期化して Server を生成。
// DB 接続、リポジトリ・サービス・ハンドラの依存注入、ルータ設定をまとめて実施。
func New(cfg *config.Config) (*Server, error) {
	// Echo 本体の初期化と共通ミドルウェア設定
	e := echo.New()
	e.HideBanner = true
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.RequestID())
	e.Use(echomiddleware.CSRFWithConfig(echomiddleware.CSRFConfig{
		CookieName: "csrf_token",
		CookiePath: "/",
		CookieHTTPOnly: false,
		TokenLookup: "header:X-CSRF-Token",
	}))

	baseLogger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	e.Use(httpmiddleware.RequestLogger(&baseLogger))

	// DB接続の初期化
	gormDB, err := db.New(cfg.DB)
	if err != nil {
		return nil, err
	}

	// Redisの初期化
	rdb, err := redisx.NewRedis(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB) // 例: "redis:6379", "", 0
	if err != nil {
		return nil, err
	}

	// Repository 層の初期化
	sqlrepos := sqlrepository.Repos{
		User:   sqlrepository.NewUserRepo(gormDB),
		Prompt: sqlrepository.NewPromptRepositorySQL(gormDB),
		// 下に追加していく
	}

	// Redis Repos
	redisRepos := redisrepository.New(rdb)

	// 認証機能の初期化
	jwtConfig := config.LoadJWTConfig()
	jwtHandler := auth.NewJWTHandler(jwtConfig)

	// 認証ミドルウェアの設定
	authMiddleware := httpmiddleware.NewAuthenticationMiddleware(jwtHandler)

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

	// Service 層の初期化
	services := service.Services{
		Auth:  service.NewAuthService(sqlrepos.User, jwtHandler),
		// 下に追加していく
	}

	// Validator 初期化
	validate := validator.InitValidator()

	// ハンドラ群とルータの設定
	api := handler.New(services, validate, jwtHandler)

	// WebSocket Hub / Handler / Realtime Service
	hub := ws.NewHub()

	nextRoundService := round.NewNextRoundService(redisRepos.Round)
	lifepointService := round.NewLifepointService(redisRepos.Round)
	matchJudgeService := round.NewMatchJudgeService(redisRepos.Round)
	roundFlowService := round.NewRoundFlowService(
		redisRepos.Round,
		hub,
		nextRoundService,
		nil,
		lifepointService,
		matchJudgeService,
		redisRepos.Presence,
	)

	measurementService := round.NewMeasurementRoundService(redisRepos.Round)
	forceFinishService := round.NewForceFinishService(
		redisRepos.Round,
		&baseLogger,
		roundFlowService,
		measurementService,
	)

	timeoutRoundService := round.NewTimeoutRoundService(
		redisRepos.Round,
		forceFinishService,
	)
	deckGeneratorService := round.NewDeckGeneratorService(
		redisRepos.Round,
		sqlrepos.Prompt,
	)

	// Realtime Service を生成（Redis実装とHubを注入）
	realtimeService := realtime.NewMatchRealtimeService(
		redisRepos.Queue,
		redisRepos.Round,
		redisRepos.Presence,
		hub,
		&baseLogger,
		deckGeneratorService,
		roundFlowService,
		measurementService,
	)

	roundFlowService.SetTimeoutService(timeoutRoundService)

	// WebSocket Handler を生成（Service には realtimeService を渡す）
	webSocketHandler := &ws.Handler{
		Hub:       hub,
		Service:   realtimeService,
		Presence:  redisRepos.Presence,
		TokenAuth: *jwtHandler,
	}

	// matchmaker 起動（0.5s間隔など好みで）
	realtimeContext, realtimeCancel := context.WithCancel(context.Background())
	realtimeService.StartMatchmaker(realtimeContext, 500*time.Millisecond)

	return &Server{
		Echo:             e,
		API:              api,
		AuthMiddleware:   authMiddleware,
		WebSocketHandler: webSocketHandler,
		RealtimeCancel:   realtimeCancel,
	}, nil
}
