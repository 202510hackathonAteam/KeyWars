package websocket

import (
	"context"
	"errors"
	"sync"

	domain "keywars/backend/internal/domain/port"
)

//
// ==== 定数・エラー定義 ====
//

// 1ルームあたりの最大収容人数（2名マッチ）
const RoomCapacity = 2

var (
	// ルームが満員のときに返すエラー
	ErrRoomFull = errors.New("room full")

	// 同じルームにすでに参加している場合に返す（冪等動作用）
	ErrAlreadyJoined = errors.New("already joined the room")
)

//
// ==== Hub 構造体 ====
//

// Hub は、room 単位で WebSocket 接続（ClientConn）を管理し、
// 「参加 / 退出 / ブロードキャスト」などをスレッドセーフに扱う中心クラス。
//
// 複数ユーザーが同時に接続・切断しても安全に動くように
// sync.RWMutex によるロックで保護されている。
type Hub struct {
	// mutex は rooms への同時アクセスを防ぐためのミューテックス。
	mutex sync.RWMutex

	// rooms は「ルーム名 → 接続集合」を保持するマップ。
	// 各ルームの値は map[ClientConn]struct{} で集合的に管理（値は空構造体で省メモリ）。
	rooms map[string]map[domain.ClientConn]struct{}
}

// domain.Broadcaster インターフェースを満たしていることを明示的に保証。
// （これがあると、interface実装チェックがコンパイル時に行われる）
var _ domain.Broadcaster = (*Hub)(nil)

//
// ==== コンストラクタ ====
//

// NewHub は、空のルームマップを持つ新しい Hub インスタンスを生成。
func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[domain.ClientConn]struct{}),
	}
}

//
// ==== Join ====
//

// Join は、指定された roomName に clientConn を参加させる。
// - すでに同じルームにいる場合は ErrAlreadyJoined を返して終了（冪等）
// - 収容人数を超えると ErrRoomFull を返す
// - 他ルームへの移動はサポート外（必要なら Move を使う）
func (hub *Hub) Join(_ context.Context, roomName string, clientConn domain.ClientConn) error {
	hub.mutex.Lock()         // 書き込みロック開始
	defer hub.mutex.Unlock() // 関数終了時に必ず解除

	// 同じルームに既にいるなら冪等動作
	if current := clientConn.Room(); current != "" && current != roomName {
		return ErrAlreadyJoined
	}

	// 対象ルームのメンバー集合を確保
	memberSet := hub.ensureRoom(roomName)
	if _, present := memberSet[clientConn]; present {
		return ErrAlreadyJoined
	}

	// 収容上限チェック（2名制限）
	if len(memberSet) >= RoomCapacity {
		return ErrRoomFull
	}

	// 新しいクライアントをルームに追加
	memberSet[clientConn] = struct{}{}
	if c, ok := clientConn.(*Client); ok {
		c.setRoom(roomName) // ルーム名を保持
	}
	return nil
}

//
// ==== Leave ====
//

// Leave は、clientConn を所属ルームから削除する。
// - ルームが空になれば、rooms マップ自体からも削除。
// - 接続 Close はロック外で安全に行う（デッドロック防止）。
func (hub *Hub) Leave(_ context.Context, clientConn domain.ClientConn) error {
	var needClose bool

	hub.mutex.Lock()
	roomName := clientConn.Room()

	// 所属ルームがある場合のみ処理
	if roomName != "" {
		if memberSet, exists := hub.rooms[roomName]; exists {
			// 対象の接続を削除
			if _, present := memberSet[clientConn]; present {
				delete(memberSet, clientConn)
				if c, ok := clientConn.(*Client); ok {
					c.clearRoom() // 所属情報をクリア
				}
				needClose = true
			}
			// 全員いなくなったらルームを削除
			if len(memberSet) == 0 {
				delete(hub.rooms, roomName)
			}
		}
	}
	hub.mutex.Unlock()

	// ロック外で接続をクローズ（チャネル競合回避）
	if needClose {
		_ = clientConn.Close()
	}
	return nil
}

//
// ==== Broadcast ====
//

// Broadcast は、指定された roomName 内の全クライアントに対して
// 任意のメッセージを JSON で送信する。
// - ctx がキャンセルされると中断。
// - 送信失敗した接続は Leave によってクリーンアップされる。
// - ベストエフォート方式（部分的失敗を許容）。
func (hub *Hub) Broadcast(ctx context.Context, roomName string, message any) (failed int, err error) {
	// 読み取りロック下でメンバーのスナップショットを作成
	hub.mutex.RLock()
	memberSet, exists := hub.rooms[roomName]
	if !exists {
		hub.mutex.RUnlock()
		return 0, nil // ルームなし＝誰もいない
	}

	// コピーしてロック時間を短縮
	conns := make([]domain.ClientConn, 0, len(memberSet))
	for c := range memberSet {
		conns = append(conns, c)
	}
	hub.mutex.RUnlock()

	// ロック外で送信処理を実行
	for _, c := range conns {
		select {
		case <-ctx.Done():
			return failed, ctx.Err() // コンテキストキャンセル時は即終了
		default:
		}
		// 送信
		if err := c.SendJSON(ctx, message); err != nil {
			failed++
			// 失敗した接続を掃除（Closeは Leave 内で実施）
			_ = hub.Leave(context.Background(), c)
		}
	}
	return failed, nil
}

//
// ==== Members / Move ====
//

// Members は roomName の参加者スナップショットを返す（ロック短縮のためコピー）。
// 「個人ルームから試合ルームへ一括移動」で利用する。
func (hub *Hub) Members(roomName string) []domain.ClientConn {
	hub.mutex.RLock()
	defer hub.mutex.RUnlock()
	set, ok := hub.rooms[roomName]
	if !ok {
		return nil
	}
	out := make([]domain.ClientConn, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	return out
}

// Move は clientConn を現在のルームから newRoom へ「原子的に」移動させる。
// - 収容上限を尊重（満杯なら ErrRoomFull）
// - roomName(setRoom/clearRoom) を正しく更新
// - old ルームが空になれば削除
func (hub *Hub) Move(_ context.Context, clientConn domain.ClientConn, newRoom string) error {
	hub.mutex.Lock()
	defer hub.mutex.Unlock()

	// すでに同じルームなら冪等
	if clientConn.Room() == newRoom {
		return ErrAlreadyJoined
	}

	// 新ルームの収容チェック
	newSet, ok := hub.rooms[newRoom]
	if !ok {
		newSet = make(map[domain.ClientConn]struct{})
		hub.rooms[newRoom] = newSet
	}
	if len(newSet) >= RoomCapacity {
		return ErrRoomFull
	}

	// 現在のルームから除外
	if old := clientConn.Room(); old != "" {
		if oldSet, ok := hub.rooms[old]; ok {
			delete(oldSet, clientConn)
			if len(oldSet) == 0 {
				delete(hub.rooms, old)
			}
		}
	}

	// 新ルームに追加し、現在ルームを更新
	newSet[clientConn] = struct{}{}
	if c, ok := clientConn.(*Client); ok {
		c.setRoom(newRoom)
	}
	return nil
}

//
// ==== 内部ヘルパー ====
//

// ensureRoom は、指定したルーム名に対応する memberSet を返す。
// ルームが存在しない場合は新しく作成して返す。
func (hub *Hub) ensureRoom(roomName string) map[domain.ClientConn]struct{} {
	if s, ok := hub.rooms[roomName]; ok {
		return s
	}
	s := make(map[domain.ClientConn]struct{})
	hub.rooms[roomName] = s
	return s
}
