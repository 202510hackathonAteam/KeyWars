package round

import (
	"context"

	"github.com/rs/zerolog"

	"keywars/backend/internal/transport/websocket"
	"keywars/backend/internal/domain/repository"
	"keywars/backend/internal/config"
	"keywars/backend/internal/domain/constant"
	"keywars/backend/internal/domain/types"
)

type ForceFinishService interface {
	ForceFinish(ctx context.Context, matchID string) (any, error)
}

type forceFinishService struct {
	roundStateRepo			 		repository.RoundStateRepository
	logger 							 		*zerolog.Logger
	roundFlowService 		 		*RoundFlowService
	measurementRoundService *MeasurementRoundService
}

// NewForceFinishService は、
// 新しい forceFinishService インスタンスを生成して返すコンストラクタ。
func NewForceFinishService(
	roundStateRepo repository.RoundStateRepository,
	logger *zerolog.Logger,
	roundFlowService *RoundFlowService,
	measurementRoundService *MeasurementRoundService,
) *forceFinishService {
	return &forceFinishService{
		roundStateRepo:				 	 roundStateRepo,
		logger:									 logger,
		roundFlowService:				 roundFlowService,
		measurementRoundService: measurementRoundService,
	}
}

// ForceFinish は、サーバ側ロジックによりラウンドを強制終了させるメソッド。
// すでに完了済みのプレイヤーはスキップし、未完了プレイヤーのみ計測を実行する。
// measurementFinishedCount により、ラウンド終了処理（ProcessRoundResult）は
// 最後の 1 回だけ実行される。
func (s *forceFinishService) ForceFinish(ctx context.Context, matchID string) (any, error) {
	// プレイヤー情報と finish 状態をロード
	players, err := s.roundStateRepo.LoadMatchPlayers(ctx, matchID)
	if err != nil {
		return websocket.NewErrorPayload(), nil
	}
	isPlayer1AnswerFinish, err := s.roundStateRepo.IsPlayerAnswerFinishFlagExists(ctx, matchID, players.Player1ID)
	if err != nil {
		return websocket.NewErrorPayload(), nil
	}
	isPlayer2AnswerFinish, err := s.roundStateRepo.IsPlayerAnswerFinishFlagExists(ctx, matchID, players.Player2ID)
	if err != nil {
		return websocket.NewErrorPayload(), nil
	}

	// 両方終わっている場合終了
	if isPlayer1AnswerFinish && isPlayer2AnswerFinish {
		return nil, nil
	}

	// 未完了プレイヤーの計測を強制実行
	var missCount int64

	playersToCheck := []struct {
    finished bool
    userID   string
	}{
		{isPlayer1AnswerFinish, players.Player1ID},
		{isPlayer2AnswerFinish, players.Player2ID},
	}

	var measurementFinishedCount types.PlayerCount

	for _, p := range playersToCheck {
		if !p.finished {
			if err := s.measurementRoundService.SaveMeasurement(
				ctx, matchID, p.userID, missCount, constant.TriggerServerForceFinish,
			); err != nil {
				s.logger.Error().
					Err(err).
					Str("event", constant.TriggerServerForceFinish).
					Str("match_id", matchID).
					Msg("failed to SaveMeasurement (force finish)")
				return websocket.NewErrorPayload(), nil
			}
			// measurementFinishedCountを+1（全員が1回だけ実行）
			currentFinishedCount, err := s.roundStateRepo.IncrementMeasurementFinishCount(ctx, matchID)
			if err != nil {
				return websocket.NewErrorPayload(), nil
			}
			measurementFinishedCount = currentFinishedCount
		}
	}

	// 両プレイヤーが揃うまで終了処理は実行しない（まだ各プレイヤー1回実行の領域）
	if measurementFinishedCount < config.RequiredPlayers {
		return nil, nil
	}

	// プレイヤー数が規定値を超えている場合 → 本来発生しない異常状態。
	if measurementFinishedCount > config.RequiredPlayers {
		s.logger.Error().
			Str("event", constant.TriggerServerForceFinish).
			Str("match_id", matchID).
			Msg("unexpected measurement count (too large)")
		return websocket.NewErrorPayload(), nil
	}

	// =====================================
	// ここから下は「最後の1人だけ」実行する処理
	// =====================================

	// プレイヤーが規定値の場合 → ラウンドフローを実行
	if err := s.roundFlowService.ProcessRoundResult(ctx, matchID); err != nil {
		s.logger.Error().
			Err(err).
			Str("event", constant.TriggerServerForceFinish).
			Str("match_id", matchID).
			Msg("ProcessRoundResult failed")
		return websocket.NewErrorPayload(), nil
	}

	return nil, nil
}
