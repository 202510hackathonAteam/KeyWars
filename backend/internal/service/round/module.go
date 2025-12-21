package round

import "go.uber.org/fx"

var Module = fx.Module(
	"round",
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