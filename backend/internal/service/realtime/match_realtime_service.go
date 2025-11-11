package realtime

import (
	"context"
	"keywars/backend/internal/domain/repository"
	"keywars/backend/internal/transport/websocket"
	"time"
)

//
// ==== Service 構造体 ====
//

// Service は、WebSocket を介してマッチング待機・キャンセル・
// マッチ成立などのリアルタイム処理を提供するアプリケーションサービス。
//
// 責務：
//   - クライアントから受信した WebSocket メッセージ（queue.join, queue.cancel など）を処理する
//   - Redis リポジトリ（MatchQueueRepositoryRedis）を通じて待機キューを操作する
//   - Hub（WebSocket Hub）を用いてイベントをブロードキャストする
type Service struct {
	matchQueueRepository repository.MatchQueueRepository
	roundStateRepository repository.RoundStateRepository
	presenceRepository   repository.PresenceRepository
	promptRepository     repository.PromptRepository
	websocketHub         *websocket.Hub
}

// NewService は、依存する Redis リポジトリと WebSocket ハブを受け取り、
// 新しい Service インスタンスを生成して返すコンストラクタ。
func NewService(
	matchQueueRepository repository.MatchQueueRepository,
	roundState repository.RoundStateRepository,
	presence repository.PresenceRepository,
	prompt repository.PromptRepository,
	websocketHub *websocket.Hub,
) *Service {
	return &Service{
		matchQueueRepository: matchQueueRepository,
		roundStateRepository: roundState,
		presenceRepository:   presence,
		promptRepository:     prompt,
		websocketHub:         websocketHub,
	}
}

//
// ==== イベントハンドラ群 ====
//

// OnConnect は、新しい WebSocket 接続が確立された際に呼び出される。
// 接続直後のクライアントに対して歓迎メッセージを返す。
func (service *Service) OnConnect(
	ctx context.Context,
	userID string,
	roomName string,
) (any, error) {
	return map[string]any{
		"type": "welcome",
		"uid":  userID,
	}, nil
}

// OnMessage は、クライアントから受信した WebSocket メッセージを処理する。
// メッセージの `type` に応じて待機キュー操作や通知を行う。
func (service *Service) OnMessage(
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

	// --- その他（未定義メッセージ） ---
	default:
		// 他のゲーム内メッセージを追加する場合はここに分岐を書く
		return nil, nil
	}
}

// OnDisconnect は、クライアントの WebSocket 接続が切断された際に呼び出される。
// 待機中であれば Redis キューから安全に削除する。
func (service *Service) OnDisconnect(
	ctx context.Context,
	userID string,
	roomName string,
) {
	// best-effort で待機解除（例: 切断時）
	_ = service.matchQueueRepository.Cancel(context.Background(), userID)
}
