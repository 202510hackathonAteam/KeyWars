package websocket

import (
	"context"
	"encoding/json"
	"sync"

	domain "keywars/backend/internal/domain/port"
)

// Client は、1つの WebSocket 接続（1セッション）を表現する構造体。
// クライアントID（userID）と所属ルーム（room）を保持し、
// サーバからの送信データは send チャネル経由で非同期的に処理される。
type Client struct {
	// userID は、このクライアントに紐づくユーザー識別子。
	userID string

	// roomName は、このクライアントが所属するルーム名。
	roomName string

	// sendChannel は、送信専用のチャネル。
	// Writer goroutine がこのチャネルから読み取り、実際に WebSocket に書き出す。
	sendChannel chan []byte

	// closeOnce は、Close() が複数回呼ばれても1回しかチャネルを閉じないように制御する同期プリミティブ。
	closeOnce sync.Once
}

// domain.ClientConn インターフェースの実装を明示。
var _ domain.ClientConn = (*Client)(nil)

// UID は、クライアントに紐づくユーザーIDを返す。
func (client *Client) UID() string {
	return client.userID
}

// Room は、クライアントが属しているルーム名を返す。
func (client *Client) Room() string {
	return client.roomName
}

// SendJSON は、指定されたデータ構造体を JSON エンコードして
// 非同期的に送信チャネルへ積む。
// チャネルが満杯の場合はドロップ（破棄）し、
// バックプレッシャ制御は行わない設計。
// 必要に応じて「ドロップ」→「切断」などに変更可能。
func (client *Client) SendJSON(_ context.Context, messageData any) error {
	jsonBytes, err := json.Marshal(messageData)
	if err != nil {
		return err
	}

	select {
	case client.sendChannel <- jsonBytes:
	default:
		// チャネルが満杯のため、メッセージを破棄（ログ出力などは任意）
	}
	return nil
}

// Close は、送信チャネルを安全に閉じてリソースを解放する。
// closeOnce により多重 Close 呼び出しを防ぐ。
func (client *Client) Close() error {
	client.closeOnce.Do(func() {
		close(client.sendChannel)
	})
	return nil
}
