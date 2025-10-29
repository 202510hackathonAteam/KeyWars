package entity

import (
	"time"
)

type User struct {
	ID string `json:"id" gorm:"primaryKey;type:char(36);not null"`
	UserName string `json:"user_name" gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string `json:"password_hash" gorm:"type:varchar(255);not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}