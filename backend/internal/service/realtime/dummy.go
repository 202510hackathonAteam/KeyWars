package realtime

// 開発・デバッグ用の簡易リアルタイムサービス。
// WebSocket 経由で送受信を確認するための最小実装。
// ※ 本番環境では、対戦ロジックや状態同期などの本実装に差し替えること。

import (
	"context"
)

// Service は RealtimeService インターフェースを満たすダミー実装。
// WebSocket ハンドラ（Handler.Svc）から呼ばれ、
// 接続・メッセージ受信・切断イベントを処理する。
type Service struct{}

// New は Service の新しいインスタンスを生成するコンストラクタ。
// 実際の依存は無いため、単純に空構造体を返す。
func New() *Service {
	return &Service{}
}

// OnConnect はクライアントが WebSocket に接続した直後に呼ばれる。
// ここでは接続確認用に “welcome” メッセージを返す。
//
// 引数:
//
//	contextObject - コンテキスト（キャンセルなどに使用）
//	userID        - 接続中のユーザーID
//	roomName      - 入室した部屋の名前（例: "match:m123"）
//
// 戻り値:
//
//	any   - クライアントに送る初期レスポンス（JSONにシリアライズされる）
//	error - 通常は nil
func (service *Service) OnConnect(contextObject context.Context, userID, roomName string) (any, error) {
	return map[string]any{
		"type": "welcome",
		"uid":  userID,
		"room": roomName,
	}, nil
}

// OnMessage はクライアントからメッセージを受信したときに呼ばれる。
// ここでは受け取った内容をそのままオウム返しする。
//
// 引数:
//
//	contextObject - コンテキスト
//	userID        - 送信元のユーザーID
//	roomName      - 現在の部屋名
//	messageType   - メッセージ種別（例: "answer", "chat"など）
//	payload       - メッセージ本文（JSON生データ）
//
// 戻り値:
//
//	any   - クライアントに返す応答データ（JSON化されて送信）
//	error - 通常は nil
func (service *Service) OnMessage(contextObject context.Context, userID, roomName string, messageType string, payload []byte) (any, error) {
	// シンプルにエコー返し
	return map[string]any{
		"type":      "echo",
		"echoType":  messageType,
		"payload":   string(payload),
		"from":      userID,
		"room_name": roomName,
	}, nil
}

// OnDisconnect はクライアントが切断した際に呼ばれる。
// 現状は何も行わないが、ログ記録やクリーンアップに利用できる。
func (service *Service) OnDisconnect(contextObject context.Context, userID, roomName string) {
	// No-op（必要に応じて実装）
}
