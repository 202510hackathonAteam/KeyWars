package websocket

import (
	"context"
	"errors"
	"sync"

	domain "keywars/backend/internal/domain/port"
)

// ErrRoomFull は、ルームの収容上限（本実装では 2 名）に達したことを示すエラー。
var ErrRoomFull = errors.New("room full")

// Hub は、room 単位で接続（ClientConn）を管理し、ブロードキャスト機能を提供する。
// すべての公開メソッドは goroutine-safe（内部でミューテックス保護）である。
type Hub struct {
	// mutex は rooms への並行アクセスを保護するための RW ミューテックス。
	mutex sync.RWMutex

	// rooms は roomName -> (ClientConn セット) の対応表。
	// 値は空 struct を使った集合として実装する。
	rooms map[string]map[domain.ClientConn]struct{}
}

// domain.Broadcaster インターフェースを実装していることを明示。
var _ domain.Broadcaster = (*Hub)(nil)

// NewHub は、空のルームマップを持つ Hub を生成して返す。
func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[domain.ClientConn]struct{}),
	}
}

// Join は、指定された roomName に clientConn を参加させる。
// 収容上限（2名）を超える場合は ErrRoomFull を返す。
func (hub *Hub) Join(_ context.Context, roomName string, clientConn domain.ClientConn) error {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()

	if _, exists := hub.rooms[roomName]; !exists {
		hub.rooms[roomName] = make(map[domain.ClientConn]struct{})
	}

	// 収容上限チェック（2 名制限）
	if len(hub.rooms[roomName]) >= 2 {
		return ErrRoomFull
	}

	hub.rooms[roomName][clientConn] = struct{}{}
	return nil
}

// Leave は、clientConn を所属ルームから削除し、必要に応じてクライアントを Close する。
// ルームが空になった場合は rooms からルーム自体を削除する。
func (hub *Hub) Leave(_ context.Context, clientConn domain.ClientConn) error {
	var needClose bool

	hub.mutex.Lock()
	roomName := clientConn.Room()
	if roomName != "" {
		if memberSet, exists := hub.rooms[roomName]; exists {
			if _, present := memberSet[clientConn]; present {
				delete(memberSet, clientConn)
				needClose = true
			}
			if len(memberSet) == 0 {
				delete(hub.rooms, roomName)
			}
		}
	}
	hub.mutex.Unlock()

	// Close はロック外で実施（送信チャネルを安全に閉じる）
	if needClose {
		_ = clientConn.Close()
	}
	return nil
}

// Broadcast は、roomName に所属するすべてのクライアントへ message を送信する。
// 送信に失敗したクライアント数を failed として返す。
// contextObject がキャンセルされた場合は、その時点で中断し、context のエラーを返す。
func (hub *Hub) Broadcast(contextObject context.Context, roomName string, message any) (failed int, err error) {
	// 読み取りロック下でスナップショットを作る（ロック短縮のためにコピー）
	hub.mutex.RLock()
	memberSet, exists := hub.rooms[roomName]
	if !exists {
		hub.mutex.RUnlock()
		return 0, nil
	}
	connections := make([]domain.ClientConn, 0, len(memberSet))
	for clientConn := range memberSet {
		connections = append(connections, clientConn)
	}
	hub.mutex.RUnlock()

	// ロック外で送信処理
	for _, clientConn := range connections {
		select {
		case <-contextObject.Done():
			return failed, contextObject.Err()
		default:
		}
		if err := clientConn.SendJSON(contextObject, message); err != nil {
			failed++
		}
	}
	return failed, nil
}
