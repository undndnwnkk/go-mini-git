package api

import "context"

type contextKey string

const requestIDContextKey contextKey = "request_id"

func withRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDContextKey).(string)
	if !ok {
		return ""
	}

	return requestID
}
