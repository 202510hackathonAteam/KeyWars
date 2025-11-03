package websocket

import "context"

// DevTicket は開発・デバッグ専用の簡易 TicketVerifier 実装。
// トークンを実際には検証せず、フォーマット "dev:<userID>:<roomName>" を単純に分解して返す。
// 本番環境では必ず JWT や Redis-backed Verifier などに置き換えること。
type DevTicket struct{}

// VerifyRoomTicket は、"dev:<userID>:<roomName>" 形式のトークンを受け取り、
// userID（例: "u001"）と roomName（例: "m_123"）を抽出して返す。
// 不正な形式や空要素の場合は空文字を返す（err は nil のまま）。
//
// 例:
//
//	token = "dev:u001:match_10"
//	→ userID = "u001", roomName = "match_10"
//
// この関数はあくまで開発用であり、認証や署名検証は行わない。
func (DevTicket) VerifyRoomTicket(contextObject context.Context, token string) (userID, roomName string, err error) {
	const prefix = "dev:"

	// トークンが "dev:" で始まらない場合は無効
	if len(token) <= len(prefix) || token[:len(prefix)] != prefix {
		return "", "", nil
	}

	// "dev:" 以降を取得
	remainder := token[len(prefix):] // 例: "u001:m_123"
	var formatValid bool

	// コロン区切りで userID と roomName を抽出
	for i := 0; i < len(remainder); i++ {
		if remainder[i] == ':' {
			userID = remainder[:i]
			roomName = remainder[i+1:]
			formatValid = true
			break
		}
	}

	// パース結果のバリデーション
	if !formatValid || userID == "" || roomName == "" {
		return "", "", nil
	}

	return userID, roomName, nil
}
