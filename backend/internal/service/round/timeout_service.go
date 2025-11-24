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
				state, _ := s.roundStateRepo.LoadMatchState(ctx, matchID)
				if !state.Player1AnswerFinished || !state.Player2AnswerFinished {
					// 強制終了として計測機能処理を発動
					s.forceFinishService.ForceFinish(ctx, matchID)
				}
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
