package entity

import (
	"time"
)

type Difficulty struct {
	ID int `json:"id" gorm:"primaryKey;autoIncrement;not null"`
	DifficultyCode string `json:"difficulty_code" gorm:"type:varchar(25);uniqueIndex;not null"`
	TimeLimitMs int `json:"time_limit_ms" gorm:"type:int;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Prompts []Prompt `gorm:"foreignKey:DifficultyID"`
}