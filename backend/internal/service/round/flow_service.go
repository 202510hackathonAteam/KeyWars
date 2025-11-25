package round

import (
	"context"

	"keywars/backend/internal/domain/repository"
	"keywars/backend/internal/transport/websocket"
	"keywars/backend/internal/config"
)

// SetTimeoutService は、RoundFlowService に対して
// TimeoutRoundService を後から注入するためのセッターメソッド。
func (s *RoundFlowService) SetTimeoutService(timeoutRoundService *TimeoutRoundService) {
	s.timeoutRoundService = timeoutRoundService
}

// RoundFlowService は、1 ラウンドの開始・進行・終了といった
// 「ラウンド進行フロー全体」を管理するサービス
type RoundFlowService struct {
	roundStateRepo			 repository.RoundStateRepository
	websocketHub         *websocket.Hub
	nextRoundService     *NextRoundService
	timeoutRoundService  *TimeoutRoundService
}

// NewRoundFlowService は RoundFlowService のコンストラクタ。
func NewRoundFlowService(roundStateRepo repository.RoundStateRepository, websocketHub *websocket.Hub, nextRoundService *NextRoundService, timeoutRoundService *TimeoutRoundService) *RoundFlowService {
	return &RoundFlowService{
		roundStateRepo: 		 roundStateRepo,
		websocketHub: 			 websocketHub,
		nextRoundService: 	 nextRoundService,
		timeoutRoundService: timeoutRoundService,
	}
}

// StartFirstRound は、試合の最初のラウンドを開始するメソッド。
func (s *RoundFlowService) StartFirstRound(ctx context.Context, matchID, user1ID, user2ID string) error {
	// 1問目取得
	roundQuestion, err := s.nextRoundService.LoadNextPrompt(ctx, matchID, 0)
	if err != nil {
		return err
	}

	// 開始予定時刻＋終了予定時刻更新し、取得
	roundStartAtMs, roundEndAtMs, err := s.nextRoundService.SaveRoundTiming(ctx, matchID, roundQuestion.LimitMs)
	if err != nil {
		return err
	}

	// マッチ開始メッセージをルーム内の全員にブロードキャスト
	payload := websocket.NewRoundStartPayload(
		matchID,
		user1ID,
		user2ID,
		websocket.MatchState{
			Round:           	config.InitialRound,
			RoundStartAtMS: 	roundStartAtMs,
			RoundEndAtMS:   	roundEndAtMs,
			Player1Lifepoint: config.InitialLifePoint,
			Player2Lifepoint: config.InitialLifePoint,
		},
		websocket.PromptPayload{
			PromptTextJa: roundQuestion.PromptTextJa,
			TargetRomaji: roundQuestion.TargetRomaji,
			LimitMs:      roundQuestion.LimitMs,
		},
	)

	// WebSocket ブロードキャスト
	s.websocketHub.Broadcast(ctx, "match:"+matchID, payload)

	go s.timeoutRoundService.ScheduleRoundTimeoutCheck(ctx, matchID, roundEndAtMs + config.GraceMs)
	return nil
}

// RunRoundFlow は、計測完了後に実行されるメソッド。
func (s *RoundFlowService) RunRoundFlow(ctx context.Context, matchID string) error {
	// ライフポイント管理機能の実行
	// プレイヤーごとにではなく、ラウンドごとの1回処理すること

	state, err := s.roundStateRepo.LoadMatchState(ctx, matchID)
	if err != nil {
		return err
	}
	currentRound := state.Round
	lifePoint1 := state.Player1Lifepoint
	lifePoint2 := state.Player2Lifepoint

	// ===============================
	// ① 終了条件（勝敗判定へ）
	// ===============================
	if currentRound >= 20 || lifePoint1 <= 0 || lifePoint2 <= 0 {
		// 勝敗判定機能を書く

		// players, err := s.roundStateRepo.LoadMatchPlayers(ctx, matchID)
		// if err != nil {
		// 	return err
		// }
		// payload := websocket.NewMatchEndPayload(
		// 	matchID,
		// 	players.Player1ID,
		// 	players.Player2ID,
		// 	winner,
		// 	state.Player1TotalMissCount,
		// 	state.Player2TotalMissCount,
		// )
	}

	// ===============================
	// ② 続行 → 次ラウンド初期化
	// ===============================
	if err := s.roundStateRepo.InitializeNextRoundState(ctx, matchID); err != nil{
		return err
	}
	if err := s.roundStateRepo.InitializeMeasurementFinishCount(ctx, matchID); err != nil{
		return err
	}
	players, err := s.roundStateRepo.LoadMatchPlayers(ctx, matchID)
	if err != nil {
		return err
	}
	s.roundStateRepo.DeletePlayerAnswerFinishFlag(ctx, matchID, players.Player1ID)
	if err != nil {
		return err
	}
	s.roundStateRepo.DeletePlayerAnswerFinishFlag(ctx, matchID, players.Player2ID)
	if err != nil {
		return err
	}

	// 次のラウンドへ値を更新
	nextDeckIndex, nextRound, err := s.roundStateRepo.UpdateNextRoundState(ctx, matchID)

	// 次ラウンドの問題取得
	roundQuestion, err := s.nextRoundService.LoadNextPrompt(ctx, matchID, nextDeckIndex)
	if err != nil {
		return err
	}

	// 開始予定時刻＋終了予定時刻取得
	roundStartAtMs, roundEndAtMs, err := s.nextRoundService.SaveRoundTiming(ctx, matchID, roundQuestion.LimitMs)
	if err != nil {
		return err
	}

	// ===============================
	// ③ WebSocketで次ラウンド開始通知
	// ===============================
	payload := websocket.NewRoundStartPayload(
		matchID,
		players.Player1ID,
		players.Player2ID,
		websocket.MatchState{
			Round:           	nextRound,
			RoundStartAtMS: 	roundStartAtMs,
			RoundEndAtMS:   	roundEndAtMs,
			Player1Lifepoint: lifePoint1,
			Player2Lifepoint: lifePoint2,
		},
		websocket.PromptPayload{
			PromptTextJa: roundQuestion.PromptTextJa,
			TargetRomaji: roundQuestion.TargetRomaji,
			LimitMs:      roundQuestion.LimitMs,
		},
	)

	// WebSocket ブロードキャスト
	s.websocketHub.Broadcast(ctx, "match:"+matchID, payload)

	go s.timeoutRoundService.ScheduleRoundTimeoutCheck(ctx, matchID, roundEndAtMs + config.GraceMs)

	return nil
}