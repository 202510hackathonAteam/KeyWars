package round

import (
	"context"
	"errors"
	"time"

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
	loadPlayersCtx, cancelLoadPlayers := context.WithTimeout(ctx, 200*time.Millisecond)
	players, err := s.roundStateRepo.LoadMatchPlayers(loadPlayersCtx, matchID)
	cancelLoadPlayers()
	if err != nil {
		return websocket.NewErrorPayload(), nil
	}
	checkFinishCtx1, cancelCheck1 := context.WithTimeout(ctx, 200*time.Millisecond)
	isPlayer1AnswerFinish, err := s.roundStateRepo.IsPlayerAnswerFinishFlagExists(checkFinishCtx1, matchID, players.Player1ID)
	cancelCheck1()
	if err != nil {
		return websocket.NewErrorPayload(), nil
	}
	checkFinishCtx2, cancelCheck2 := context.WithTimeout(ctx, 200*time.Millisecond)
	isPlayer2AnswerFinish, err := s.roundStateRepo.IsPlayerAnswerFinishFlagExists(checkFinishCtx2, matchID, players.Player2ID)
	cancelCheck2()
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
			measurementCtx, cancelMeasurement := context.WithTimeout(ctx, 600*time.Millisecond)
			err = s.measurementRoundService.SaveMeasurement(measurementCtx, matchID, p.userID, missCount, constant.TriggerServerForceFinish)
			cancelMeasurement()
			if err != nil {
				// すでに実行済み（2回目の finish は正常扱いとして無視）
				if errors.Is(err, constant.ErrAlreadyFinished) {
					continue
				}
				s.logger.Error().
					Err(err).
					Str("event", constant.TriggerServerForceFinish).
					Str("match_id", matchID).
					Msg("failed to SaveMeasurement (force finish)")
				return websocket.NewErrorPayload(), nil
			}
			// measurementFinishedCountを+1（SaveMeasurement が成功した時だけ）
			incrementCountCtx, cancelIncrement := context.WithTimeout(ctx, 200*time.Millisecond)
			currentFinishedCount, err := s.roundStateRepo.IncrementMeasurementFinishCount(incrementCountCtx, matchID)
			cancelIncrement()
			if err != nil {
				return websocket.NewErrorPayload(), nil
			}
			measurementFinishedCount = currentFinishedCount
		}
	}

	// 強制終了では、この時点で両プレイヤーの finish が揃っている必要がある。
	// 2になっていないのは未処理の残りがあるという意味で異常。
	if measurementFinishedCount != config.RequiredPlayers {
    s.logger.Error().
			Str("event", constant.TriggerServerForceFinish).
			Str("match_id", matchID).
			Msg("ForceFinish did not complete two players – logic error")
    return websocket.NewErrorPayload(), nil
	}

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
