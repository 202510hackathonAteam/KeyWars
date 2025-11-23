package round

import (
	"context"

	"keywars/backend/internal/domain/repository"
)

type ForceFinishService interface {
	ForceFinish(ctx context.Context, matchID string)
}

type forceFinishService struct {
	roundStateRepo			 		repository.RoundStateRepository
	roundFlowService 		 		*RoundFlowService
	measurementRoundService *MeasurementRoundService
}

// NewForceFinishService は、
// 新しい forceFinishService インスタンスを生成して返すコンストラクタ。
func NewForceFinishService(
	roundStateRepo repository.RoundStateRepository,
	roundFlowService *RoundFlowService,
	measurementRoundService *MeasurementRoundService,
) *forceFinishService {
	return &forceFinishService{
		roundStateRepo:				 	 roundStateRepo,
		roundFlowService:				 roundFlowService,
		measurementRoundService: measurementRoundService,
	}
}

// 途中
// ForceFinish は、サーバ側の内部ロジックによりラウンドを強制的に終了させるためのメソッド。
func (s *forceFinishService) ForceFinish(ctx context.Context, matchID string) {
	return
}