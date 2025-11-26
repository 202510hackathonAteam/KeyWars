package redisrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"

	drepo "keywars/backend/internal/domain/repository"
	"keywars/backend/internal/domain/model"
	"keywars/backend/internal/config"
	"keywars/backend/internal/domain/types"
)

// RoundStateRepositoryRedis は、対戦進行中の「メタ情報・状態・イベント・デッキ」を
// Redis 上の複数キーに分割して管理するリポジトリ実装。
// キー構成：
//   - match:{matchID}         ... メタ情報（status, created_at, player1, player2, winner_user_id）
//   - match:{matchID}:state   ... 進行状態（deck_index, round, round_start_at_ms, round_end_at_ms,
//                                     player1_lifepoint, player2_lifepoint,
//                                     player1_total_miss_count, player2_total_miss_count）
//   - match:{matchID}:events  ... イベント Streams（answer などの出来事）
//   - match:{matchID}:deck    ... 出題デッキ（LIST; 要素はJSON文字列）
//   - match:{matchID}:measurement_finished_count
//     ... ラウンド内で「計測完了した人数」を保持するカウンタ（0 → 1 → 2）
//   - match:{matchID}:answer_finish_flag:{userID}
//     ... 各プレイヤーの finish トリガー受付フラグ（SetNX による冪等制御用）
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
		pipeline.Expire(contextObject, matchKey+suffix, config.MatchExpiryOnStart)
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
		pipeline.PExpire(contextObject, matchKey+suffix, config.MatchExpiryOnFinish)
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
			"prompt_text_ja": p.PromptTextJa,
			"target_romaji": p.TargetRomaji,
			"limit_ms": p.TimeLimitMs,
		}
		b, _ := json.Marshal(payload)
		args = append(args, b)
	}

	pipe := repository.redisClient.TxPipeline()
	if len(args) > 0 {
		pipe.RPush(ctx, keyDeck, args...)
	}
	// 試合開始時に 1h で揃える（Meta が無いケースでも Deck 側へ設定しておく）
	pipe.Expire(ctx, keyDeck, config.MatchExpiryOnStart)
	pipe.Expire(ctx, keyMeta, config.MatchExpiryOnStart)

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

	var nextPromptRaw map[string]interface{}
	if err := json.Unmarshal([]byte(nextPromptJson), &nextPromptRaw); err != nil {
		return nil, err
	}

	return &model.DeckPrompt{
		PromptTextJa: fmt.Sprintf("%v", nextPromptRaw["prompt_text_ja"]),
		TargetRomaji: fmt.Sprintf("%v", nextPromptRaw["target_romaji"]),
		LimitMs:      int64(nextPromptRaw["limit_ms"].(float64)),
	}, nil
}

// StoreFinishEvent は、回答確定時のイベント（miss数・終了時刻・ラウンド情報）を
// Redis Streams に保存するメソッド。
func (repository *RoundStateRepositoryRedis) StoreFinishEvent(
	ctx context.Context,
	matchID,
	userID string,
	round,
	missCount,
	finishAtMs int64,
) error {
	eventsKey := fmt.Sprintf("match:%s:events", matchID)

	_, err := repository.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: eventsKey,
		Values: map[string]interface{}{
			"player_id": userID,
			"round": round,
			"miss_count": missCount,
			"finish_at_ms": finishAtMs,
		},
	}).Result()

	return err
}

// LoadFinishEventsByRound は、Redis Stream に記録された MatchFinishEvents から
// 指定ラウンドのイベントだけを最大 RequiredPlayers 件（通常2件）読み込み、返すメソッド。
func (repository *RoundStateRepositoryRedis) LoadFinishEventsByRound(ctx context.Context, matchID string, round int64) ([]model.MatchFinishEvents, error) {
	eventsKey := fmt.Sprintf("match:%s:events", matchID)
	streamMessages, err := repository.redisClient.XRange(ctx, eventsKey, "-", "+").Result()
	if err != nil {
		return nil, err
	}

	var finishEvents []model.MatchFinishEvents

	// 取得した全イベントを走査
	for _, streamMessage := range streamMessages {
		// Values は map[string]interface{}
		eventValues := streamMessage.Values

		// round フィールド取得（int64）
		eventRound := parseInt64FromStream(eventValues, "round")
		if eventRound != round {
			continue
		}

		// 各値を取得
		eventPlayerID := parseStringFromStream(eventValues, "player_id")
		eventMissCount := parseInt64FromStream(eventValues, "miss_count")
		eventFinishAtMs := parseInt64FromStream(eventValues, "finish_at_ms")

		// 結果に追加
		finishEvents = append(finishEvents, model.MatchFinishEvents{
			PlayerID:   eventPlayerID,
			Round:      eventRound,
			MissCount:  eventMissCount,
			FinishAtMs: eventFinishAtMs,
		})

		// 規定値そろったら終了
		if len(finishEvents) == int(config.RequiredPlayers) {
			break
		}
	}

	if len(finishEvents) < int(config.RequiredPlayers) {
		return nil, fmt.Errorf("finish events not ready: got %d, need %d",
			len(finishEvents), config.RequiredPlayers)
	}

	return finishEvents, nil
}

// InitializeNextRoundState は、次ラウンド開始のために
// ラウンド内で使用する一時的な state（予定時刻など）を初期化するメソッド。
func (repository *RoundStateRepositoryRedis) InitializeNextRoundState(ctx context.Context, matchID string) error {
	stateKey := fmt.Sprintf("match:%s:state", matchID)

	return repository.redisClient.HSet(ctx, stateKey, map[string]interface{}{
		"round_start_at_ms": 0,
		"round_end_at_ms": 0,
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
		DeckIndex:      	parseInt64FromHash(matchStateRow, "deck_index"),
		Round:          	parseInt64FromHash(matchStateRow, "round"),
		RoundStartAtMS:  	parseInt64FromHash(matchStateRow, "round_start_at_ms"),
		RoundEndAtMS:   	parseInt64FromHash(matchStateRow, "round_end_at_ms"),
		Player1Lifepoint: parseInt64FromHash(matchStateRow, "player1_lifepoint"),
		Player2Lifepoint: parseInt64FromHash(matchStateRow, "player2_lifepoint"),
		Player1TotalMissCount: parseInt64FromHash(matchStateRow, "player1_total_miss_count"),
		Player2TotalMissCount: parseInt64FromHash(matchStateRow, "player2_total_miss_count"),
	}

	return state, nil
}

// UpdateTotalMissCount は、指定されたプレイヤー（player1 / player2）の
// 累計ミス数カウントに missCount を加算するメソッド。
func (repository *RoundStateRepositoryRedis) UpdateTotalMissCount(
	ctx context.Context,
	matchID,
	playerField string,
	missCount int64,
) error {
	stateKey := fmt.Sprintf("match:%s:state", matchID)
	totalMissCountField := playerField + "_total_miss_count"

	_, err := repository.redisClient.HIncrBy(
		ctx,
		stateKey,
		totalMissCountField,
		missCount,
	).Result()

	return err
}

// ReduceLifepoint は、指定されたプレイヤーのライフポイントを指定したダメージ分だけ減算するメソッド。
func (repository *RoundStateRepositoryRedis) ReduceLifepoint(ctx context.Context, matchID string, playerField string, damage int64) (int64, error) {
	stateKey := fmt.Sprintf("match:%s:state", matchID)
	playerLifepointField := playerField + "_lifepoint"

	lifepoint, err := repository.redisClient.HIncrBy(
		ctx,
		stateKey,
		playerLifepointField,
		-damage,
	).Result()

	if err != nil {
		return 0, err
	}

	if lifepoint < 0 {
		lifepoint = 0
		_ = repository.redisClient.HSet(ctx, stateKey, playerLifepointField, 0).Err()
	}

	return lifepoint, nil
}

// UpdateNextRoundState は、次ラウンドへ進むために deck_index と round を
// 1つプラスして更新するメソッド。
func (repository *RoundStateRepositoryRedis) UpdateNextRoundState(ctx context.Context, matchID string) (int64, int64, error) {
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

	return repository.redisClient.Set(ctx, measurementFinishedCountKey, 0, config.MatchExpiryOnStart).Err()
}

// InitializeMeasurementFinishCount は、指定された matchID に紐づく
// 「計測完了人数（measurement_finished_count）」カウンタを 0 に初期化するメソッド。
func (repository *RoundStateRepositoryRedis) InitializeMeasurementFinishCount(ctx context.Context, matchID string) error {
	measurementFinishedCountKey := fmt.Sprintf("match:%s:measurement_finished_count", matchID)

	return repository.redisClient.Set(ctx, measurementFinishedCountKey, 0, config.MatchExpiryOnStart).Err()
}

// IncrementMeasurementFinishCount は、指定された matchID に紐づく
// 「計測完了人数（measurement_finished_count）」カウンタを +1 するメソッド。
func (repository *RoundStateRepositoryRedis) IncrementMeasurementFinishCount(ctx context.Context, matchID string) (types.PlayerCount, error) {
	measurementFinishedCountKey := fmt.Sprintf("match:%s:measurement_finished_count", matchID)

	measurementFinishedCountInt64, err := repository.redisClient.Incr(ctx, measurementFinishedCountKey).Result()
	if err != nil {
		return 0, err
	}

	return types.PlayerCount(measurementFinishedCountInt64), nil
}

// RegisterPlayerAnswerFinishFlag は、プレイヤーの finish トリガーを
// 原子的に「初回のみ」受け付けるメソッド。
func (repository *RoundStateRepositoryRedis) RegisterPlayerAnswerFinishFlag(ctx context.Context, matchID, userID string) (bool, error) {
	flagKey := fmt.Sprintf("match:%s:answer_finish_flag:%s", matchID, userID)

	isFirstFinished, err := repository.redisClient.SetNX(ctx, flagKey, true, config.MatchExpiryOnStart).Result()
	if err != nil {
		return false, err
	}

	return isFirstFinished, nil
}

// IsPlayerAnswerFinishFlagExists は、指定したプレイヤーの finish フラグキーが
// Redis 上に存在するかどうかを返すメソッド。
func (repository *RoundStateRepositoryRedis) IsPlayerAnswerFinishFlagExists(ctx context.Context, matchID, userID string) (bool, error) {
	flagKey := fmt.Sprintf("match:%s:answer_finish_flag:%s", matchID, userID)

	isPlayerFinishFlagExists, err := repository.redisClient.Exists(ctx, flagKey).Result()
	if err != nil {
		return false, err
	}

	return isPlayerFinishFlagExists==1, nil
}

// DeletePlayerAnswerFinishFlag は、プレイヤーの finish フラグキーを削除し、
// 次のラウンドで再び RegisterPlayerFinishFlag の SetNX が成功するようにリセットするメソッド。
func (repository *RoundStateRepositoryRedis) DeletePlayerAnswerFinishFlag(ctx context.Context, matchID, userID string) error {
	flagKey := fmt.Sprintf("match:%s:answer_finish_flag:%s", matchID, userID)
	return repository.redisClient.Del(ctx, flagKey).Err()
}


// ---- ヘルパー関数 ----
// parseInt64FromHash は、Redis HGETALL の結果(map[string]string)から
// 指定したフィールド名の値を int64 にパースして返す関数。
func parseInt64FromHash(raw map[string]string, field string) int64 {
	rawValue, exists := raw[field]
	if !exists {
		return 0
  }
	parsedValue, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		return 0
	}

	return parsedValue
}

// parseStringFromStream は Redis Streams の Values(map[string]interface{}) から
// 指定フィールドを string として取り出す関数。
func parseStringFromStream(streamValues map[string]interface{}, fieldName string) string {
	fieldRawValue, exists := streamValues[fieldName]
	if !exists {
		return ""
	}

	fieldStringValue, isString := fieldRawValue.(string)
	if !isString {
		return ""
	}

	return fieldStringValue
}

// parseInt64FromStream は Redis Streams の Values(map[string]interface{}) から
// 指定フィールドを int64 として取り出す関数。
func parseInt64FromStream(streamValues map[string]interface{}, fieldName string) int64 {
	rawValue, exists := streamValues[fieldName]
	if !exists {
		return 0
  }
	parsedValueString, ok := rawValue.(string)
	if !ok {
		return 0
	}

	parsedValue, err := strconv.ParseInt(parsedValueString, 10, 64)
	if err != nil {
		return 0
	}

	return parsedValue
}
