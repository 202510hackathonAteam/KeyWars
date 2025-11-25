package round

import (
	"context"

	"keywars/backend/internal/domain/repository"
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

// 途中
// SaveMeasurement は、1 プレイヤー分の計測データを保存するメソッド。
func (s *MeasurementRoundService) SaveMeasurement(ctx context.Context, matchID, userID string, missCount int64) error {
	// 計測機能の実行
	return nil
}