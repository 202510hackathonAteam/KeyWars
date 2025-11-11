package repository

import "context"

// PromptWithDifficulty は、JOIN 結果を受けるための DTO。
// gorm:"column:..." を付けて列名と確実に一致させます。
// json タグは、フロントにそのまま渡す場合にも便利です。
type PromptWithDifficulty struct {
	ID              int    `gorm:"column:id"               json:"id"`
	PromptTextJa    string `gorm:"column:prompt_text_ja"   json:"prompt_text_ja"`
	TargetRomaji    string `gorm:"column:target_romaji"    json:"target_romaji"`
	DifficultyLevel int    `gorm:"column:difficulty_level" json:"difficulty_level"`
	TimeLimitMs     int    `gorm:"column:time_limit_ms"    json:"time_limit_ms"`
}

// PromptRepository は、デッキ取得用のインターフェース。
// 難易度別取得が外でも必要なら、GetPromptsByDifficulty もここに追加してOK。
type PromptRepository interface {
	GetDeckPrompts(ctx context.Context) ([]PromptWithDifficulty, error)
	// もし使うなら↓も追加
	// GetPromptsByDifficulty(ctx context.Context, difficultyCode string, limit int) ([]PromptWithDifficulty, error)
}
