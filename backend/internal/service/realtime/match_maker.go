package realtime

import (
	"context"
	"time"
)

//
// ==== マッチメイカー（定期実行タスク） ====
//

// StartMatchmaker は、一定間隔でマッチング処理を試行するバックグラウンドループを開始する。
// Redis の待機キューに 2 名以上が存在する場合、Dequeue して新しいマッチを生成する。
// tick には試行間隔（例: 500ms, 1s など）を指定する。
func (service *MatchRealtimeService) StartMatchmaker(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// コンテキストがキャンセルされた場合はループを終了
				return

			case <-ticker.C:
				// tick ごとにマッチング処理を試行
				service.tryMakeMatch(ctx)
			}
		}
	}()
}

//
// ==== 内部マッチングロジック ====
//

// tryMakeMatch は、Redis の待機キューから 2 名を取り出し、
// 新しいマッチを初期化して各クライアントへ通知する。
// 取り出しは排他ロックにより、同時実行を防止する。
func (service *MatchRealtimeService) tryMakeMatch(ctx context.Context) {
	// DequeuePairAndInitMatch:
	//   - 2名を ZPOPMIN で取り出す
	//   - match:{matchID} / match:{matchID}:state を初期化
	dequeueCtx, cancelDequeue := context.WithTimeout(ctx, 500*time.Millisecond)
	user1ID, user2ID, matchID, _, err := service.matchQueueRepository.DequeuePairAndInitMatch(dequeueCtx)
	cancelDequeue()
	if err != nil || matchID == "" {
		// 競合発生 or 2名未満の場合は何もしない
		return
	}
	// log.Printf("[matchmaker] matched: matchID=%s p1=%s p2=%s", matchID, user1ID, user2ID)

	// 初期化
	initMeasurementCtx, cancelInitMeasurement := context.WithTimeout(ctx, 300*time.Millisecond)
	err = service.roundStateRepository.InitMeasurementFinishCount(initMeasurementCtx, matchID)
	cancelInitMeasurement()
	if err != nil {
		return
	}

	// 問題抽出
	deckLoadCtx, cancelDeckLoad := context.WithTimeout(ctx, 700*time.Millisecond)
	deck, err := service.promptRepository.GetDeckPrompts(deckLoadCtx)
	cancelDeckLoad()
	if err == nil && len(deck) == 20 {
		saveDeckCtx, cancelSaveDeck := context.WithTimeout(ctx, 700*time.Millisecond)
		_ = service.roundStateRepository.SaveDeck(saveDeckCtx, matchID, deck)
		cancelSaveDeck()
	}

	// --- 1) 両者の個人ルームへ「マッチ成立」通知 ---
	_, _ = service.websocketHub.Broadcast(ctx, "user:"+user1ID, map[string]any{
		"type":     "match.found",
		"matchId":  matchID,
		"opponent": user2ID,
	})
	_, _ = service.websocketHub.Broadcast(ctx, "user:"+user2ID, map[string]any{
		"type":     "match.found",
		"matchId":  matchID,
		"opponent": user1ID,
	})

	// --- 2) 両者をマッチルームへ移動 ---
	// ルーム名は "match:<matchID>" とする
	matchRoomName := "match:" + matchID
	_ = service.moveClientIfConnected(user1ID, matchRoomName)
	_ = service.moveClientIfConnected(user2ID, matchRoomName)

	now := time.Now().UnixMilli()
	_ = service.presenceRepository.SetIngame(ctx, user1ID, matchID, now)
	_ = service.presenceRepository.SetIngame(ctx, user2ID, matchID, now)

	// --- 3) 第1ラウンドを開始 ---
	if err := service.roundFlowService.StartFirstRound(ctx, matchID, user1ID, user2ID); err != nil {
    service.logger.Error().
			Err(err).
			Str("matchID", matchID).
			Msg("failed to start first round")
    return
	}
}

//
// ==== 内部ヘルパー ====
//

// moveClientIfConnected は、指定されたユーザーIDのクライアントが現在オンラインであれば、
// 新しいルーム（newRoomName）へ安全に移動させる。
// 切断済み（Hubに存在しない）場合は no-op（何もしない）。
func (service *MatchRealtimeService) moveClientIfConnected(userID string, newRoomName string) error {
	// 個人ルーム（user:<userID>）に現在接続中のクライアントを取得
	clientConnections := service.websocketHub.Members("user:" + userID)
	if len(clientConnections) == 0 {
		// 未接続または直前に切断された場合
		return nil
	}

	// 想定上、同一ユーザーに対して複数の接続は存在しない。
	// 仮に複数ある場合は、最初の1件のみを対象とする。
	service.websocketHub.Move(context.Background(), clientConnections[0], newRoomName)
	return nil
}
