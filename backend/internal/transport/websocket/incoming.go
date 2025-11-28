package websocket

// PlayerAnswerFinishedPayload は、プレイヤーが自分の解答を終了した際に、
// サーバへ送信する WebSocket メッセージのペイロードを表す構造体。
type PlayerAnswerFinishedPayload struct {
	MatchID string `json:"match_id"`
	MissCount int64 `json:"miss_count"`
}