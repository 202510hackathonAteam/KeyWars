package websocket

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
	"context"

	domain "keywars/backend/internal/domain/port"
	"keywars/backend/internal/infra/auth"
	"keywars/backend/internal/config"

	"github.com/gorilla/websocket"
)

const (
	// readLimit は 1 メッセージあたりの最大受信サイズ（バイト）。
	readLimit = 1 << 20

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
	Hub       *Hub
	Service   domain.RealtimeService
	TokenAuth auth.JWTHandler
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
	// WebSocket接続生存期間用
	connCtx, connCancel := context.WithCancel(context.Background())
	defer connCancel()

	// --- 1) 認証 ---
	cookie, err := request.Cookie("access_token")
	if err != nil || cookie.Value == "" {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := handler.TokenAuth.VerifyAccessToken(cookie.Value)
	if err != nil {
		http.Error(writer, "unauthorized", http.StatusUnauthorized)
		return
	}
	roomName := "user:" + userID

	// --- 2) Upgrade ---
	wsConn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		log.Println("[ws] upgrade error:", err)
		return
	}

	// 接続インスタンス（送信用チャネル付き）
	clientConn := &Client{
		userID:      userID,
		roomName:    "",
		sendChannel: make(chan []byte, sendBufSize),
	}

	// --- 3) Hub.Join（マッチング待機は個人ルームへ） ---
	if err := handler.Hub.Join(roomName, clientConn); err != nil {
		log.Println("[WS] Hub.Join error:", err)
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

			case <-connCtx.Done():
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
	onConnectCtx, onConnectcancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer onConnectcancel()
	reply, err := handler.Service.OnConnect(onConnectCtx, userID, clientConn.Room())
	if err == nil && onConnectCtx.Err() != nil {
		err = onConnectCtx.Err()
	}
	if err != nil {
    log.Println("[WS-OnConnectError]", err)
    // エラー理由をクライアントへ送信（任意）
    _ = wsConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "on connect failed"),
			time.Now().Add(writeWait),
    )
    wsConn.Close()
    return
	}

	if reply != nil {
		_ = clientConn.SendJSON(reply)
	}

	// --- 5) reader ループ（Pong/ReadDeadline 込み） ---
	wsConn.SetReadLimit(readLimit)
	_ = wsConn.SetReadDeadline(time.Now().Add(config.PongWait))
	wsConn.SetPongHandler(func(_ string) error {
		_ = wsConn.SetReadDeadline(time.Now().Add(config.PongWait))
		_ = handler.Service.OnHeartbeat(connCtx, userID)
		return nil
	})
	wsConn.SetCloseHandler(func(_ int, _ string) error {
		connCancel()
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
		reply, err := handler.Service.OnMessage(connCtx, userID, incoming.Type, incoming.Body)
		if err != nil {
			log.Println("[WS-OnMessageError]", err)

			_ = clientConn.SendJSON(NewErrorPayload())

			continue
		}

		if reply != nil {
			_ = clientConn.SendJSON(reply)
		}
	}

	// --- 6) 終了処理 ---
	_ = handler.Hub.Leave(clientConn)

	onDisconnectCtx, onDisconnectcancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer onDisconnectcancel()
	handler.Service.OnDisconnect(onDisconnectCtx, userID)

	<-doneChan
}
