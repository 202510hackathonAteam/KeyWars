package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"keywars/backend/internal/service"
)

// AuthHandler は、認証関連の HTTP リクエストを処理するハンドラの定義。
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler は、AuthService を受け取り AuthHandler を生成。
func NewAuthHandler(service service.AuthService) *AuthHandler {
	return &AuthHandler{authService: service}
}

// --- 一時的な動作確認用エンドポイント --- //
func (h *AuthHandler) Hello(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "Hello, Echo is working!")
}