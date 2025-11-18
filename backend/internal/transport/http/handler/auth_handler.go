package handler

import (
	"net/http"
	"context"
	"time"
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	goValidator "github.com/go-playground/validator/v10"

	"keywars/backend/internal/infra/auth"
	"keywars/backend/internal/service"
	"keywars/backend/internal/transport/http/response"
	"keywars/backend/internal/util/validator"
	"keywars/backend/internal/util/cookie"
)

// AuthHandler は、認証関連の HTTP リクエストを処理するハンドラの定義。
type AuthHandler struct {
	authService service.AuthService
	validate *goValidator.Validate
	jwtHandler *auth.JWTHandler
}

// NewAuthHandler は、AuthService を受け取り AuthHandler を生成。
func NewAuthHandler(service service.AuthService, validate *goValidator.Validate, jwtHandler *auth.JWTHandler) *AuthHandler {
	return &AuthHandler{
		authService: service,
		validate: validate,
		jwtHandler: jwtHandler,
	}
}

// SignUpRequest は、ユーザー新規登録時のリクエストボディを表す構造体。
type SignUpRequest struct {
	Username string `json:"user_name" validate:"required,min=3,max=255,username_format"`
	Password string `json:"password" validate:"required,min=8,max=64,password_format"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
}

// Signup は新規ユーザー登録のエンドポイント。
func (h *AuthHandler) Signup(c echo.Context) error {
	logger := zerolog.Ctx(c.Request().Context())

	// 入力の受け取り
	body := &SignUpRequest{}
	if err := c.Bind(body); err != nil {
		logger.Warn().Err(err).Msg("failed binding body")
		return response.Respond(c, http.StatusBadRequest, nil)
	}

	// 必須チェックなど軽バリデーション
	if body.Username == "" || body.Password == "" || body.ConfirmPassword == "" {
		return response.Respond(c, http.StatusBadRequest, echo.Map{
			"message": "すべての項目を入力してください。",
		})
	}
	if body.Password != body.ConfirmPassword {
		return response.Respond(c, http.StatusBadRequest, echo.Map{
			"message": "パスワードが一致していません。同じパスワードを入力してください。",
		})
	}

	// リクエストの形式を検証（validateタグに基づく構文チェック）
	if err := h.validate.Struct(body); err != nil {
		message := validator.TranslateError(err)
		return response.Respond(c, http.StatusBadRequest, echo.Map{
			"message": message,
		})
	}

	// リクエスト全体の処理時間を制限
	timeoutCtx, cancel := context.WithTimeout(c.Request().Context(), time.Second*10)
	defer cancel()

	// ユーザー登録処理
	accessToken, refreshToken, err := h.authService.Signup(timeoutCtx, body.Username, body.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserExists):
			logger.Warn().Err(err).Str("username", body.Username).Msg("/auth/signup conflict: user already exists")
			return response.Respond(c, http.StatusConflict, echo.Map{
				"message": "ユーザー名がすでに使われています。",
			})
		default:
			logger.Error().Err(err).Msgf("/auth/signup failed: internal error (%T: %v)", err, err)
			return response.Respond(c, http.StatusInternalServerError, nil)
		}
	}

	// Cookie保存
	cookie.SetTokens(
		c,
		accessToken,
		refreshToken,
		h.jwtHandler.Config.AccessTokenExpiry,
		h.jwtHandler.Config.RefreshTokenExpiry,
	)

	return response.Respond(c, http.StatusCreated, nil)
}

// SigninRequest は、ユーザーサインイン時のリクエストボディを表す構造体。
type SigninRequest struct {
	Username string `json:"user_name"`
	Password string `json:"password"`
}

// Signin は既存ユーザーの認証エンドポイント。
func (h *AuthHandler) Signin(c echo.Context) error {
	logger := zerolog.Ctx(c.Request().Context())

	// 入力の受け取り
	body := &SigninRequest{}
	if err := c.Bind(body); err != nil {
		logger.Warn().Err(err).Msg("failed binding body")
		return response.Respond(c, http.StatusBadRequest, nil)
	}

	// 必須チェックなど軽バリデーション
	if body.Username == "" || body.Password == "" {
		return response.Respond(c, http.StatusBadRequest, echo.Map{
			"message": "すべての項目を入力してください。",
		})
	}

	// リクエスト全体の処理時間を制限
	timeoutCtx, cancel := context.WithTimeout(c.Request().Context(), time.Second*10)
	defer cancel()

	// サービス層でサインイン処理（重複確認・ハッシュ・作成）
	accessToken, refreshToken, err := h.authService.Signin(timeoutCtx, body.Username, body.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound),
				 errors.Is(err, service.ErrInvalidCredentials):
			logger.Warn().Err(err).Str("username", body.Username).Msg("/auth/signin failed: user not found")
			return response.Respond(c, http.StatusConflict, echo.Map{
				"message": "ログインに失敗しました。入力内容をご確認ください。",
			})
		default:
			logger.Error().Err(err).Msg("/auth/signin failed: internal error")
			return response.Respond(c, http.StatusInternalServerError, nil)
		}
	}

	// Cookie保存
	cookie.SetTokens(
		c,
		accessToken,
		refreshToken,
		h.jwtHandler.Config.AccessTokenExpiry,
		h.jwtHandler.Config.RefreshTokenExpiry,
	)

	return response.Respond(c, http.StatusOK, nil)
}

// Signout はログアウト処理を行うエンドポイント。
// 現在のセッション（JWT）は stateless なため、
// クッキーを削除（上書き）するだけでログアウトが完了する。
func(h *AuthHandler) Signout(c echo.Context) error {
	// Cookie削除
	cookie.ClearTokens(c)

	return response.Respond(c, http.StatusNoContent, nil)
}

// Refresh は、リフレッシュトークンを用いてアクセストークンのみを再発行
func (h *AuthHandler) Refresh(c echo.Context) error {
	logger := zerolog.Ctx(c.Request().Context())

	// refresh_token クッキーの取得と基本チェック
	refreshTokenCookie, err := c.Cookie("refresh_token")
	if err != nil {
		return response.Respond(c, http.StatusUnauthorized, nil)
	}
	refreshToken := refreshTokenCookie.Value
	if refreshToken == "" {
		return response.Respond(c, http.StatusUnauthorized, nil)
	}

	// リフレッシュトークンの検証、アクセストークンの再発行処理
	accessToken, err := h.authService.Refresh(refreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			logger.Warn().Err(err).Msg("/auth/refresh failed: invalid refresh token")
			return response.Respond(c, http.StatusUnauthorized, nil)
		default:
			logger.Error().Err(err).Msg("/auth/refresh failed: internal error")
			return response.Respond(c, http.StatusInternalServerError, nil)
		}
	}

	// アクセストークンをクッキーへ再設定
	cookie.SetAccessToken(c, accessToken, h.jwtHandler.Config.AccessTokenExpiry)

	return response.Respond(c, http.StatusOK, nil)
}

// Check は、アクセストークンの有効性を確認するエンドポイント。
func (h *AuthHandler) Check(c echo.Context) error {
	logger := zerolog.Ctx(c.Request().Context())

	// アクセストークンの取得
	accessTokenCookie, err := c.Cookie("access_token")
	if err != nil {
		return response.Respond(c, http.StatusUnauthorized, nil)
	}
	accessToken := accessTokenCookie.Value
	if accessToken == "" {
		return response.Respond(c, http.StatusUnauthorized, nil)
	}

	// JWTの検証
	_, err = h.jwtHandler.VerifyAccessToken(accessToken)
	if err != nil {
		logger.Warn().Err(err).Msg("/auth/check failed: access token verification")
		return response.Respond(c, http.StatusUnauthorized, nil)
	}

	return response.Respond(c, http.StatusOK, nil)
}