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

const (
	readLimit    = 1 << 20
	pongWait     = 60 * time.Second
	writeWait    = 10 * time.Second
	pingInterval = 25 * time.Second
	sendBufSize  = 256
)

type TicketVerifier interface {
	VerifyRoomTicket(ctx context.Context, token string) (uid, room string, err error)
}

type Handler struct {
	Hub      *Hub
	Svc      domain.RealtimeService
	Verifier TicketVerifier
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:    1024,
	WriteBufferSize:   1024,
	EnableCompression: true,
	CheckOrigin:       func(r *http.Request) bool { return true }, // 本番は限定
}

type inMsg struct {
	Type string          `json:"type"`
	Body json.RawMessage `json:"body"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// --- 認証（roomTicket 必須） ---
	if h.Verifier == nil {
		http.Error(w, "server verifier not configured", http.StatusServiceUnavailable)
		return
	}
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
		log.Println("[ws] upgrade error:", err)
		return
	}
	// conn.Close() は writer 側で行う（CloseMessage送信のため）

	cli := &client{
		uid:  uid,
		room: room,
		send: make(chan []byte, sendBufSize),
	}

	// --- Join ---
	if h.Hub != nil {
		if err := h.Hub.Join(ctx, room, cli); err != nil {
			if errors.Is(err, ErrRoomFull) {
				_ = conn.WriteControl(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "room full"),
					time.Now().Add(writeWait))
			}
			_ = conn.Close()
			return
		}
	}

	// --- writer goroutine（Ping込み）を先に起動 ---
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer conn.Close()

		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()

		for {
			select {
			case msg, ok := <-cli.send:
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					// close frame を送って終了
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
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-ctx.Done():
				// サーバ都合で閉じる
				_ = conn.WriteControl(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseServiceRestart, "server stopping"),
					time.Now().Add(writeWait))
				return
			}
		}
	}()

	// --- 接続直後の初期メッセージ（writer 起動後に） ---
	if h.Svc != nil {
		if rep, err := h.Svc.OnConnect(ctx, uid, room); err == nil && rep != nil {
			_ = cli.SendJSON(ctx, rep)
		}
	}

	// --- reader ループ（Pong/ReadDeadline込み） ---
	conn.SetReadLimit(readLimit)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	conn.SetCloseHandler(func(code int, text string) error {
		// 必要ならログ
		return nil
	})

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var m inMsg
		if err := json.Unmarshal(data, &m); err != nil {
			// ここで軽いエラー応答を返してもOK
			continue
		}
		if h.Svc != nil {
			if rep, err := h.Svc.OnMessage(ctx, uid, room, m.Type, m.Body); err == nil && rep != nil {
				_ = cli.SendJSON(ctx, rep)
			}
		}
		if h.Svc == nil {
			_ = cli.SendJSON(ctx, map[string]any{
				"echo": string(data),
			})
			continue
		}
	}

	// --- 終了処理 ---
	if h.Hub != nil {
		_ = h.Hub.Leave(ctx, cli) // 先にHubから外す
	}
	if h.Svc != nil {
		h.Svc.OnDisconnect(ctx, uid, room)
	}
	_ = cli.Close() // send を閉じて writer 終了を促す
	<-done          // writer の終了待ち（CloseMessage 送出）
}
