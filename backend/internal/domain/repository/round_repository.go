package repository

import "context"

// AnswerApplyArg は、回答処理（answer）の適用時に必要なデータをまとめた引数構造体。
// Redis の match:{matchID}:state や Streams に対して状態を更新する際に使用される。
type AnswerApplyArg struct {
	MatchID              string
	OpponentUserID       string
	NewOpponentLifePoint int64
	NextDeckIndex        int64
	CurrentServerTimeMs  int64

	// イベントストリーム（match:{matchID}:events）に追加するフィールド群
	// 例: {"answer_user_id": "u1", "word": "apple", "correct": "1"}
	EventFields map[string]string
}

// RoundStateRepository は、対戦中の進行状態（ラウンド状態）を管理するリポジトリインターフェース。
// Redis の `match:{matchID}:state` や `match:{matchID}:events` に対する操作を抽象化する。
type RoundStateRepository interface {
	// 新しいマッチ用のメタ情報を初期化する。
	CreateMeta(contextObject context.Context, matchID, user1ID, user2ID string, currentTimeMs int64) error

	// マッチの開始フラグを立てる。
	Start(contextObject context.Context, matchID string) error

	// マッチを終了状態に更新する。
	Finish(contextObject context.Context, matchID, winnerUserID string) error

	// 出題デッキを初期化時に一度だけ保存する。
	// match:{matchID}:deck に全単語を保存しておき、以降の出題でインデックス参照する。
	SaveDeck(ctx context.Context, matchID string, deck []PromptWithDifficulty) error

	// 指定したデッキインデックスの単語を取得する。
	// Redis の配列から deckIndex 番目の単語を返す。
	GetDeckItem(contextObject context.Context, matchID string, deckIndex int64) (string, error)

	// プレイヤーの回答を反映し、次の状態を更新する。
	ApplyAnswer(contextObject context.Context, answerArg AnswerApplyArg) (eventID string, turn int64, err error)
}
