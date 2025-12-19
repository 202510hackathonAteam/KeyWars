package port

import (
	"context"
)

type Broadcaster interface {
	Join(room string, c ClientConn) error
	Leave(c ClientConn) error
	Broadcast(ctx context.Context, room string, v any) (failed int, err error)
}

type ClientConn interface {
	SendJSON(ctx context.Context, v any) error
	Close() error
	UID() string
	Room() string
}

type RealtimeService interface {
	// 接続直後に実行（部屋参加や初期メッセージ返却など）
	OnConnect(ctx context.Context, uid, room string) (any, error)
	// 対象ユーザーの生存確認
	OnHeartbeat(ctx context.Context, userID string) error
	// クライアント→サーバのアプリケーションメッセージ処理
	OnMessage(ctx context.Context, uid string, typ string, payload []byte) (maybeReply any, err error)
	// 切断時の後片付け
	OnDisconnect(ctx context.Context, uid string)
}
