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
	websocketHub *websocket.Hub,
	logger *zerolog.Logger,
	deckGeneratorService *round.DeckGeneratorService,
	roundFlowService *round.RoundFlowService,
	measurementRoundService *round.MeasurementRoundService,
) *MatchRealtimeService {
	return &MatchRealtimeService{
		matchQueueRepo: 	 			 matchQueueRepo,
		roundStateRepo: 	 			 roundStateRepo,
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
// 接続直後のクライアントを現在の試合状態に同期し、
// 必要に応じて復帰通知・通常歓迎メッセージのいずれかを返す。
func (s *MatchRealtimeService) OnConnect(
	ctx context.Context,
	userID string,
	roomName string,
) (any, error) {
	nowMs := time.Now().UnixMilli()

	// --- 1) 途中復帰できるか？ ---
	if payload, ok := s.tryRestore(ctx, userID, nowMs); ok {
		return payload, nil
	}

	// --- 2) それ以外（通常接続） ---
	return websocket.NewWelcomePayload(userID), nil
}

// OnMessage は、クライアントから受信した WebSocket メッセージを処理する。
// メッセージの `type` に応じて待機キュー操作や通知を行う。
func (s *MatchRealtimeService) OnMessage(
	ctx context.Context,
	userID string,
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
			s.logger.Error().
        Err(err).
        Str("event", constant.EventQueueJoin).
        Msg("failed to enqueue user into match queue")
			return nil, err
		}

		// クライアントに成功レスポンスを返す
		return websocket.NewQueueJoinedPayload(currentTimeMs), nil

	// --- マッチ待機キャンセル ---
	case "queue.left":
		// best-effort：失敗しても致命的ではない
		_ = s.matchQueueRepo.Cancel(ctx, userID)

		return websocket.NewQueueLeftPayload(), nil

	// --- ラウンド完了 ---
	case constant.TriggerAnswerFinish:
		var payload websocket.PlayerAnswerFinishedPayload
		if err := json.Unmarshal(messagePayload, &payload); err != nil {
			s.logger.Error().
				Err(err).
				Str("event", constant.TriggerAnswerFinish).
				Str("match_id", payload.MatchID).
				Msg("failed to unmarshal payload")
			return nil, err
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
			return nil, err
		}
		if userID != players.Player1ID && userID != players.Player2ID {
      s.logger.Warn().
        Str("event", constant.TriggerAnswerFinish).
        Str("match_id", payload.MatchID).
        Msg("unauthorized player")
			return nil, err
		}

		// 計測機能実行（各プレイヤーが1回だけ実行）
		measurementCtx, cancelMeasurement := context.WithTimeout(ctx, 600*time.Millisecond)
		err = s.measurementRoundService.SaveMeasurement(measurementCtx, payload.MatchID, userID, payload.MissCount, constant.TriggerAnswerFinish)
		cancelMeasurement()
		if err != nil {
			// すでに実行済み（2回目の finish は正常扱いとして無視）
			if errors.Is(err, constant.ErrAlreadyFinished) {
        return nil, nil
			}
			s.logger.Error().
        Err(err).
        Str("event", constant.TriggerAnswerFinish).
        Str("match_id", payload.MatchID).
        Msg("failed to SaveMeasurement (answer finish)")
			return nil, err
		}

		// measurementFinishedCountを+1（全員が1回だけ実行）
		incrementCountCtx, cancelIncrement := context.WithTimeout(ctx, 200*time.Millisecond)
		measurementFinishedCount, err := s.roundStateRepo.IncrementMeasurementFinishCount(incrementCountCtx, payload.MatchID)
		cancelIncrement()
		if err != nil {
			return nil, err
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
			return nil, err
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
			return nil, err
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
			return nil, err
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
			return nil, err
		}
		if userID != players.Player1ID && userID != players.Player2ID {
      s.logger.Warn().
        Str("event", constant.TriggerAnswerTimeout).
        Str("match_id", payload.MatchID).
        Msg("unauthorized player")
			return nil, err
		}

		// 計測機能実行（各プレイヤーが1回だけ実行）
		measurementCtx, cancelMeasurement := context.WithTimeout(ctx, 600*time.Millisecond)
		err = s.measurementRoundService.SaveMeasurement(measurementCtx, payload.MatchID, userID, payload.MissCount, constant.TriggerAnswerTimeout)
		cancelMeasurement()
		if err != nil {
			// すでに実行済み（2回目の finish は正常扱いとして無視）
			if errors.Is(err, constant.ErrAlreadyFinished) {
        return nil, nil
			}
			s.logger.Error().
        Err(err).
        Str("event", constant.TriggerAnswerTimeout).
        Str("match_id", payload.MatchID).
        Msg("failed to SaveMeasurement (answer timeout)")
			return nil, err
		}

		// measurementFinishedCountを+1（全員が1回だけ実行）
		incrementCountCtx, cancelIncrement := context.WithTimeout(ctx, 200*time.Millisecond)
		measurementFinishedCount, err := s.roundStateRepo.IncrementMeasurementFinishCount(incrementCountCtx, payload.MatchID)
		cancelIncrement()
		if err != nil {
			return nil, err
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
			return nil, err
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
			return nil, err
		}

		return nil, nil

	// --- その他（未定義メッセージ） ---
	default:
		// 他のゲーム内メッセージを追加する場合はここに分岐を書く
		return nil, nil
	}
}

// OnDisconnect は WebSocket 切断という事実を受け取り、
// 待機キューからの除外を best-effort で行う。
// 失敗時もエラーは返さず、内部でログ出力のみ行う。
func (s *MatchRealtimeService) OnDisconnect(
	ctx context.Context,
	userID string,
) {
	if err := s.matchQueueRepo.Cancel(ctx, userID); err != nil {
		s.logger.Warn().
			Err(err).
			Str("event", constant.EventMatchDisconnect).
			Str("user_id", userID).
			Msg("failed to cancel match queue on disconnect")
	}
}

// tryRestore は、WebSocket 接続時の途中復帰判定および復帰処理。
// 対象ユーザーが進行中試合に参加している場合のみ、ルーム移動と
// 復帰用ペイロード生成を行い、復帰不可の場合は処理を行わない。
func (s *MatchRealtimeService) tryRestore(
	ctx context.Context,
	userID string,
	nowMs int64,
) (any, bool) {
	// 1) ユーザーが参加中の試合を引く
	matchID, err := s.roundStateRepo.LoadUserActiveMatchID(ctx, userID)
	if err != nil {
		return nil, false
	}

	// 2) 試合の state が生きているか確認
	restoredState, err := s.roundStateRepo.LoadMatchState(ctx, matchID)
	if err != nil || restoredState == nil {
		return nil, false
	}

	// ここまで来た場合、途中復帰できる
	// 3) 接続をマッチルームへ移動
	userRoom := "user:" + userID
	matchRoom := "match:" + matchID

	conns := s.websocketHub.Members(userRoom)
	if len(conns) > 0 {
		_ = s.websocketHub.Move(conns[0], matchRoom)
	}

	// 4) 復帰用ペイロードを返す
	return websocket.NewMatchRestorePayload(
		matchID,
		websocket.MatchRestoreState{
			Round:            restoredState.Round,
			Player1Lifepoint: restoredState.Player1Lifepoint,
			Player2Lifepoint: restoredState.Player2Lifepoint,
		},
	), true
}

