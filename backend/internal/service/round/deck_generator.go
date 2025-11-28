package round

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"keywars/backend/internal/domain/repository"

	"github.com/redis/go-redis/v9"
)

type DeckGeneratorService struct {
	dbRepo repository.PromptRepository
	redis  *redis.Client
}

func NewDeckGeneratorService(dbRepo repository.PromptRepository, redisClient *redis.Client) *DeckGeneratorService {
	return &DeckGeneratorService{
		dbRepo: dbRepo,
		redis:  redisClient,
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
	prompts, err := s.dbRepo.GetDeckPrompts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get prompts: %w", err)
	}

	key := fmt.Sprintf("match:%s:deck", matchID)
	pipeline := s.redis.TxPipeline()

	for _, p := range prompts {
		item := DeckItem{
			PromptTextJa:    p.PromptTextJa,
			TargetRomaji:    p.TargetRomaji,
			LimitMs:    		 p.TimeLimitMs,
		}
		jsonBytes, _ := json.Marshal(item)
		pipeline.RPush(ctx, key, jsonBytes)
	}

	pipeline.Expire(ctx, key, 1*time.Hour) // デッキTTLは1時間
	_, err = pipeline.Exec(ctx)
	return err
}
