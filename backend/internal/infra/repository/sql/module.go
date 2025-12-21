package sql

import "go.uber.org/fx"

var Module = fx.Module(
	"sql",
	fx.Provide(
		// SQL repos
		NewUserRepo,
		NewPromptRepositorySQL,
	),
)