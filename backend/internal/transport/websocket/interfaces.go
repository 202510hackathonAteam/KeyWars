package websocket

import (
	"context"
)

type ClientConn interface {
	WriteJSON(v any) error
	Close() error
}

type MatchRealtimeService interface {
	// 接続直後に実行（部屋参加や初期メッセージ返却など）
	OnConnect(ctx context.Context, userID string) (any, error)
	// クライアント→サーバのアプリケーションメッセージ処理
	OnMessage(ctx context.Context, userID string, typ string, payload []byte) (maybeReply any, err error)
	// 切断時の後片付け
	OnDisconnect(ctx context.Context, userID string)
}
