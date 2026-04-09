package notion

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestReadCompletePageAggregatesPropertyItemsAndMarkdown(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/pages/page_123":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":       "page_123",
					"url":      "https://www.notion.so/page_123",
					"in_trash": false,
					"parent": map[string]any{
						"type":    "page_id",
						"page_id": "parent_demo",
					},
					"properties": map[string]any{
						"Name": map[string]any{
							"id":   "title_prop",
							"type": "title",
							"title": []map[string]any{
								{
									"type":       "text",
									"plain_text": "主页标题",
									"text": map[string]any{
										"content": "主页标题",
									},
								},
							},
						},
						"Stakeholders": map[string]any{
							"id":   "people_prop",
							"type": "people",
							"people": []map[string]any{
								{"id": "user_1"},
							},
						},
					},
				}), nil
			case "/v1/pages/page_123/properties/title_prop":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object":   "list",
					"results":  []map[string]any{{"object": "property_item", "id": "title_item_1"}},
					"has_more": false,
				}), nil
			case "/v1/pages/page_123/properties/people_prop":
				if request.URL.Query().Get("start_cursor") == "" {
					return jsonResponse(t, http.StatusOK, map[string]any{
						"object":      "list",
						"results":     []map[string]any{{"object": "property_item", "id": "people_item_1"}},
						"next_cursor": "cursor_people_2",
						"has_more":    true,
					}), nil
				}
				if request.URL.Query().Get("start_cursor") != "cursor_people_2" {
					t.Fatalf("unexpected people property page token: %s", request.URL.Query().Get("start_cursor"))
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object":   "list",
					"results":  []map[string]any{{"object": "property_item", "id": "people_item_2"}},
					"has_more": false,
				}), nil
			case "/v1/pages/page_123/markdown":
				if request.URL.Query().Get("include_transcript") != "true" {
					t.Fatalf("unexpected include_transcript query: %s", request.URL.Query().Get("include_transcript"))
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object":            "page_markdown",
					"id":                "page_123",
					"markdown":          "# Root\n\n正文",
					"truncated":         true,
					"unknown_block_ids": []string{"blk_unknown_1", "blk_unknown_2"},
				}), nil
			case "/v1/pages/blk_unknown_1/markdown":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object":            "page_markdown",
					"id":                "blk_unknown_1",
					"markdown":          "## Nested 1",
					"truncated":         false,
					"unknown_block_ids": []string{"blk_unknown_1_child"},
				}), nil
			case "/v1/pages/blk_unknown_1_child/markdown":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object":            "page_markdown",
					"id":                "blk_unknown_1_child",
					"markdown":          "### Nested child",
					"truncated":         false,
					"unknown_block_ids": []string{},
				}), nil
			case "/v1/pages/blk_unknown_2/markdown":
				return jsonResponse(t, http.StatusNotFound, map[string]any{
					"object":  "error",
					"status":  404,
					"code":    "object_not_found",
					"message": "Could not find block",
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.ReadCompletePage(context.Background(), testStaticProfile(), map[string]any{
		"page_id":                 "page_123",
		"include_property_items":  true,
		"property_item_page_size": 100,
		"include_markdown":        true,
		"include_transcript":      true,
		"expand_unknown_blocks":   true,
		"unknown_block_limit":     5,
	})
	if appErr != nil {
		t.Fatalf("ReadCompletePage returned error: %+v", appErr)
	}

	page := data["page"].(map[string]any)
	if page["page_id"] != "page_123" {
		t.Fatalf("unexpected page payload: %+v", page)
	}

	propertyItems := data["property_items"].(map[string]any)
	itemsByName := propertyItems["items_by_name"].(map[string]any)
	if len(itemsByName) != 2 {
		t.Fatalf("unexpected completed property item count: %+v", itemsByName)
	}
	peopleItem := itemsByName["Stakeholders"].(map[string]any)
	peopleItems := peopleItem["items"].([]map[string]any)
	if len(peopleItems) != 2 {
		t.Fatalf("expected paginated property items to be merged: %+v", peopleItem)
	}

	markdown := data["markdown"].(map[string]any)
	resolvedUnknownBlocks := markdown["resolved_unknown_blocks"].([]map[string]any)
	if len(resolvedUnknownBlocks) != 2 {
		t.Fatalf("unexpected resolved unknown blocks: %+v", resolvedUnknownBlocks)
	}
	unknownBlockErrors := markdown["unknown_block_errors"].([]map[string]any)
	if len(unknownBlockErrors) != 1 {
		t.Fatalf("unexpected unknown block errors: %+v", unknownBlockErrors)
	}
	withAppendices, _ := markdown["markdown_with_appendices"].(string)
	if withAppendices == "" || !containsAll(withAppendices, "# Root", "## Nested 1", "### Nested child") {
		t.Fatalf("unexpected markdown_with_appendices: %s", withAppendices)
	}
}

func TestReadCompletePageCanSkipPropertyItemsAndMarkdown(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/pages/page_123" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			return jsonResponse(t, http.StatusOK, map[string]any{
				"id":       "page_123",
				"url":      "https://www.notion.so/page_123",
				"in_trash": false,
				"properties": map[string]any{
					"title": map[string]any{
						"id":   "title_prop",
						"type": "title",
						"title": []map[string]any{
							{
								"type":       "text",
								"plain_text": "只读基础信息",
								"text": map[string]any{
									"content": "只读基础信息",
								},
							},
						},
					},
				},
			}), nil
		},
	})

	data, appErr := client.ReadCompletePage(context.Background(), testStaticProfile(), map[string]any{
		"page_id":                "page_123",
		"include_property_items": false,
		"include_markdown":       false,
	})
	if appErr != nil {
		t.Fatalf("ReadCompletePage returned error: %+v", appErr)
	}

	propertyItems := data["property_items"].(map[string]any)
	if propertyItems["enabled"] != false {
		t.Fatalf("expected property items to be disabled: %+v", propertyItems)
	}
	markdown := data["markdown"].(map[string]any)
	if markdown["enabled"] != false {
		t.Fatalf("expected markdown to be disabled: %+v", markdown)
	}
}

func containsAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}
