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
	MatchExpiryOnStart  time.Duration = 1 * time.Hour
	MatchExpiryOnFinish time.Duration = 10 * time.Minute
)