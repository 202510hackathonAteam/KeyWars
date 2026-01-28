package sql

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"
	"github.com/go-sql-driver/mysql"

	domainmodel "keywars/backend/internal/domain/model"
	"keywars/backend/internal/domain/repository"
	inframodel "keywars/backend/internal/infra/repository/sql/model"
)

var _ repository.UserRepository = (*userRepo)(nil)

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

// toRecord は domainmodel.User を inframodel.User に変換する関数。
func toRecord(u *domainmodel.User) *inframodel.User {
	if u == nil {
		return nil
	}
	return &inframodel.User{
		ID: u.ID,
		UserName: u.UserName,
		PasswordHash: u.PasswordHash,
	}
}

// toDomain は inframodel.User を domainmodel.User に変換します。
func toDomain(u *inframodel.User) *domainmodel.User {
	if u == nil {
		return nil
	}
	return &domainmodel.User{
		ID: u.ID,
		UserName: u.UserName,
		PasswordHash: u.PasswordHash,
	}
}

// Create は新規ユーザーをデータベースに登録する処理。
func (r *userRepo) Create(ctx context.Context, user *domainmodel.User) (string, error) {
	userRecord := toRecord(user)

	// データベース登録処理
	err := r.db.WithContext(ctx).Create(userRecord).Error

	// MySQL 固有のエラー判定処理
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		// user_name のユニーク制約違反を検出する処理
		if strings.Contains(strings.ToLower(mysqlErr.Message), "ux_users_user_name") {
			return "", errors.New("user_name already exists")
		}
	}

	// その他のエラー処理
	if err != nil {
		return "", err
	}

	return userRecord.ID, nil
}

// ExistsByUsername は、指定されたユーザー名のレコードが存在するかを確認する関数。
func (r *userRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	// 検索条件の設定とデータ取得
	var userRecord inframodel.User
	err := r.db.WithContext(ctx).
		Where("user_name = ?", username).
		Select("id").
		First(&userRecord).Error

	// レコード未存在時の処理
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}

	// その他のエラー処理
	if err != nil {
		return false, err
	}

	return true, nil
}

// FindByID は、指定されたユーザーIDに一致するユーザーを取得する関数。
func (r *userRepo) FindByID(ctx context.Context, userID string) (*domainmodel.User, error) {
	var userRecord inframodel.User
	err := r.db.WithContext(ctx).
		Where("id = ?", userID).
		First(&userRecord).Error

	// レコード未存在時の処理
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	// その他のエラー処理
	if err != nil {
		return nil, err
	}

	return toDomain(&userRecord), nil
}

// FindUserByUsername は、指定されたユーザー名に一致するユーザーを取得する関数。
func (r *userRepo) FindUserByUsername(ctx context.Context, username string) (*domainmodel.User, error) {
	var userRecord inframodel.User
	err := r.db.WithContext(ctx).
		Where("user_name = ?", username).
		First(&userRecord).Error

	// レコード未存在時の処理
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	// その他のエラー処理
	if err != nil {
		return nil, err
	}

	return toDomain(&userRecord), nil
}

// FindUserNameByUserID は、指定されたユーザーIDに一致するユーザー名を取得する関数。
func (r *userRepo) FindUserNameByUserID(ctx context.Context, userID string) (string, error) {
	var userName string
	err := r.db.WithContext(ctx).
		Table("users").
		Select("user_name").
		Where("id = ?", userID).
		Scan(&userName).Error

	if err != nil {
		return "", err
	}

	return userName, nil
}