package websocket

// イベント種別（type）を定数化
const (
	TypeActiveMatchExists = "match.active_exists"
	TypeQueueJoined = "queue.joined"
	TypeQueueLeft   = "queue.cancelled"

	TypeWelcome = "welcome"
	TypeMatchRestore = "match.restore"
	TypeMatchStart = "match.start"
	TypeMatchEnd   = "match.end"

	TypeError = "error"
)

// --- 共通で使う小さな型 ---

// MatchState は試合の進行状態（match.start などで配信）
type MatchState struct {
	Round          	 int64 `json:"round"`
	RoundStartAtMs 	 int64 `json:"round_start_at_ms"`
	RoundEndAtMs   	 int64 `json:"round_end_at_ms"`
	Player1Lifepoint int64 `json:"player1_lifepoint"`
	Player2Lifepoint int64 `json:"player2_lifepoint"`
}

type PromptPayload struct {
	PromptTextJa string `json:"prompt_text_ja"`
	TargetRomaji string `json:"target_romaji"`
	LimitMs      int64  `json:"limit_ms"`
}

// --- 各イベントのペイロード定義 ---

// ActiveMatchExistsPayload: すでに進行中の試合があることを通知
type ActiveMatchExistsPayload struct {
	Type string `json:"type"`
	At 	 int64	`json:"at"`
}

func NewActiveMatchExistsPayload(atMs int64) ActiveMatchExistsPayload {
	return ActiveMatchExistsPayload{
		Type: TypeActiveMatchExists,
		At:		atMs,
	}
}

// QueueJoinedPayload: キュー参加通知
type QueueJoinedPayload struct {
	Type string `json:"type"` // "queue.joined"
	At   int64  `json:"at"`   // enqueueしたサーバ時刻（ms）
}

func NewQueueJoinedPayload(atMs int64) QueueJoinedPayload {
	return QueueJoinedPayload{
		Type: TypeQueueJoined,
		At:   atMs,
	}
}

// QueueLeftPayload: キュー離脱（キャンセル）通知
type QueueLeftPayload struct {
	Type   string `json:"type"`   // "queue.cancelled"
	// Reason string `json:"reason"` // "canceled" など（任意）
}

func NewQueueLeftPayload() QueueLeftPayload {
	return QueueLeftPayload{
		Type:   TypeQueueLeft,
	}
}

// WelcomePayload: WebSocket 接続直後の初回メッセージ
type WelcomePayload struct {
	Type string `json:"type"`
	UserID string `json:"user_id"`
}

func NewWelcomePayload(userID string) WelcomePayload {
	return WelcomePayload{
		Type: TypeWelcome,
		UserID: userID,
	}
}

type MatchRestoreState struct {
	Round          	 int64 `json:"round"`
	Player1Lifepoint int64 `json:"player1_lifepoint"`
	Player2Lifepoint int64 `json:"player2_lifepoint"`
}

// MatchRestorePayload: 途中復帰通知（現在の試合状態を同期するためのメッセージ）
type MatchRestorePayload struct {
	Type    string     `json:"type"`		// "match.restore"
	MatchID string     `json:"matchId"`
	State   MatchRestoreState `json:"state"`
}

func NewMatchRestorePayload(matchID string, state MatchRestoreState) MatchRestorePayload {
	return MatchRestorePayload{
		Type: TypeMatchRestore,
		MatchID: matchID,
		State: state,
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
	Result  string `json:"result"`	// "win" | "draw"
	Winner  string `json:"winner"`  // 勝者ユーザーID（未決なら空文字でも可）
	Player1 string `json:"player1"`
	Player2 string `json:"player2"`
	Player1TotalMissCount int64 `json:"player1_total_miss_count"`
	Player2TotalMissCount int64 `json:"player2_total_miss_count"`
	Player1TotalDamageDealt int64 `json:"player1_total_damage_dealt"`
	Player2TotalDamageDealt int64 `json:"player2_total_damage_dealt"`
}

func NewMatchEndPayload(matchID, result, winnerUserID, player1, player2 string, player1TotalMissCount, player2TotalMissCount, player1TotalDamageDealt, player2TotalDamageDealt int64) MatchEndPayload {
	return MatchEndPayload{
		Type:    TypeMatchEnd,
		MatchID: matchID,
		Result:  result,
		Winner:  winnerUserID,
		Player1: player1,
		Player2: player2,
		Player1TotalMissCount: player1TotalMissCount,
		Player2TotalMissCount: player2TotalMissCount,
		Player1TotalDamageDealt: player1TotalDamageDealt,
		Player2TotalDamageDealt: player2TotalDamageDealt,
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
