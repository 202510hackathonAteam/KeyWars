package service

import (
	"context"
	"keywars/backend/internal/domain/repository"
)

type MatchService struct {
	queue repository.MatchQueueRepository
	round repository.RoundStateRepository
}

func NewMatchService(q repository.MatchQueueRepository, r repository.RoundStateRepository) *MatchService {
	return &MatchService{queue: q, round: r}
}

func (s *MatchService) JoinQueue(ctx context.Context, uid string, nowMs int64) error {
	return s.queue.Enqueue(ctx, uid, nowMs)
}

func (s *MatchService) TryMatch(ctx context.Context) (mid, u1, u2 string, err error) {
	u1, u2, mid, _, err = s.queue.DequeuePairAndInitMatch(ctx)
	return
}
