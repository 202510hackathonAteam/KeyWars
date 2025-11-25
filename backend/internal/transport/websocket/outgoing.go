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
	RoundStartAtMS 	 int64 `json:"round_start_at_ms"`
	RoundEndAtMS   	 int64 `json:"round_end_at_ms"`
	Round          	 int64 `json:"round"`
	Player1Lifepoint int64 `json:"player1_lifepoint"`
	Player2Lifepoint int64 `json:"player2_lifepoint"`
}

type PromptPayload struct {
	PromptTextJa string `json:"prompt_text_ja"`
	TargetRomaji string `json:"target_romaji"`
	LimitMs      int64  `json:"limit_ms"`
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
	// Reason string `json:"reason"` // "canceled" など（任意）
}

func NewQueueLeftPayload() QueueLeftPayload {
	return QueueLeftPayload{
		Type:   TypeQueueLeft,
	}
}

// MatchFoundPayload: マッチ成立通知（個人ルーム user:<uid> 宛に送る）
type MatchFoundPayload struct {
	Type     string `json:"type"`    // "match.found"
	MatchID  string `json:"match_id"` // 例: "cd4d6af01a..."
	Opponent string `json:"opponent"`
}

func NewMatchFoundPayload(matchID, opponentUserID string) MatchFoundPayload {
	return MatchFoundPayload{
		Type:     TypeMatchFound,
		MatchID:  matchID,
		Opponent: opponentUserID,
	}
}

// RoundStartPayload: 試合開始通知（ルーム match:<matchId> にブロードキャスト）
type RoundStartPayload struct {
	Type    string     		`json:"type"`    // "match.start"
	MatchID string     		`json:"match_id"` // 例: "cd4d6af01a..."
	Player1 string     		`json:"player1"`
	Player2 string     		`json:"player2"`
	State   MatchState 		`json:"state"`
	Prompt  PromptPayload `json:"prompt"`
}

func NewRoundStartPayload(matchID, player1, player2 string, state MatchState, prompt PromptPayload) RoundStartPayload {
	return RoundStartPayload{
		Type:    TypeMatchStart,
		MatchID: matchID,
		Player1: player1,
		Player2: player2,
		State:   state,
		Prompt:  prompt,
	}
}

// MatchEndPayload: 試合終了通知（勝者など）
type MatchEndPayload struct {
	Type    string `json:"type"`    // "match.end"
	MatchID string `json:"match_id"` // 例: "cd4d6af01a..."
	Winner  string `json:"winner"`  // 勝者ユーザーID（未決なら空文字でも可）
	Player1 string `json:"player1"`
	Player2 string `json:"player2"`
	Player1TotalMissCount int64 `json:"player1_total_miss_count"`
	Player2TotalMissCount int64 `json:"player2_total_miss_count"`
}

func NewMatchEndPayload(matchID, player1, player2, winnerUserID string, player1TotalMissCount, player2TotalMissCount int64) MatchEndPayload {
	return MatchEndPayload{
		Type:    TypeMatchEnd,
		MatchID: matchID,
		Player1: player1,
		Player2: player2,
		Winner:  winnerUserID,
		Player1TotalMissCount: player1TotalMissCount,
		Player2TotalMissCount: player2TotalMissCount,
	}
}

// ErrorPayload: 汎用エラー通知（プロトコル違反など）
type ErrorPayload struct {
	Type string `json:"type"`    // "error"
}

func NewErrorPayload() ErrorPayload {
	return ErrorPayload{
		Type: TypeError,
	}
}
