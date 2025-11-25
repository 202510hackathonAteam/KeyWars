package websocket

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
	"context"

	domain "keywars/backend/internal/domain/port"
	"keywars/backend/internal/domain/repository"
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
	Hub       *Hub
	Service   domain.RealtimeService
	Presence  repository.PresenceRepository
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
	requestCtx := request.Context()

	// WebSocket処理用のタイムアウト付きコンテキストを生成
	timeoutCtx, cancel := context.WithTimeout(requestCtx, time.Second*10)
	defer cancel()

	// 🔍 ここ！Upgrade 前の通常 HTTP リクエストなので全部見える
	log.Println("[WS] Cookie Header:", request.Header.Get("Cookie"))
	log.Println("[WS] X-CSRF-Token Header:", request.Header.Get("X-CSRF-Token"))
	log.Println("[WS] Origin:", request.Header.Get("Origin"))
	log.Println("[WS] User-Agent:", request.Header.Get("User-Agent"))

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
	if handler.Hub != nil {
		if err := handler.Hub.Join(timeoutCtx, roomName, clientConn); err != nil {
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

			case <-timeoutCtx.Done():
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
		if reply, err := handler.Service.OnConnect(timeoutCtx, userID, clientConn.Room()); err == nil && reply != nil {
			_ = clientConn.SendJSON(timeoutCtx, reply)
		}
	}

	// --- 5) reader ループ（Pong/ReadDeadline 込み） ---
	wsConn.SetReadLimit(readLimit)
	_ = wsConn.SetReadDeadline(time.Now().Add(pongWait))
	wsConn.SetPongHandler(func(_ string) error {
		_ = wsConn.SetReadDeadline(time.Now().Add(pongWait))
		if handler.Presence != nil {
			_ = handler.Presence.Heartbeat(timeoutCtx, userID, time.Now().UnixMilli())
		}
		return nil
	})
	wsConn.SetCloseHandler(func(_ int, _ string) error {
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
		if handler.Service != nil {
			room := clientConn.Room()
			if reply, err := handler.Service.OnMessage(timeoutCtx, userID, room, incoming.Type, incoming.Body); err == nil && reply != nil {
				_ = clientConn.SendJSON(timeoutCtx, reply)
			}
		}
		if handler.Service == nil {
			_ = clientConn.SendJSON(timeoutCtx, map[string]any{
				"echo": string(rawData),
			})
			continue
		}
	}

	// --- 6) 終了処理 ---
	if handler.Hub != nil {
		_ = handler.Hub.Leave(timeoutCtx, clientConn)
	}
	if handler.Presence != nil {
		_ = handler.Presence.Disconnect(timeoutCtx, userID, time.Now().UnixMilli())
	}
	<-doneChan
}
