package service

import (
	"fmt"
	"context"
	"errors"
	"strings"

	"keywars/backend/internal/domain/repository"
	"keywars/backend/internal/domain/model"
	"keywars/backend/internal/infra/auth"
)

var (
	ErrUserExists         = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenMismatch      = errors.New("token mismatch")
	ErrInvalidToken       = errors.New("invalid token")
)

// AuthService は、認証関連ユースケースのインターフェースの定義。
type AuthService interface {
	Signup(ctx context.Context, username, passwordPlain string) (string, string, error)
	Signin(ctx context.Context, username, passwordPlain string) (string, string, error)
	Refresh(refreshToken string) (string, error)
}

// authService は AuthService インターフェースの具象実装。
type authService struct{
	userRepo repository.UserRepository
	jwtHandler *auth.JWTHandler
}

// NewAuthService は UserRepository を受け取り、AuthServiceを生成。
func NewAuthService(userRepo repository.UserRepository, jwtHandler *auth.JWTHandler) *authService {
	return &authService{
		userRepo: userRepo,
		jwtHandler: jwtHandler,
	}
}

// Signup は、新規ユーザー登録を行うメソッド。
func (s *authService) Signup(ctx context.Context, username, passwordPlain string) (string, string, error) {
	// 重複チェック
	usernameExists, err := s.userRepo.ExistsByUsername(ctx, username)
	if err != nil {
		return "", "", fmt.Errorf("check username exists: %w", err)
	}
	if usernameExists {
		return "", "", ErrUserExists
	}
	
	// ユーザーデータの作成
	user := model.NewUser(username)
	err = user.SetPassword(passwordPlain)
	if err != nil {
		return "", "", fmt.Errorf("hash password: %w", err)
	}

	// ユーザーをデータベースに登録
	userID, err := s.userRepo.Create(ctx, user)
	if err != nil {
		if strings.Contains(err.Error(), "user_name already exists") {
			return "", "", ErrUserExists
		}
		return "", "", fmt.Errorf("create user: %w", err)
	}

	// トークン生成
	accessToken, refreshToken, err := s.jwtHandler.GenerateTokens(userID)
	if err != nil {
		return "", "", fmt.Errorf("generate tokens: %w", err)
	}

	return accessToken, refreshToken, nil
}

// Signin は既存ユーザーの認証を行うメソッド。
func (s *authService) Signin(ctx context.Context, username, passwordPlain string) (string, string, error) {
	// 認証対象の取得
	user, err := s.userRepo.FindUserByUsername(ctx, username)
	if err != nil {
		return "", "", fmt.Errorf("retrieve user: %w", err)
	}
	if user == nil {
		return "", "", ErrUserNotFound
	}

	// パスワード検証
	err = user.ValidatePassword(passwordPlain)
	if errors.Is(err, model.ErrInvalidPassword) {
		return "", "", ErrInvalidCredentials
	}
	if err != nil {
		return "", "", fmt.Errorf("verify password:: %w", err)
	}

	// トークン生成
	accessToken, refreshToken, err := s.jwtHandler.GenerateTokens(user.ID)
	if err != nil {
		return "", "", fmt.Errorf("generate tokens: %w", err)
	}

	return accessToken, refreshToken, nil
}

// Refresh は、アクセストークンを再発行するメソッド。
func (s *authService) Refresh(refreshToken string) (string, error) {
	// リフレッシュトークンの検証
	userID, err := s.jwtHandler.VerifyRefreshToken(refreshToken)
	if err != nil {
		return "", ErrInvalidToken
	}

	// 新しい access_token を生成
	accessToken, err := s.jwtHandler.GenerateAccessToken(userID)
	if err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}

	return accessToken, err
}