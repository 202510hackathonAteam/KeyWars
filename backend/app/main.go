// backend/main.go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// WebSocketアップグレーダ
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 開発中なので全許可（本番では制限を）
	},
}

type Session struct {
	ID     string
	UserID string
	Conn   *websocket.Conn
}

var sessionCounter = 0
var sessions = make(map[string]*Session)

func main() {
	http.HandleFunc("/ws", handleWebSocket)
	fmt.Println("🚀 サーバー起動中: ws://localhost:8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// クエリから user_id を取得
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// セッションIDを採番
	sessionCounter++
	sessionID := fmt.Sprintf("session_%03d", sessionCounter)

	sess := &Session{
		ID:     sessionID,
		UserID: userID,
		Conn:   conn,
	}
	sessions[sessionID] = sess

	log.Printf("✅ 新規セッション: %s (user_id=%s)", sessionID, userID)

	// フロントにセッションIDを返す
	conn.WriteJSON(map[string]string{
		"type":       "session_start",
		"session_id": sessionID,
	})

	// メッセージ受信ループ
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("切断: user_id=%s (%v)", userID, err)
			delete(sessions, sessionID)
			break
		}
		log.Printf("📩 受信: %+v", msg)
	}
}
