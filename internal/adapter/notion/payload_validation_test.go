package notion

import (
	"net/http"
	"strings"
	"testing"
)

func TestBuildUpdatePagePayloadSupportsArchivedAliasIconAndCover(t *testing.T) {
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

	if payload["in_trash"] != true {
		t.Fatalf("expected in_trash=true, got: %+v", payload)
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

func TestBuildUpdatePagePayloadSupportsFileUploadIcon(t *testing.T) {
	payload, appErr := buildUpdatePagePayload(map[string]any{
		"icon": map[string]any{
			"file_upload_id": "fu_demo",
		},
	})
	if appErr != nil {
		t.Fatalf("buildUpdatePagePayload returned error: %+v", appErr)
	}
	icon := payload["icon"].(map[string]any)
	if icon["type"] != "file_upload" {
		t.Fatalf("unexpected icon payload: %+v", icon)
	}
	fileUpload := icon["file_upload"].(map[string]any)
	if fileUpload["id"] != "fu_demo" {
		t.Fatalf("unexpected file_upload icon payload: %+v", icon)
	}
}

func TestBuildUpdatePagePayloadSupportsInTrashField(t *testing.T) {
	payload, appErr := buildUpdatePagePayload(map[string]any{
		"in_trash": true,
	})
	if appErr != nil {
		t.Fatalf("buildUpdatePagePayload returned error: %+v", appErr)
	}
	if payload["in_trash"] != true {
		t.Fatalf("expected in_trash=true, got: %+v", payload)
	}
}

func TestBuildUpdatePagePayloadSupportsTemplateLockAndEraseContent(t *testing.T) {
	payload, appErr := buildUpdatePagePayload(map[string]any{
		"is_locked":     true,
		"erase_content": true,
		"template": map[string]any{
			"type":        "template_id",
			"template_id": "tpl_demo",
			"timezone":    "Asia/Shanghai",
		},
	})
	if appErr != nil {
		t.Fatalf("buildUpdatePagePayload returned error: %+v", appErr)
	}
	if payload["is_locked"] != true || payload["erase_content"] != true {
		t.Fatalf("unexpected page update payload: %+v", payload)
	}
	template := payload["template"].(map[string]any)
	if template["type"] != "template_id" || template["template_id"] != "tpl_demo" || template["timezone"] != "Asia/Shanghai" {
		t.Fatalf("unexpected template payload: %+v", template)
	}
}

func TestBuildUpdatePagePayloadRejectsInvalidTemplate(t *testing.T) {
	_, appErr := buildUpdatePagePayload(map[string]any{
		"template": map[string]any{
			"type": "template_id",
		},
	})
	if appErr == nil {
		t.Fatal("expected buildUpdatePagePayload to reject missing template_id")
	}
	if appErr.Code != "INVALID_INPUT" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

func TestBuildBlockSupportsFileUploadBackedMediaBlocks(t *testing.T) {
	imagePayload, appErr := buildBlock(map[string]any{
		"type":           "image",
		"file_upload_id": "fu_image_demo",
	})
	if appErr != nil {
		t.Fatalf("buildBlock returned error for image file_upload: %+v", appErr)
	}
	imageBody := imagePayload["image"].(map[string]any)
	if imageBody["type"] != "file_upload" {
		t.Fatalf("unexpected image payload: %+v", imageBody)
	}
	if imageBody["file_upload"].(map[string]any)["id"] != "fu_image_demo" {
		t.Fatalf("unexpected image file_upload payload: %+v", imageBody)
	}

	filePayload, appErr := buildBlock(map[string]any{
		"type": "file",
		"file": map[string]any{
			"type": "file_upload",
			"file_upload": map[string]any{
				"id": "fu_file_demo",
			},
		},
	})
	if appErr != nil {
		t.Fatalf("buildBlock returned error for file file_upload: %+v", appErr)
	}
	fileBody := filePayload["file"].(map[string]any)
	if fileBody["type"] != "file_upload" {
		t.Fatalf("unexpected file payload: %+v", fileBody)
	}
	if fileBody["file_upload"].(map[string]any)["id"] != "fu_file_demo" {
		t.Fatalf("unexpected file file_upload payload: %+v", fileBody)
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

func TestBuildBlockSupportsProviderNativeBodies(t *testing.T) {
	payload, appErr := buildBlock(map[string]any{
		"type": "paragraph",
		"paragraph": map[string]any{
			"color": "blue_background",
			"rich_text": []map[string]any{
				{
					"type": "text",
					"text": map[string]any{
						"content": "Provider 段落",
					},
				},
			},
			"children": []map[string]any{
				{
					"type": "to_do",
					"to_do": map[string]any{
						"checked": true,
						"rich_text": []map[string]any{
							{
								"type": "text",
								"text": map[string]any{
									"content": "Provider 子项",
								},
							},
						},
					},
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("buildBlock returned error: %+v", appErr)
	}

	body := payload["paragraph"].(map[string]any)
	if body["color"] != "blue_background" {
		t.Fatalf("unexpected paragraph body: %+v", body)
	}
	richText := body["rich_text"].([]map[string]any)
	if richText[0]["text"].(map[string]any)["content"] != "Provider 段落" {
		t.Fatalf("unexpected paragraph rich_text: %+v", richText)
	}
	children := body["children"].([]map[string]any)
	toDo := children[0]["to_do"].(map[string]any)
	if toDo["checked"] != true {
		t.Fatalf("unexpected to_do body: %+v", toDo)
	}
	toDoRichText := toDo["rich_text"].([]map[string]any)
	if toDoRichText[0]["text"].(map[string]any)["content"] != "Provider 子项" {
		t.Fatalf("unexpected to_do rich_text: %+v", toDoRichText)
	}
}

func TestBuildBlockTopLevelFieldsOverrideProviderNativeBodies(t *testing.T) {
	payload, appErr := buildBlock(map[string]any{
		"type": "paragraph",
		"text": "顶层正文优先",
		"paragraph": map[string]any{
			"rich_text": []map[string]any{
				{
					"type": "text",
					"text": map[string]any{
						"content": "不应被保留",
					},
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("buildBlock returned error: %+v", appErr)
	}

	body := payload["paragraph"].(map[string]any)
	richText := body["rich_text"].([]map[string]any)
	if richText[0]["text"].(map[string]any)["content"] != "顶层正文优先" {
		t.Fatalf("unexpected rich_text precedence: %+v", richText)
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
