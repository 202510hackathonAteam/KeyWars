package sql

import (
	"gorm.io/gorm"
	"keywars/backend/internal/domain/repository"
)

type Repos struct {
	User  repository.UserRepository
}

func New(db *gorm.DB) *Repos {
	return &Repos{
		User:  NewUserRepo(db),
	}
}