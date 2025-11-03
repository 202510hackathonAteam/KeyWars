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
	// roundStateRepository は、試合状態（state, deck, events）を管理するドメインリポジトリ。
	roundStateRepository repository.RoundStateRepository

	// problemRepository は「出題デッキを問題リポジトリから組む」場合に利用する（任意依存）。
	// problemRepository repository.ProblemRepository // デッキ生成用（未実装なら nil でOK）
}

// NewRoundService は RoundService を生成する。
func NewRoundService(roundStateRepository repository.RoundStateRepository) *RoundService {
	return &RoundService{roundStateRepository: roundStateRepository}
}

// WithProblemRepo は「出題デッキを問題リポジトリから組む」場合に設定する。
// func (service *RoundService) WithProblemRepo(problemRepo repository.ProblemRepository) *RoundService {
// 	service.problemRepository = problemRepo
// 	return service
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
func (service *RoundService) SaveDeckOnce(contextObject context.Context, matchID string, deckItems []DeckItem) error {
	if len(deckItems) == 0 {
		return errors.New("empty deck")
	}
	// JSON文字列の配列に変換
	jsonArray := make([]string, 0, len(deckItems))
	for _, deckItem := range deckItems {
		bytesJSON, marshalErr := json.Marshal(deckItem)
		if marshalErr != nil {
			return marshalErr
		}
		jsonArray = append(jsonArray, string(bytesJSON))
	}
	return service.roundStateRepository.SaveDeckOnce(contextObject, matchID, jsonArray)
}

// GetDeckItem は deck_index で1件取得（JSON文字列）→ DeckItem にデコードする。
func (service *RoundService) GetDeckItem(contextObject context.Context, matchID string, deckIndex int64) (DeckItem, error) {
	jsonString, err := service.roundStateRepository.GetDeckItem(contextObject, matchID, deckIndex)
	if err != nil {
		return DeckItem{}, err
	}
	var deckItem DeckItem
	if unmarshalErr := json.Unmarshal([]byte(jsonString), &deckItem); unmarshalErr != nil {
		return DeckItem{}, unmarshalErr
	}
	return deckItem, nil
}

// -----------------------------
// メタ（開始/終了）
// -----------------------------

// CreateAndStart はメタ生成→開始（TTL付与）まで一括で行う。
// すでに外側で CreateMeta 済みなら Start だけを呼んでOK。
func (service *RoundService) CreateAndStart(contextObject context.Context, matchID, user1ID, user2ID string, currentTimeMs int64) error {
	if err := service.roundStateRepository.CreateMeta(contextObject, matchID, user1ID, user2ID, currentTimeMs); err != nil {
		return err
	}
	return service.roundStateRepository.Start(contextObject, matchID)
}

// Start は既存メタ（p1/p2等が入っている）を playing にする。
func (service *RoundService) Start(contextObject context.Context, matchID string) error {
	return service.roundStateRepository.Start(contextObject, matchID)
}

// Finish は winner を設定し TTL を短縮する。
func (service *RoundService) Finish(contextObject context.Context, matchID, winnerUserID string) error {
	return service.roundStateRepository.Finish(contextObject, matchID, winnerUserID)
}

// -----------------------------
// 回答確定（ApplyAnswer）
// -----------------------------

// AnswerInput はサービス層が受ける“回答確定”の入力。
// - NewOpponentLifePoint:   攻撃を受ける側（相手）の新しいLP（呼び出し元で計算して渡す）
// - Advance:                次の問題へ進めるか（true の時 NextDeckIndex を使う）
// - NextDeckIndex:          進める場合の deck_index
// - CurrentServerTimeMs:    サーバ時刻ms（未指定=0なら time.Now().UnixMilli() を使用）
type AnswerInput struct {
	MatchID              string
	AttackerUserID       string
	OpponentUserID       string
	NewOpponentLifePoint int64

	Advance       bool
	NextDeckIndex int64 // Advance=false のときは無視

	// イベント詳細（STREAMに保存する付加情報）
	PromptID        int64
	Correct         bool
	Damage          int64
	Chars           int64
	Misses          int64
	ClientElapsedMS int64
	TimeOK          bool
	First           bool

	CurrentServerTimeMs int64 // 0なら自動設定
}

// ApplyAnswer は 1回答の確定処理をトリガする。
// - state の LP / turn / deck_idx / q_started_at_ms（必要時）を更新
// - events に XADD（turn, server_ts 等含む）
// - last_event_id は repository 側でベストエフォート更新
//
// 注意: LP の厳密更新（現在LPの取得→newLP算出→更新の衝突対策）は repository 側のWATCHで担保される想定。
//
//	ここでは“新LPを引数で受ける”方針にして、計算は呼び出し元で行わせる。
func (service *RoundService) ApplyAnswer(contextObject context.Context, input AnswerInput) (eventID string, turn int64, err error) {
	// 必須入力のバリデーション
	if input.MatchID == "" || input.AttackerUserID == "" || input.OpponentUserID == "" {
		return "", 0, errors.New("invalid input (matchID/attackerUserID/opponentUserID required)")
	}
	// サーバ時刻が未指定の場合は現在時刻を設定
	if input.CurrentServerTimeMs == 0 {
		input.CurrentServerTimeMs = time.Now().UnixMilli()
	}

	// NextDeckIndex の扱い（Advance=false の場合は -1 として repo 側で無視させる）
	nextDeckIndex := int64(-1)
	if input.Advance {
		nextDeckIndex = input.NextDeckIndex
	}

	// events に入れる付加フィールドの生成（予約語以外）
	eventFields := map[string]string{
		"uid":        input.AttackerUserID,                         // 攻撃者
		"deck_idx":   strconv.FormatInt(nextDeckIndex, 10),         // -1 の場合は「進めていない」
		"promptId":   strconv.FormatInt(input.PromptID, 10),        // 出題ID
		"correct":    boolTo01(input.Correct),                      // "1"/"0"
		"dmg":        strconv.FormatInt(input.Damage, 10),          // 与ダメ
		"chars":      strconv.FormatInt(input.Chars, 10),           // 入力文字数
		"misses":     strconv.FormatInt(input.Misses, 10),          // ミス回数
		"elapsed_ms": strconv.FormatInt(input.ClientElapsedMS, 10), // クライアント計測
		"time_ok":    boolTo01(input.TimeOK),                       // 制限時間内か
		"first":      boolTo01(input.First),                        // 先着か
		// server_ts と turn は repository 側で付与される（ApplyAnswer の戻り値で取得）
	}

	// リポジトリ層へ委譲（※ AnswerApplyArg は「長い命名」版を想定）
	return service.roundStateRepository.ApplyAnswer(contextObject, repository.AnswerApplyArg{
		MatchID:              input.MatchID,
		OpponentUserID:       input.OpponentUserID,
		NewOpponentLifePoint: input.NewOpponentLifePoint,
		NextDeckIndex:        nextDeckIndex,
		CurrentServerTimeMs:  input.CurrentServerTimeMs,
		EventFields:          eventFields,
	})
}

// -----------------------------
// ユーティリティ
// -----------------------------

// boolTo01 は、bool を "1"/"0" の文字列へ変換する補助関数。
// イベントフィールドに布値を入れるときの表現を統一する。
func boolTo01(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
