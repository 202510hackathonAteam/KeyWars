package model

type DeckPayload struct {
	PromptTextJa string `json:"prompt_text_ja"`
	TargetRomaji string `json:"target_romaji"`
	LimitMs      int64  `json:"limit_ms"`
}