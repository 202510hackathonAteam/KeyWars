package model

import (
  "time"
  "github.com/google/uuid"
  "gorm.io/gorm"
)

// User はアプリケーションに登録されたユーザーを表すモデル。
type User struct {
	ID string `gorm:"column:id;type:char(36);primaryKey"`
	UserName string `gorm:"column:user_name;type:varchar(255);not null;uniqueIndex:ux_users_user_name"`
	PasswordHash string `gorm:"column:password_hash;type:varchar(255);not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
}

// TableName は GORM に明示的にテーブル名を指定するためのメソッド。
func (User) TableName() string {
	return "users"
}

// BeforeCreate はレコード作成前に呼ばれる GORM のフック。
// ID が未設定の場合、自動的に UUID を割り当てる。
func (user *User) BeforeCreate(transaction *gorm.DB) error {
  if user.ID == "" {
    user.ID = uuid.NewString()
  }
  return nil
}
