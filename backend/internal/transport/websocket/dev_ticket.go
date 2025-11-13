package websocket

import (
	"context"
	"errors"
	"strings"
)

// DevTicket は開発・デバッグ専用の簡易 TicketVerifier 実装。
// トークンは "dev:<userID>:<roomName>" 形式を受理する。
// 本番では JWT/署名検証などの仕組みに置き換えること。
type DevTicket struct{}

// VerifyRoomTicket は "dev:<userID>:<roomName>" を分解して返す。
// 形式不正の場合は ("" ,"" , error) を返す。
func (DevTicket) VerifyRoomTicket(_ context.Context, token string) (string, string, error) {
	if !strings.HasPrefix(token, "dev:") {
		return "", "", errors.New("invalid token: missing 'dev:' prefix")
	}
	parts := strings.SplitN(token, ":", 3)
	if len(parts) != 3 || parts[1] == "" || parts[2] == "" {
		return "", "", errors.New("invalid token: want 'dev:<userID>:<roomName>'")
	}
	userID := parts[1]
	roomName := parts[2]
	return userID, roomName, nil
}
