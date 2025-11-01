package repository

import "context"

type MatchQueueRepository interface {
	Enqueue(ctx context.Context, userID string, enqueueAtMs int64) error
	Cancel(ctx context.Context, userID string) error
	Score(ctx context.Context, userID string) (float64, error)
	DequeuePairAndInitMatch(ctx context.Context) (u1, u2, mid, lockToken string, err error)
	ReleaseLock(ctx context.Context, token string) error
}
