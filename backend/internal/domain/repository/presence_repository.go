package repository

import "context"

// PresenceRepository は、ユーザーの接続状態（Presence）を管理するリポジトリインターフェース。
// Redisキー: user:{userID}:presence（HASH + TTL 30s）
type PresenceRepository interface {
	//   心拍。updated_at=now を更新し、PEXPIRE 30s（延長）のみ行う。
	Heartbeat(ctx context.Context, userID string, currentTimeMs int64) error

	//   対戦開始。status=ingame, match_id=matchID, updated_at=now, PEXPIRE 30s。
	SetIngame(ctx context.Context, userID string, matchID string, currentTimeMs int64) error

	//   切断。socket_count を-1し、updated_at=now, PEXPIRE 30s。
	//   socket_count<=0 のときは status=offline に落とし、socket_count=0 へ補正。
	Disconnect(ctx context.Context, userID string, currentTimeMs int64) error

	//   現在のpresence（HASH）を map[string]string で返す（キーが無ければ空）。
	Get(ctx context.Context, userID string) (map[string]string, error)
}
