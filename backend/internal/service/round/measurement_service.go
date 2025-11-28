package round

import (
	"context"
	"time"
	"fmt"

	"keywars/backend/internal/domain/repository"
	"keywars/backend/internal/domain/constant"
)

// MeasurementRoundService は、ラウンド中の各プレイヤーの解答状況を監視し、
// 制限時間の到達や全プレイヤーの回答完了を検知して、適切なタイミングで
// ラウンド終了処理を実行するサービス。
type MeasurementRoundService struct {
	roundStateRepo 	 repository.RoundStateRepository
}

// NewMeasurementRoundService は MeasurementRoundService のコンストラクタ。
func NewMeasurementRoundService(roundStateRepo repository.RoundStateRepository) *MeasurementRoundService {
	return &MeasurementRoundService{
		roundStateRepo:   roundStateRepo,
	}
}

// SaveMeasurement は、1プレイヤー分の計測結果（finish 時刻・ミス数・イベント）を保存し、
// 累計ミス数を加算する処理を行うメソッド。
func (s *MeasurementRoundService) SaveMeasurement(ctx context.Context, matchID, userID string, missCount int64, trigger string) error {	
	// 冪等性チェック（finish 済みなら弾く）
	finishFlagCtx, cancelFinishFlag := context.WithTimeout(ctx, 300*time.Millisecond)
	isFirstFinished, err := s.roundStateRepo.RegisterPlayerAnswerFinishFlag(finishFlagCtx, matchID, userID)
	cancelFinishFlag()
	if err != nil {
		return err
	}
	if !isFirstFinished {
		return constant.ErrAlreadyFinished
	}

	// 状態読み取り
	loadStateCtx, cancelLoadState := context.WithTimeout(ctx, 300*time.Millisecond)
	state, err := s.roundStateRepo.LoadMatchState(loadStateCtx, matchID)
	cancelLoadState()
	if err != nil {
		return err
	}

	// 終了時刻の決定
	var finishAtMs int64
	nowAtMs := time.Now().UnixMilli()
	switch trigger {
	case constant.TriggerAnswerFinish:
		if state.RoundEndAtMs < nowAtMs {
			finishAtMs = state.RoundEndAtMs
		} else {
			finishAtMs = nowAtMs
		}
	case constant.TriggerAnswerTimeout:
		finishAtMs = state.RoundEndAtMs
	case constant.TriggerServerForceFinish:
		finishAtMs = state.RoundEndAtMs
	default:
		return fmt.Errorf("unexpected finish trigger: %s", trigger)
	}

	// finish イベントの保存（Stream に追加）
	storeEventCtx, cancelStoreEvent := context.WithTimeout(ctx, 300*time.Millisecond)
	err = s.roundStateRepo.StoreFinishEvent(storeEventCtx, matchID, userID, state.Round, missCount, finishAtMs)
	cancelStoreEvent()
	if err != nil {
		return err
	}

	// プレイヤー判定
	loadPlayersCtx, cancelLoadPlayers := context.WithTimeout(ctx, 300*time.Millisecond)
	players, err := s.roundStateRepo.LoadMatchPlayers(loadPlayersCtx, matchID)
	cancelLoadPlayers()
	if err != nil {
		return err
	}

	var playerField string
	switch userID {
	case players.Player1ID:
		playerField = "player1"
	case players.Player2ID:
		playerField = "player2"
	default:
		return fmt.Errorf("userID %s is not in this match", userID)
	}

	// 累計ミス数の更新（atomic increment）
	updateMissCtx, cancelUpdateMiss := context.WithTimeout(ctx, 200*time.Millisecond)
	err = s.roundStateRepo.UpdateTotalMissCount(updateMissCtx, matchID, playerField, missCount)
	cancelUpdateMiss()
	if err != nil {
		return err
	}

	return nil
}
