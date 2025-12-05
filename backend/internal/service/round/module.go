package round

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(
		NewDeckGeneratorService,
		NewNextRoundService,
		NewMeasurementRoundService,
		NewLifepointService,
		NewMatchJudgeService,
		NewRoundFlowService,
		NewForceFinishService,
		NewTimeoutRoundService,
	),
)