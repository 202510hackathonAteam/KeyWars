package sql

import (
	"context"
	"math/rand"
	"time"

	"gorm.io/gorm"

	"keywars/backend/internal/config"
	domainmodel "keywars/backend/internal/domain/model"
	"keywars/backend/internal/domain/repository"
)

var _ repository.PromptRepository = (*PromptRepositorySQL)(nil)

type PromptRepositorySQL struct {
	db *gorm.DB
}

// NewPromptRepositorySQL は、SQL を用いた PromptRepository の生成。
func NewPromptRepositorySQL(db *gorm.DB) repository.PromptRepository {
	return &PromptRepositorySQL{db: db}
}

// 指定 difficulty_code ごとに n 件取得する関数
func (r *PromptRepositorySQL) GetPromptsByDifficulty(ctx context.Context, difficultyCode string, limitCount int) ([]domainmodel.DeckPrompt, error) {
	var promptList []domainmodel.DeckPrompt

	err := r.db.WithContext(ctx).
		Table("prompts AS p").
		Select(`
			p.prompt_text_ja AS prompt_text_ja,
			p.target_romaji AS target_romaji,
			d.time_limit_ms AS limit_ms
		`).
		Joins("JOIN difficulties d ON p.difficulty_id = d.id").
		Where("d.difficulty_code = ?", difficultyCode).
		Order("RAND()").
		Limit(limitCount).
		Scan(&promptList).Error

	return promptList, err
}

// easy/normal/hard を固定数ずつ取得して、難易度事にランダム順にして、難易度順に合体する関数
func (r *PromptRepositorySQL) GetDeckPrompts(ctx context.Context) ([]domainmodel.DeckPrompt, error) {
	allPrompts := make([]domainmodel.DeckPrompt, 0, 20)

	easyPrompts, err := r.GetPromptsByDifficulty(ctx, "easy", config.Deck.EasyCount)
	if err != nil {
		return nil, err
	}
	normalPrompts, err := r.GetPromptsByDifficulty(ctx, "normal", config.Deck.NormalCount)
	if err != nil {
		return nil, err
	}
	hardPrompts, err := r.GetPromptsByDifficulty(ctx, "hard", config.Deck.HardCount)
	if err != nil {
		return nil, err
	}

	// 各難易度内のランダム化
	randomGenerator := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomGenerator.Shuffle(len(easyPrompts), func(i, j int) {
		easyPrompts[i], easyPrompts[j] = easyPrompts[j], easyPrompts[i]
	})
	randomGenerator.Shuffle(len(normalPrompts), func(i, j int) {
		normalPrompts[i], normalPrompts[j] = normalPrompts[j], normalPrompts[i]
	})
	randomGenerator.Shuffle(len(hardPrompts), func(i, j int) {
		hardPrompts[i], hardPrompts[j] = hardPrompts[j], hardPrompts[i]
	})

	// 難易度順に結合
	allPrompts = append(allPrompts, easyPrompts...)
	allPrompts = append(allPrompts, normalPrompts...)
	allPrompts = append(allPrompts, hardPrompts...)

	return allPrompts, nil
}
