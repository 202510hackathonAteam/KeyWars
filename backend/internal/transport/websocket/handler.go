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
	// readLimit は 1 メッセージあたりの最大受信サイズ（バイト）。
	readLimit = 1 << 20

	// pongWait は最後の Pong 受信から次の Pong までの猶予時間。
	pongWait = 60 * time.Second

	// writeWait は各フレーム送信の書き込みタイムアウト。
	writeWait = 10 * time.Second

	// pingInterval はサーバ側からの Ping を送る間隔。
	pingInterval = 25 * time.Second

	// sendBufSize は送信チャネルのバッファサイズ（メッセージ数）。
	sendBufSize = 256
)

// TicketVerifier は、クエリ等で渡される「ルーム参加用トークン」を検証し、
// ユーザーIDとルーム名を返す責務を持つ。
type TicketVerifier interface {
	VerifyRoomTicket(ctx context.Context, token string) (userID, roomName string, err error)
}

// Handler は WebSocket エンドポイントのハンドラ。
// - Hub: 接続の出入りとブロードキャストを司る
// - Service: アプリ固有の接続/メッセージ/切断処理
// - Verifier: 参加用トークンの検証
type Handler struct {
	Hub      *Hub
	Service  domain.RealtimeService
	Verifier TicketVerifier
}

// upgrader は HTTP から WebSocket へのアップグレード設定。
// 本番では CheckOrigin の制約を強めること。
var upgrader = websocket.Upgrader{
	ReadBufferSize:    1024,
	WriteBufferSize:   1024,
	EnableCompression: true,
	CheckOrigin:       func(_ *http.Request) bool { return true }, // 本番はオリジンを限定
}

// IncomingMessage はクライアントから受信するメッセージの基本スキーマ。
type IncomingMessage struct {
	Type string          `json:"type"`
	Body json.RawMessage `json:"body"`
}

// ServeHTTP は WebSocket エンドポイントのエントリポイント。
// 1) トークン検証 → 2) Upgrade → 3) Hub への Join → 4) writer 起動 → 5) reader ループ → 6) クリーンアップ
func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	requestContext := request.Context()

	// --- 1) 認証（roomTicket 必須） ---
	if handler.Verifier == nil {
		http.Error(writer, "server verifier not configured", http.StatusServiceUnavailable)
		return
	}
	token := request.URL.Query().Get("token")
	if token == "" {
		http.Error(writer, "token is required", http.StatusBadRequest)
		return
	}
	userID, roomName, err := handler.Verifier.VerifyRoomTicket(requestContext, token)
	if err != nil || userID == "" || roomName == "" {
		http.Error(writer, "invalid token", http.StatusUnauthorized)
		return
	}

	// --- 2) Upgrade ---
	wsConn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Println("[ws] upgrade error:", err)
		return
	}
	// Close は writer 側で行う（CloseMessage 送信のため）

	// 接続インスタンス（送信用チャネル付き）
	clientConn := &Client{
		userID:      userID,
		roomName:    roomName,
		sendChannel: make(chan []byte, sendBufSize),
	}

	// --- 3) Hub.Join ---
	if handler.Hub != nil {
		if err := handler.Hub.Join(requestContext, roomName, clientConn); err != nil {
			if errors.Is(err, ErrRoomFull) {
				_ = wsConn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "room full"),
					time.Now().Add(writeWait),
				)
			}
			_ = wsConn.Close()
			return
		}
	}

	// --- 4) writer goroutine（Ping 送信込み）---
	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		defer wsConn.Close()

		pingTicker := time.NewTicker(pingInterval)
		defer pingTicker.Stop()

		for {
			select {
			case messageBytes, ok := <-clientConn.sendChannel:
				_ = wsConn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					// close frame を送って終了
					_ = wsConn.WriteMessage(websocket.CloseMessage, []byte{})
					return
				}
				frameWriter, err := wsConn.NextWriter(websocket.TextMessage)
				if err != nil {
					return
				}
				if _, err := frameWriter.Write(messageBytes); err != nil {
					_ = frameWriter.Close()
					return
				}
				if err := frameWriter.Close(); err != nil {
					return
				}

			case <-pingTicker.C:
				_ = wsConn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := wsConn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}

			case <-requestContext.Done():
				// サーバ都合で閉じる
				_ = wsConn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseServiceRestart, "server stopping"),
					time.Now().Add(writeWait),
				)
				return
			}
		}
	}()

	// --- 接続直後の初期メッセージ（writer 起動後に） ---
	if handler.Service != nil {
		if reply, err := handler.Service.OnConnect(requestContext, userID, roomName); err == nil && reply != nil {
			_ = clientConn.SendJSON(requestContext, reply)
		}
	}

	// --- 5) reader ループ（Pong/ReadDeadline 込み） ---
	wsConn.SetReadLimit(readLimit)
	_ = wsConn.SetReadDeadline(time.Now().Add(pongWait))
	wsConn.SetPongHandler(func(string) error {
		_ = wsConn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	wsConn.SetCloseHandler(func(_ int, _ string) error {
		// 必要ならログ
		return nil
	})

	for {
		_, rawData, err := wsConn.ReadMessage()
		if err != nil {
			break
		}
		var incoming IncomingMessage
		if err := json.Unmarshal(rawData, &incoming); err != nil {
			// ここで軽いエラー応答を返してもOK
			continue
		}
		if handler.Service != nil {
			if reply, err := handler.Service.OnMessage(requestContext, userID, roomName, incoming.Type, incoming.Body); err == nil && reply != nil {
				_ = clientConn.SendJSON(requestContext, reply)
			}
		}
		if handler.Service == nil {
			_ = clientConn.SendJSON(requestContext, map[string]any{
				"echo": string(rawData),
			})
			continue
		}
	}

	// --- 6) 終了処理 ---
	if handler.Hub != nil {
		_ = handler.Hub.Leave(requestContext, clientConn) // 先に Hub から外す
	}
	if handler.Service != nil {
		handler.Service.OnDisconnect(requestContext, userID, roomName)
	}
	_ = clientConn.Close() // sendChannel を閉じて writer を終了させる
	<-doneChan             // writer の終了待ち（CloseMessage 送出）
}
