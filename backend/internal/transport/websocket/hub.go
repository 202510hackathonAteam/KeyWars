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
	// 接続（ClientConn）は揮発的であり、切断や再接続により随時入れ替わる。
	connsByUser map[string]map[ClientConn]struct{}
}

//
// ==== コンストラクタ ====
//

// NewHub は、空の接続管理マップを初期化した Hub インスタンスを生成。
func NewHub() *Hub {
	return &Hub{
		connsByUser: make(map[string]map[ClientConn]struct{}),
	}
}

//
// ==== Join ====
//

// JoinUser は、指定された userID に対して
// WebSocket 接続（ClientConn）を配送対象として登録する。
//
// 本関数は接続管理のための内部処理であり、
// 参加人数や試合状態などのドメイン制約は扱わない。
// 同一接続の再登録は冪等に処理される。
func (hub *Hub) JoinUser(userID string, clientConn ClientConn) {
	hub.mutex.Lock()         // 書き込みロック開始
	defer hub.mutex.Unlock() // 関数終了時に必ず解除

	if hub.connsByUser[userID] == nil {
		hub.connsByUser[userID] = make(map[ClientConn]struct{})
	}
	hub.connsByUser[userID][clientConn] = struct{}{}
}

//
// ==== Leave ====
//

// Leave は、指定された ClientConn を
// Hub の配送対象から削除する。
//
// 本関数は接続登録の解除のみを行い、
// WebSocket 接続の Close は行わない。
func (hub *Hub) Leave(conn ClientConn) {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()

	for userID, conns := range hub.connsByUser {
		if _, exists := conns[conn]; exists {
			delete(conns, conn)
			if len(conns) == 0 {
				delete(hub.connsByUser, userID)
			}
			break
		}
	}
}

// LeaveUser は、指定されたユーザーに紐づくすべての ClientConn を
// Hub の配送対象から削除する。
//
// 主に「試合終了」「強制退室」など、業務ロジック上の理由により
// 当該ユーザーへイベントを送信すべきでなくなった場合に使用する。
//
// 本関数は WebSocket 接続の Close は行わず、
// あくまで Hub 内の配送対象からの除外のみを行う。
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
// 本関数は以下の特性を持つ：
//   - userID を宛先解決の唯一の基準とする
//   - 接続単位（ClientConn）でのベストエフォート送信
//   - 送信に失敗した接続は自動的にクリーンアップされる
//
// WebSocket の到達保証は行わず、
// 再送や状態同期は上位レイヤに委ねる。
func (hub *Hub) DispatchToUser(userID string, message any) {
	// 対象 userID に紐づく接続のスナップショットを取得する。
	hub.mutex.RLock()
	conns := make([]ClientConn, 0, len(hub.connsByUser[userID]))
	for c := range hub.connsByUser[userID] {
		conns = append(conns, c)
	}
	hub.mutex.RUnlock()

	for _, conn := range conns {
		if err := conn.WriteJSON(message); err != nil {
			// 壊れた接続は掃除
			hub.Leave(conn)
		}
	}
}

// DispatchAndCloseUser は、指定された userID に紐づくすべての ClientConn に対して
// メッセージを配送したうえで、直ちに WebSocket 接続をクローズする。
//
// 本関数は主に「match.end」など、
// メッセージ送信後に当該ユーザーの接続ライフサイクルを
// 確実に終了させる必要があるケースで使用する。
//
// 処理の流れは以下の通り：
//   1. 対象 userID に紐づく接続のスナップショットを取得
//   2. 各接続に対して message を送信（ベストエフォート）
//   3. 送信後、接続を明示的に Close
//
// 本関数は WebSocket の Close を伴うため、
// 通常のイベント通知には DispatchToUser を使用し、
// 明示的に切断を伴う場面でのみ使用すること。
func (hub *Hub) DispatchAndCloseUser(userID string, message any) {
	// 対象 userID に紐づく接続のスナップショットを取得する。
	hub.mutex.RLock()
	conns := make([]ClientConn, 0, len(hub.connsByUser[userID]))
	for c := range hub.connsByUser[userID] {
		conns = append(conns, c)
	}
	hub.mutex.RUnlock()

	for _, conn := range conns {
		if err := conn.WriteJSON(message); err != nil {
			// 壊れた接続は掃除
			hub.Leave(conn)
		}
		conn.Close()
	}
}