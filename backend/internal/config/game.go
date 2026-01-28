package config

import (
	"time"

	"keywars/backend/internal/domain/types"
)

// ===== ゲーム基本設定 =====
const (
	RequiredPlayers  types.PlayerCount = 2
	InitialRound     int64 = 1
	InitialDeckIndex int64 = 0
)

// ===== ゲーム数値設定 =====
const (
	InitialLifePoint int64 = 150
	GraceMs          int64 = 2000
)

// ===== マッチ有効期限 =====
const (
	MatchExpiryOnStart  time.Duration = 15 * time.Minute
	MatchExpiryOnFinish time.Duration = 10 * time.Minute
)

// ===== デッキ設定 =====

type DeckConfig struct {
	TotalCount int
	EasyCount   int
	NormalCount int
	HardCount   int
}

// Deck はゲームの問題出題設定
var Deck = DeckConfig{
	TotalCount: 20,
	EasyCount:   6,
	NormalCount: 6,
	HardCount:   8,
}