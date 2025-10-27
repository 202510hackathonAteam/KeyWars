package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"keywars/backend/internal/app"
	"keywars/backend/internal/config"
)

// main は、アプリケーションのエントリーポイント。
// 設定読み込み・サーバ初期化・Graceful シャットダウン制御を担当。
func main() {
	// 設定ファイルの読み込み
	cfg := config.Load()

	// アプリケーションサーバーの初期化
	server, err := app.New(&cfg)
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}
	e := server.Echo

	// ポート番号の決定
	port := cfg.Server.Port
	if port == "" {
		port = os.Getenv("SERVER_PORT")
	}
	if port == "" {
		port = "8080"
	}

	// シグナル受信用チャネルの初期化
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Gracefulシャットダウン処理（最大10秒待機）
	go func() {
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := e.Shutdown(ctx); err != nil {
			e.Logger.Errorf("shutdown error: %v", err)
		}
	}()

	// サーバー起動
	// Graceful シャットダウン時の ErrServerClosed は正常終了として扱う。
	if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
		e.Logger.Fatalf("start error: %v", err)
	}
}