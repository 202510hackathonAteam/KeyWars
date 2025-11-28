package handler

import (
	goValidator "github.com/go-playground/validator/v10"

	"keywars/backend/internal/infra/auth"
	"keywars/backend/internal/service"
)

// API は、アプリケーションの HTTP ハンドラ群をまとめたエントリーポイントの定義。
// 各機能のハンドラ（Auth など）を保持し、ルータから呼び出される。
type API struct {
	Auth *AuthHandler
	// 下に他のハンドラーを追加していく
}

// New は、service 層の集約を受け取り、API 構造体を生成。
// 各ハンドラへ対応する service を注入して初期化。
func New(services service.Services, validate *goValidator.Validate, jwtHandler *auth.JWTHandler) *API {
	return &API{
		Auth: NewAuthHandler(services.Auth, validate, jwtHandler),
		// 下に他のハンドラーを追加していく
	}
}
