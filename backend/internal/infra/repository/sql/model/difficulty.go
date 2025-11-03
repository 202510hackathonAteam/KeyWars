package model

import (
	"time"
)

// Difficulty はゲーム内の難易度レベルを表すモデル。
type Difficulty struct {
	ID int `gorm:"column:id;primaryKey;autoIncrement;not null"`
	DifficultyCode string `gorm:"column:difficulty_code;type:varchar(25);uniqueIndex;not null"`
	TimeLimitMs int `gorm:"column:time_limit_ms;type:int;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`

	Prompts []Prompt `gorm:"foreignKey:DifficultyID"`
}

// TableName は GORM に明示的にテーブル名を指定するためのメソッド。
func (Difficulty) TableName() string {
	return "difficulties"
}