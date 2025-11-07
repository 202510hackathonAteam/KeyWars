package repository

import "context"

// PresenceRepository は、ユーザーの接続状態（Presence）を管理するリポジトリインターフェース。
// このインターフェースは、ユーザーのオンライン／オフライン状態、ゲーム中ステータス、
// ハートビート（生存信号）更新などの操作を定義する。
//
// 想定される Redis 構造例:
//   - presence:{userID} : HASH
//     ├─ status: "online" | "ingame" | "offline"
//     ├─ match_id: "m123"（ingame時のみ）
//     ├─ last_seen_ms: UNIXミリ秒
//     └─ updated_at_ms: UNIXミリ秒
type PresenceRepository interface {
	// SetOnline は、ユーザーを「オンライン」状態に設定する。
	// ユーザーがログインまたは接続確立したタイミングで呼び出される。
	//
	// Redis想定操作:
	//   HSET presence:{userID} status "online" last_seen_ms <currentTimeMs>
	//
	// 引数:
	//   contextObject   - コンテキスト（キャンセルやタイムアウト制御に使用）
	//   userID          - 対象のユーザーID
	//   currentTimeMs   - 状態更新時のUNIXミリ秒
	//
	// 戻り値:
	//   error - 成功時は nil、失敗時はエラーを返す
	SetOnline(contextObject context.Context, userID string, currentTimeMs int64) error

	// Heartbeat は、ユーザーがまだ接続中であることを示す「生存信号」を送る。
	// 一定間隔で呼び出され、last_seen_ms を更新する。
	//
	// Redis想定操作:
	//   HSET presence:{userID} last_seen_ms <currentTimeMs>
	//
	// 引数:
	//   contextObject   - コンテキスト
	//   userID          - 対象のユーザーID
	//   currentTimeMs   - 現在のUNIXミリ秒
	//
	// 戻り値:
	//   error - 成功時は nil、失敗時はエラーを返す
	Heartbeat(contextObject context.Context, userID string, currentTimeMs int64) error

	// SetIngame は、ユーザーを「対戦中（ingame）」状態に設定する。
	// 対戦中のマッチIDと時刻を記録し、presence 情報を更新する。
	//
	// Redis想定操作:
	//   HSET presence:{userID} status "ingame" match_id <matchID> updated_at_ms <currentTimeMs>
	//
	// 引数:
	//   contextObject   - コンテキスト
	//   userID          - 対象のユーザーID
	//   matchID         - 現在のマッチID
	//   currentTimeMs   - 状態変更時のUNIXミリ秒
	//
	// 戻り値:
	//   error - 成功時は nil、失敗時はエラーを返す
	SetIngame(contextObject context.Context, userID string, matchID string, currentTimeMs int64) error

	// Disconnect は、ユーザーを「オフライン」状態に設定する。
	// 通信断・ログアウトなどにより接続が切れた際に呼び出される。
	//
	// Redis想定操作:
	//   HSET presence:{userID} status "offline" last_seen_ms <currentTimeMs>
	//
	// 引数:
	//   contextObject   - コンテキスト
	//   userID          - 対象のユーザーID
	//   currentTimeMs   - 切断検知時のUNIXミリ秒
	//
	// 戻り値:
	//   error - 成功時は nil、失敗時はエラーを返す
	Disconnect(contextObject context.Context, userID string, currentTimeMs int64) error

	// Get は、指定されたユーザーのプレゼンス情報を取得する。
	// Redis HASH を map[string]string として返す。
	//
	// Redis想定操作:
	//   HGETALL presence:{userID}
	//
	// 引数:
	//   contextObject - コンテキスト
	//   userID        - 対象のユーザーID
	//
	// 戻り値:
	//   map[string]string - 現在の presence 情報（status, match_id, last_seen_ms など）
	//   error             - 取得エラー
	Get(contextObject context.Context, userID string) (map[string]string, error)
}
