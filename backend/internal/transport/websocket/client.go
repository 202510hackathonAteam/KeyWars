package websocket

import (
	"context"
	"encoding/json"
	"sync"

	domain "keywars/backend/internal/domain/port"
)

// client は1接続=1セッションの論理表現。
// 送信は send チャネルに積み、writer goroutine がWebSocketへ書き出す。
type client struct {
	uid       string
	room      string
	send      chan []byte
	closeOnce sync.Once
}

var _ domain.ClientConn = (*client)(nil)

func (c *client) UID() string  { return c.uid }
func (c *client) Room() string { return c.room }

// SendJSON は非同期（内部バッファへ積む）。満杯時の扱いは「捨てる」方針。
// ※ バックプレッシャの方針は必要に応じて差し替えてOK。
func (c *client) SendJSON(_ context.Context, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	select {
	case c.send <- b:
	default:
		// バッファ満杯→ドロップ（またはキック等の方針に変更してOK）
	}
	return nil
}

func (c *client) Close() error {
	c.closeOnce.Do(func() { close(c.send) })
	return nil
}
