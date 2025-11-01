package repository

import "context"

type AnswerApplyArg struct {
	MID         string
	OppUID      string
	NewOppLP    int64
	NextDeckIdx int64
	NowMs       int64
	EventFields map[string]string
}

type RoundStateRepository interface {
	CreateMeta(ctx context.Context, mid, u1, u2 string, nowMs int64) error
	Start(ctx context.Context, mid string) error
	Finish(ctx context.Context, mid, winnerUID string) error
	SaveDeckOnce(ctx context.Context, mid string, deckJSON []string) error
	GetDeckItem(ctx context.Context, mid string, deckIdx int64) (string, error)
	ApplyAnswer(ctx context.Context, a AnswerApplyArg) (eventID string, turn int64, err error)
}
