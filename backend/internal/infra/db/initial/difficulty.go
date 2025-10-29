package initial

import (
	"gorm.io/gorm"
	"keywars/backend/internal/domain/entity"
)

// loadDifficulties は、難易度マスターデータを初期投入する関数。
func loadDifficulties(db *gorm.DB) error {
	// 登録する初期データ一覧
	difficulties := []entity.Difficulty{
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