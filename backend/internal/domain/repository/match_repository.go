package repository

import "context"

// マッチング待機キューの操作を定義するリポジトリインターフェース。
type MatchQueueRepository interface {
	// ユーザーをマッチング待機キューに追加する。
	Enqueue(contextObject context.Context, userID string, enqueueAtMs int64) error

	// 指定されたユーザーを待機キューから削除する。
	Cancel(contextObject context.Context, userID string) error

	// S指定ユーザーの現在のスコア（待機時刻）を取得する。
	Score(contextObject context.Context, userID string) (float64, error)

	// 2名をキューから取り出し、新しいマッチを初期化する。
	DequeuePairAndInitMatch(contextObject context.Context) (user1ID, user2ID, matchID, lockToken string, err error)

	// RDequeuePairAndInitMatch で取得したロックキーを解放する。
	ReleaseLock(contextObject context.Context, lockToken string) error
}
