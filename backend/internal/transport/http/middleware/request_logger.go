package middleware

import (
	"context"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

// RequestLogger は、各リクエストに一意のロガーを生成して Context に注入するミドルウェア。
// これにより、handler や service 層で zerolog.Ctx(ctx) を通して
// request_id / route / method を含むログが自動出力できる。
func RequestLogger(baseLogger *zerolog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			request := c.Request()
			response := c.Response()

			// RequestID を Request、Response の順で取得
			requestID := request.Header.Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = response.Header().Get(echo.HeaderXRequestID)
			}

			// リクエスト専用ロガーを生成
			requestLogger := baseLogger.With().
				Str("request_id", requestID).
				Str("route", c.Path()).
				Str("method", request.Method).
				Logger()

			// Contextにロガーを紐づけ（zerolog.Ctx(ctx)で取得可能）
			requestCtx := requestLogger.WithContext(request.Context())
			if requestID != "" {
				requestCtx = context.WithValue(requestCtx, ctxKeyRequestID, requestID)
			}
			c.SetRequest(request.WithContext(requestCtx))

			return next(c)
		}
	}
}