package websocket

// イベント種別（type）を定数化
const (
	TypeQueueJoined = "queue.joined"
	TypeQueueLeft   = "queue.left"

	TypeMatchFound = "match.found"
	TypeMatchStart = "match.start"
	TypeMatchEnd   = "match.end"

	TypeError = "error"
)

// --- 共通で使う小さな型 ---

// MatchState は試合の進行状態（match.start などで配信）
type MatchState struct {
	DeckIdx      int   `json:"deck_idx"`
	QStartedAtMS int64 `json:"q_started_at_ms"`
	Turn         int   `json:"turn"`
}

// --- 各イベントのペイロード定義 ---

// QueueJoinedPayload: キュー参加通知
type QueueJoinedPayload struct {
	Type string `json:"type"` // "queue.joined"
	At   int64  `json:"at"`   // enqueueしたサーバ時刻（ms）
}

func NewQueueJoinedPayload(atMS int64) QueueJoinedPayload {
	return QueueJoinedPayload{
		Type: TypeQueueJoined,
		At:   atMS,
	}
}

// QueueLeftPayload: キュー離脱（キャンセル）通知
type QueueLeftPayload struct {
	Type   string `json:"type"`   // "queue.left"
	Reason string `json:"reason"` // "canceled" など（任意）
}

func NewQueueLeftPayload(reason string) QueueLeftPayload {
	return QueueLeftPayload{
		Type:   TypeQueueLeft,
		Reason: reason,
	}
}

// MatchFoundPayload: マッチ成立通知（個人ルーム user:<uid> 宛に送る）
type MatchFoundPayload struct {
	Type     string `json:"type"`    // "match.found"
	MatchID  string `json:"matchId"` // 例: "cd4d6af01a..."
	Opponent string `json:"opponent"`
}

func NewMatchFoundPayload(matchID, opponentUserID string) MatchFoundPayload {
	return MatchFoundPayload{
		Type:     TypeMatchFound,
		MatchID:  matchID,
		Opponent: opponentUserID,
	}
}

// MatchStartPayload: 試合開始通知（ルーム match:<matchId> にブロードキャスト）
type MatchStartPayload struct {
	Type    string     `json:"type"`    // "match.start"
	MatchID string     `json:"matchId"` // 例: "cd4d6af01a..."
	P1      string     `json:"p1"`
	P2      string     `json:"p2"`
	State   MatchState `json:"state"`
}

func NewMatchStartPayload(matchID, p1, p2 string, state MatchState) MatchStartPayload {
	return MatchStartPayload{
		Type:    TypeMatchStart,
		MatchID: matchID,
		P1:      p1,
		P2:      p2,
		State:   state,
	}
}

// MatchEndPayload: 試合終了通知（勝者など）
type MatchEndPayload struct {
	Type    string `json:"type"`    // "match.end"
	MatchID string `json:"matchId"` // 例: "cd4d6af01a..."
	Winner  string `json:"winner"`  // 勝者ユーザーID（未決なら空文字でも可）
	Reason  string `json:"reason"`  // "timeout" / "resign" / "finished" など
}

func NewMatchEndPayload(matchID, winnerUserID, reason string) MatchEndPayload {
	return MatchEndPayload{
		Type:    TypeMatchEnd,
		MatchID: matchID,
		Winner:  winnerUserID,
		Reason:  reason,
	}
}

// ErrorPayload: 汎用エラー通知（プロトコル違反など）
type ErrorPayload struct {
	Type    string `json:"type"`    // "error"
	Code    string `json:"code"`    // アプリ内のエラーコード（例: "bad_request"）
	Message string `json:"message"` // 人間可読メッセージ
}

func NewErrorPayload(code, message string) ErrorPayload {
	return ErrorPayload{
		Type:    TypeError,
		Code:    code,
		Message: message,
	}
}
