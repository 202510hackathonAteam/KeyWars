package config

import "time"

const (
	InitialDeckIndex int64 = 0
	InitialRound int64 = 1
	InitialLifePoint int64 = 100
	GraceMs int64 = 2000
	RequiredPlayers int64 = 2
	MatchExpiryOnStart time.Duration = 1 * time.Hour
	MatchExpiryOnFinish time.Duration = 10 * time.Minute
)