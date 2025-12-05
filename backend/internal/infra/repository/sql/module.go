package sql

import "go.uber.org/fx"

var Module = fx.Options(
	fx.Provide(
		// SQL repos
		NewUserRepo,
		NewPromptRepositorySQL,
	),
)