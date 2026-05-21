package middleware

import "context"

type contextKey string

const userUUIDContextKey = contextKey("user_uuid")

func WithUserUUID(ctx context.Context, uuid string) context.Context {
	return context.WithValue(ctx, userUUIDContextKey, uuid)
}

func UserUUIDFromContext(ctx context.Context) string {
	uuid, _ := ctx.Value(userUUIDContextKey).(string)
	return uuid
}
