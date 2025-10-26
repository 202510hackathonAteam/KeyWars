package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig は、JWT の発行および検証に必要な設定値の定義。
type JWTConfig struct {
	IssuerName string
	HMACSecretKey []byte
	AccessTokenTTL time.Duration
}

// JWTHandler は、JWT の生成および検証処理を担当する構造体の定義。
type JWTHandler struct {
	Config JWTConfig
}

// NewJWTHandler は、指定された設定をもとに JWTHandler を初期化して返却。
func NewJWTHandler(config JWTConfig) *JWTHandler {
	return &JWTHandler{Config: config}
}

// GenerateAccessToken は、userID を Subject に埋め込んだアクセストークンの生成。
// トークン文字列と有効期限を返却。署名に失敗した場合はエラーを返す。
func (h *JWTHandler) GenerateAccessToken(userID string) (tokenString string, expiresAt time.Time, err error) {
	// --- トークンの有効期限を設定 ---
	expiresAt = time.Now().Add(h.Config.AccessTokenTTL)

	// --- JWT クレームの作成 ---
	claims := jwt.RegisteredClaims{
		Subject: userID,
		Issuer: h.Config.IssuerName,
		IssuedAt: jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		// 必要に応じて Audience / NotBefore なども追加
	}

	// --- HMAC 署名によるトークン生成 ---
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString(h.Config.HMACSecretKey)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// VerifyAccessToken は、アクセストークンの検証処理。
// トークンが有効な場合は Subject（userID）を返却。
// 署名方式の不一致、期限切れ、Issuer 不一致、Subject 欠落などの場合はエラーを返す。
func (h *JWTHandler) VerifyAccessToken(tokenString string) (string, error) {
	var claims jwt.RegisteredClaims

	// --- トークンの解析と署名検証 ---
	parsedToken, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		// 署名方式の検証（HMACSecretKey 以外は拒否）
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

	// --- クレーム内容の検証 ---
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