package initial

import (
	"gorm.io/gorm"
)

// difficultyRow は、初期データ投入専用のDBモデル。
type difficultyRow struct {
	ID int `gorm:"column:id;primaryKey"`
	DifficultyCode string `gorm:"column:difficulty_code"`
	TimeLimitMs int `gorm:"column:time_limit_ms"`
}

// TableName は、GORM に使用させるテーブル名を明示的に指定。
func (difficultyRow) TableName() string {
	return "difficulties"
}

// loadDifficulties は、難易度マスターデータを初期投入する関数。
func loadDifficulties(db *gorm.DB) error {
	// 登録する初期データ一覧
	difficulties := []difficultyRow{
		{
			ID: 1,
			DifficultyCode: "easy",
			TimeLimitMs: 6000,
		},
		{
			ID: 2,
			DifficultyCode: "normal",
			TimeLimitMs: 9000,
		},
		{
			ID: 3,
			DifficultyCode: "hard",
			TimeLimitMs: 11000,
		},
	}

	// 各難易度データをDBへ登録（存在しない場合のみ作成）
	for _, difficulty := range difficulties {
		if err := db.FirstOrCreate(&difficulty).Error; err != nil {
			return err
		}
	}

	return nil
}