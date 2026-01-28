package websocket

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/gorilla/websocket"
)

//
// ==== 定数・エラー定義 ====
//

// 送信バッファが満杯でメッセージを送信できなかった場合に返すエラー
var ErrSendBufferFull = errors.New("send buffer full")

// クライアント接続がすでに閉じられている場合に返すエラー
var ErrClientClosed = errors.New("client closed")

//
// ==== Client 構造体 ====
//

// Client は、1つの WebSocket 接続（1ユーザーセッション）を表す。
// 各クライアントは1ユーザーIDと所属ルームを持ち、
// サーバーからクライアントへの送信は非同期チャネル経由で行われる。
type Client struct {
	wsConn *websocket.Conn
	// sendChannel は、サーバーからクライアントへ送信するメッセージを保持するチャネル。
	// 書き込みは WriteJSON() から行われ、読み取りは Writer goroutine が担当する。
	sendChannel chan []byte

	// mutex は、sendChannel と close 操作を直列化してデータ競合や panic を防ぐためのロック。
	mutex sync.Mutex

	// closed は、チャネルがすでに閉じられているかどうかを示すフラグ。
	closed bool

	// closeOnce は、Close() が複数回呼ばれてもチャネルを1度しか閉じないようにする制御構造。
	closeOnce sync.Once
}

// ClientConn インターフェースを満たしていることをコンパイル時に保証。
var _ ClientConn = (*Client)(nil)

//
// ==== パブリックメソッド ====
//

// WriteJSON は、指定された任意の構造体 messageData を JSON にシリアライズし、
// 非同期送信チャネル（sendChannel）へ送信要求を enqueue する。
//
// 本メソッドは非ブロッキングであり、
// ネットワーク送信の完了や到達保証は行わない。
func (client *Client) WriteJSON(messageData any) error {
	jsonBytes, err := json.Marshal(messageData)
	if err != nil {
		return err
	}

	client.mutex.Lock()
	defer client.mutex.Unlock()

	// 非ブロッキング送信：チャネルが満杯なら破棄
	select {
	case client.sendChannel <- jsonBytes:
		return nil
	default:
		return ErrSendBufferFull
	}
}

// Close は、送信チャネルを安全に閉じてリソースを解放する。
// closeOnce により、Close が複数回呼ばれても
// チャネルの close は 1 度しか実行されず、panic は発生しない。
func (client *Client) Close() error {
	client.closeOnce.Do(func() {
		close(client.sendChannel)
	})
	return nil
}
