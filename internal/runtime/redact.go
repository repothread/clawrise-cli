package runtime

import "strings"

var sensitiveKeyFragments = []string{
	"token",
	"secret",
	"password",
	"authorization",
	"credential",
	"api_key",
	"apikey",
	"access_key",
}

// redactValue 会递归处理 map、slice 和字符串。
func redactValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSensitiveKey(key) {
				result[key] = "***"
				continue
			}
			result[key] = redactValue(item)
		}
		return result
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, redactValue(item))
		}
		return result
	case string:
		return redactString(typed)
	default:
		return value
	}
}

func redactError(err *ErrorBody) *ErrorBody {
	if err == nil {
		return nil
	}
	return &ErrorBody{
		Code:         err.Code,
		Message:      redactString(err.Message),
		Retryable:    err.Retryable,
		UpstreamCode: err.UpstreamCode,
		HTTPStatus:   err.HTTPStatus,
	}
}

func cloneIdempotencyState(state *IdempotencyState) *IdempotencyState {
	if state == nil {
		return nil
	}
	return &IdempotencyState{
		Key:       state.Key,
		Status:    state.Status,
		Persisted: state.Persisted,
		UpdatedAt: state.UpdatedAt,
	}
}

func isSensitiveKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, part := range sensitiveKeyFragments {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
}

func redactString(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return "Bearer ***"
	}
	return value
}
