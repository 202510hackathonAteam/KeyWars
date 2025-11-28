package repository

import (
	"context"

	"keywars/backend/internal/domain/model"
)

// PromptRepository は、デッキ取得用のインターフェース。
// 難易度別取得が外でも必要なら、GetPromptsByDifficulty もここに追加してOK。
type PromptRepository interface {
	GetDeckPrompts(ctx context.Context) ([]model.DeckPrompt, error)
}
