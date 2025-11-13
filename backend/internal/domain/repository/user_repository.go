package repository

import (
	"context"

	domainmodel "keywars/backend/internal/domain/model"
)

// UserRepository はユーザー情報の取得・保存などを行うインターフェース
type UserRepository interface{
	Create(ctx context.Context, user *domainmodel.User) error
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	FindByID(ctx context.Context, userID string) (*domainmodel.User, error)
	FindUserByUsername(ctx context.Context, username string) (*domainmodel.User, error)
}