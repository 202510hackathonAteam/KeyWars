package round

import (
	"context"
	"time"

	"keywars/backend/internal/domain/repository"
)

// NextRoundService は、次のラウンドに必要な問題（プロンプト）情報を
// 取得・生成するためのサービス。
type NextRoundService struct {
	roundStateRepo repository.RoundStateRepository
}

// NewNextRoundService は NextRoundService のコンストラクタ。
func NewNextRoundService(roundStateRepo repository.RoundStateRepository) *NextRoundService {
	return &NextRoundService{
		roundStateRepo: roundStateRepo,
	}
}

// PromptInfo は、1ラウンド分の出題情報を表す構造体。
type PromptInfo struct {
	PromptTextJa string `json:"prompt_text_ja"`
	TargetRomaji string `json:"target_romaji"`
	LimitMs int64 `json:"limit_ms"`
}

// LoadNextPrompt は、次のラウンドの問題を取得するメソッド。
func (s *NextRoundService) LoadNextPrompt(ctx context.Context, matchID string, deckIndex int64) (*PromptInfo, error) {
	// 次ラウンドの問題の取得
	nextPrompt, err := s.roundStateRepo.LoadDeckPrompt(ctx, matchID, deckIndex)
	if err != nil {
		return nil, err
	}

	return &PromptInfo{
		PromptTextJa: nextPrompt.PromptTextJa,
		TargetRomaji: nextPrompt.TargetRomaji,
		LimitMs: 			nextPrompt.LimitMs,
	}, nil
}

// SaveRoundTiming は、開始/終了予定時刻を作成して Redis に保存するメソッド。
func (s *NextRoundService) SaveRoundTiming(ctx context.Context, matchID string, limitMs int64) (int64, int64, error) {
	// 開始予定時刻と終了予定時刻を作成
	roundStartDelay := 1500 * time.Millisecond
	roundStartAtMs := time.Now().Add(roundStartDelay).UnixMilli()
	roundEndAtMs := roundStartAtMs + limitMs

	// 開始予定時刻と終了予定時刻をRedisに保存
	if err := s.roundStateRepo.UpdateRoundTiming(
		ctx, matchID, roundStartAtMs, roundEndAtMs,
	); err != nil {
		return 0, 0, err
	}

	return roundStartAtMs, roundEndAtMs, nil
}

// SaveFirstRoundTiming は初回ラウンドだけ開始までの余裕時間を長く取るメソッド。
// UIロードやフロント側での初回処理のため17秒遅延している。
func (s *NextRoundService) SaveFirstRoundTiming(ctx context.Context, matchID string, limitMs int64) (int64, int64, error) {
	// 開始予定時刻と終了予定時刻を作成
	roundStartDelay := 17000 * time.Millisecond
	roundStartAtMs := time.Now().Add(roundStartDelay).UnixMilli()
	roundEndAtMs := roundStartAtMs + limitMs

	// 開始予定時刻と終了予定時刻をRedisに保存
	if err := s.roundStateRepo.UpdateRoundTiming(
		ctx, matchID, roundStartAtMs, roundEndAtMs,
	); err != nil {
		return 0, 0, err
	}

	return roundStartAtMs, roundEndAtMs, nil
}