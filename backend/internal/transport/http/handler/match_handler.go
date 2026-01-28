package handler

import (
	"fmt"
	"net/http"
	"context"
	"time"
	"encoding/json"
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"keywars/backend/internal/service"
	"keywars/backend/internal/transport/http/response"
	"keywars/backend/internal/transport/websocket"
)

// MatchHandler は、試合関連の HTTP リクエストを処理するハンドラの定義。
type MatchHandler struct {
	matchService service.MatchService
}

// NewMatchHandler は、MatchService を受け取り MatchHandler を生成。
func NewMatchHandler(service service.MatchService) *MatchHandler {
	return &MatchHandler{
		matchService: service,
	}
}

// FrontendState は、ユーザーIDに紐づく試合のフロントエンド再構築用の
//（frontend_state）を取得して返却するエンドポイント。
func (h *MatchHandler) FrontendState(c echo.Context) error {
	userID := c.Get("userID").(string)

	timeoutCtx, cancel := context.WithTimeout(c.Request().Context(), time.Second*10)
	defer cancel()

	payloadBytes, err := h.matchService.FrontendState(timeoutCtx, userID)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 状態がまだ無いだけ（正常）
			return response.Respond(c, http.StatusNoContent, nil)
		}
		return response.Respond(c, http.StatusInternalServerError, nil)
	}

	payload, err := decodeTypedPayload(payloadBytes)
	if err != nil {
		return response.Respond(c, http.StatusBadRequest, nil)
	}

	return response.Respond(c, http.StatusOK, payload)
}

func decodeTypedPayload(payloadBytes []byte) (any, error) {
	var base struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(payloadBytes, &base); err != nil {
		return nil, err
	}

	switch base.Type {
	case "match.start":
		var payload websocket.RoundStartPayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case "match.end":
		var payload websocket.MatchEndPayload
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	default:
		return nil, fmt.Errorf("unknown payload type: %s", base.Type)
	}
}