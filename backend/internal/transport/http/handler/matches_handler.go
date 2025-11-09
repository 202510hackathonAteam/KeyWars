package handler

import (
	"net/http"
	"time"

	"keywars/backend/internal/service"

	"github.com/labstack/echo/v4"
)

// MatchesHandler は、マッチング機能に関する HTTP ハンドラです。
// HTTP リクエストを受け取り、MatchService（ビジネスロジック）を呼び出して JSON を返します。
type MatchesHandler struct {
	MatchService *service.MatchService
}

// NewMatchesHandler は、MatchService を受け取り MatchesHandler を生成します。
func NewMatchesHandler(matchService *service.MatchService) *MatchesHandler {
	return &MatchesHandler{
		MatchService: matchService,
	}
}

// --- レスポンス DTO（Data Transfer Object） ---

// JoinQueueResponse は、待機キュー追加成功時に返す JSON の形です。
type JoinQueueResponse struct {
	Status      string `json:"status"`
	UserID      string `json:"user_id"`
	EnqueueAtMs int64  `json:"enqueue_at_ms"`
}

// NoMatchResponse は、マッチ未成立時に返す JSON の形です。
type NoMatchResponse struct {
	Matched bool `json:"matched"`
}

// TryMatchResponse は、マッチ成立時に返す JSON の形です。
type TryMatchResponse struct {
	Matched bool   `json:"matched"`
	MatchID string `json:"match_id"`
	Player1 string `json:"player1"`
	Player2 string `json:"player2"`
}

// JoinQueue は、ユーザーをマッチング待機キューへ追加します。
// 例: POST /api/matches/queue/join?user_id=123
func (h *MatchesHandler) JoinQueue(echoContext echo.Context) error {
	// 1) 入力の取り出し・検証
	userID := echoContext.QueryParam("user_id")
	if userID == "" {
		return echoContext.JSON(http.StatusBadRequest, map[string]string{
			"error": "missing query parameter: user_id",
		})
	}

	// 2) サービス呼び出し（契約に合わせて現在時刻 ms を付与）
	nowUnixMilli := time.Now().UnixMilli()
	requestContext := echoContext.Request().Context()

	if err := h.MatchService.JoinQueue(requestContext, userID, nowUnixMilli); err != nil {
		return echoContext.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// 3) 成功レスポンス
	return echoContext.JSON(http.StatusOK, JoinQueueResponse{
		Status:      "queued",
		UserID:      userID,
		EnqueueAtMs: nowUnixMilli,
	})
}

// TryMatch は、待機キューから 2 名を取り出してマッチを初期化します。
// 例: POST /api/matches/queue/try
func (h *MatchesHandler) TryMatch(echoContext echo.Context) error {
	requestContext := echoContext.Request().Context()

	matchID, userID1, userID2, err := h.MatchService.TryMatch(requestContext)
	if err != nil {
		return echoContext.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// マッチ未成立（待機人数が 2 名未満）の場合
	if matchID == "" {
		return echoContext.JSON(http.StatusOK, NoMatchResponse{Matched: false})
	}

	// マッチ成立
	return echoContext.JSON(http.StatusOK, TryMatchResponse{
		Matched: true,
		MatchID: matchID,
		Player1: userID1,
		Player2: userID2,
	})
}
