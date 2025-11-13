package seed

import (
	"gorm.io/gorm"
)

// SeedDevelopmentData は、開発環境やテスト環境向けに、ダミーデータを投入する関数。
func SeedDevelopmentData(db *gorm.DB) error {
	return db.Transaction(func(transaction *gorm.DB) error {
		// ユーザーデータの投入
		if err := seedUsers(transaction); err != nil {
			return err
		}

		return nil
	})
}