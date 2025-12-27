package websocket

import (
	"sync"
)

//
// ==== Hub 構造体 ====
//

// Hub は、userID をキーとして WebSocket 接続（ClientConn）を管理し、
// 指定された userID に対してメッセージを配送するための中継コンポーネントである。
//
// Hub は接続の生死や配送先解決のみを責務とし、
// 試合参加・人数制限・マッチ状態といったドメインルールは扱わない。
// 複数 goroutine から安全に利用できるよう、内部状態は mutex により保護される。
type Hub struct {
	// mutex は rooms への同時アクセスを防ぐためのミューテックス。
	mutex sync.RWMutex
	// connsByUser は userID ごとに現在有効な WebSocket 接続を保持する。
	// 本 Hub では「1ユーザー = 1接続」を前提とし、
	// 新しい接続が登録された場合は、既存の接続をクローズして上書きする。
	connsByUser map[string]*Client
}

//
// ==== コンストラクタ ====
//

// NewHub は、空の接続管理マップを初期化した Hub インスタンスを生成。
func NewHub() *Hub {
	return &Hub{
		connsByUser: make(map[string]*Client),
	}
}

//
// ==== Join ====
//

// JoinUser は、指定された userID に対して
// 現在有効な WebSocket 接続を登録する。
//
// 同一 userID に対して既存の接続が存在する場合は、
// その接続をクローズしたうえで新しい接続で上書きする。
//
// 本関数は「接続管理」のみを責務とし、
// 試合参加・人数制限・マッチ状態などの
// ドメインロジックは一切扱わない。
func (hub *Hub) JoinUser(userID string, conn *Client) {
	hub.mutex.Lock()         // 書き込みロック開始
	defer hub.mutex.Unlock() // 関数終了時に必ず解除

	if existingConn, ok := hub.connsByUser[userID]; ok {
    _ = existingConn.Close()
	}
	hub.connsByUser[userID] = conn
}


//
// ==== Leave ====
//

// LeaveUser は、指定された userID に紐づく
// WebSocket 接続を Hub の配送対象から解除する。
//
// 本関数は Hub 内の接続参照を削除するのみで、
// WebSocket 接続の Close は行わない。
func (hub *Hub) LeaveUser(userID string) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()

	delete(hub.connsByUser, userID)
}

//
// ==== DispatchToUser ====
//

// DispatchToUser は、指定された userID に紐づく
// 現在有効なすべてのクライアント接続に対して、
// メッセージを配送する。
//
// 本 Hub では「1ユーザー = 1接続」を前提とするため、
// 配送対象は常に高々 1 接続である。
//
// WebSocket の到達保証は行わず、
// 再送や状態同期は上位レイヤに委ねる。
func (hub *Hub) DispatchToUser(userID string, message any) {
	// 対象 userID に紐づく接続のスナップショットを取得する。
	hub.mutex.RLock()
	conn, ok := hub.connsByUser[userID]
	hub.mutex.RUnlock()

	if !ok {
		return
	}

	if err := conn.WriteJSON(message); err != nil {
		// 書き込みに失敗した接続は Hub から除外する
		hub.LeaveUser(userID)
	}
}

// DispatchAndCloseUser は、指定された userID に紐づく
// 現在の接続に対してメッセージを送信し、
// 直後に WebSocket 接続をクローズする。
//
// 本関数は match.end / kick / ban など、
// 業務的に「この接続を確実に終了させる」必要がある場合にのみ使用する。
func (hub *Hub) DispatchAndCloseUser(userID string, message any) {
	// 対象 userID に紐づく接続のスナップショットを取得する。
	hub.mutex.RLock()
	client, ok := hub.connsByUser[userID]
	hub.mutex.RUnlock()

	if !ok {
		return
	}

	_ = client.WriteJSON(message)

	client.Close()

	hub.mutex.Lock()
	delete(hub.connsByUser, userID)
	hub.mutex.Unlock()
}