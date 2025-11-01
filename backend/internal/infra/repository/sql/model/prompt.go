package model

import (
	"time"
)

// Prompt はタイピングゲームにおける「お題」を表すモデル。
// 各お題は特定の難易度（Difficulty）に属する。
type Prompt struct {
	ID int `gorm:"column:id;primaryKey;autoIncrement;not null"`
	DifficultyID int `gorm:"column:difficulty_id;type:int;not null"`
	PromptTextJa string `gorm:"column:prompt_text_ja;type:VARCHAR(255);not null"`
	TargetRomaji string `gorm:"column:target_romaji;type:VARCHAR(255);not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`

	Difficulty Difficulty `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}

// TableName は GORM に明示的にテーブル名を指定するためのメソッド。
func (Prompt) TableName() string {
	return "prompts"
}