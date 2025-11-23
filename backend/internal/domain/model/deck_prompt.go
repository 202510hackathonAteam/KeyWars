package model

// DeckPrompt は、デッキに保存されている1問ぶんのデータから
// この段階で必要となる最小限の情報だけを切り出した構造体。
type DeckPrompt struct {
	PromptTextJa string
	TargetRomaji string
	LimitMs    	 int64
}