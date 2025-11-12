package auth

import (
	"fmt"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"keywars/backend/internal/config"
)



// JWTHandler は、JWT の生成および検証処理を担当する構造体の定義。
type JWTHandler struct {
	Config config.JWTConfig
}

// NewJWTHandler は、指定された設定をもとに JWTHandler を初期化して返却。
func NewJWTHandler(config config.JWTConfig) *JWTHandler {
	return &JWTHandler{Config: config}
}

// GenerateAccessToken は、アクセストークンを生成。
func (h *JWTHandler) GenerateAccessToken(userID string) (accessToken string, err error) {
	accessExpiresAt := time.Now().Add(h.Config.AccessTokenExpiry)

	accessTokenClaims := jwt.RegisteredClaims{
		Subject: userID,
		Issuer: h.Config.IssuerName,
		IssuedAt: jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
	}

	// アクセストークン生成
	accessJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, accessTokenClaims)
	accessToken, err = accessJWT.SignedString(h.Config.HMACSecretKey)
	if err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}

	return accessToken, nil
}

// GenerateRefreshToken は、リフレッシュトークンを生成。
func (h *JWTHandler) GenerateRefreshToken(userID string) (refreshToken string, err error) {
	refreshExpiresAt := time.Now().Add(h.Config.RefreshTokenExpiry)

	refreshTokenClaims := jwt.RegisteredClaims{
		Subject: userID,
		Issuer: h.Config.IssuerName,
		IssuedAt: jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(refreshExpiresAt),
	}

	// リフレッシュトークン生成
	refreshJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshTokenClaims)
	refreshToken, err = refreshJWT.SignedString(h.Config.HMACSecretKey)
	if err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}

	return refreshToken, nil
}

// GenerateTokens は、アクセストークンおよびリフレッシュトークンを生成。
func (h *JWTHandler) GenerateTokens(userID string) (accessToken string, refreshToken string, err error) {
	accessToken, err = h.GenerateAccessToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err = h.GenerateRefreshToken(userID)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// VerifyAccessToken は、アクセストークンの検証処理。
func (h *JWTHandler) VerifyAccessToken(accessTokenString string) (string, error) {
	var accessTokenClaims jwt.RegisteredClaims

	// アクセストークンの解析と署名検証
	parsedAccessToken, err := jwt.ParseWithClaims(accessTokenString, &accessTokenClaims, func(accessToken *jwt.Token) (interface{}, error) {
		// 署名方式の検証（HMACSecretKey 以外は拒否）
		if _, ok := accessToken.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return h.Config.HMACSecretKey, nil
	})
	if err != nil {
		return "", err
	}
	if !parsedAccessToken.Valid {
		return "", errors.New("invalid token")
	}

	// クレーム内容の検証
	if accessTokenClaims.Issuer != h.Config.IssuerName {
		return "", errors.New("invalid issuer")
	}
	if accessTokenClaims.ExpiresAt == nil || time.Now().After(accessTokenClaims.ExpiresAt.Time) {
		return "", errors.New("token expired")
	}
	if accessTokenClaims.Subject == "" {
		return "", errors.New("missing subject")
	}

	return accessTokenClaims.Subject, nil
}

// VerifyRefreshToken は、リフレッシュトークンの検証処理。
func (h *JWTHandler) VerifyRefreshToken(refreshTokenString string) (string, error) {
	var refreshTokenClaims jwt.RegisteredClaims

	// リフレッシュトークンの解析と署名検証
	parsedRefreshToken, err := jwt.ParseWithClaims(refreshTokenString, &refreshTokenClaims, func(refreshToken *jwt.Token) (interface{}, error) {
		// 署名方式の検証（HMACSecretKey 以外は拒否）
		if _, ok := refreshToken.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return h.Config.HMACSecretKey, nil
	})
	if err != nil {
		return "", err
	}
	if !parsedRefreshToken.Valid {
		return "", errors.New("invalid token")
	}

	// クレーム内容の検証
	if refreshTokenClaims.Issuer != h.Config.IssuerName {
		return "", errors.New("invalid issuer")
	}
	if refreshTokenClaims.ExpiresAt == nil || time.Now().After(refreshTokenClaims.ExpiresAt.Time) {
		return "", errors.New("token expired")
	}
	if refreshTokenClaims.Subject == "" {
		return "", errors.New("missing subject")
	}

	return refreshTokenClaims.Subject, nil
}