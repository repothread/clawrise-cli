package adapter

import (
	"strings"
	"testing"
)

func TestRedactDebugValueRedactsSensitiveAndContentFields(t *testing.T) {
	redacted := RedactDebugValue(map[string]any{
		"access_token":  "secret-token",
		"authorization": "Bearer secret-token",
		"plain_text":    "Hello Clawrise",
		"text": map[string]any{
			"content": "Nested content",
		},
		"url":  "https://example.com/path?q=1",
		"safe": "visible",
	}).(map[string]any)

	if redacted["access_token"] != "***" {
		t.Fatalf("expected access_token to be redacted: %+v", redacted)
	}
	if redacted["authorization"] != "***" {
		t.Fatalf("expected authorization to be redacted by key: %+v", redacted)
	}
	if redacted["plain_text"] == "Hello Clawrise" {
		t.Fatalf("expected plain_text to be summarized: %+v", redacted)
	}
	text := redacted["text"].(map[string]any)
	if text["content"] == "Nested content" {
		t.Fatalf("expected nested content to be summarized: %+v", text)
	}
	if !strings.Contains(redacted["url"].(string), "example.com") || redacted["url"] == "https://example.com/path?q=1" {
		t.Fatalf("expected url to be summarized with host only: %+v", redacted["url"])
	}
	if redacted["safe"] != "visible" {
		t.Fatalf("expected safe field to remain visible: %+v", redacted["safe"])
	}
}
