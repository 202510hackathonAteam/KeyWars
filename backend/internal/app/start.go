package app

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/fx"

	"keywars/backend/internal/service/realtime"
)

// StartServer は、サーバー起動処理とリアルタイム処理のライフサイクル管理。
func StartServer(
	lifecycle fx.Lifecycle,
	server *Server,
	matchMakerService *realtime.MatchMakerService,
) {
	e := server.Echo
	
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// Realtime Matchmaker 起動
			matchMakerCtx, cancel := context.WithCancel(context.Background())
			server.MatchMakerCancel = cancel

			matchMakerService.StartMatchmaker(matchMakerCtx, 500*time.Millisecond)

			// Echo サーバー起動（非同期）
			go func() {
				if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
					e.Logger.Fatalf("start error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Realtime の goroutine 停止
			if server.MatchMakerCancel != nil {
				server.MatchMakerCancel()
			}

			shutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			return e.Shutdown(shutCtx)
		},
	})
}