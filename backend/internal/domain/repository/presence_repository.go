package repository

import "context"

type PresenceRepository interface {
	SetOnline(ctx context.Context, uid string, nowMs int64) error
	Heartbeat(ctx context.Context, uid string, nowMs int64) error
	SetIngame(ctx context.Context, uid, mid string, nowMs int64) error
	Disconnect(ctx context.Context, uid string, nowMs int64) error
	Get(ctx context.Context, uid string) (map[string]string, error)
}
