package round

// ConnectForceFinishServices は、ForceFinishService に RoundFlowService を紐づけるための依存関係接続処理。
func ConnectForceFinishServices(
	forceFinishService ForceFinishService,
	roundFlowService *RoundFlowService,
) {
	forceFinishService.SetRoundFlowService(roundFlowService)
}