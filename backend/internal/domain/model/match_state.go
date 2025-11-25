package model

// MatchState は、試合中のプレイヤー状態（デッキ位置、ターン、LPなど）を表す構造体。
type MatchState struct {
	DeckIndex int64
	Round int64
	Player1AnswerFinished bool
	Player2AnswerFinished bool
	Player1Lifepoint int64
	Player2Lifepoint int64
}