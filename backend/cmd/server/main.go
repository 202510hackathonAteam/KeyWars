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

func main() {
	cfg := config.Load()

	server, err := app.New(&cfg)
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}
	e := server.Echo

	port := cfg.Server.Port
	if port == "" {
		if v := os.Getenv("PORT"); v != "" {
			port = v
		} else {
			port = "8080"
		}
	}

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

	// サーバーを起動する（Gracefulシャットダウン時の ErrServerClosed は無視する）
	if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
		e.Logger.Fatalf("start error: %v", err)
	}
}