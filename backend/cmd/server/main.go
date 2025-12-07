package main

import (
	"go.uber.org/fx"

	"keywars/backend/internal/di"
)

// main は、DI モジュールを使ってアプリケーションを起動。
func main() {
	fx.New(di.Module).Run()
}
