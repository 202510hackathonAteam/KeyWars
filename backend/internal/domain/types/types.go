package types

// ======== ID 系 ========
type (
	PlayerID string
	RoomID string
	MatchID string
)

// ======== 数値系（ゲーム状態） ========
type (
	RoundNumber int
	LifePoint int
	TimeMs int64
)

// ======== カウント系（人数など） ========
type (
	MissCount int
	PlayerCount int
)