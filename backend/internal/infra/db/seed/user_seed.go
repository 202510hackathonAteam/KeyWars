package seed

import (
	"gorm.io/gorm"
)

// User はアプリケーションに登録されたユーザーを表すモデル。
type userRow struct {
	ID string `gorm:"column:id;primaryKey"`
	UserName string `gorm:"column:user_name"`
	PasswordHash string `gorm:"column:password_hash"`
}

// TableName は GORM に明示的にテーブル名を指定するためのメソッド。
func (userRow) TableName() string {
	return "users"
}

// seedUsers は、ユーザーのデモデータを初期投入する関数。
func seedUsers(db *gorm.DB) error {
	users := []userRow{
		{
			ID: "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d",
			UserName: "testuser1",
			PasswordHash: "npR6sxACVE3YZSqMVCQlXQ.6Ik22LF3pb72tmCRQAjR7hi0eTTt2do9Waak0pBVWdE",
		},
		{
			ID: "2b3c4d5e-6f7a-4b8c-9d0e-1f2a3b4c5d6e",
			UserName: "testuser2",
			PasswordHash: "/dtD53fg5uK4UeDWKk28JA.6wHReeX2gXRFI47srXspkkJV181th7/ERN4SrGjmf3s",
		},
		{
			ID: "3c4d5e6f-7a8b-4c9d-8e1f-2a3b4c5d6e7f",
			UserName: "testuser3",
			PasswordHash: "Oj1QOYP4CZ8ETtImxkmBJQ.h6YH/Hl10PIRs3GUr4BfjNSKHmATcaq4Z53z+sjQDP4",
		},
		{
			ID: "4d5e6f7a-8b9c-4d0e-9f2a-3b4c5d6e7f8a",
			UserName: "testuser4",
			PasswordHash: "u3GhXPAdrzPQfraNzyF8ZA.5lLUQOkCuN2MHfWdf00qjjptikJjNZTsJMQOpLee1k8",
		},
	}

	// 各ユーザーデータをDBへ登録（存在しない場合のみ作成）
	for _, user := range users {
		if err := db.FirstOrCreate(&user).Error; err != nil {
			return err
		}
	}

	return nil
}
