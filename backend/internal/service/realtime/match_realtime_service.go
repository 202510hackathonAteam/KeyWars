package realtime

import (
	"context"
	"time"
	"encoding/json"
	"github.com/rs/zerolog"

	"keywars/backend/internal/domain/repository"
	"keywars/backend/internal/transport/websocket"
	"keywars/backend/internal/service/round"
	"keywars/backend/internal/config"
)

//
// ==== Service 構造体 ====
//

// MatchRealtimeService は、WebSocket を介してマッチング待機・キャンセル・
// マッチ成立などのリアルタイム処理を提供するアプリケーションサービス。
//
// 責務：
//   - クライアントから受信した WebSocket メッセージ（queue.join, queue.cancel など）を処理する
//   - Redis リポジトリ（MatchQueueRepositoryRedis）を通じて待機キューを操作する
//   - Hub（WebSocket Hub）を用いてイベントをブロードキャストする
type MatchRealtimeService struct {
	matchQueueRepository 		repository.MatchQueueRepository
	roundStateRepository 		repository.RoundStateRepository
	presenceRepository   		repository.PresenceRepository
	promptRepository     		repository.PromptRepository
	websocketHub         		*websocket.Hub
	logger 							 		*zerolog.Logger
	roundFlowService 		 		*round.RoundFlowService
	measurementRoundService *round.MeasurementRoundService
}

// NewMatchRealtimeService は、依存する Redis リポジトリと WebSocket ハブを受け取り、
// 新しい MatchRealtimeService インスタンスを生成して返すコンストラクタ。
func NewMatchRealtimeService(
	matchQueueRepository repository.MatchQueueRepository,
	roundState repository.RoundStateRepository,
	presence repository.PresenceRepository,
	prompt repository.PromptRepository,
	websocketHub *websocket.Hub,
	logger *zerolog.Logger,
	roundFlowService *round.RoundFlowService,
	measurementRoundService *round.MeasurementRoundService,
) *MatchRealtimeService {
	return &MatchRealtimeService{
		matchQueueRepository: 	 matchQueueRepository,
		roundStateRepository: 	 roundState,
		presenceRepository:   	 presence,
		promptRepository:     	 prompt,
		websocketHub:         	 websocketHub,
		logger:									 logger,
		roundFlowService:				 roundFlowService,
		measurementRoundService: measurementRoundService,
	}
}

//
// ==== イベントハンドラ群 ====
//

// OnConnect は、新しい WebSocket 接続が確立された際に呼び出される。
// 接続直後のクライアントに対して歓迎メッセージを返す。
func (service *MatchRealtimeService) OnConnect(
	ctx context.Context,
	userID string,
	roomName string,
) (any, error) {
	nowMs := time.Now().UnixMilli()

	// Presence の状態を取得
	if service.presenceRepository != nil {
		presenceMap, err := service.presenceRepository.Get(ctx, userID)
		if err == nil && len(presenceMap) > 0 {
			status := presenceMap["status"]
			matchID := presenceMap["match_id"]

			if (status == "ingame" || status == "reconnecting") && matchID != "" {
				matchRoomName := "match:" + matchID
				userRoomName := "user:" + userID

				clientConnections := service.websocketHub.Members(userRoomName)
				if len(clientConnections) > 0 {
					_ = service.websocketHub.Move(ctx, clientConnections[0], matchRoomName)

					restoredState, _ := service.roundStateRepository.LoadMatchState(ctx, matchID)

					_ = service.presenceRepository.SetIngame(ctx, userID, matchID, nowMs)

					return map[string]any{
						"type":    "match.restore",
						"matchId": matchID,
						"state":   restoredState,
					}, nil
				}
			}
		}
		_ = service.presenceRepository.SetOnline(ctx, userID, nowMs)
	}

	return map[string]any{
		"type": "welcome",
		"uid":  userID,
	}, nil
}

// OnMessage は、クライアントから受信した WebSocket メッセージを処理する。
// メッセージの `type` に応じて待機キュー操作や通知を行う。
func (service *MatchRealtimeService) OnMessage(
	ctx context.Context,
	userID string,
	roomName string,
	messageType string,
	messagePayload []byte,
) (any, error) {

	switch messageType {

	// --- マッチ待機参加 ---
	case "queue.join":
		currentTimeMs := time.Now().UnixMilli()

		// Redis の待機キューに追加
		err := service.matchQueueRepository.Enqueue(ctx, userID, currentTimeMs)
		if err != nil {
			return map[string]any{
				"type":  "queue.error",
				"error": err.Error(),
			}, nil
		}

		// クライアントに成功レスポンスを返す
		return map[string]any{
			"type": "queue.joined",
			"at":   currentTimeMs,
		}, nil

	// --- マッチ待機キャンセル ---
	case "queue.cancel":
		// best-effort：失敗しても致命的ではない
		_ = service.matchQueueRepository.Cancel(ctx, userID)

		return map[string]any{
			"type": "queue.cancelled",
		}, nil

	// --- ラウンド完了 ---
	case "round.finish":
		var payload websocket.RoundEndPayload
		if err := json.Unmarshal(messagePayload, &payload); err != nil {
			return websocket.NewErrorPayload(), nil
		}

		// プレイヤー認証
		players, err := service.roundStateRepository.LoadMatchPlayers(ctx, payload.MatchID)
		if err != nil {
			return websocket.NewErrorPayload(), nil
    }
		if userID != players.Player1ID && userID != players.Player2ID {
			return websocket.NewErrorPayload(), nil
		}

		// 計測機能実行
		err = service.measurementRoundService.SaveMeasurement(ctx, payload.MatchID, userID, payload.MissCount)
		if err != nil {
			return websocket.NewErrorPayload(), nil
		}

		// measurementFinishedCountを+1
		measurementFinishedCount, err := service.roundStateRepository.IncrementMeasurementFinishCount(ctx, payload.MatchID)
		if err != nil {
			return websocket.NewErrorPayload(), nil
		}

		// 2人揃ったか確認
		if measurementFinishedCount < config.RequiredPlayers {
			return nil, nil
		}

		// ラウンドフローを実行（メイン処理）
		err = service.roundFlowService.RunRoundFlow(ctx, payload.MatchID)
		if err != nil {
			return websocket.NewErrorPayload(), nil
		}

		return nil, nil

	// --- 時間切れ ---
	case "round.timeout":
		var payload websocket.RoundEndPayload
		if err := json.Unmarshal(messagePayload, &payload); err != nil {
			return websocket.NewErrorPayload(), nil
		}

		// プレイヤー認証
		players, err := service.roundStateRepository.LoadMatchPlayers(ctx, payload.MatchID)
		if err != nil {
			return websocket.NewErrorPayload(), nil
    }
		if userID != players.Player1ID && userID != players.Player2ID {
			return websocket.NewErrorPayload(), nil
		}

		// 計測機能実行
		err = service.measurementRoundService.SaveMeasurement(ctx, payload.MatchID, userID, payload.MissCount)
		if err != nil {
			return websocket.NewErrorPayload(), nil
		}

		// measurementFinishedCountを+1
		measurementFinishedCount, err := service.roundStateRepository.IncrementMeasurementFinishCount(ctx, payload.MatchID)
		if err != nil {
			return websocket.NewErrorPayload(), nil
		}

		// 2人揃ったか確認
		if measurementFinishedCount < config.RequiredPlayers {
			return nil, nil
		}

		// ラウンドフローを実行（メイン処理）
		err = service.roundFlowService.RunRoundFlow(ctx, payload.MatchID)
		if err != nil {
			return websocket.NewErrorPayload(), nil
		}

		return nil, nil

	// --- その他（未定義メッセージ） ---
	default:
		// 他のゲーム内メッセージを追加する場合はここに分岐を書く
		return nil, nil
	}
}

// OnDisconnect は、クライアントの WebSocket 接続が切断された際に呼び出される。
// 待機中であれば Redis キューから安全に削除する。
func (service *MatchRealtimeService) OnDisconnect(
	ctx context.Context,
	userID string,
	roomName string,
) {
	// best-effort で待機解除（例: 切断時）
	_ = service.matchQueueRepository.Cancel(context.Background(), userID)
}
