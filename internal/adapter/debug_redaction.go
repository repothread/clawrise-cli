package adapter

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"unicode/utf8"
)

var sensitiveDebugKeyFragments = []string{
	"token",
	"secret",
	"password",
	"authorization",
	"credential",
	"api_key",
	"apikey",
	"access_key",
	"cookie",
	"signature",
}

var contentDebugKeys = map[string]struct{}{
	"plain_text": {},
	"content":    {},
	"markdown":   {},
}

var urlDebugKeys = map[string]struct{}{
	"url":          {},
	"avatar_url":   {},
	"cover_url":    {},
	"download_url": {},
	"href":         {},
	"link":         {},
}

// RedactDebugValue 会递归脱敏 provider debug 输出中的敏感字段和值。
func RedactDebugValue(value any) any {
	return redactDebugValue("", value)
}

func redactDebugValue(key string, value any) any {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for childKey, item := range typed {
			if isSensitiveDebugKey(childKey) {
				result[childKey] = "***"
				continue
			}
			result[childKey] = redactDebugValue(childKey, item)
		}
		return result
	case []any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, redactDebugValue(key, item))
		}
		return result
	case string:
		return redactDebugString(key, typed)
	default:
		if value == nil {
			return nil
		}
		if _, ok := value.([]byte); ok {
			return summarizeRedactedString("binary", fmt.Sprintf("%d", len(value.([]byte))))
		}
		typedValue := reflect.ValueOf(value)
		if typedValue.Kind() != reflect.Slice {
			return value
		}
		result := make([]any, 0, typedValue.Len())
		for index := 0; index < typedValue.Len(); index++ {
			result = append(result, redactDebugValue(key, typedValue.Index(index).Interface()))
		}
		return result
	}
}

func redactDebugString(key string, value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}

	switch {
	case isSensitiveDebugKey(key):
		return "***"
	case isContentDebugKey(key):
		return summarizeRedactedString("content", trimmed)
	case isURLDebugKey(key):
		return summarizeRedactedURL(trimmed)
	}

	lowered := strings.ToLower(trimmed)
	switch {
	case strings.HasPrefix(lowered, "bearer "):
		return "Bearer ***"
	case strings.HasPrefix(lowered, "basic "):
		return "Basic ***"
	case looksSensitiveDebugString(trimmed):
		return summarizeRedactedString("secret", trimmed)
	default:
		return value
	}
}

func isSensitiveDebugKey(key string) bool {
	key = normalizeDebugKey(key)
	for _, fragment := range sensitiveDebugKeyFragments {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}

func isContentDebugKey(key string) bool {
	_, ok := contentDebugKeys[normalizeDebugKey(key)]
	return ok
}

func isURLDebugKey(key string) bool {
	_, ok := urlDebugKeys[normalizeDebugKey(key)]
	return ok
}

func normalizeDebugKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func summarizeRedactedString(kind string, value string) string {
	return fmt.Sprintf("[REDACTED %s len=%d]", kind, utf8.RuneCountInString(value))
}

func summarizeRedactedURL(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || strings.TrimSpace(parsed.Host) == "" {
		return summarizeRedactedString("url", value)
	}
	return fmt.Sprintf("[REDACTED url host=%s]", parsed.Host)
}

func looksSensitiveDebugString(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, " ") {
		return false
	}
	if strings.Contains(value, ".") || strings.Contains(value, "_") || strings.Contains(value, "-") {
		return utf8.RuneCountInString(value) >= 24
	}
	return false
}
