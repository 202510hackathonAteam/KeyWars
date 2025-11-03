package repository

import "context"

// MatchQueueRepository は、マッチング待機キューの操作を定義するリポジトリインターフェース。
// このインターフェースは、待機中ユーザーの登録・削除・取得や、
// ペアマッチング処理およびロック管理を抽象化する。
type MatchQueueRepository interface {
	// Enqueue は、ユーザーをマッチング待機キューに追加する。
	//
	// 引数:
	//   contextObject  - コンテキスト（キャンセルやタイムアウト制御に使用）
	//   userID         - 待機中ユーザーの識別子
	//   enqueueAtMs    - キュー投入時刻のUNIXミリ秒
	//
	// 戻り値:
	//   error - 成功時は nil、失敗時はエラーを返す
	Enqueue(contextObject context.Context, userID string, enqueueAtMs int64) error

	// Cancel は、指定されたユーザーを待機キューから削除する。
	//
	// 引数:
	//   contextObject - コンテキスト
	//   userID        - キャンセル対象のユーザーID
	//
	// 戻り値:
	//   error - 成功時は nil、失敗時はエラーを返す
	Cancel(contextObject context.Context, userID string) error

	// Score は、指定ユーザーの現在のスコア（待機時刻）を取得する。
	// ZSET のスコアに対応し、ユーザーが存在しない場合は 0 を返す。
	//
	// 引数:
	//   contextObject - コンテキスト
	//   userID        - 対象ユーザーID
	//
	// 戻り値:
	//   float64 - スコア値（存在しない場合は 0）
	//   error   - 取得エラー
	Score(contextObject context.Context, userID string) (float64, error)

	// DequeuePairAndInitMatch は、2名をキューから取り出し、新しいマッチを初期化する。
	// この操作は排他ロックの下で実行される。
	//
	// 戻り値:
	//   user1ID    - マッチングされた1人目のユーザーID
	//   user2ID    - マッチングされた2人目のユーザーID
	//   matchID    - 生成されたマッチID
	//   lockToken  - ロック識別用トークン（後で ReleaseLock に渡す）
	//   err        - 失敗時のエラー
	DequeuePairAndInitMatch(contextObject context.Context) (user1ID, user2ID, matchID, lockToken string, err error)

	// ReleaseLock は、DequeuePairAndInitMatch で取得したロックキーを解放する。
	// ロック識別用トークンが一致した場合のみ削除を行う。
	//
	// 引数:
	//   contextObject - コンテキスト
	//   lockToken     - ロックトークン（識別用）
	//
	// 戻り値:
	//   error - 成功時は nil、失敗時はエラーを返す
	ReleaseLock(contextObject context.Context, lockToken string) error
}
