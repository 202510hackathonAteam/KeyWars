package repository

import "context"

// PresenceRepository は、ユーザーの接続状態（Presence）を管理するリポジトリインターフェース。
// Redisキー: user:{userID}:presence（HASH + TTL 30s）
type PresenceRepository interface {
	Connect(ctx context.Context, userID string, nowUnixMilli int64) error

	//   心拍。updated_at=now を更新し、PEXPIRE 30s（延長）のみ行う。
	Heartbeat(ctx context.Context, userID string, currentTimeMs int64) error

	//   切断。socket_count を-1し、updated_at=now, PEXPIRE 30s。
	Disconnect(ctx context.Context, userID string, currentTimeMs int64) error
}
