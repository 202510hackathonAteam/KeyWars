package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	drepo "keywars/backend/internal/domain/repository"
)

// クライアントから飛んでくる回答メッセージのペイロード例
//
//	{
//	  "matchId": "xxxx",
//	  "isCorrect": true,
//	  "serverElapsedMs": 1234,
//	  "limitMs": 6000,
//	  "totalMissCount": 2,
//	  "opponentUserId": "u002"
//	}
type AnswerMessagePayload struct {
	MatchID         string `json:"matchId"`
	IsCorrect       bool   `json:"isCorrect"`
	ServerElapsedMs int64  `json:"serverElapsedMs"`
	LimitMs         int64  `json:"limitMs"`
	TotalMissCount  int    `json:"totalMissCount"`
	OpponentUserID  string `json:"opponentUserId"`
}

// handleAnswerMessage は、"answer.submit" メッセージ専用の処理。
func (service *Service) handleAnswerMessage(
	ctx context.Context,
	userID string,
	roomName string,
	rawPayload []byte,
) (any, error) {
	var payload AnswerMessagePayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return map[string]any{
			"type":  "answer.error",
			"error": "invalid payload",
		}, nil
	}

	// ① ジャッジ（attack_logic.go のロジックを使う）
	judgeResult := JudgeAnswer(
		payload.IsCorrect,
		payload.ServerElapsedMs,
		payload.LimitMs,
		payload.TotalMissCount,
	)

	// ダメージ 0 かつ問題終了じゃないなら、何も起こさず ACK だけ返すなど
	if !judgeResult.IsQuestionFinished {
		return map[string]any{
			"type":      "answer.accepted",
			"matchId":   payload.MatchID,
			"isCorrect": payload.IsCorrect,
		}, nil
	}

	// ② LP 計算（lp_logic.go の ApplyDamage を利用）
	var newOpponentLP int64
	if judgeResult.IsAttackSuccess {
		// opponent の現在LP は本当は state から取るべき
		// ここでは例として 100 から計算
		currentOpponentLP := int64(100) // TODO: redis から取得する
		newOpponentLP = ApplyDamage(currentOpponentLP, judgeResult.DamageToOpponent)
	}

	// ③ Redis 側 state 更新（AnswerApplyArg 経由）
	//   - LP 更新
	//   - turn+1 / deck_idx / q_started_at_ms 更新
	//   - events ストリームへ "answer" イベント追加
	answerArgs := drepo.AnswerApplyArg{
		MatchID:              payload.MatchID,
		OpponentUserID:       payload.OpponentUserID,
		NewOpponentLifePoint: newOpponentLP,
		CurrentServerTimeMs:  time.Now().UnixMilli(),
		NextDeckIndex:        -1, // ここは「次の問題へ進むか」で決める
		EventFields: map[string]string{
			"is_correct":       boolToString(payload.IsCorrect),
			"damage":           int64ToString(judgeResult.DamageToOpponent),
			"total_miss_count": intToString(payload.TotalMissCount),
		},
	}

	eventID, turn, err := service.roundStateRepository.ApplyAnswer(ctx, answerArgs)
	if err != nil {
		return map[string]any{
			"type":  "answer.error",
			"error": "failed to apply answer",
		}, nil
	}

	// ④ ルーム内の全員に結果をブロードキャスト
	matchRoomName := "match:" + payload.MatchID
	_, _ = service.websocketHub.Broadcast(ctx, matchRoomName, map[string]any{
		"type":            "answer.result",
		"matchId":         payload.MatchID,
		"by":              userID,
		"isCorrect":       payload.IsCorrect,
		"damage":          judgeResult.DamageToOpponent,
		"newOpponentLP":   newOpponentLP,
		"turn":            turn,
		"eventId":         eventID,
		"serverElapsedMs": payload.ServerElapsedMs,
	})

	// 呼び出し元のクライアントには簡単な ACK を返す
	return map[string]any{
		"type":      "answer.applied",
		"matchId":   payload.MatchID,
		"eventId":   eventID,
		"isCorrect": payload.IsCorrect,
	}, nil
}

// 補助の変換関数
func boolToString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
func int64ToString(v int64) string { return fmt.Sprintf("%d", v) }
func intToString(v int) string     { return fmt.Sprintf("%d", v) }
