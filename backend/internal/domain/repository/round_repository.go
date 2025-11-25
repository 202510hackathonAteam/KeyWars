package repository

import (
	"context"

	"keywars/backend/internal/domain/model"
)

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

	// 指定したデッキインデックスのmatch:{matchID}:deck の JSON を構造体に変換する。
	LoadDeckPrompt(ctx context.Context, matchID string, deckIndex int64) (*model.DeckPrompt, error)

	// プレイヤーの回答を反映し、次の状態を更新する。
	ApplyAnswer(contextObject context.Context, answerArg AnswerApplyArg) (eventID string, round int64, err error)

	// 次ラウンド開始のために
	// ラウンド内で使用する一時的な state（予定時刻や計測フラグなど）を初期化する。
	InitializeNextRoundState(ctx context.Context, matchID string) error

	// Redis の match:{matchID}:state に保存されている現在の試合状態を取得する。
	LoadMatchState(ctx context.Context, matchID string) (*model.MatchState, error)

	// 次ラウンドへ進むために deck_index と round を1つプラスして更新する。
	UpdateNextRound(ctx context.Context, matchID string) (int64, int64, error)

	// ラウンドの開始予定時刻と終了予定時刻を、Redis の match:{matchID}:state に保存する。
	UpdateRoundTiming(ctx context.Context, matchID string, roundStartAtMs int64, roundEndAtMs int64) error

	// Redis に保存されたmatch:{matchID} のプレイヤー情報（player1 / player2）を取得する。
	LoadMatchPlayers(ctx context.Context, matchID string) (*model.MatchPlayers, error)

	// 試合開始直後に呼び出される初期化処理する。
	InitMeasurementFinishCount(ctx context.Context, matchID string) error

	// 指定された matchID に紐づく「計測完了人数（measurement_finished_count）」カウンタを 0 に初期化する。
	InitializeMeasurementFinishCount(ctx context.Context, matchID string) error

	// 指定された matchID に紐づく「計測完了人数（measurement_finished_count）」カウンタを +1 する。
	IncrementMeasurementFinishCount(ctx context.Context, matchID string) (int64, error)
}
