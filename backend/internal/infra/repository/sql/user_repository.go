package sql

import (
	"gorm.io/gorm"
	"keywars/backend/internal/domain/repository"
)

// ダミー実装
type userRepo struct{
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) repository.UserRepository {
	return &userRepo{db: db}
}