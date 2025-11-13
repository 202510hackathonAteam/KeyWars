package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	timeCost = 1
	memoryCost = 64 * 1024
	threads = 4
	keyLength = 32
	saltLength = 16
)

// Hash は、平文パスワードを安全にハッシュ化して、ソルト付き文字列として返す関数。
func Hash(passwordPlain []byte) (string, error) {
	// ランダムなソルト生成
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// ハッシュ生成
	hash := argon2.IDKey(passwordPlain, salt, timeCost, memoryCost, threads, keyLength)

	// エンコード
	encoded := fmt.Sprintf("%s.%s",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

// Verify は、入力された平文パスワードがハッシュと一致するかを検証する関数。
func Verify(passwordPlain []byte, passwordHash string) (bool, error) {
	// ハッシュ形式確認
	parts := strings.Split(passwordHash, ".")
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid hash format")
	}

	// Base64デコード
	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// 平文パスワードを同じパラメータ・ソルトで再ハッシュ化
	newHash := argon2.IDKey(passwordPlain, salt, timeCost, memoryCost, threads, uint32(len(hash)))

	// タイミング攻撃を防ぐため、定数時間比較で一致確認
	if subtle.ConstantTimeCompare(hash, newHash) != 1 {
		return false, nil
	}

	return true, nil
}