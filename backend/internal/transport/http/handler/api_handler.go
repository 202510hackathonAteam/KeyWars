package handler

import (
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
func New(services service.Services) *API {
  return &API{
    Auth: NewAuthHandler(services.Auth),
    // 下に他のハンドラーを追加していく
  }
}