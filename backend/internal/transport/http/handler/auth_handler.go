package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"keywars/backend/internal/service"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(s service.AuthService) *AuthHandler {
	return &AuthHandler{authService: s}
}

// --- 一時的な動作確認用エンドポイント --- //
func (h *AuthHandler) Hello(ctx echo.Context) error {
	return ctx.String(http.StatusOK, "Hello, Echo is working!")
}