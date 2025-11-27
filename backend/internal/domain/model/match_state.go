package model

// MatchState は、試合中のプレイヤー状態（デッキ位置、ラウンド数、LPなど）を表す構造体。
type MatchState struct {
	DeckIndex int64
	Round int64
	RoundStartAtMs int64
	RoundEndAtMs int64
	Player1Lifepoint int64
	Player2Lifepoint int64
	Player1TotalMissCount int64
	Player2TotalMissCount int64
}