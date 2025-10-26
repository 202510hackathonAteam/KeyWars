package service

import (
	"keywars/backend/internal/domain/repository"
)

// Repositories は、service が依存するリポジトリ群の定義。
type Repositories struct {
	User repository.UserRepository
	// 下に他のリポジトリを追加していく
}

// Services は、service 群の集約。
type Services struct {
	Auth AuthService
	// 下に他のサービスを追加していく
}

// NewServices は、Repositories から Services を生成。
func NewServices(repos Repositories) Services {
	return Services{
		Auth: NewAuthService(repos.User),
		// 下に他のサービスを追加していく
	}
}