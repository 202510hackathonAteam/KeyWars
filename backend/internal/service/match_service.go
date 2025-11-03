package service

import (
	"context"
	"keywars/backend/internal/domain/repository"
)

// MatchService は、マッチング処理のビジネスロジックを提供するサービス層。
// 主に MatchQueueRepository（待機キュー管理）と RoundStateRepository（試合状態管理）
// の2つのリポジトリを統合して、マッチングの実行・状態初期化などを行う。
type MatchService struct {
	// matchQueueRepository は、待機中ユーザーの登録・削除・マッチ生成などを担当する。
	matchQueueRepository repository.MatchQueueRepository

	// roundStateRepository は、マッチ進行中の状態（ラウンド、デッキ、イベントなど）を管理する。
	roundStateRepository repository.RoundStateRepository
}

// NewMatchService は、MatchQueueRepository と RoundStateRepository を受け取り、
// 新しい MatchService インスタンスを初期化して返す。
//
// 引数:
//
//	matchQueueRepository  - マッチング待機キュー操作を行うリポジトリ
//	roundStateRepository  - 試合状態を管理するリポジトリ
//
// 戻り値:
//
//	*MatchService - 初期化済みのサービスインスタンス
func NewMatchService(matchQueueRepository repository.MatchQueueRepository, roundStateRepository repository.RoundStateRepository) *MatchService {
	return &MatchService{
		matchQueueRepository: matchQueueRepository,
		roundStateRepository: roundStateRepository,
	}
}

// JoinQueue は、指定されたユーザーをマッチング待機キューに追加する。
// サービス層では、リポジトリへの委譲とエラーハンドリングのみ行う。
// 実際のキュー挙動（ZADD NXなど）は repository 側で実装される。
//
// 引数:
//
//	contextObject   - コンテキスト（キャンセルやタイムアウト制御に使用）
//	userID          - 待機ユーザーの識別子
//	currentTimeMs   - キュー投入時刻（UNIXミリ秒）
//
// 戻り値:
//
//	error - 成功時は nil、失敗時はエラーを返す
func (service *MatchService) JoinQueue(contextObject context.Context, userID string, currentTimeMs int64) error {
	return service.matchQueueRepository.Enqueue(contextObject, userID, currentTimeMs)
}

// TryMatch は、マッチング待機キューから2名を取り出し、新しいマッチを初期化する。
// 内部的にはリポジトリの DequeuePairAndInitMatch を呼び出し、
// 2名のユーザーが揃えばマッチIDを生成して返す。
// 待機者が1名以下の場合は空文字列を返す（＝マッチ未成立）。
//
// 引数:
//
//	contextObject - コンテキスト
//
// 戻り値:
//
//	matchID  - 生成されたマッチID（例: "a9f0d7b2..."）
//	user1ID  - 1人目のユーザーID
//	user2ID  - 2人目のユーザーID
//	error    - 失敗時のエラー
func (service *MatchService) TryMatch(contextObject context.Context) (matchID, user1ID, user2ID string, err error) {
	user1ID, user2ID, matchID, _, err = service.matchQueueRepository.DequeuePairAndInitMatch(contextObject)
	return
}
