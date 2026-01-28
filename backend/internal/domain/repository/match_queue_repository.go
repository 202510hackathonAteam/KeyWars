package repository

import "context"

type MatchQueueRepository interface {
	Enqueue(ctx context.Context, userID string, enqueueAtMs int64) error
	Cancel(ctx context.Context, userID string) error
	DequeuePairAndInitMatch(ctx context.Context) (user1ID, user2ID, matchID string, err error)
}
