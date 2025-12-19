package realtime

import (
	"context"
	"time"

	"keywars/backend/internal/transport/websocket"
)

//
// ==== マッチメイカー（定期実行タスク） ====
//

// StartMatchmaker は、一定間隔でマッチング処理を試行するバックグラウンドループを開始する。
// Redis の待機キューに 2 名以上が存在する場合、Dequeue して新しいマッチを生成する。
// tick には試行間隔（例: 500ms, 1s など）を指定する。
func (s *MatchRealtimeService) StartMatchmaker(ctx context.Context, interval time.Duration) {
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
				s.tryMakeMatch(ctx)
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
func (s *MatchRealtimeService) tryMakeMatch(ctx context.Context) {
	// DequeuePairAndInitMatch:
	//   - 2名を ZPOPMIN で取り出す
	//   - match:{matchID} / match:{matchID}:state を初期化
	dequeueCtx, cancelDequeue := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancelDequeue()
	user1ID, user2ID, matchID, _, err := s.matchQueueRepo.DequeuePairAndInitMatch(dequeueCtx)
	if err != nil || matchID == "" {
		// 競合発生 or 2名未満の場合は何もしない
		return
	}
	// log.Printf("[matchmaker] matched: matchID=%s p1=%s p2=%s", matchID, user1ID, user2ID)

	// 初期化
	initMeasurementCtx, cancelInitMeasurement := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancelInitMeasurement()
	err = s.roundStateRepo.InitMeasurementFinishCount(initMeasurementCtx, matchID)
	if err != nil {
		return
	}

	// 問題抽出
	deckSaveCtx, cancelDeckSave := context.WithTimeout(ctx, 700*time.Millisecond)
	defer cancelDeckSave()
	err = s.deckGeneratorService.GenerateAndSaveDeck(deckSaveCtx, matchID)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("match_id", matchID).
			Msg("failed to generate prompts")
		return
	}

	// ユーザー → 試合の対応関係を確定
	s.roundStateRepo.SetUserActiveMatch(ctx, user1ID, matchID)
	s.roundStateRepo.SetUserActiveMatch(ctx, user2ID, matchID)

	// マッチ成立通知（即時 push）
  s.notifyMatchFoundIfConnected(user1ID, matchID, user2ID)
  s.notifyMatchFoundIfConnected(user2ID, matchID, user1ID)

	// 第1ラウンド開始（1回だけ）
	if err := s.roundFlowService.StartFirstRound(ctx, matchID, user1ID, user2ID); err != nil {
    s.logger.Error().
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
func (s *MatchRealtimeService) moveClientIfConnected(userID string, newRoomName string) error {
	// 個人ルーム（user:<userID>）に現在接続中のクライアントを取得
	clientConnections := s.websocketHub.Members("user:" + userID)
	if len(clientConnections) == 0 {
		// 未接続または直前に切断された場合
		return nil
	}

	// 想定上、同一ユーザーに対して複数の接続は存在しない。
	// 仮に複数ある場合は、最初の1件のみを対象とする。
	s.websocketHub.Move(clientConnections[0], newRoomName)
	return nil
}

// notifyMatchFoundIfConnected は、接続中ユーザーに対するマッチ成立の即時通知処理。
// WebSocket 接続が存在する場合のみ通知とルーム移動を行い、
// 未接続の場合は OnConnect による再送に委ねる実装。
func (s *MatchRealtimeService) notifyMatchFoundIfConnected(
  userID, matchID, opponentID string,
) {
  conns := s.websocketHub.Members("user:" + userID)
  if len(conns) == 0 {
    return // 未接続なら後回し
  }

  _ = s.websocketHub.Move(conns[0], "match:"+matchID)
  _ = conns[0].SendJSON(
    context.Background(),
    websocket.NewMatchFoundPayload(matchID, opponentID),
  )
}