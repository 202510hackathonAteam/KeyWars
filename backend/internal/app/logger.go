package app

import (
	"os"

	"github.com/rs/zerolog"
)

// NewLogger は、標準出力に書き込む Zerolog ロガーを生成。
func NewLogger() *zerolog.Logger {
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger()
	return &logger
}