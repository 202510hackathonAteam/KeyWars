package repository

import (
	"context"

	"keywars/backend/internal/domain/model"
	"keywars/backend/internal/domain/types"
)

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

	// Redis Stream に記録された FinishEvent から
	// 指定ラウンドのイベントだけを最大 RequiredPlayers 件（通常2件）読み込み、返す。
	LoadFinishEventsByRound(ctx context.Context, matchID string, round int64) ([]model.MatchFinishEvents, error)

	// 回答確定時のイベント（miss数・終了時刻・ラウンド情報）を Redis Streams に保存する。
	StoreFinishEvent(ctx context.Context, matchID, userID string, round, missCount, finishAtMs int64) error

	// 次ラウンド開始のために
	// ラウンド内で使用する一時的な state（予定時刻など）を初期化する。
	InitializeNextRoundState(ctx context.Context, matchID string) error

	// Redis の match:{matchID}:state に保存されている現在の試合状態を取得する。
	LoadMatchState(ctx context.Context, matchID string) (*model.MatchState, error)

	// 指定されたプレイヤー（player1 / player2）の累計ミス数カウントに missCount を加算する。
	UpdateTotalMissCount(ctx context.Context, matchID, playerField string, missCount int64) error

	// 指定されたプレイヤーのライフポイントを指定したダメージ分だけ減算する。
	ReduceLifepoint(ctx context.Context, matchID, playerField string, damage int64) error

	// 次ラウンドへ進むために deck_index と round を1つプラスして更新する。
	UpdateNextRoundState(ctx context.Context, matchID string) (int64, int64, error)

	// ラウンドの開始予定時刻と終了予定時刻を、Redis の match:{matchID}:state に保存する。
	UpdateRoundTiming(ctx context.Context, matchID string, roundStartAtMs int64, roundEndAtMs int64) error

	// Redis に保存されたmatch:{matchID} のプレイヤー情報（player1 / player2）を取得する。
	LoadMatchPlayers(ctx context.Context, matchID string) (*model.MatchPlayers, error)

	// 試合開始直後に呼び出される初期化処理する。
	InitMeasurementFinishCount(ctx context.Context, matchID string) error

	// 指定された matchID に紐づく「計測完了人数（measurement_finished_count）」カウンタを 0 に初期化する。
	InitializeMeasurementFinishCount(ctx context.Context, matchID string) error

	// 指定された matchID に紐づく「計測完了人数（measurement_finished_count）」カウンタを +1 する。
	IncrementMeasurementFinishCount(ctx context.Context, matchID string) (types.PlayerCount, error)

	// プレイヤーの finish トリガーを原子的に「初回のみ」受け付ける。
	RegisterPlayerAnswerFinishFlag(ctx context.Context, matchID, userID string) (bool, error)

	// 指定したプレイヤーの finish フラグキーが Redis 上に存在するかどうかを返す。
	IsPlayerAnswerFinishFlagExists(ctx context.Context, matchID, userID string) (bool, error)

	// プレイヤーの finish フラグキーを削除し、
	// 次のラウンドで再び RegisterPlayerFinishFlag の SetNX が成功するようにリセットする。
	DeletePlayerAnswerFinishFlag(ctx context.Context, matchID, userID string) error
}
