package entity

import (
	"time"
)

type Prompt struct {
	ID int `json:"id" gorm:"primaryKey;autoIncrement;not null"`
	DifficultyID int `json:"difficulty_id" gorm:"type:int;not null"`
	PromptTextJa string `json:"prompt_text_ja" gorm:"type:VARCHAR(255);not null"`
	TargetRomaji string `json:"target_romaji" gorm:"type:VARCHAR(255);not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Difficulty Difficulty `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
}