package sql

import (
	"gorm.io/gorm"
	"keywars/backend/internal/domain/repository"
)

// Repos は、domain 層で定義された各リポジトリインターフェースを
// GORM を用いて実装した構造体をまとめた集約の定義。
// アプリ全体で使用するリポジトリ群を一括で初期化・注入する際に利用。
type Repos struct {
	User repository.UserRepository
	// 下に他のリポジトリを追加していく
}

// New は、*gorm.DB を受け取り Repos 構造体を生成。
// 各リポジトリの GORM 実装を初期化して返却。
func New(db *gorm.DB) *Repos {
	return &Repos{
		User: NewUserRepo(db),
		// 下に他のリポジトリを追加していく
	}
}