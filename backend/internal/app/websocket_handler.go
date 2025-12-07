package app

import (
	"keywars/backend/internal/infra/auth"
	redisrepository "keywars/backend/internal/domain/repository"
	"keywars/backend/internal/service/realtime"
	ws "keywars/backend/internal/transport/websocket"
)

// NewWebSocketHandler は、WebSocket ハンドラーの生成。
func NewWebSocketHandler(
	hub *ws.Hub,
	presenceRepo redisrepository.PresenceRepository,
	realtimeService *realtime.MatchRealtimeService,
	jwtHandler *auth.JWTHandler,
) *ws.Handler {
	return &ws.Handler{
		Hub: hub,
		Service: realtimeService,
		Presence: presenceRepo,
		TokenAuth: *jwtHandler,
	}
}