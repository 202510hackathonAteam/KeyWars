package round

import (
	"fmt"
	"context"

	"keywars/backend/internal/domain/repository"
	"keywars/backend/internal/domain/model"
)

type LifepointService struct {
	roundStateRepo repository.RoundStateRepository
}

// NewLifepointService は LifepointService のコンストラクタ。
func NewLifepointService(roundStateRepo repository.RoundStateRepository) *LifepointService {
	return &LifepointService{
		roundStateRepo: roundStateRepo,
	}
}

// ApplyRoundDamage は、指定された matchID のラウンド終了後に、
// 各プレイヤーの回答完了時刻・ミス数をもとに
// 与えるダメージを計算し、ライフポイントへ反映するメソッド。
func (s *LifepointService) ApplyRoundDamage(ctx context.Context, matchID string) (int64, int64, error) {
	// マッチ状態のロード
	state, err := s.roundStateRepo.LoadMatchState(ctx, matchID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to load match state: %w", err)
	}

	// 今ラウンドのプレイヤー分の試合結果の取得
	events, err := s.roundStateRepo.LoadFinishEventsByRound(ctx, matchID, state.Round)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to load finish events: %w", err)
	}

	// プレイヤーID情報をロード
	players, err := s.roundStateRepo.LoadMatchPlayers(ctx, matchID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to load match players: %w", err)
	}

	// イベントを player1 / player2 に振り分け
	var player1Event *model.MatchFinishEvents
	var player2Event *model.MatchFinishEvents

	for i := range events {
		event := &events[i]

		switch event.PlayerID{
		case players.Player1ID:
			player1Event = event
		case players.Player2ID:
			player2Event = event
		default:
			return 0, 0, fmt.Errorf("unknown playerId in FinishEvents: %s", event.PlayerID)
		}
	}

	// 必要分の件数が揃っているかチェック
	if player1Event == nil || player2Event == nil {
		return 0, 0, fmt.Errorf("required finish events not found")
	}

	// ダメージ計算
	var player1Damage int64 = 0
	var player2Damage int64 = 0

	// 先に答えた方が勝ち → 負けた側へ 10 ダメージ
	if player1Event.FinishAtMs < player2Event.FinishAtMs {
		player2Damage = 10
	}
	if player2Event.FinishAtMs < player1Event.FinishAtMs {
		player1Damage = 10
	}

	// ミス数に応じた追加ダメージ（2 × missCount）
	if player1Event.MissCount > 0 {
		player1Damage += player1Event.MissCount * 2
	}
	if player2Event.MissCount > 0 {
		player2Damage += player2Event.MissCount * 2
	}

	player1Lifepoint := state.Player1Lifepoint
	player2Lifepoint := state.Player2Lifepoint

	// ライフポイント減算を適用
	if player1Damage > 0 {
		player1Lifepoint, err = s.roundStateRepo.ReduceLifepoint(ctx, matchID, "player1", player1Damage)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to apply damage player1: %w", err)
		}
	}

	if player2Damage > 0 {
		player2Lifepoint, err = s.roundStateRepo.ReduceLifepoint(ctx, matchID, "player2", player2Damage)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to apply damage player2: %w", err)
		}
	}

	return player1Lifepoint, player2Lifepoint, nil
}
