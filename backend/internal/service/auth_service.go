package service

import (
	"keywars/backend/internal/domain/repository"
)

// AuthService は、認証関連ユースケースのインターフェースの定義。
type AuthService interface {
	// 将来的には Login, Register などを追加していく
}

// authService は AuthService インターフェースの具象実装。
type authService struct{
	user repository.UserRepository
}

// NewAuthService は UserRepository を受け取り、AuthServiceを生成。
func NewAuthService(user repository.UserRepository) AuthService {
	return &authService{
		user: user,
	}
}