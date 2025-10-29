package sql

import (
	"strings"
	"gorm.io/gorm"
	"github.com/google/uuid"
	"keywars/backend/internal/domain/entity"
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

// Create は、Userエンティティをデータベースに新規登録。
// UUIDの重複が発生した場合は、新しいUUIDを再生成して再試行する。
func (repo *userRepo) Create(user *entity.User) error {
	// ID が未設定の場合のみ UUID を自動生成
	for {
		repo.ensureUUID(user)

		// INSERT（CREATE）処理を実行
		err := repo.db.Create(user).Error
		// 成功した場合は終了
		if err == nil {
			return nil
		}

		// UUID 重複による一意制約エラーの場合のみ、再試行
		if isDuplicateKeyError(err) {
			user.ID = ""
			continue
		}

		// その他のエラーはそのまま返却
		return err
	}
}

// ensureUUID は、ユーザーIDが空の場合にUUIDを自動生成して設定。
func (repo *userRepo) ensureUUID(user *entity.User) {
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
}

// isDuplicateKeyError は、MySQLの一意制約違反エラーかどうかを判定。
func isDuplicateKeyError(err error) bool {
	return strings.Contains(err.Error(), "Duplicate entry")
}
