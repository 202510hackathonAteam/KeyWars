package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"keywars/backend/internal/app"
	"keywars/backend/internal/config"

	"keywars/backend/internal/service/realtime"
	ws "keywars/backend/internal/transport/websocket"

	"github.com/labstack/echo/v4"
)

func main() {
	cfg := config.Load()
	fmt.Println("=== Config Check ===")
	fmt.Printf("User: %s\n", cfg.DB.User)
	fmt.Printf("Password: %s\n", cfg.DB.Password)
	fmt.Printf("Host: %s\n", cfg.DB.Host)
	fmt.Printf("Port: %s\n", cfg.DB.Port)
	fmt.Printf("Name: %s\n", cfg.DB.Name)
	fmt.Printf("Server Port: %s\n", cfg.Server.Port)
	fmt.Printf("DSN: %s\n", cfg.DB.DSN())

	server, err := app.New(&cfg)
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
	}
	e := server.Echo

	// ====== WebSocket配線 ======
	// Hub（room管理）
	hub := ws.NewHub()

	// RealtimeService（ここは好きな実装に差し替え）
	rtSvc := realtime.New()

	// チケット検証（本番はJWT実装に差し替え）
	verifier := ws.DevTicket{}

	wsHandler := &ws.Handler{
		Hub:      hub,
		Svc:      rtSvc,
		Verifier: verifier,
	}

	// Echoにマウント（Anyにしておくとプロキシ環境でも融通が利く）
	e.Any("/ws", echo.WrapHandler(wsHandler))
	// ====== ここまで ======

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
