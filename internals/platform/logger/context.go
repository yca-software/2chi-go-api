package logger

import "context"

func ContextExtractor(ctx context.Context) map[string]any {
	return map[string]any{
		"request_id": ctx.Value("request_id"),
		"user_id":    ctx.Value("user_id"),
		"api_key_id": ctx.Value("api_key_id"),
	}
}
