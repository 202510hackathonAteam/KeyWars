package middleware

// contextKey は、context.Context に格納される値のキーを型安全に管理するための独自型
type contextKey string

const (
	CtxUserID contextKey = "user_id"
	ctxKeyRequestID contextKey = "request_id"
)