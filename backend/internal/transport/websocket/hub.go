package websocket

import (
	"context"
	"errors"
	"sync"

	domain "keywars/backend/internal/domain/port"
)

var ErrRoomFull = errors.New("room full")

// Hub は room 単位の接続を管理し、ブロードキャストを提供する。
// goroutine-safe。
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[domain.ClientConn]struct{}
}

var _ domain.Broadcaster = (*Hub)(nil)

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]map[domain.ClientConn]struct{})}
}

func (h *Hub) Join(_ context.Context, room string, cc domain.ClientConn) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.rooms[room]; !ok {
		h.rooms[room] = make(map[domain.ClientConn]struct{})
	}
	// 2名制限
	if len(h.rooms[room]) >= 2 {
		return ErrRoomFull
	}
	h.rooms[room][cc] = struct{}{}
	return nil
}

func (h *Hub) Leave(_ context.Context, cc domain.ClientConn) error {

	var needClose bool

	h.mu.Lock()
	room := cc.Room()
	if room != "" {
		if m, ok := h.rooms[room]; ok {
			if _, exists := m[cc]; exists {
				delete(m, cc)
				needClose = true
			}
			if len(m) == 0 {
				delete(h.rooms, room)
			}
		}
	}
	h.mu.Unlock()

	if needClose {
		_ = cc.Close() // 送信チャネルを安全に閉じる（ロック外）
	}
	return nil
}

func (h *Hub) Broadcast(ctx context.Context, room string, v any) (failed int, err error) {
	h.mu.RLock()
	m, ok := h.rooms[room]
	if !ok {
		h.mu.RUnlock()
		return 0, nil
	}
	conns := make([]domain.ClientConn, 0, len(m))
	for cc := range m {
		conns = append(conns, cc)
	}
	h.mu.RUnlock()

	for _, cc := range conns {
		select {
		case <-ctx.Done():
			return failed, ctx.Err()
		default:
		}
		if err := cc.SendJSON(ctx, v); err != nil {
			failed++
		}
	}
	return failed, nil
}
