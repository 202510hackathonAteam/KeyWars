package handler

import (
	"github.com/labstack/echo/v4"
)

// Respond は、HTTP レスポンスを JSON 形式で返す共通ヘルパー関数。
func Respond(c echo.Context, code int, body any) error {
	return c.JSON(code, body)
}