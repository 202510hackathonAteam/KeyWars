package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig は JWT の発行および検証に必要な設定値を保持する構造体。
type JWTConfig struct {
	IssuerName string
	HMACSecretKey []byte
	AccessTokenTTL time.Duration
}

// NewJWTHandler は 指定された設定をもとに JWTHandler を初期化して返す。
type JWTHandler struct {
	Config JWTConfig
}

// JWTHandler は JWT の生成および検証処理を担当する構造体。
func NewJWTHandler(config JWTConfig) *JWTHandler {
	return &JWTHandler{Config: config}
}

// GenerateAccessToken は userID を Subject に埋め込んだアクセストークンを生成し、
// そのトークン文字列と有効期限を返す。
func (h *JWTHandler) GenerateAccessToken(userID string) (tokenString string, expiresAt time.Time, err error) {
	expiresAt = time.Now().Add(h.Config.AccessTokenTTL)

	claims := jwt.RegisteredClaims{
		Subject: userID,
		Issuer: h.Config.IssuerName,
		IssuedAt: jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		// 必要に応じて Audience / NotBefore なども追加
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString(h.Config.HMACSecretKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// VerifyAccessToken は アクセストークンを検証し、有効な場合は埋め込まれた userID を返す。
func (h *JWTHandler) VerifyAccessToken(tokenString string) (string, error) {
	var claims jwt.RegisteredClaims

	parsedToken, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return h.Config.HMACSecretKey, nil
	})
	if err != nil {
		return "", err
	}
	if !parsedToken.Valid {
		return "", errors.New("invalid token")
	}

	if claims.Issuer != h.Config.IssuerName {
		return "", errors.New("invalid issuer")
	}
	if claims.ExpiresAt == nil || time.Now().After(claims.ExpiresAt.Time) {
		return "", errors.New("token expired")
	}
	if claims.Subject == "" {
		return "", errors.New("missing subject")
	}

	return claims.Subject, nil
}