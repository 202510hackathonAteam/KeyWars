package round

import (
	"context"
	"time"

	"keywars/backend/internal/domain/repository"
)

// TimeoutRoundService は、ラウンドの制限時間を監視し、
// 時間切れ時にラウンド終了処理を自動発火させるサービス。
type TimeoutRoundService struct {
	roundStateRepo 	 	 repository.RoundStateRepository
	forceFinishService ForceFinishService
}

// NewTimeoutRoundService は TimeoutRoundService のコンストラクタ。
func NewTimeoutRoundService(roundStateRepo repository.RoundStateRepository, forceFinishService ForceFinishService) *TimeoutRoundService {
	return &TimeoutRoundService{
		roundStateRepo:			roundStateRepo,
		forceFinishService: forceFinishService,
	}
}

// ScheduleRoundTimeoutCheck は、ラウンドの制限時間(deadlineMs)を監視し、
// 締切に到達した時点で両プレイヤーの回答状況を確認するメソッド。
// 
// 締切に到達した時点で一度だけ両プレイヤーの回答状況を確認し、
// 必要と判断された場合は試合の強制終了を発動する。
func (s *TimeoutRoundService) ScheduleRoundTimeoutCheck(ctx context.Context, matchID string, deadlineMs int64) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	// 常駐ループ開始
	for {
		select {
		// 現在時刻を取得し、ラウンドの締切（deadlineMs）を超えているか判定
		case <-ticker.C:
			now := time.Now().UnixMilli()
			if now >= deadlineMs {
				if s.shouldTriggerForceFinish(ctx, matchID) {
					s.forceFinishService.ForceFinish(context.Background(), matchID)
				}
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// shouldTriggerForceFinish は、ラウンドのタイムアウト到達時に
// 「試合を強制終了すべきか」を判定するためのフェイルセーフ判定メソッド。
func (s *TimeoutRoundService) shouldTriggerForceFinish(
	ctx context.Context,
	matchID string,
) bool {
	loadStateCtx, cancelLoadState := context.WithTimeout(ctx, 300*time.Millisecond)
	players, err := s.roundStateRepo.LoadMatchPlayers(loadStateCtx, matchID)
	cancelLoadState()
	if loadStateCtx.Err() != nil || err != nil {
		return true
	}
	checkFinishCtx1, cancelCheck1 := context.WithTimeout(ctx, 200*time.Millisecond)
	isPlayer1AnswerFinish, err := s.roundStateRepo.IsPlayerAnswerFinishFlagExists(
		checkFinishCtx1, matchID, players.Player1ID,
	)
	cancelCheck1()
	if checkFinishCtx1.Err() != nil || err != nil {
		return true
	}
	checkFinishCtx2, cancelCheck2 := context.WithTimeout(ctx, 200*time.Millisecond)
	isPlayer2AnswerFinish, err := s.roundStateRepo.IsPlayerAnswerFinishFlagExists(
		checkFinishCtx2, matchID, players.Player2ID,
	)
	cancelCheck2()
	if checkFinishCtx2.Err() != nil || err != nil {
		return true
	}

	// 回答未完了のため、強制終了を発動する条件に該当
	if !isPlayer1AnswerFinish || !isPlayer2AnswerFinish {
		return true
	}
	return false
}
