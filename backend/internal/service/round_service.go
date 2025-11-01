// internal/service/round_service.go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"keywars/backend/internal/domain/repository"
)

// RoundService は「試合（ラウンド）進行」のユースケースをまとめたサービス層。
// - メタ作成/開始/終了
// - デッキ保存（冪等）/要素取得
// - 回答確定（state更新 + events追記のトリガ）
//
// ※ 実際の原子的更新は repository.RoundStateRepository が担う。
//
//	ここでは入力値の整形、イベントフィールドの構成、簡易リトライなどを行う。
type RoundService struct {
	roundRepo repository.RoundStateRepository
	// problemRepo repository.ProblemRepository // デッキ生成用（未実装なら nil でOK）
}

// NewRoundService は RoundService を生成する。
func NewRoundService(round repository.RoundStateRepository) *RoundService {
	return &RoundService{roundRepo: round}
}

// WithProblemRepo は「出題デッキを問題リポジトリから組む」場合に設定する。
// func (s *RoundService) WithProblemRepo(p repository.ProblemRepository) *RoundService {
// 	s.problemRepo = p
// 	return s
// }

// -----------------------------
// デッキ関連
// -----------------------------

// DeckItem は deck 要素のJSONスキーマに対応するDTO。
type DeckItem struct {
	PID       int64  `json:"pid"`
	Surface   string `json:"surface"`
	Reading   string `json:"reading"`
	Diff      int    `json:"diff"`
	CharCount int    `json:"char_count"`
	LimitMS   int64  `json:"limit_ms"`
}

// SaveDeckOnce はデッキ（20問想定）を JSON 文字列として保存する（冪等）。
func (s *RoundService) SaveDeckOnce(ctx context.Context, mid string, items []DeckItem) error {
	if len(items) == 0 {
		return errors.New("empty deck")
	}
	// JSON文字列の配列に変換
	arr := make([]string, 0, len(items))
	for _, it := range items {
		b, err := json.Marshal(it)
		if err != nil {
			return err
		}
		arr = append(arr, string(b))
	}
	return s.roundRepo.SaveDeckOnce(ctx, mid, arr)
}

// GetDeckItem は deck_idx で1件取得（JSON文字列）→ DeckItem にデコードする。
func (s *RoundService) GetDeckItem(ctx context.Context, mid string, deckIdx int64) (DeckItem, error) {
	js, err := s.roundRepo.GetDeckItem(ctx, mid, deckIdx)
	if err != nil {
		return DeckItem{}, err
	}
	var di DeckItem
	if err := json.Unmarshal([]byte(js), &di); err != nil {
		return DeckItem{}, err
	}
	return di, nil
}

// -----------------------------
// メタ（開始/終了）
// -----------------------------

// CreateAndStart はメタ生成→開始（TTL付与）まで一括で行う。
// すでに外側で CreateMeta 済みなら Start だけを呼んでOK。
func (s *RoundService) CreateAndStart(ctx context.Context, mid, u1, u2 string, nowMs int64) error {
	if err := s.roundRepo.CreateMeta(ctx, mid, u1, u2, nowMs); err != nil {
		return err
	}
	return s.roundRepo.Start(ctx, mid)
}

// Start は既存メタ（p1/p2等が入っている）を playing にする。
func (s *RoundService) Start(ctx context.Context, mid string) error {
	return s.roundRepo.Start(ctx, mid)
}

// Finish は winner を設定し TTL を短縮する。
func (s *RoundService) Finish(ctx context.Context, mid, winnerUID string) error {
	return s.roundRepo.Finish(ctx, mid, winnerUID)
}

// -----------------------------
// 回答確定（ApplyAnswer）
// -----------------------------

// AnswerInput はサービス層が受ける“回答確定”の入力。
// - NewOppLP:   攻撃を受ける側（相手）の新しいLP（呼び出し元で計算して渡す）
// - Advance:    次の問題へ進めるか（true の時 NextDeckIdx を使う）
// - NextDeckIdx: 進める場合の deck_idx
// - NowMs:      サーバ時刻ms（未指定=0なら time.Now().UnixMilli() を使用）
type AnswerInput struct {
	MID         string
	AttackerUID string
	OpponentUID string
	NewOppLP    int64

	Advance     bool
	NextDeckIdx int64 // Advance=false のときは無視

	// イベント詳細（STREAMに保存する付加情報）
	PromptID        int64
	Correct         bool
	Damage          int64
	Chars           int64
	Misses          int64
	ClientElapsedMS int64
	TimeOK          bool
	First           bool

	NowMs int64 // 0なら自動設定
}

// ApplyAnswer は 1回答の確定処理をトリガする。
// - state の LP / turn / deck_idx / q_started_at_ms（必要時）を更新
// - events に XADD（turn, server_ts 等含む）
// - last_event_id は repo 側でベストエフォート更新
//
// 注意: LP の厳密更新（現在LPの取得→newLP算出→更新の衝突対策）は repo 側のWATCHで担保される想定。
//
//	ここでは“新LPを引数で受ける”方針にして、計算は呼び出し元で行わせる。
func (s *RoundService) ApplyAnswer(ctx context.Context, in AnswerInput) (eventID string, turn int64, err error) {
	if in.MID == "" || in.AttackerUID == "" || in.OpponentUID == "" {
		return "", 0, errors.New("invalid input (mid/attacker/opponent required)")
	}
	if in.NowMs == 0 {
		in.NowMs = time.Now().UnixMilli()
	}

	// NextDeckIdx の扱い
	nextIdx := int64(-1)
	if in.Advance {
		nextIdx = in.NextDeckIdx
	}

	// events に入れる付加フィールドの生成
	ev := map[string]string{
		"uid":        in.AttackerUID,
		"deck_idx":   strconv.FormatInt(nextIdx, 10), // 参照用。-1 の場合は進めていない
		"promptId":   strconv.FormatInt(in.PromptID, 10),
		"correct":    bool01(in.Correct),
		"dmg":        strconv.FormatInt(in.Damage, 10),
		"chars":      strconv.FormatInt(in.Chars, 10),
		"misses":     strconv.FormatInt(in.Misses, 10),
		"elapsed_ms": strconv.FormatInt(in.ClientElapsedMS, 10),
		"time_ok":    bool01(in.TimeOK),
		"first":      bool01(in.First),
		// server_ts と turn は repo 側で追加される（ApplyAnswer戻りで受け取る）
	}

	return s.roundRepo.ApplyAnswer(ctx, repository.AnswerApplyArg{
		MID:         in.MID,
		OppUID:      in.OpponentUID,
		NewOppLP:    in.NewOppLP,
		NextDeckIdx: nextIdx,
		NowMs:       in.NowMs,
		EventFields: ev,
	})
}

// -----------------------------
// ユーティリティ
// -----------------------------

func bool01(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
