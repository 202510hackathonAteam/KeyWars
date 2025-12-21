package di

import (
	"go.uber.org/fx"

	"keywars/backend/internal/config"
	"keywars/backend/internal/app"
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

var Module = fx.Options(
	config.Module,
	db.Module,
	redisx.Module,

	// SQL repos
	sqlrepository.Module,

	// Redis
	redisrepository.Module,

	// JWT
	auth.Module,

	// Service
	service.Module,
	round.Module,
	realtime.Module,

	// middleware / validator
	httpmiddleware.Module,
	validator.Module,

	// handler
	handler.Module,

	// websocket
	ws.Module,

	app.Module,
)