package app

import (
	"keywars/backend/internal/infra/auth"
	"keywars/backend/internal/service/realtime"
	ws "keywars/backend/internal/transport/websocket"
)

// NewWebSocketHandler は、WebSocket ハンドラーの生成。
func NewWebSocketHandler(
	hub *ws.Hub,
	realtimeService *realtime.MatchRealtimeService,
	jwtHandler *auth.JWTHandler,
) *ws.Handler {
	return &ws.Handler{
		Hub: hub,
		Service: realtimeService,
		TokenAuth: *jwtHandler,
	}
}