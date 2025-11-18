package service

import (
	"keywars/backend/internal/domain/repository"
	"keywars/backend/internal/infra/auth"
)

// Repositories は、service が依存するリポジトリ群の定義。
type Repositories struct {
	User  repository.UserRepository
	Queue repository.MatchQueueRepository
	Round repository.RoundStateRepository
	// 下に他のリポジトリを追加していく
}

// Services は、service 群の集約。
type Services struct {
	Auth  AuthService
	Round *RoundService
	// 下に他のサービスを追加していく
}

// NewServices は、Repositories から Services を生成。
func NewServices(repos Repositories, jwtHandler *auth.JWTHandler) Services {
	return Services{
		Auth:  NewAuthService(repos.User, jwtHandler),
		Round: NewRoundService(repos.Round),
		// 下に他のサービスを追加していく
	}
}
