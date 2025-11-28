package round

import (
	"context"
	"fmt"

	"keywars/backend/internal/domain/repository"
)

type DeckGeneratorService struct {
	roundStateRepo repository.RoundStateRepository
	promptRepo 		 repository.PromptRepository
}

func NewDeckGeneratorService(roundStateRepo repository.RoundStateRepository, promptRepo repository.PromptRepository) *DeckGeneratorService {
	return &DeckGeneratorService{
		roundStateRepo: roundStateRepo,
		promptRepo:  		promptRepo,
	}
}

// DeckItem は Redis に保存する出題情報
type DeckItem struct {
	PromptTextJa string `json:"prompt_text_ja"`
	TargetRomaji string `json:"target_romaji"`
	LimitMs    	 int    `json:"limit_ms"`
}

// GenerateAndSaveDeck は、指定 matchID のデッキを作成して Redis に保存する。
func (s *DeckGeneratorService) GenerateAndSaveDeck(ctx context.Context, matchID string) error {
	// SQLから問題取得
	prompts, err := s.promptRepo.GetDeckPrompts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get prompts: %w", err)
	}

	// Redisに問題保存
	if err := s.roundStateRepo.SaveDeck(ctx, matchID, prompts); err != nil {
    return fmt.Errorf("failed to save deck: %w", err)
	}

	return err
}
