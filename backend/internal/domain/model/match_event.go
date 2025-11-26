package model

// MatchFinishEvents は、1 プレイヤー分のラウンド終了情報（FinishEvent）を表す構造体。
type MatchFinishEvents struct {
	PlayerID string
	Round int64
	MissCount int64
	FinishAtMs int64
}