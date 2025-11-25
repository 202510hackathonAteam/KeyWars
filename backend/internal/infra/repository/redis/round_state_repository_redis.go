package redisrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	drepo "keywars/backend/internal/domain/repository"
	"keywars/backend/internal/domain/model"
)

const (
	matchTTL        = time.Hour        // 試合開始時 EXPIRE 1h
	matchTTLPostFin = 10 * time.Minute // 終了後 PEXPIRE 10m は別箇所で
)

// RoundStateRepositoryRedis は、対戦進行中の「メタ情報・状態・イベント・デッキ」を
// Redis 上の複数キーに分割して管理するリポジトリ実装。
// キー構成：
//   - match:{matchID}         ... メタ情報（status, created_at, player1, player2, winner_user_id）
//   - match:{matchID}:state   ... 進行状態（deck_index, round_start_at_ms, round_end_at_ms, round, p{userID}:lp, last_event_id）
//   - match:{matchID}:events  ... イベント Streams（answer などの出来事）
//   - match:{matchID}:deck    ... 出題デッキ（LIST; 要素はJSON文字列）
type RoundStateRepositoryRedis struct {
	// redisClient は go-redis v9 のクライアント。
	// 1インスタンスを本構造体で共有して各操作に使用する。
	redisClient *redis.Client
}

// NewRoundStateRepositoryRedis は、外部から提供された Redis クライアントで実装を初期化する。
// DI しやすいコンストラクタ。
func NewRoundStateRepositoryRedis(redisClient *redis.Client) *RoundStateRepositoryRedis {
	return &RoundStateRepositoryRedis{redisClient: redisClient}
}

// CreateMeta は、マッチのメタ情報を新規作成する。
// 役割：参加者IDや作成時刻を保存し、初期状態を "waiting" に設定する。
// ここでは state/events/deck には触れず、責務を分離している。
func (repository *RoundStateRepositoryRedis) CreateMeta(contextObject context.Context, matchID, user1ID, user2ID string, currentTimeMs int64) error {
	metaKey := fmt.Sprintf("match:%s", matchID)
	return repository.redisClient.HSet(contextObject, metaKey,
		"status", "waiting",
		"created_at", currentTimeMs,
		"player1", user1ID,
		"player2", user2ID,
	).Err()
}

// Start は、マッチを playing 状態へ遷移し、関連キーへ TTL を設定する。
// 意図：進行中の試合データが放置されても自動的に回収されるよう GC を効かせる。
// TTL は運用方針に応じて調整可。
func (repository *RoundStateRepositoryRedis) Start(contextObject context.Context, matchID string) error {
	matchKey := fmt.Sprintf("match:%s", matchID)
	pipeline := repository.redisClient.TxPipeline()

	// ステータスを playing に更新
	pipeline.HSet(contextObject, matchKey, "status", "playing")

	for _, suffix := range []string{"", ":state", ":events", ":deck"} {
		pipeline.Expire(contextObject, matchKey+suffix, matchTTL)
	}

	_, err := pipeline.Exec(contextObject)
	return err
}

// Finish は、マッチを finished 状態に更新し、勝者を記録したうえで短い TTL に切り替える。
// 意図：終了後しばらくは参照できるが、不要に残り続けないようにする。
func (repository *RoundStateRepositoryRedis) Finish(contextObject context.Context, matchID, winnerUserID string) error {
	matchKey := fmt.Sprintf("match:%s", matchID)
	pipeline := repository.redisClient.TxPipeline()

	// ステータスと勝者IDを保存
	pipeline.HSet(contextObject, matchKey,
		"status", "finished",
		"winner_user_id", winnerUserID,
	)

	// 終了後は 10 分で掃除（ミリ秒精度で設定）
	for _, suffix := range []string{"", ":state", ":events", ":deck"} {
		pipeline.PExpire(contextObject, matchKey+suffix, matchTTLPostFin)
	}

	_, err := pipeline.Exec(contextObject)
	return err
}

// SaveDeckOnce は、出題デッキ(LIST)を「未作成のときだけ」保存する（冪等）。
// 競合時は WATCH により存在チェックと追加を疑似原子的に実施する。
func (repository *RoundStateRepositoryRedis) SaveDeck(ctx context.Context, matchID string, deck []drepo.PromptWithDifficulty) error {
	keyDeck := fmt.Sprintf("match:%s:deck", matchID)
	keyMeta := fmt.Sprintf("match:%s", matchID) // TTL を揃える対象

	// JSON 化して RPUSH
	args := make([]interface{}, 0, len(deck))
	for _, p := range deck {
		// フロントが欲しい形に最低限整える（必要に応じて項目名調整）
		payload := map[string]any{
			"promptTextJa": p.PromptTextJa,
			"reading":    	p.TargetRomaji,
			"limit_ms":   	p.TimeLimitMs,
		}
		b, _ := json.Marshal(payload)
		args = append(args, b)
	}

	pipe := repository.redisClient.TxPipeline()
	if len(args) > 0 {
		pipe.RPush(ctx, keyDeck, args...)
	}
	// 試合開始時に 1h で揃える（Meta が無いケースでも Deck 側へ設定しておく）
	pipe.Expire(ctx, keyDeck, matchTTL)
	pipe.Expire(ctx, keyMeta, matchTTL)

	_, err := pipe.Exec(ctx)
	return err
}

// LoadDeckPrompt は、指定したデッキインデックスのmatch:{matchID}:deck の JSON を構造体に変換するメソッド。
func (repository *RoundStateRepositoryRedis) LoadDeckPrompt(ctx context.Context, matchID string, deckIndex int64) (*model.DeckPrompt, error) {
	deckKey := fmt.Sprintf("match:%s:deck", matchID)
	nextPromptJson, err := repository.redisClient.LIndex(ctx, deckKey, deckIndex).Result()
	if err != nil {
		return nil, err
	}

	var nextPrompt model.DeckPrompt
	if err := json.Unmarshal([]byte(nextPromptJson), &nextPrompt); err != nil {
		return nil, err
	}

	return &nextPrompt, nil
}

// ApplyAnswer は、回答結果を state に反映し（LP, deck_index, q_started_at_ms, round）、
// 続けて events Streams に answer イベントを追加する 2 フェーズ処理。
// Phase A: WATCH + TxPipelined で round 競合を回避しつつ state 更新
// Phase B: XADD でイベントを記録し、last_event_id を best-effort で更新
func (repository *RoundStateRepositoryRedis) ApplyAnswer(contextObject context.Context, answerArgs drepo.AnswerApplyArg) (eventID string, round int64, err error) {
	stateKey := fmt.Sprintf("match:%s:state", answerArgs.MatchID)
	eventsKey := fmt.Sprintf("match:%s:events", answerArgs.MatchID)

	// --- Phase A: 状態更新（round の整合性確保のため WATCH を使用）---
	err = repository.redisClient.Watch(contextObject, func(transaction *redis.Tx) error {
		// 現在の round を取得。未設定なら 0 とみなす。
		values, err := transaction.HMGet(contextObject, stateKey, "round").Result()
		if err != nil {
			return err
		}

		var currentRound int64
		if len(values) > 0 && values[0] != nil {
			// HMGet は interface{} を返すため、string にキャストしてから parse
			currentRound, _ = strconv.ParseInt(values[0].(string), 10, 64)
		}

		// TxPipelined: WATCH 中の原子的更新
		_, err = transaction.TxPipelined(contextObject, func(pipeliner redis.Pipeliner) error {
			// 対戦相手の LP を更新。キーは p{userID}:lp（例: p123:lp）
			pipeliner.HSet(contextObject, stateKey, fmt.Sprintf("p%s:lp", answerArgs.OpponentUserID), answerArgs.NewOpponentLifePoint)

			// ターンを +1（競合があれば WATCH により失敗→再実行 or エラー）
			pipeliner.HIncrBy(contextObject, stateKey, "round", 1)

			// 次の問題へ進む場合のみ、deck_idx と q_started_at_ms を更新
			if answerArgs.NextDeckIndex >= 0 {
				pipeliner.HSet(contextObject, stateKey,
					"deck_index", answerArgs.NextDeckIndex,
					"q_started_at_ms", answerArgs.CurrentServerTimeMs,
				)
			}
			return nil
		})
		if err == nil {
			// 呼び出し側に「更新後の round 」を返す
			round = currentRound + 1
		}
		return err
	}, stateKey)
	if err != nil {
		return "", 0, err
	}

	// --- Phase B: イベント追加（予約済みフィールドを除外して積む）---
	eventValues := []interface{}{
		"type", "answer",
		"round", strconv.FormatInt(round, 10),
		"server_ts", strconv.FormatInt(answerArgs.CurrentServerTimeMs, 10),
	}
	for key, value := range answerArgs.EventFields {
		// 予約済みキーは上書きしない
		if key == "type" || key == "round" || key == "server_ts" {
			continue
		}
		eventValues = append(eventValues, key, value)
	}

	// Streams にイベントを追加。成功時のみ last_event_id を best-effort で更新。
	eventID, err = repository.redisClient.XAdd(contextObject, &redis.XAddArgs{
		Stream: eventsKey,
		Values: eventValues,
	}).Result()
	if err == nil {
		_ = repository.redisClient.HSet(contextObject, stateKey, "last_event_id", eventID).Err()
	}
	return
}

// InitializeNextRoundState は、次ラウンド開始のために
// ラウンド内で使用する一時的な state（予定時刻や計測フラグなど）を初期化するメソッド。
func (repository *RoundStateRepositoryRedis) InitializeNextRoundState(ctx context.Context, matchID string) error {
	stateKey := fmt.Sprintf("match:%s:state", matchID)

	return repository.redisClient.HSet(ctx, stateKey, map[string]interface{}{
		"round_start_at_ms": 0,
		"round_end_at_ms": 0,
		"player1_answer_finished": false,
		"player2_answer_finished": false,
	}).Err()
}

// LoadMatchState は、Redis の match:{matchID}:state に保存されている現在の試合状態を取得するメソッド。
func (repository *RoundStateRepositoryRedis) LoadMatchState(ctx context.Context, matchID string) (*model.MatchState, error) {
	stateKey := fmt.Sprintf("match:%s:state", matchID)

	// Redis から hash を取得
	matchStateRow, err := repository.redisClient.HGetAll(ctx, stateKey).Result()
	if err != nil {
		return nil, err
	}
	if len(matchStateRow) == 0 {
		return nil, fmt.Errorf("state not found")
	}

	// int64に変換してセット
	state := &model.MatchState{
		DeckIndex:        		 					parseInt64Field(matchStateRow, "deck_index"),
		Round:            		 					parseInt64Field(matchStateRow, "round"),
		Player1AnswerFinished: 					parseBoolField(matchStateRow, "player1_answer_finished"),
		Player2AnswerFinished: 					parseBoolField(matchStateRow, "player2_answer_finished"),
		Player1Lifepoint: 		 					parseInt64Field(matchStateRow, "player1_lifepoint"),
		Player2Lifepoint: 		 					parseInt64Field(matchStateRow, "player2_lifepoint"),
	}

	return state, nil
}

// UpdateNextRound は、次ラウンドへ進むために deck_index と round を
// 1つプラスして更新するメソッド。
func (repository *RoundStateRepositoryRedis) UpdateNextRound(ctx context.Context, matchID string) (int64, int64, error) {
	stateKey := fmt.Sprintf("match:%s:state", matchID)

	nextDeckIndex, err := repository.redisClient.HIncrBy(ctx, stateKey, "deck_index", 1).Result()
	if err != nil {
		return 0, 0, err
	}

	nextRound, err := repository.redisClient.HIncrBy(ctx, stateKey, "round", 1).Result()
	if err != nil {
		return 0, 0, err
	}

	return nextDeckIndex, nextRound, nil
}

// UpdateRoundTiming は、ラウンドの開始予定時刻と終了予定時刻を
// Redis の match:{matchID}:state に保存するメソッド。
func (repository *RoundStateRepositoryRedis) UpdateRoundTiming(ctx context.Context, matchID string, roundStartAtMs, roundEndAtMs int64) error {
	stateKey := fmt.Sprintf("match:%s:state", matchID)
	return repository.redisClient.HSet(ctx, stateKey, map[string]interface{}{
		"round_start_at_ms": roundStartAtMs,
		"round_end_at_ms":   roundEndAtMs,
	}).Err()
}

// LoadMatchPlayers は Redis に保存された
// match:{matchID} のプレイヤー情報（player1 / player2）を取得するメソッド。
func (repository *RoundStateRepositoryRedis) LoadMatchPlayers(ctx context.Context, matchID string) (*model.MatchPlayers, error) {
	matchKey := fmt.Sprintf("match:%s", matchID)

	data, err := repository.redisClient.HGetAll(ctx, matchKey).Result()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("match not found: %s", matchID)
	}

	players := &model.MatchPlayers{
		Player1ID: data["player1"],
		Player2ID: data["player2"],
	}

	return players, nil
}

// InitMeasurementFinishCountは、試合開始直後に呼び出される初期化処理するメソッド
func (repository *RoundStateRepositoryRedis) InitMeasurementFinishCount(ctx context.Context, matchID string) error {
	measurementFinishedCountKey := fmt.Sprintf("match:%s:measurement_finished_count", matchID)

	return repository.redisClient.Set(ctx, measurementFinishedCountKey, 0, 1*time.Hour).Err()
}

// InitializeMeasurementFinishCount は、指定された matchID に紐づく
// 「計測完了人数（measurement_finished_count）」カウンタを 0 に初期化するメソッド。
func (repository *RoundStateRepositoryRedis) InitializeMeasurementFinishCount(ctx context.Context, matchID string) error {
	measurementFinishedCountKey := fmt.Sprintf("match:%s:measurement_finished_count", matchID)

	return repository.redisClient.Set(ctx, measurementFinishedCountKey, 0, 1*time.Hour).Err()
}

// IncrementMeasurementFinishCount は、指定された matchID に紐づく
// 「計測完了人数（measurement_finished_count）」カウンタを +1 するメソッド。
func (repository *RoundStateRepositoryRedis) IncrementMeasurementFinishCount(ctx context.Context, matchID string) (int64, error) {
	measurementFinishedCountKey := fmt.Sprintf("match:%s:measurement_finished_count", matchID)

	measurementFinishedCount, err := repository.redisClient.Incr(ctx, measurementFinishedCountKey).Result()
	if err != nil {
		return 0, err
	}

	return measurementFinishedCount, nil
}


// ---- ヘルパー関数 ----
// parseInt64Field は、Redis HGETALL の結果(map[string]string)から
// 指定したフィールド名の値を int64 にパースして返す関数。
func parseInt64Field(row map[string]string, field string) int64 {
	rowValue, exists := row[field]
	if !exists {
		return 0
  }
	parsedValue, err := strconv.ParseInt(rowValue, 10, 64)
	if err != nil {
		return 0
	}

	return parsedValue
}

// parseBoolField は、Redis HGETALL の結果(map[string]string)から
// 指定したフィールド名の値を bool にパースして返す関数。
func parseBoolField(row map[string]string, field string) bool {
	rowValue, exists := row[field]
	if !exists {
		return false
  }
	parsedValue, err := strconv.ParseBool(rowValue)
	if err != nil {
		return false
	}

	return parsedValue
}
