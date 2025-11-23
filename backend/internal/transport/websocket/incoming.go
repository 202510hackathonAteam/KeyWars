package websocket

// RoundEndPayload は、ラウンド終了時にクライアントがサーバへ送信する
// WebSocket メッセージのペイロードを表す構造体。
type RoundEndPayload struct {
	MatchID string `json:"match_id"`
	MissCount int64 `json:"miss_count"`
}