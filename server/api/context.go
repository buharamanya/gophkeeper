package api

import "context"

// contextKey - типизированный ключ для контекста
type contextKey string

// String возвращает строковое представление ключа
func (c contextKey) String() string {
	return "gophkeeper context key " + string(c)
}

// Ключи контекста
var (
	userIDKey    = contextKey("userID")
	userLoginKey = contextKey("userLogin")
)

// GetUserID извлекает userID из контекста
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}

// GetUserLogin извлекает userLogin из контекста
func GetUserLogin(ctx context.Context) (string, bool) {
	userLogin, ok := ctx.Value(userLoginKey).(string)
	return userLogin, ok
}

// WithUserID добавляет userID в контекст
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithUserLogin добавляет userLogin в контекст
func WithUserLogin(ctx context.Context, userLogin string) context.Context {
	return context.WithValue(ctx, userLoginKey, userLogin)
}
