package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	domain "keywars/backend/internal/domain/port"

	"github.com/gorilla/websocket"
)

// roomTicket を検証して UID / Room を取り出す疎結合IF
type TicketVerifier interface {
	VerifyRoomTicket(ctx context.Context, token string) (uid string, room string, err error)
}

type Handler struct {
	Hub      *Hub
	Svc      domain.RealtimeService
	Verifier TicketVerifier
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: 本番は許可するオリジンを限定
		return true
	},
}

type inMsg struct {
	Type string          `json:"type"`
	Body json.RawMessage `json:"body"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// --- 認証（roomTicket 必須） ---
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}
	uid, room, err := h.Verifier.VerifyRoomTicket(ctx, token)
	if err != nil || uid == "" || room == "" {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// --- Upgrade ---
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()

	cli := &client{
		uid:  uid,
		room: room,
		send: make(chan []byte, 64),
	}

	// --- Join ---
	if err := h.Hub.Join(ctx, room, cli); err != nil {
		if errors.Is(err, ErrRoomFull) {
			_ = conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "room full"),
				time.Now().Add(2*time.Second))
		}
		return
	}
	defer func() {
		_ = h.Hub.Leave(ctx, cli)
		h.Svc.OnDisconnect(ctx, uid, room)
	}()

	// --- 接続直後の初期メッセージ（必要なら） ---
	if rep, err := h.Svc.OnConnect(ctx, uid, room); err == nil && rep != nil {
		_ = cli.SendJSON(ctx, rep)
	}

	// --- writer goroutine（Ping込み） ---
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(25 * time.Second) // Ping間隔
		defer ticker.Stop()

		for {
			select {
			case msg, ok := <-cli.send:
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if !ok {
					_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				w, err := conn.NextWriter(websocket.TextMessage)
				if err != nil {
					return
				}
				if _, err := w.Write(msg); err != nil {
					_ = w.Close()
					return
				}
				if err := w.Close(); err != nil {
					return
				}

			case <-ticker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	// --- reader ループ（Pong/ReadDeadline込み） ---
	conn.SetReadLimit(1 << 20)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	conn.SetCloseHandler(func(code int, text string) error {
		// 既定動作（Close送出）に任せる。必要ならログを追加。
		return nil
	})

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			// 切断（defer で OnDisconnect 済み）
			return
		}
		var m inMsg
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		rep, err := h.Svc.OnMessage(ctx, uid, room, m.Type, m.Body)
		if err == nil && rep != nil {
			_ = cli.SendJSON(ctx, rep)
		}
	}
}
