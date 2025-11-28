package realtime

import (
	"context"
	"errors"
	"time"
	"encoding/json"

	"github.com/rs/zerolog"

	"keywars/backend/internal/config"
	"keywars/backend/internal/domain/constant"
	"keywars/backend/internal/domain/repository"
	"keywars/backend/internal/service/round"
	"keywars/backend/internal/transport/websocket"
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
	matchQueueRepo 					repository.MatchQueueRepository
	roundStateRepo 					repository.RoundStateRepository
	presenceRepo   					repository.PresenceRepository
	websocketHub         		*websocket.Hub
	logger 							 		*zerolog.Logger
	deckGeneratorService		*round.DeckGeneratorService
	roundFlowService 		 		*round.RoundFlowService
	measurementRoundService *round.MeasurementRoundService
}

// NewMatchRealtimeService は、依存する Redis リポジトリと WebSocket ハブを受け取り、
// 新しい MatchRealtimeService インスタンスを生成して返すコンストラクタ。
func NewMatchRealtimeService(
	matchQueueRepo repository.MatchQueueRepository,
	roundStateRepo repository.RoundStateRepository,
	presenceRepo repository.PresenceRepository,
	websocketHub *websocket.Hub,
	logger *zerolog.Logger,
	deckGeneratorService *round.DeckGeneratorService,
	roundFlowService *round.RoundFlowService,
	measurementRoundService *round.MeasurementRoundService,
) *MatchRealtimeService {
	return &MatchRealtimeService{
		matchQueueRepo: 	 			 matchQueueRepo,
		roundStateRepo: 	 			 roundStateRepo,
		presenceRepo:   	 			 presenceRepo,
		websocketHub:         	 websocketHub,
		logger:									 logger,
		deckGeneratorService:    deckGeneratorService,
		roundFlowService:				 roundFlowService,
		measurementRoundService: measurementRoundService,
	}
}

//
// ==== イベントハンドラ群 ====
//

// OnConnect は、新しい WebSocket 接続が確立された際に呼び出される。
// 接続直後のクライアントに対して歓迎メッセージを返す。
func (s *MatchRealtimeService) OnConnect(
	ctx context.Context,
	userID string,
	roomName string,
) (any, error) {
	nowMs := time.Now().UnixMilli()

	// Presence が無い＝このユーザーは試合参加中ではない。
	// 復帰対象が無いため通常接続として welcome を返す。
	if s.presenceRepo == nil {
		return websocket.NewWelcomePayload(userID), nil
	}

	presenceMap, err := s.presenceRepo.Get(ctx, userID)
	if err != nil || len(presenceMap) == 0 {
		_ = s.presenceRepo.SetOnline(ctx, userID, nowMs)
		return websocket.NewWelcomePayload(userID), nil
	}

	status := presenceMap["status"]
	matchID := presenceMap["match_id"]

	// 途中復帰の前提条件をすべてチェック（否定条件は即 return）
	if !(status == "ingame" || status == "reconnecting") {
		_ = s.presenceRepo.SetOnline(ctx, userID, nowMs)
		return websocket.NewWelcomePayload(userID), nil
	}

	if matchID == "" {
		_ = s.presenceRepo.SetOnline(ctx, userID, nowMs)
		return websocket.NewWelcomePayload(userID), nil
	}

	// state がない = 終了済み or 試合破棄 → 復帰できない
	restoredState, err := s.roundStateRepo.LoadMatchState(ctx, matchID)
	if err != nil || restoredState == nil {
		_ = s.presenceRepo.SetOnline(ctx, userID, nowMs)
		return websocket.NewWelcomePayload(userID), nil
	}

	// ここまで来た場合、途中復帰できる
	matchRoom := "match:" + matchID
	userRoom := "user:" + userID
	conns := s.websocketHub.Members(userRoom)

	if len(conns) > 0 {
		_ = s.websocketHub.Move(ctx, conns[0], matchRoom)
	}

	_ = s.presenceRepo.SetIngame(ctx, userID, matchID, nowMs)

	return websocket.NewMatchRestorePayload(
		matchID,
		websocket.MatchState{
			Round:            restoredState.Round,
			RoundStartAtMS:   restoredState.RoundStartAtMs,
			RoundEndAtMS:     restoredState.RoundEndAtMs,
			Player1Lifepoint: restoredState.Player1Lifepoint,
			Player2Lifepoint: restoredState.Player2Lifepoint,
		},
	), nil
}

// OnMessage は、クライアントから受信した WebSocket メッセージを処理する。
// メッセージの `type` に応じて待機キュー操作や通知を行う。
func (s *MatchRealtimeService) OnMessage(
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
		err := s.matchQueueRepo.Enqueue(ctx, userID, currentTimeMs)
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
	case "queue.left":
		// best-effort：失敗しても致命的ではない
		_ = s.matchQueueRepo.Cancel(ctx, userID)

		return map[string]any{
			"type": "queue.cancelled",
		}, nil

	// --- ラウンド完了 ---
	case constant.TriggerAnswerFinish:
		var payload websocket.PlayerAnswerFinishedPayload
		if err := json.Unmarshal(messagePayload, &payload); err != nil {
			s.logger.Error().
				Err(err).
				Str("event", constant.TriggerAnswerFinish).
				Str("match_id", payload.MatchID).
				Msg("failed to unmarshal payload")
			return websocket.NewErrorPayload(), nil
		}

		// プレイヤー認証
		loadPlayersCtx, cancelLoadPlayers := context.WithTimeout(ctx, 200*time.Millisecond)
		players, err := s.roundStateRepo.LoadMatchPlayers(loadPlayersCtx, payload.MatchID)
		cancelLoadPlayers()
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("event", constant.TriggerAnswerFinish).
				Str("match_id", payload.MatchID).
				Msg("failed to load match players")
			return websocket.NewErrorPayload(), nil
		}
		if userID != players.Player1ID && userID != players.Player2ID {
      s.logger.Warn().
        Str("event", constant.TriggerAnswerFinish).
        Str("match_id", payload.MatchID).
        Msg("unauthorized player")
			return websocket.NewErrorPayload(), nil
		}

		// 計測機能実行（各プレイヤーが1回だけ実行）
		if err := s.measurementRoundService.SaveMeasurement(ctx, payload.MatchID, userID, payload.MissCount, constant.TriggerAnswerFinish); err != nil {
			// すでに実行済み（2回目の finish は正常扱いとして無視）
			if errors.Is(err, constant.ErrAlreadyFinished) {
        return nil, nil
			}
			s.logger.Error().
        Err(err).
        Str("event", constant.TriggerAnswerFinish).
        Str("match_id", payload.MatchID).
        Msg("failed to SaveMeasurement (answer finish)")
			return websocket.NewErrorPayload(), nil
		}

		// measurementFinishedCountを+1（全員が1回だけ実行）
		incrementCountCtx, cancelIncrement := context.WithTimeout(ctx, 200*time.Millisecond)
		measurementFinishedCount, err := s.roundStateRepo.IncrementMeasurementFinishCount(incrementCountCtx, payload.MatchID)
		cancelIncrement()
		if err != nil {
			return websocket.NewErrorPayload(), nil
		}

		// 両プレイヤーが揃うまで終了処理は実行しない（まだ各プレイヤー1回実行の領域）
		if measurementFinishedCount < config.RequiredPlayers {
			return nil, nil
		}

		// プレイヤー数が規定値を超えている場合 → 本来発生しない異常状態。
		if measurementFinishedCount > config.RequiredPlayers {
			s.logger.Error().
        Str("event", constant.TriggerAnswerFinish).
        Str("match_id", payload.MatchID).
        Msg("unexpected measurement count (too large)")
			return websocket.NewErrorPayload(), nil
		}

		// =====================================
		// ここから下は「最後の1人だけ」実行する処理
		// =====================================

		// プレイヤーが規定値の場合 → ラウンドフローを実行
		if err := s.roundFlowService.ProcessRoundResult(ctx, payload.MatchID); err != nil {
			s.logger.Error().
        Err(err).
        Str("event", constant.TriggerAnswerFinish).
        Str("match_id", payload.MatchID).
        Msg("ProcessRoundResult failed")
			return websocket.NewErrorPayload(), nil
		}

		return nil, nil

	// --- 時間切れ ---
	case constant.TriggerAnswerTimeout:
		var payload websocket.PlayerAnswerFinishedPayload
		if err := json.Unmarshal(messagePayload, &payload); err != nil {
			s.logger.Error().
				Err(err).
				Str("event", constant.TriggerAnswerTimeout).
				Str("match_id", payload.MatchID).
				Msg("failed to unmarshal payload")
			return websocket.NewErrorPayload(), nil
		}

		// プレイヤー認証
		loadPlayersCtx, cancelLoadPlayers := context.WithTimeout(ctx, 200*time.Millisecond)
		players, err := s.roundStateRepo.LoadMatchPlayers(loadPlayersCtx, payload.MatchID)
		cancelLoadPlayers()
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("event", constant.TriggerAnswerTimeout).
				Str("match_id", payload.MatchID).
				Msg("failed to load match players")
			return websocket.NewErrorPayload(), nil
		}
		if userID != players.Player1ID && userID != players.Player2ID {
      s.logger.Warn().
        Str("event", constant.TriggerAnswerTimeout).
        Str("match_id", payload.MatchID).
        Msg("unauthorized player")
			return websocket.NewErrorPayload(), nil
		}

		// 計測機能実行（各プレイヤーが1回だけ実行）
		if err := s.measurementRoundService.SaveMeasurement(ctx, payload.MatchID, userID, payload.MissCount, constant.TriggerAnswerTimeout); err != nil {
			// すでに実行済み（2回目の finish は正常扱いとして無視）
			if errors.Is(err, constant.ErrAlreadyFinished) {
        return nil, nil
			}
			s.logger.Error().
        Err(err).
        Str("event", constant.TriggerAnswerTimeout).
        Str("match_id", payload.MatchID).
        Msg("failed to SaveMeasurement (answer timeout)")
			return websocket.NewErrorPayload(), nil
		}

		// measurementFinishedCountを+1（全員が1回だけ実行）
		incrementCountCtx, cancelIncrement := context.WithTimeout(ctx, 200*time.Millisecond)
		measurementFinishedCount, err := s.roundStateRepo.IncrementMeasurementFinishCount(incrementCountCtx, payload.MatchID)
		cancelIncrement()
		if err != nil {
			return websocket.NewErrorPayload(), nil
		}

		// 両プレイヤーが揃うまで終了処理は実行しない（まだ各プレイヤー1回実行の領域）
		if measurementFinishedCount < config.RequiredPlayers {
			return nil, nil
		}

		// プレイヤー数が規定値を超えている場合 → 本来発生しない異常状態。
		if measurementFinishedCount > config.RequiredPlayers {
			s.logger.Error().
        Str("event", constant.TriggerAnswerTimeout).
        Str("match_id", payload.MatchID).
        Msg("unexpected measurement count (too large)")
			return websocket.NewErrorPayload(), nil
		}

		// =====================================
		// ここから下は「最後の1人だけ」実行する処理
		// =====================================

		// プレイヤーが規定値の場合 → ラウンドフローを実行
		if err := s.roundFlowService.ProcessRoundResult(ctx, payload.MatchID); err != nil {
			s.logger.Error().
        Err(err).
        Str("event", constant.TriggerAnswerTimeout).
        Str("match_id", payload.MatchID).
        Msg("ProcessRoundResult failed")
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
func (s *MatchRealtimeService) OnDisconnect(
	ctx context.Context,
	userID string,
	roomName string,
) {
	// best-effort で待機解除（例: 切断時）
	_ = s.matchQueueRepo.Cancel(context.Background(), userID)
}
