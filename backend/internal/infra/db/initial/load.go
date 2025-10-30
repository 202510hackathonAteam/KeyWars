package initial

import (
	"gorm.io/gorm"
)

// LoadInitialData はアプリ起動時に実行される初期データ投入関数。
func LoadInitialData(db *gorm.DB) error {
	return db.Transaction(func(transaction *gorm.DB) error {
		// 難易度データの投入
		if err := loadDifficulties(transaction); err != nil {
			return err
		}

		// お題データの投入
		if err := loadPrompts(transaction); err != nil {
			return err
		}

		return nil
	})
}