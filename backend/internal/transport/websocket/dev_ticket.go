package websocket

// 動作確認用、あとで消す

import "context"

// dev:<uid>:<room> 形式のトークンを受け入れるだけ（検証なし）
type DevTicket struct{}

func (DevTicket) VerifyRoomTicket(ctx context.Context, token string) (uid, room string, err error) {
	// 例: token = "dev:u001:m_123"
	const prefix = "dev:"
	if len(token) <= len(prefix) || token[:len(prefix)] != prefix {
		return "", "", nil
	}
	rest := token[len(prefix):] // u001:m_123
	var ok bool
	for i := 0; i < len(rest); i++ {
		if rest[i] == ':' {
			uid = rest[:i]
			room = rest[i+1:]
			ok = true
			break
		}
	}
	if !ok || uid == "" || room == "" {
		return "", "", nil
	}
	return uid, room, nil
}
