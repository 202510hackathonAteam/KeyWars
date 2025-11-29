package round

import (
	"context"
	"fmt"
	"time"

	"keywars/backend/internal/config"
	"keywars/backend/internal/domain/repository"
	"keywars/backend/internal/transport/websocket"
)

// SetTimeoutService は、RoundFlowService に対して
// TimeoutRoundService を後から注入するためのセッターメソッド。
func (s *RoundFlowService) SetTimeoutService(timeoutRoundService *TimeoutRoundService) {
	s.timeoutRoundService = timeoutRoundService
}

// RoundFlowService は、1 ラウンドの開始・進行・終了といった
// 「ラウンド進行フロー全体」を管理するサービス
type RoundFlowService struct {
	roundStateRepo      repository.RoundStateRepository
	websocketHub        *websocket.Hub
	nextRoundService    *NextRoundService
	timeoutRoundService *TimeoutRoundService
	lifepointService    *LifepointService
	matchJudgeService   *MatchJudgeService
	timeoutCancelMap    map[string]context.CancelFunc
	presenceRepo        repository.PresenceRepository
}

// NewRoundFlowService は RoundFlowService のコンストラクタ。
func NewRoundFlowService(
	roundStateRepo repository.RoundStateRepository,
	websocketHub *websocket.Hub,
	nextRoundService *NextRoundService,
	timeoutRoundService *TimeoutRoundService,
	lifepointService *LifepointService,
	matchJudgeService *MatchJudgeService,
	presenceRepo repository.PresenceRepository,
) *RoundFlowService {
	return &RoundFlowService{
		roundStateRepo:      roundStateRepo,
		websocketHub:        websocketHub,
		nextRoundService:    nextRoundService,
		timeoutRoundService: timeoutRoundService,
		lifepointService:    lifepointService,
		matchJudgeService:   matchJudgeService,
		timeoutCancelMap:    make(map[string]context.CancelFunc),
		presenceRepo:        presenceRepo,
	}
}

// StartFirstRound は、試合の最初のラウンドを開始するメソッド。
func (s *RoundFlowService) StartFirstRound(ctx context.Context, matchID, user1ID, user2ID string) error {
	// 1問目取得
	loadCtx, cancelLoad := context.WithTimeout(ctx, 500*time.Millisecond)
	roundQuestion, err := s.nextRoundService.LoadNextPrompt(loadCtx, matchID, 0)
	cancelLoad()
	if err != nil {
		return err
	}

	// 開始予定時刻＋終了予定時刻更新し、取得
	timingCtx, cancelTiming := context.WithTimeout(ctx, 500*time.Millisecond)
	roundStartAtMs, roundEndAtMs, err := s.nextRoundService.SaveFirstRoundTiming(timingCtx, matchID, roundQuestion.LimitMs)
	cancelTiming()
	if err != nil {
		return err
	}

	// マッチ開始メッセージをルーム内の全員にブロードキャスト
	payload := websocket.NewRoundStartPayload(
		matchID,
		user1ID,
		user2ID,
		websocket.MatchState{
			Round:            config.InitialRound,
			RoundStartAtMS:   roundStartAtMs,
			RoundEndAtMS:     roundEndAtMs,
			Player1Lifepoint: config.InitialLifePoint,
			Player2Lifepoint: config.InitialLifePoint,
		},
		websocket.PromptPayload{
			PromptTextJa: roundQuestion.PromptTextJa,
			TargetRomaji: roundQuestion.TargetRomaji,
			LimitMs:      roundQuestion.LimitMs,
		},
	)

	s.websocketHub.Broadcast(ctx, "match:"+matchID, payload)

	timeoutCtx, timeoutCancel := context.WithCancel(context.Background())
	s.timeoutCancelMap[matchID] = timeoutCancel
	go s.timeoutRoundService.ScheduleRoundTimeoutCheck(timeoutCtx, matchID, roundEndAtMs+config.GraceMs)
	return nil
}

// ProcessRoundResult は、両プレイヤーの回答が揃ったあとに呼ばれ、
// ダメージ計算 → 勝敗判定 → 次ラウンド準備 → WebSocket通知 を実行するメソッド。
// ラウンド終了後の全処理を一括で実行するフロー関数。
func (s *RoundFlowService) ProcessRoundResult(ctx context.Context, matchID string) error {
	// 前ラウンドの timeout goroutine を停止
	if cancel, ok := s.timeoutCancelMap[matchID]; ok {
		cancel()
		delete(s.timeoutCancelMap, matchID)
	}

	// ラウンド結果に基づきダメージを適用
	applyCtx, cancelApply := context.WithTimeout(ctx, 600*time.Millisecond)
	player1Lifepoint, player2Lifepoint, err := s.lifepointService.ApplyRoundDamage(applyCtx, matchID)
	cancelApply()
	if err != nil {
		return fmt.Errorf("failed to apply round damage: %w", err)
	}

	loadStateCtx, cancelLoadState := context.WithTimeout(ctx, 300*time.Millisecond)
	state, err := s.roundStateRepo.LoadMatchState(loadStateCtx, matchID)
	cancelLoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}
	currentRound := state.Round

	// ===============================
	// ① 終了条件（勝敗判定へ）
	// ===============================
	if currentRound >= 20 || player1Lifepoint <= 0 || player2Lifepoint <= 0 {
		// 勝敗判定
		judgeCtx, cancelJudge := context.WithTimeout(ctx, 300*time.Millisecond)
		matchJudgeResult, err := s.matchJudgeService.JudgeMatchResult(judgeCtx, matchID, player1Lifepoint, player2Lifepoint)
		cancelJudge()
		if err != nil {
			return fmt.Errorf("failed to judge match result: %w", err)
		}

		loadPlayersCtx, cancelLoadPlayers := context.WithTimeout(ctx, 200*time.Millisecond)
		players, err := s.roundStateRepo.LoadMatchPlayers(loadPlayersCtx, matchID)
		cancelLoadPlayers()
		if err != nil {
			return fmt.Errorf("failed to load players: %w", err)
		}
		payload := websocket.NewMatchEndPayload(
			matchID,
			string(matchJudgeResult.ResultType),
			matchJudgeResult.WinnerPlayerID,
			players.Player1ID,
			players.Player2ID,
			state.Player1TotalMissCount,
			state.Player2TotalMissCount,
			matchJudgeResult.Player1TotalDamageDealt,
			matchJudgeResult.Player2TotalDamageDealt,
		)

		// match.end をフロントへ送信
		s.websocketHub.Broadcast(ctx, "match:"+matchID, payload)

		if err := s.completeMatch(ctx, matchID, players.Player1ID, players.Player2ID); err != nil {
			return fmt.Errorf("completeMatch failed")
		}

		return nil
	}

	// ===============================
	// ② 続行 → 次ラウンド初期化
	// ===============================
	initNextCtx, cancelNextState := context.WithTimeout(ctx, 300*time.Millisecond)
	err = s.roundStateRepo.InitializeNextRoundState(initNextCtx, matchID)
	cancelNextState()
	if err != nil {
		return fmt.Errorf("failed to prepare next round state: %w", err)
	}

	resetMeasurementCtx, cancelResetMeasurement := context.WithTimeout(ctx, 300*time.Millisecond)
	err = s.roundStateRepo.InitializeMeasurementFinishCount(resetMeasurementCtx, matchID)
	cancelResetMeasurement()
	if err != nil {
		return fmt.Errorf("failed to reset measurement finish count: %w", err)
	}

	playersCtx, cancelPlayers := context.WithTimeout(ctx, 200*time.Millisecond)
	players, err := s.roundStateRepo.LoadMatchPlayers(playersCtx, matchID)
	cancelPlayers()
	if err != nil {
		return fmt.Errorf("failed to load players: %w", err)
	}

	clearPlayer1Ctx, cancelClearPlayer1 := context.WithTimeout(ctx, 200*time.Millisecond)
	err = s.roundStateRepo.DeletePlayerAnswerFinishFlag(clearPlayer1Ctx, matchID, players.Player1ID)
	cancelClearPlayer1()
	if err != nil {
		return fmt.Errorf("failed to clear p1 flag: %w", err)
	}
	clearPlayer2Ctx, cancelClearPlayer2 := context.WithTimeout(ctx, 200*time.Millisecond)
	err = s.roundStateRepo.DeletePlayerAnswerFinishFlag(clearPlayer2Ctx, matchID, players.Player2ID)
	cancelClearPlayer2()
	if err != nil {
		return fmt.Errorf("failed to clear p2 flag: %w", err)
	}

	// デッキ番号・ラウンド番号の更新
	updateNextRoundCtx, cancelUpdateNextRound := context.WithTimeout(ctx, 300*time.Millisecond)
	nextDeckIndex, nextRound, err := s.roundStateRepo.UpdateNextRoundState(updateNextRoundCtx, matchID)
	cancelUpdateNextRound()
	if err != nil {
		return fmt.Errorf("failed to update next round state: %w", err)
	}

	// 次ラウンドの問題・時間情報を取得
	loadPromptCtx, cancelLoadPrompt := context.WithTimeout(ctx, 600*time.Millisecond)
	roundQuestion, err := s.nextRoundService.LoadNextPrompt(loadPromptCtx, matchID, nextDeckIndex)
	cancelLoadPrompt()
	if err != nil {
		return fmt.Errorf("failed to load next prompt: %w", err)
	}

	// 開始予定時刻＋終了予定時刻取得
	timingCtx, cancelTiming := context.WithTimeout(ctx, 600*time.Millisecond)
	roundStartAtMs, roundEndAtMs, err := s.nextRoundService.SaveRoundTiming(timingCtx, matchID, roundQuestion.LimitMs)
	cancelTiming()
	if err != nil {
		return fmt.Errorf("failed to save round timing: %w", err)
	}

	// ===============================
	// ③ WebSocketで次ラウンド開始通知
	// ===============================
	payload := websocket.NewRoundStartPayload(
		matchID,
		players.Player1ID,
		players.Player2ID,
		websocket.MatchState{
			Round:            nextRound,
			RoundStartAtMS:   roundStartAtMs,
			RoundEndAtMS:     roundEndAtMs,
			Player1Lifepoint: player1Lifepoint,
			Player2Lifepoint: player2Lifepoint,
		},
		websocket.PromptPayload{
			PromptTextJa: roundQuestion.PromptTextJa,
			TargetRomaji: roundQuestion.TargetRomaji,
			LimitMs:      roundQuestion.LimitMs,
		},
	)

	s.websocketHub.Broadcast(ctx, "match:"+matchID, payload)

	timeoutCtx, timeoutCancel := context.WithCancel(context.Background())
	s.timeoutCancelMap[matchID] = timeoutCancel
	go s.timeoutRoundService.ScheduleRoundTimeoutCheck(timeoutCtx, matchID, roundEndAtMs+config.GraceMs)

	return nil
}

// completeMatch は、試合の終了後に実行される「後処理専用」の関数。
func (s *RoundFlowService) completeMatch(ctx context.Context, matchID, player1ID, player2ID string) error {
	cleanupCtx, cancelCleanup := context.WithTimeout(ctx, 200*time.Millisecond)
	err := s.roundStateRepo.CleanupMatch(cleanupCtx, matchID, player1ID, player2ID)
	cancelCleanup()
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	roomName := fmt.Sprintf("match:%s", matchID)

	// Hub から全メンバーを除外
	conns := s.websocketHub.Members(roomName)
	for _, conn := range conns {
		_ = s.websocketHub.Leave(ctx, conn)
	}

	// ★ 重要：presence を "online" に戻して match_id を消す
	if s.presenceRepo != nil {
		now := time.Now().UnixMilli()
		_ = s.presenceRepo.SetOnline(ctx, player1ID, now)
		_ = s.presenceRepo.SetOnline(ctx, player2ID, now)
	}

	return nil
}
