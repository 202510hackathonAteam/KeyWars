package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"

	_ "embed"
	domainmodel "keywars/backend/internal/domain/model"
	"keywars/backend/internal/domain/repository"
	inframodel "keywars/backend/internal/infra/repository/redis/model"
	"keywars/backend/internal/config"
	"keywars/backend/internal/domain/types"
)

var _ repository.RoundStateRepository = (*RoundStateRepositoryRedis)(nil)

//go:embed scripts/set_key_with_ttl.lua
var setKeyWithTTLScriptSource string

var setKeyWithTTLScript = redis.NewScript(setKeyWithTTLScriptSource)

var criticalScripts = map[string]string{
	"set_key_with_ttl.lua": setKeyWithTTLScriptSource,
}

// init は、重要な Lua スクリプトが正しく embed されていることを起動時に検証する関数。
// embed に失敗した場合でも Go のコンパイルや Redis Script の実行自体は成功してしまい、
// 実行時に静かに不整合が発生するため、ここで fail-fast させる。
func init() {
	for name, src := range criticalScripts {
		if len(src) == 0 {
			panic("embed failed: " + name)
		}
	}
}

// RoundStateRepositoryRedis は、対戦進行中の「メタ情報・状態・イベント・デッキ」を
// Redis 上の複数キーに分割して管理するリポジトリ実装。
// キー構成：
// 	 - user_active_match:{userID} ... ユーザーIDに紐づいたマッチID（matchID）
//   - match:{matchID}         ... メタ情報（player1, player2）
//   - match:{matchID}:state   ... 進行状態（deck_index, round, round_start_at_ms, round_end_at_ms,
//                                     player1_lifepoint, player2_lifepoint,
//                                     player1_total_miss_count, player2_total_miss_count）
// 	 - match:{matchID}:frontend_state ... フロントエンド再構築用の最新スナップショット
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

// NewRoundStateRepositoryRedis は、Redis を用いた RoundStateRepository の生成。
func NewRoundStateRepositoryRedis(redisClient *redis.Client) repository.RoundStateRepository {
	return &RoundStateRepositoryRedis{redisClient: redisClient}
}

// CleanupMatch は、1つのマッチが完全に終了した後に呼び出される
// 「再戦に影響する一時データのみ」を安全に削除するクリーンアップ処理するメソッド。
func (r *RoundStateRepositoryRedis) CleanupMatch(ctx context.Context, matchID, user1ID, user2ID string) error {
	keys := []string{
		fmt.Sprintf("user_active_match:%s", user1ID),
		fmt.Sprintf("user_active_match:%s", user2ID),
		fmt.Sprintf("match:%s", matchID),
		fmt.Sprintf("match:%s:frontend_state", matchID),
		fmt.Sprintf("match:%s:state", matchID),
		fmt.Sprintf("match:%s:measurement_finished_count", matchID),
		fmt.Sprintf("match:%s:answer_finish_flag:%s", matchID, user1ID),
		fmt.Sprintf("match:%s:answer_finish_flag:%s", matchID, user2ID),
	}

	return r.redisClient.Del(ctx, keys...).Err()
}

// SetActiveMatchForUsers は、指定された複数ユーザーを同一試合に原子的に紐づけるメソッド。
// 全ユーザー分の対応関係が成功した場合のみ確定し、
// 途中失敗による部分的な保存は発生しない。
func (r *RoundStateRepositoryRedis) SetActiveMatchForUsers(
	ctx context.Context,
	user1ID,
	user2ID,
	matchID string,
) error {
	keys := []string{
		fmt.Sprintf("user_active_match:%s", user1ID),
		fmt.Sprintf("user_active_match:%s", user2ID),
	}
	return setKeyWithTTLScript.
		Run(ctx, r.redisClient, keys, matchID, config.MatchExpiryOnStart.Milliseconds()).
		Err()
}

// LoadUserActiveMatchID は、ユーザーが現在参加している試合の matchID を取得するメソッド。
// user_active_match:{userID} に保存された逆引きインデックスを参照し、
// 試合に参加していない場合は Redis のエラーをそのまま返す。
func (r *RoundStateRepositoryRedis) LoadUserActiveMatchID(
	ctx context.Context,
	userID string,
) (string, error) {
	userMatchKey := fmt.Sprintf("user_active_match:%s", userID)
	return r.redisClient.Get(ctx, userMatchKey).Result()
}

// SaveDeck は、試合で使用する出題デッキ（20問分）を Redis に保存するメソッド。
func (r *RoundStateRepositoryRedis) SaveDeck(ctx context.Context, matchID string, deck []domainmodel.DeckPrompt) error {
	deckKey := fmt.Sprintf("match:%s:deck", matchID)
	metaKey := fmt.Sprintf("match:%s", matchID) // TTL を揃える対象

	pipeline := r.redisClient.TxPipeline()

	for _, prompt := range deck {
		// Redis に保存したい形（DTO）へ変換する
		payload := inframodel.DeckPayload{
			PromptTextJa: prompt.PromptTextJa,
			TargetRomaji: prompt.TargetRomaji,
			LimitMs:			prompt.LimitMs,
		}
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		pipeline.RPush(ctx, deckKey, jsonData)
	}

	// 試合開始時に 1h で揃える（Meta が無いケースでも Deck 側へ設定しておく）
	pipeline.Expire(ctx, deckKey, config.MatchExpiryOnStart)
	pipeline.Expire(ctx, metaKey, config.MatchExpiryOnStart)

	_, err := pipeline.Exec(ctx)

	// 保存後の件数チェック（20問保証）
	length, err := r.redisClient.LLen(ctx, deckKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check deck length: %w", err)
	}
	if length != int64(config.Deck.TotalCount) {
		return fmt.Errorf("deck incomplete: expected %d, got %d", config.Deck.TotalCount, length)
	}

	return err
}

// LoadDeckPrompt は、指定したデッキインデックスのmatch:{matchID}:deck の JSON を構造体に変換するメソッド。
func (r *RoundStateRepositoryRedis) LoadDeckPrompt(ctx context.Context, matchID string, deckIndex int64) (*domainmodel.DeckPrompt, error) {
	deckKey := fmt.Sprintf("match:%s:deck", matchID)
	nextPromptJson, err := r.redisClient.LIndex(ctx, deckKey, deckIndex).Result()
	if err != nil {
		return nil, err
	}

	var nextPromptRaw map[string]interface{}
	if err := json.Unmarshal([]byte(nextPromptJson), &nextPromptRaw); err != nil {
		return nil, err
	}

	return &domainmodel.DeckPrompt{
		PromptTextJa: fmt.Sprintf("%v", nextPromptRaw["prompt_text_ja"]),
		TargetRomaji: fmt.Sprintf("%v", nextPromptRaw["target_romaji"]),
		LimitMs:      int64(nextPromptRaw["limit_ms"].(float64)),
	}, nil
}

// StoreFinishEvent は、回答確定時のイベント（miss数・終了時刻・ラウンド情報）を
// Redis Streams に保存するメソッド。
func (r *RoundStateRepositoryRedis) StoreFinishEvent(
	ctx context.Context,
	matchID,
	userID string,
	round,
	missCount,
	finishAtMs int64,
) error {
	eventsKey := fmt.Sprintf("match:%s:events", matchID)

	_, err := r.redisClient.XAdd(ctx, &redis.XAddArgs{
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
func (r *RoundStateRepositoryRedis) LoadFinishEventsByRound(ctx context.Context, matchID string, round int64) ([]domainmodel.MatchFinishEvents, error) {
	eventsKey := fmt.Sprintf("match:%s:events", matchID)
	streamMessages, err := r.redisClient.XRange(ctx, eventsKey, "-", "+").Result()
	if err != nil {
		return nil, err
	}

	var finishEvents []domainmodel.MatchFinishEvents

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
		finishEvents = append(finishEvents, domainmodel.MatchFinishEvents{
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
func (r *RoundStateRepositoryRedis) InitializeNextRoundState(ctx context.Context, matchID string) error {
	stateKey := fmt.Sprintf("match:%s:state", matchID)

	return r.redisClient.HSet(ctx, stateKey, map[string]interface{}{
		"round_start_at_ms": 0,
		"round_end_at_ms": 0,
	}).Err()
}

// LoadMatchState は、Redis の match:{matchID}:state に保存されている現在の試合状態を取得するメソッド。
func (r *RoundStateRepositoryRedis) LoadMatchState(ctx context.Context, matchID string) (*domainmodel.MatchState, error) {
	stateKey := fmt.Sprintf("match:%s:state", matchID)

	// Redis から hash を取得
	matchStateRow, err := r.redisClient.HGetAll(ctx, stateKey).Result()
	if err != nil {
		return nil, err
	}
	if len(matchStateRow) == 0 {
		return nil, fmt.Errorf("state not found")
	}

	// int64に変換してセット
	state := &domainmodel.MatchState{
		DeckIndex:      	parseInt64FromHash(matchStateRow, "deck_index"),
		Round:          	parseInt64FromHash(matchStateRow, "round"),
		RoundStartAtMs:  	parseInt64FromHash(matchStateRow, "round_start_at_ms"),
		RoundEndAtMs:   	parseInt64FromHash(matchStateRow, "round_end_at_ms"),
		Player1Lifepoint: parseInt64FromHash(matchStateRow, "player1_lifepoint"),
		Player2Lifepoint: parseInt64FromHash(matchStateRow, "player2_lifepoint"),
		Player1TotalMissCount: parseInt64FromHash(matchStateRow, "player1_total_miss_count"),
		Player2TotalMissCount: parseInt64FromHash(matchStateRow, "player2_total_miss_count"),
	}

	return state, nil
}

// UpdateTotalMissCount は、指定されたプレイヤー（player1 / player2）の
// 累計ミス数カウントに missCount を加算するメソッド。
func (r *RoundStateRepositoryRedis) UpdateTotalMissCount(
	ctx context.Context,
	matchID,
	playerField string,
	missCount int64,
) error {
	stateKey := fmt.Sprintf("match:%s:state", matchID)
	totalMissCountField := playerField + "_total_miss_count"

	_, err := r.redisClient.HIncrBy(
		ctx,
		stateKey,
		totalMissCountField,
		missCount,
	).Result()

	return err
}

// ReduceLifepoint は、指定されたプレイヤーのライフポイントを指定したダメージ分だけ減算するメソッド。
func (r *RoundStateRepositoryRedis) ReduceLifepoint(ctx context.Context, matchID string, playerField string, damage int64) (int64, error) {
	stateKey := fmt.Sprintf("match:%s:state", matchID)
	playerLifepointField := playerField + "_lifepoint"

	lifepoint, err := r.redisClient.HIncrBy(
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
		_ = r.redisClient.HSet(ctx, stateKey, playerLifepointField, 0).Err()
	}

	return lifepoint, nil
}

// UpdateNextRoundState は、次ラウンドへ進むために deck_index と round を
// 1つプラスして更新するメソッド。
func (r *RoundStateRepositoryRedis) UpdateNextRoundState(ctx context.Context, matchID string) (int64, int64, error) {
	stateKey := fmt.Sprintf("match:%s:state", matchID)

	nextDeckIndex, err := r.redisClient.HIncrBy(ctx, stateKey, "deck_index", 1).Result()
	if err != nil {
		return 0, 0, err
	}

	nextRound, err := r.redisClient.HIncrBy(ctx, stateKey, "round", 1).Result()
	if err != nil {
		return 0, 0, err
	}

	return nextDeckIndex, nextRound, nil
}

// UpdateRoundTiming は、ラウンドの開始予定時刻と終了予定時刻を
// Redis の match:{matchID}:state に保存するメソッド。
func (r *RoundStateRepositoryRedis) UpdateRoundTiming(ctx context.Context, matchID string, roundStartAtMs, roundEndAtMs int64) error {
	stateKey := fmt.Sprintf("match:%s:state", matchID)
	return r.redisClient.HSet(ctx, stateKey, map[string]interface{}{
		"round_start_at_ms": roundStartAtMs,
		"round_end_at_ms":   roundEndAtMs,
	}).Err()
}

// SetFrontendState は、フロントエンド再構築用状態を Redis に保存するメソッド。
func (r *RoundStateRepositoryRedis) SetFrontendState(ctx context.Context, matchID string, payloadBytes []byte) error {
	frontendStateKey := fmt.Sprintf("match:%s:frontend_state", matchID)

	return r.redisClient.Set(ctx, frontendStateKey, payloadBytes, config.MatchExpiryOnStart).Err()
}

// LoadFrontendState は、指定された試合のフロントエンド再構築用状態を
// Redis から取得するメソッド。
func (r *RoundStateRepositoryRedis) LoadFrontendState(ctx context.Context, matchID string) ([]byte, error) {
	frontendStateKey := fmt.Sprintf("match:%s:frontend_state", matchID)

	payloadBytes, err := r.redisClient.Get(ctx, frontendStateKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, redis.Nil
		}
		return nil, err
	}

	return payloadBytes, nil
}

// DeleteFrontendState は、指定された試合のフロントエンド再構築用状態を
// Redis から完全に削除するメソッド。
func (r *RoundStateRepositoryRedis) DeleteFrontendState(ctx context.Context, matchID string) error {
	frontendStateKey := fmt.Sprintf("match:%s:frontend_state", matchID)
	return r.redisClient.Del(ctx, frontendStateKey).Err()
}

// LoadMatchPlayers は Redis に保存された
// match:{matchID} のプレイヤー情報（player1 / player2）を取得するメソッド。
func (r *RoundStateRepositoryRedis) LoadMatchPlayers(ctx context.Context, matchID string) (*domainmodel.MatchPlayers, error) {
	matchKey := fmt.Sprintf("match:%s", matchID)

	data, err := r.redisClient.HGetAll(ctx, matchKey).Result()
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("match not found: %s", matchID)
	}

	players := &domainmodel.MatchPlayers{
		Player1ID: data["player1"],
		Player2ID: data["player2"],
	}

	return players, nil
}

// InitMeasurementFinishCountは、試合開始直後に呼び出される初期化処理するメソッド
func (r *RoundStateRepositoryRedis) InitMeasurementFinishCount(ctx context.Context, matchID string) error {
	measurementFinishedCountKey := fmt.Sprintf("match:%s:measurement_finished_count", matchID)

	return r.redisClient.Set(ctx, measurementFinishedCountKey, 0, config.MatchExpiryOnStart).Err()
}

// InitializeMeasurementFinishCount は、指定された matchID に紐づく
// 「計測完了人数（measurement_finished_count）」カウンタを 0 に初期化するメソッド。
func (r *RoundStateRepositoryRedis) InitializeMeasurementFinishCount(ctx context.Context, matchID string) error {
	measurementFinishedCountKey := fmt.Sprintf("match:%s:measurement_finished_count", matchID)

	return r.redisClient.Set(ctx, measurementFinishedCountKey, 0, config.MatchExpiryOnStart).Err()
}

// IncrementMeasurementFinishCount は、指定された matchID に紐づく
// 「計測完了人数（measurement_finished_count）」カウンタを +1 するメソッド。
func (r *RoundStateRepositoryRedis) IncrementMeasurementFinishCount(ctx context.Context, matchID string) (types.PlayerCount, error) {
	measurementFinishedCountKey := fmt.Sprintf("match:%s:measurement_finished_count", matchID)

	measurementFinishedCountInt64, err := r.redisClient.Incr(ctx, measurementFinishedCountKey).Result()
	if err != nil {
		return 0, err
	}

	return types.PlayerCount(measurementFinishedCountInt64), nil
}

// RegisterPlayerAnswerFinishFlag は、プレイヤーの finish トリガーを
// 原子的に「初回のみ」受け付けるメソッド。
func (r *RoundStateRepositoryRedis) RegisterPlayerAnswerFinishFlag(ctx context.Context, matchID, userID string) (bool, error) {
	flagKey := fmt.Sprintf("match:%s:answer_finish_flag:%s", matchID, userID)

	isFirstFinished, err := r.redisClient.SetNX(ctx, flagKey, true, config.MatchExpiryOnStart).Result()
	if err != nil {
		return false, err
	}

	return isFirstFinished, nil
}

// IsPlayerAnswerFinishFlagExists は、指定したプレイヤーの finish フラグキーが
// Redis 上に存在するかどうかを返すメソッド。
func (r *RoundStateRepositoryRedis) IsPlayerAnswerFinishFlagExists(ctx context.Context, matchID, userID string) (bool, error) {
	flagKey := fmt.Sprintf("match:%s:answer_finish_flag:%s", matchID, userID)

	isPlayerFinishFlagExists, err := r.redisClient.Exists(ctx, flagKey).Result()
	if err != nil {
		return false, err
	}

	return isPlayerFinishFlagExists==1, nil
}

// DeletePlayerAnswerFinishFlag は、プレイヤーの finish フラグキーを削除し、
// 次のラウンドで再び RegisterPlayerFinishFlag の SetNX が成功するようにリセットするメソッド。
func (r *RoundStateRepositoryRedis) DeletePlayerAnswerFinishFlag(ctx context.Context, matchID, userID string) error {
	flagKey := fmt.Sprintf("match:%s:answer_finish_flag:%s", matchID, userID)
	return r.redisClient.Del(ctx, flagKey).Err()
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
