package model

import (
	"fmt"
	"errors"

	"github.com/alexedwards/argon2id"
)

// User はアプリケーション内のユーザー情報を表すドメインモデル
type User struct {
	ID string
	UserName string
	PasswordHash string
}

// NewUser は、登録前のユーザーデータを整形するための User 構造体生成関数。
func NewUser(username string) *User {
  return &User{
    UserName: username,
  }
}

var ErrInvalidPassword = errors.New("invalid password")

// SetPassword は、平文パスワードをハッシュ化してユーザー構造体に設定するメソッド。
func (u *User) SetPassword(passwordPlain string) error {
  passwordHash, err := argon2id.CreateHash(passwordPlain, argon2id.DefaultParams)
  if err != nil {
    return err
  }
  u.PasswordHash = passwordHash
  return nil
}

// ValidatePassword は、入力された平文パスワードがユーザーのハッシュ済みパスワードと一致するかを検証するメソッド。
func (u *User) ValidatePassword(passwordPlain string) error {
  ok, err := argon2id.ComparePasswordAndHash(passwordPlain, u.PasswordHash)
  if err != nil {
		return fmt.Errorf("failed to verify password: %w", err)
	}
	if !ok {
		return ErrInvalidPassword
	}
  return nil
}