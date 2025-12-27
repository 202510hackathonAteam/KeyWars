package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"context"

	"keywars/backend/internal/infra/auth"

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

// Handler は WebSocket エンドポイントのハンドラ。
// - Hub: 接続の出入りとブロードキャストを司る
// - Service: アプリ固有の接続/メッセージ/切断処理
// - Verifier: 参加用トークンの検証
type Handler struct {
	hub       *Hub
	service   MatchRealtimeService
	tokenAuth *auth.JWTHandler
}

// NewWebSocketHandler は、WebSocket ハンドラーの生成。
func NewWebSocketHandler(
	hub *Hub,
	service MatchRealtimeService,
	tokenAuth *auth.JWTHandler,
) *Handler {
	return &Handler{
		hub: hub,
		service: service,
		tokenAuth: tokenAuth,
	}
}

// upgrader は HTTP から WebSocket へのアップグレード設定。
// 本番では CheckOrigin の制約を強めること。
var upgrader = websocket.Upgrader{
	ReadBufferSize:    1024,
	WriteBufferSize:   1024,
	EnableCompression: false,
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
	// WebSocket接続生存期間用
	connCtx, closeConn := context.WithCancel(context.Background())

	// --- 1) 認証 ---
	cookie, err := request.Cookie("access_token")
	if err != nil || cookie.Value == "" {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := handler.tokenAuth.VerifyAccessToken(cookie.Value)
	if err != nil {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}

	// --- 2) Upgrade ---
	wsConn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Println("[ws] upgrade error:", err)
		return
	}

	// 接続インスタンス（送信用チャネル付き）
	client := &Client{
		wsConn: wsConn,
		sendChannel: make(chan []byte, sendBufSize),
	}

	// --- 3) Hub.JoinUser（マッチング待機は個人ルームへ） ---
	handler.hub.JoinUser(userID, client)

	// --- 4) writer goroutine（Ping 送信込み）---
	go func() {
		pingTicker := time.NewTicker(pingInterval)
		defer pingTicker.Stop()

		for {
			select {
			case messageBytes, ok := <-client.sendChannel:
				_ = wsConn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					// close frame を送って終了
					_ = wsConn.WriteControl(
						websocket.CloseMessage,
						websocket.FormatCloseMessage(
							websocket.CloseNormalClosure,
							"connection closed",
						),
						time.Now().Add(writeWait),
					)
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

			case <-connCtx.Done():
				// サーバ都合で閉じる
				_ = wsConn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(
						websocket.CloseNormalClosure,
						"session ended",
					),
					time.Now().Add(writeWait),
				)
				return
			}
		}
	}()

	// --- 接続直後の初期メッセージ（writer 起動後に） ---
	onConnectCtx, onConnectcancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer onConnectcancel()
	reply, err := handler.service.OnConnect(onConnectCtx, userID)
	if err == nil && onConnectCtx.Err() != nil {
		err = onConnectCtx.Err()
	}
	if err != nil {
    log.Println("[WS-OnConnectError]", err)
    // エラー理由をクライアントへ送信（任意）
    _ = wsConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(
				websocket.CloseInternalServerErr,
				"on connect failed",
			),
			time.Now().Add(writeWait),
    )
    wsConn.Close()
    return
	}

	if reply != nil {
		_ = client.WriteJSON(reply)
	}

	// --- 5) reader ループ（Pong/ReadDeadline 込み） ---
	wsConn.SetReadLimit(readLimit)
	_ = wsConn.SetReadDeadline(time.Now().Add(pongWait))
	wsConn.SetPongHandler(func(_ string) error {
		_ = wsConn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, rawData, err := wsConn.ReadMessage()
		if err != nil {
			log.Println("[WS-ReadError]", err)
			break
		}
		var incoming IncomingMessage
		if err := json.Unmarshal(rawData, &incoming); err != nil {
			continue
		}
		reply, err := handler.service.OnMessage(connCtx, userID, incoming.Type, incoming.Body)
		if err != nil {
			log.Println("[WS-OnMessageError]", err)
			_ = client.WriteJSON(NewErrorPayload())
			continue
		}

		if reply != nil {
			_ = client.WriteJSON(reply)
		}
	}

	// --- 6) 終了処理 ---
	closeConn()

	handler.hub.LeaveUser(userID)

	onDisconnectCtx, onDisconnectcancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer onDisconnectcancel()
	handler.service.OnDisconnect(onDisconnectCtx, userID)
}
