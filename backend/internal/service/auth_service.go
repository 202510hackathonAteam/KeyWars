package service

import (
	"keywars/backend/internal/domain/repository"
)

type AuthService interface {
	// 将来的には Login, Register などを追加していく
}

// authService は AuthService インターフェースの具象実装。
type authService struct{}

// NewAuthService は AuthService のコンストラクタ。
func NewAuthService(users repository.UserRepository) AuthService {
	return &authService{}
}