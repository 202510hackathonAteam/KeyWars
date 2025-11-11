package repository

import "context"

// AnswerApplyArg は、回答処理（answer）の適用時に必要なデータをまとめた引数構造体。
// Redis の match:{matchID}:state や Streams に対して状態を更新する際に使用される。
type AnswerApplyArg struct {
	// MatchID: 対戦中のマッチID
	MatchID string

	// OpponentUserID: 対戦相手ユーザーID（攻撃対象）
	OpponentUserID string

	// NewOpponentLifePoint: 回答結果により更新された相手の残りLP（ライフポイント）
	NewOpponentLifePoint int64

	// NextDeckIndex: 次に出題するデッキ（問題）のインデックス（例: 0→1→2...）
	NextDeckIndex int64

	// CurrentServerTimeMs: 現在のサーバー時刻（UNIXミリ秒）
	CurrentServerTimeMs int64

	// EventFields: イベントストリーム（match:{matchID}:events）に追加するフィールド群
	// 例: {"answer_user_id": "u1", "word": "apple", "correct": "1"}
	EventFields map[string]string
}

// RoundStateRepository は、対戦中の進行状態（ラウンド状態）を管理するリポジトリインターフェース。
// Redis の `match:{matchID}:state` や `match:{matchID}:events` に対する操作を抽象化する。
type RoundStateRepository interface {
	// CreateMeta は、新しいマッチ用のメタ情報を初期化する。
	// Redis 側では HASH を生成し、プレイヤー情報や初期値（LP, deck_index など）を登録する。
	//
	// 引数:
	//   contextObject  - コンテキスト（キャンセルやタイムアウト制御に使用）
	//   matchID        - マッチID
	//   user1ID        - プレイヤー1のユーザーID
	//   user2ID        - プレイヤー2のユーザーID
	//   currentTimeMs  - サーバー時刻（ミリ秒）
	//
	// 戻り値:
	//   error - 成功時は nil、失敗時はエラーを返す
	CreateMeta(contextObject context.Context, matchID, user1ID, user2ID string, currentTimeMs int64) error

	// Start は、マッチの開始フラグを立てる。
	// 状態を waiting → playing に変更し、開始時刻などを記録する。
	//
	// 引数:
	//   contextObject - コンテキスト
	//   matchID       - 対象マッチID
	//
	// 戻り値:
	//   error - 成功時は nil、失敗時はエラーを返す
	Start(contextObject context.Context, matchID string) error

	// Finish は、マッチを終了状態に更新する。
	// 状態を finished にし、勝者IDなどを保存する。
	//
	// 引数:
	//   contextObject - コンテキスト
	//   matchID       - マッチID
	//   winnerUserID  - 勝者のユーザーID
	//
	// 戻り値:
	//   error - 成功時は nil、失敗時はエラーを返す
	Finish(contextObject context.Context, matchID, winnerUserID string) error

	// SaveDeck は、出題デッキを初期化時に一度だけ保存する。
	// 例: match:{matchID}:deck に全単語を保存しておき、以降の出題でインデックス参照する。
	//
	// 引数:
	//   contextObject - コンテキスト
	//   matchID       - マッチID
	//   deckItems     - 出題単語のリスト（JSON化された配列）
	//
	// 戻り値:
	//   error - 成功時は nil、失敗時はエラーを返す
	SaveDeck(ctx context.Context, matchID string, deck []PromptWithDifficulty) error

	// GetDeckItem は、指定したデッキインデックスの単語を取得する。
	// Redis の配列から deckIndex 番目の単語を返す。
	//
	// 引数:
	//   contextObject - コンテキスト
	//   matchID       - マッチID
	//   deckIndex     - 取得するインデックス（0-based）
	//
	// 戻り値:
	//   string - 取得した単語
	//   error  - エラー（存在しない場合など）
	GetDeckItem(contextObject context.Context, matchID string, deckIndex int64) (string, error)

	// ApplyAnswer は、プレイヤーの回答を反映し、次の状態を更新する。
	// 内部的には:
	//   - state の更新（LP減少、deck_index進行など）
	//   - events への追加（Streamsに回答イベントをPush）
	// を行う。
	//
	// 引数:
	//   contextObject - コンテキスト
	//   answerArg     - 回答処理用の引数構造体（AnswerApplyArg）
	//
	// 戻り値:
	//   eventID - 生成されたイベントのStreams ID（例: "1734350000123-0"）
	//   turn    - 次のターン番号（例: 1→2→3...）
	//   err     - エラー（処理失敗時）
	ApplyAnswer(contextObject context.Context, answerArg AnswerApplyArg) (eventID string, turn int64, err error)
}
