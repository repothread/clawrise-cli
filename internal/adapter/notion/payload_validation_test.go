package notion

import (
	"net/http"
	"strings"
	"testing"
)

func TestBuildUpdatePagePayloadSupportsArchivedIconAndCover(t *testing.T) {
	// 页面更新是 live 测试里最常见的清理与装饰入口，这里把关键字段的构造补齐单测。
	payload, appErr := buildUpdatePagePayload(map[string]any{
		"title":    "已更新页面",
		"archived": true,
		"icon":     "🧪",
		"cover":    "https://example.com/cover.png",
	})
	if appErr != nil {
		t.Fatalf("buildUpdatePagePayload returned error: %+v", appErr)
	}

	if payload["archived"] != true {
		t.Fatalf("expected archived=true, got: %+v", payload)
	}
	icon := payload["icon"].(map[string]any)
	if icon["type"] != "emoji" || icon["emoji"] != "🧪" {
		t.Fatalf("unexpected icon payload: %+v", icon)
	}
	cover := payload["cover"].(map[string]any)
	if cover["type"] != "external" {
		t.Fatalf("unexpected cover payload: %+v", cover)
	}
	properties := payload["properties"].(map[string]any)
	titleProperty := properties["title"].(map[string]any)
	titleItems := titleProperty["title"].([]map[string]any)
	if titleItems[0]["text"].(map[string]any)["content"] != "已更新页面" {
		t.Fatalf("unexpected title payload: %+v", titleItems)
	}
}

func TestBuildUpdatePagePayloadRejectsEmptyInput(t *testing.T) {
	_, appErr := buildUpdatePagePayload(map[string]any{
		"page_id": "page_demo",
	})
	if appErr == nil {
		t.Fatal("expected buildUpdatePagePayload to reject empty update payload")
	}
	if appErr.Code != "INVALID_INPUT" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

func TestBuildUpdatePageMarkdownPayloadSupportsReplaceAndRangeCommands(t *testing.T) {
	replacePayload, appErr := buildUpdatePageMarkdownPayload(map[string]any{
		"type": "replace_content",
		"replace_content": map[string]any{
			"new_str":                "# 新内容",
			"allow_deleting_content": true,
		},
	})
	if appErr != nil {
		t.Fatalf("buildUpdatePageMarkdownPayload returned error for replace_content: %+v", appErr)
	}
	if replacePayload["type"] != "replace_content" {
		t.Fatalf("unexpected replace_content payload: %+v", replacePayload)
	}

	rangePayload, appErr := buildUpdatePageMarkdownPayload(map[string]any{
		"replace_content_range": map[string]any{
			"content":                "Delta",
			"content_range":          "Alpha...Beta",
			"allow_deleting_content": true,
		},
	})
	if appErr != nil {
		t.Fatalf("buildUpdatePageMarkdownPayload returned error for replace_content_range: %+v", appErr)
	}
	if rangePayload["type"] != "replace_content_range" {
		t.Fatalf("expected payload type to be inferred: %+v", rangePayload)
	}
}

func TestBuildCreateCommentPayloadValidatesParentsAndAttachments(t *testing.T) {
	// 评论接口要求 parent 互斥，这里把有效负载与错误路径一起补上。
	_, appErr := buildCreateCommentPayload(map[string]any{
		"text":          "冲突评论",
		"page_id":       "page_123",
		"discussion_id": "discussion_123",
	})
	if appErr == nil {
		t.Fatal("expected buildCreateCommentPayload to reject multiple parents")
	}

	payload, appErr := buildCreateCommentPayload(map[string]any{
		"block_id": "block_123",
		"text":     "块评论",
		"attachments": []any{
			map[string]any{
				"name": "evidence.txt",
			},
		},
		"display_name": map[string]any{
			"type": "text",
			"name": "Clawrise CI",
		},
	})
	if appErr != nil {
		t.Fatalf("buildCreateCommentPayload returned error: %+v", appErr)
	}
	if payload["parent"].(map[string]any)["block_id"] != "block_123" {
		t.Fatalf("unexpected parent payload: %+v", payload)
	}
	if len(payload["attachments"].([]map[string]any)) != 1 {
		t.Fatalf("unexpected attachments payload: %+v", payload["attachments"])
	}
}

func TestBuildBlockSupportsTableInferenceAndTableRowPlainText(t *testing.T) {
	tablePayload, appErr := buildBlock(map[string]any{
		"type":              "table",
		"has_column_header": true,
		"rows": []any{
			map[string]any{
				"type":  "table_row",
				"cells": []any{"H1", "H2"},
			},
			map[string]any{
				"type":  "table_row",
				"cells": []any{"R1C1", "R1C2"},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("buildBlock returned error: %+v", appErr)
	}
	tableBody := tablePayload["table"].(map[string]any)
	if tableBody["table_width"] != 2 {
		t.Fatalf("expected inferred table_width=2, got: %+v", tableBody)
	}

	plainText := extractTableRowPlainText(map[string]any{
		"table_row": map[string]any{
			"cells": []any{
				[]any{
					map[string]any{
						"type":       "text",
						"plain_text": "H1",
					},
				},
				[]any{
					map[string]any{
						"type":       "text",
						"plain_text": "H2",
					},
				},
			},
		},
	})
	if plainText != "H1 | H2" {
		t.Fatalf("unexpected table row plain text: %s", plainText)
	}
}

func TestNormalizeNotionFileObjectAndHTTPErrorMapping(t *testing.T) {
	emojiPayload, appErr := normalizeNotionFileObject("✅", true)
	if appErr != nil {
		t.Fatalf("normalizeNotionFileObject returned error: %+v", appErr)
	}
	if emojiPayload["type"] != "emoji" {
		t.Fatalf("unexpected emoji payload: %+v", emojiPayload)
	}

	if _, appErr := normalizeNotionFileObject(123, true); appErr == nil {
		t.Fatal("expected normalizeNotionFileObject to reject invalid value type")
	}

	rateLimitedErr := normalizeNotionHTTPError(http.StatusTooManyRequests, http.Header{
		"Retry-After": []string{"3"},
	}, mustJSON(t, map[string]any{
		"object":  "error",
		"status":  429,
		"code":    "rate_limited",
		"message": "slow down",
	}))
	if rateLimitedErr.Code != "RATE_LIMITED" || !rateLimitedErr.Retryable {
		t.Fatalf("unexpected rate limit mapping: %+v", rateLimitedErr)
	}
	if !strings.Contains(rateLimitedErr.Message, "Retry-After: 3") {
		t.Fatalf("expected retry-after hint in message: %+v", rateLimitedErr)
	}

	plainBodyErr := normalizeNotionHTTPError(http.StatusBadGateway, http.Header{}, []byte("gateway exploded"))
	if plainBodyErr.Code != "UPSTREAM_TEMPORARY_FAILURE" || !plainBodyErr.Retryable {
		t.Fatalf("unexpected plain body error mapping: %+v", plainBodyErr)
	}
}
