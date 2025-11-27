package round

import (
	"fmt"
	"context"

	"keywars/backend/internal/domain/repository"
	"keywars/backend/internal/config"
)

type MatchJudgeService struct {
	roundStateRepo repository.RoundStateRepository
}

// NewMatchJudgeService は MatchJudgeService のコンストラクタ。
func NewMatchJudgeService(roundStateRepo repository.RoundStateRepository) *MatchJudgeService {
	return &MatchJudgeService{
		roundStateRepo:			roundStateRepo,
	}
}

// =====================
// Result Type Enum
// =====================

// ResultType は勝敗の種別を表す列挙型。
type ResultType string

const (
	ResultUnknown ResultType = "unknown"
	ResultWin ResultType = "win"
	ResultDraw ResultType = "draw"
)

// MatchJudgeResult は、勝敗判定後の結果データ。
type MatchJudgeResult struct {
	WinnerPlayerID string
	ResultType ResultType
	Player1TotalDamageDealt int64
	Player2TotalDamageDealt int64
}

// JudgeMatchResult は、プレイヤーの最終ライフポイントから勝敗を判定し、
// 勝者ID・結果タイプ（勝利 or 引き分け）・与ダメージ量をまとめて返すメソッド。
func (s *MatchJudgeService) JudgeMatchResult(ctx context.Context, matchID string, player1Lifepoint, player2Lifepoint int64) (*MatchJudgeResult, error) {
	// プレイヤー情報の取得
	players, err := s.roundStateRepo.LoadMatchPlayers(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("failed to load players: %w", err)
	}

	var resultType ResultType
	var winnerPlayerID string

	// 勝者・結果タイプの決定
	switch {
	case player1Lifepoint == player2Lifepoint:
		resultType = ResultDraw
		winnerPlayerID = ""
	case player1Lifepoint < player2Lifepoint:
		resultType = ResultWin
		winnerPlayerID = players.Player2ID
	case player2Lifepoint < player1Lifepoint:
		resultType = ResultWin
		winnerPlayerID = players.Player1ID
	}

	// トータルダメージ計算
	// プレイヤー1がプレイヤー2に与えたダメージ
	player1TotalDamageDealt := config.InitialLifePoint - player2Lifepoint
	// プレイヤー2がプレイヤー1に与えたダメージ
	player2TotalDamageDealt := config.InitialLifePoint - player1Lifepoint

	matchJudgeResult := &MatchJudgeResult{
		WinnerPlayerID: winnerPlayerID,
		ResultType: resultType,
		Player1TotalDamageDealt: player1TotalDamageDealt,
		Player2TotalDamageDealt: player2TotalDamageDealt,
	}

	return matchJudgeResult, nil
}