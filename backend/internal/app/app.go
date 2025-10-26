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
	sqlrepository "keywars/backend/internal/infra/repository/sql"
	"keywars/backend/internal/transport/http/handler"
	httpmiddleware "keywars/backend/internal/transport/http/middleware"
	"keywars/backend/internal/transport/http/router"
	"keywars/backend/internal/service"
)

type Server struct {
	Echo *echo.Echo
}

func New(config *config.Config) (*Server, error) {
	e := echo.New()
	e.HideBanner = true
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.RequestID())

	gormDB, err := db.New(config.DB)
	if err != nil {
		return nil, err
	}

	repo := sqlrepository.New(gormDB)

	jwtHandler := auth.NewJWTHandler(auth.JWTConfig{
		IssuerName: "keywars",
		HMACSecretKey: []byte(os.Getenv("JWT_SECRET")),
		AccessTokenTTL: 24 * time.Hour,
	})

	authMiddleware := httpmiddleware.NewAuthenticationMiddleware(jwtHandler)

	if origin := os.Getenv("CORS_ALLOWED_ORIGIN"); origin != "" {
		e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
			AllowOrigins: []string{origin},
			AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
			AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
			AllowCredentials: true,
		}))
	}

	authService := service.NewAuthService(repo.User)

	api := handler.New(authService)
	router.SetupRouter(e, api, authMiddleware)

	return &Server{Echo: e}, nil
}
