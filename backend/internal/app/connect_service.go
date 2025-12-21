package app

import "keywars/backend/internal/service/round"

// ConnectForceFinishServices は、ForceFinishService に RoundFlowService を紐づけるための依存関係接続処理。
func ConnectForceFinishServices(
	forceFinishService round.ForceFinishService,
	roundFlowService *round.RoundFlowService,
) {
	forceFinishService.SetRoundFlowService(roundFlowService)
}