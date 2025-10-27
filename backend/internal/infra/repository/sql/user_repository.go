package sql

import (
	"gorm.io/gorm"
	"keywars/backend/internal/domain/repository"
)

// userRepo は、domain 層の UserRepository を GORM を用いて実装した構造体の定義。
// データベース操作を担当し、domain 層からの要求を SQL に変換して処理。
type userRepo struct{
	db *gorm.DB
}

// ダミー実装
// NewUserRepo は、*gorm.DB を受け取り userRepo を生成。
// domain/repository.UserRepository インターフェースを実装した具体型を返却。
func NewUserRepo(db *gorm.DB) repository.UserRepository {
	return &userRepo{db: db}
}