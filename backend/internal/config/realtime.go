package config

import "time"

// ===== 接続・リアルタイム制御 =====
const (
	PongWait    time.Duration = 30 * time.Second
	PresenceTTL time.Duration = PongWait + 5 * time.Second
)