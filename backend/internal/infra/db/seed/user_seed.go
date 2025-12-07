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
			PasswordHash: "$argon2id$v=19$m=65536,t=1,p=12$kp5iwllSZiltdIFbAn8l3w$lP0oFmykA6WIyrzf5xsRaiGQP7b7iETrGRt/kRZMO90",
		},
		{
			ID: "2b3c4d5e-6f7a-4b8c-9d0e-1f2a3b4c5d6e",
			UserName: "testuser2",
			PasswordHash: "$argon2id$v=19$m=65536,t=1,p=12$n4CYCbQzzeMUlP1zFWWe+w$iwVdTLyDYwb4VXJGQxC0LNLbbNjR4wUqD6/46ZlhXzk",
		},
		{
			ID: "3c4d5e6f-7a8b-4c9d-8e1f-2a3b4c5d6e7f",
			UserName: "testuser3",
			PasswordHash: "$argon2id$v=19$m=65536,t=1,p=12$gn8vbsM657bgiEXwIRKpaQ$a5zLXEyDVgdBj3EJJoisjv6oOOCEn3I8xPIMg0/pZRQ",
		},
		{
			ID: "4d5e6f7a-8b9c-4d0e-9f2a-3b4c5d6e7f8a",
			UserName: "testuser4",
			PasswordHash: "$argon2id$v=19$m=65536,t=1,p=12$HrvNaqxagJ23sM6ZnZe3Bg$J8jzw5yicIkjGHKPNl4m0oXlZn9kP8mDpwnsibuWTKM",
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
