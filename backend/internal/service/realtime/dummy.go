package realtime

// 動作確認用、あとで消す

import (
	"context"
)

type Service struct{}

func New() *Service { return &Service{} }

func (s *Service) OnConnect(ctx context.Context, uid, room string) (any, error) {
	return map[string]any{"type": "welcome", "uid": uid, "room": room}, nil
}
func (s *Service) OnMessage(ctx context.Context, uid, room string, typ string, payload []byte) (any, error) {
	// シンプルにオウム返し
	return map[string]any{"type": "echo", "echoType": typ, "payload": string(payload)}, nil
}
func (s *Service) OnDisconnect(ctx context.Context, uid, room string) {}
