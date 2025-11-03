package sql

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"github.com/go-sql-driver/mysql"

	"keywars/backend/internal/domain/repository"
	"keywars/backend/internal/infra/repository/sql/model"
)

// userRepo は、domain 層の UserRepository を GORM を用いて実装した構造体の定義。
// データベース操作を担当し、domain 層からの要求を SQL に変換して処理。
type userRepo struct{
	db *gorm.DB
}

// NewUserRepo は、*gorm.DB を受け取り userRepo を生成。
// domain/repository.UserRepository インターフェースを実装した具体型を返却。
func NewUserRepo(db *gorm.DB) repository.UserRepository {
	return &userRepo{db: db}
}

// Create は新規ユーザーをデータベースに登録する処理。
func (r *userRepo) Create(ctx context.Context, user *model.User) error {
	// コンテキストを付与してユーザー情報を登録する処理。
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		// MySQL固有のエラー型を判定する処理。
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			// user_name のユニーク制約違反を検出する処理。
			if strings.Contains(strings.ToLower(mysqlErr.Message), "ux_users_user_name") {
        return errors.New("user_name already exists")
			}
		}
		return err
	}
	return nil
}
