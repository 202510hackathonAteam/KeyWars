package sql

import (
	"context"
	"math/rand"
	"time"

	domainrepo "keywars/backend/internal/domain/repository"

	"gorm.io/gorm"
)

type PromptRepositorySQL struct {
	db *gorm.DB
}

func NewPromptRepositorySQL(db *gorm.DB) *PromptRepositorySQL {
	return &PromptRepositorySQL{db: db}
}

// 指定 difficulty_code ごとに n 件取得する関数
func (repository *PromptRepositorySQL) GetPromptsByDifficulty(ctx context.Context, difficultyCode string, limitCount int) ([]domainrepo.PromptWithDifficulty, error) {
	var promptList []domainrepo.PromptWithDifficulty

	err := repository.db.WithContext(ctx).
		Table("prompts AS p").
		Select(`
			p.id,
			p.prompt_text_ja,
			p.target_romaji,
			d.id AS difficulty_level,
			d.time_limit_ms
		`).
		Joins("JOIN difficulties d ON p.difficulty_id = d.id").
		Where("d.difficulty_code = ?", difficultyCode).
		Order("RAND()").
		Limit(limitCount).
		Scan(&promptList).Error

	return promptList, err
}

// easy/normal/hard を固定数ずつ取得して、難易度事にランダム順にして、難易度順に合体する関数
func (repository *PromptRepositorySQL) GetDeckPrompts(ctx context.Context) ([]domainrepo.PromptWithDifficulty, error) {
	allPrompts := make([]domainrepo.PromptWithDifficulty, 0, 20)

	easyPrompts, err := repository.GetPromptsByDifficulty(ctx, "easy", 6)
	if err != nil {
		return nil, err
	}
	normalPrompts, err := repository.GetPromptsByDifficulty(ctx, "normal", 6)
	if err != nil {
		return nil, err
	}
	hardPrompts, err := repository.GetPromptsByDifficulty(ctx, "hard", 8)
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
