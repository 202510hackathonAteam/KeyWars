package app

import (
	"context"

	"github.com/labstack/echo/v4"

	ws "keywars/backend/internal/transport/websocket"
)

// Server は、Echo インスタンスと各種ハンドラをまとめたアプリケーションサーバーの構造体。
type Server struct {
	Echo             *echo.Echo
	AuthMiddleware   echo.MiddlewareFunc
	WebSocketHandler *ws.Handler
	MatchMakerCancel context.CancelFunc
}

// NewServer は、渡されたコンポーネントを束ねて Server を構築するコンストラクタ。
func NewServer(
	e *echo.Echo,
	authMiddleware echo.MiddlewareFunc,
	webSocketHandler *ws.Handler,
) (*Server, error) {
	return &Server{
		Echo:             e,
		AuthMiddleware:   authMiddleware,
		WebSocketHandler: webSocketHandler,
		MatchMakerCancel: nil,
	}, nil
}
