package service

import (
	"context"

	"keywars/backend/internal/domain/repository"
)

// MatchService は、試合関連ユースケースのインターフェースの定義。
type MatchService interface {
	FrontendState(ctx context.Context, userID string) ([]byte, error)
}

// matchService は MatchService インターフェースの具象実装。
type matchService struct{
	roundStateRepo repository.RoundStateRepository
}

// NewMatchService は RoundStateRepository を受け取り、MatchServiceを生成。
func NewMatchService(
	roundStateRepo repository.RoundStateRepository,
) MatchService {
	return &matchService{
		roundStateRepo: roundStateRepo,
	}
}

// FrontendState は、ユーザーIDに紐づく試合に対するフロントエンド再構築用の
// 最新スナップショット（frontend_state）を取得するメソッド。
func (s *matchService) FrontendState(ctx context.Context, userID string) ([]byte, error) {
	// 試合参加チェック
	matchID, err := s.roundStateRepo.LoadUserActiveMatchID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// データ取得
	payloadBytes, err := s.roundStateRepo.LoadFrontendState(ctx, matchID)
	if err != nil {
		return nil, err
	}

	return payloadBytes, nil
}