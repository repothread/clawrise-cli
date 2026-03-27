package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestSearchSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/search" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", request.Method)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode search request: %v", err)
			}
			if payload["query"] != "项目" {
				t.Fatalf("unexpected query: %+v", payload["query"])
			}
			if payload["page_size"] != float64(10) {
				t.Fatalf("unexpected page_size: %+v", payload["page_size"])
			}
			filter := payload["filter"].(map[string]any)
			if filter["value"] != "page" {
				t.Fatalf("unexpected filter: %+v", filter)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"results": []map[string]any{
					{
						"object": "page",
						"id":     "page_123",
						"url":    "https://www.notion.so/page_123",
						"properties": map[string]any{
							"title": map[string]any{
								"title": []map[string]any{
									{
										"type":       "text",
										"plain_text": "项目周报",
										"text": map[string]any{
											"content": "项目周报",
										},
									},
								},
							},
						},
					},
					{
						"object": "data_source",
						"id":     "ds_123",
						"url":    "https://www.notion.so/ds_123",
						"title": []map[string]any{
							{
								"type":       "text",
								"plain_text": "项目库",
								"text": map[string]any{
									"content": "项目库",
								},
							},
						},
					},
				},
				"next_cursor": "cursor_next",
				"has_more":    true,
			}), nil
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.Search(context.Background(), testStaticProfile(), map[string]any{
		"query":     "项目",
		"page_size": 10,
		"filter": map[string]any{
			"property": "object",
			"value":    "page",
		},
	})
	if appErr != nil {
		t.Fatalf("Search returned error: %+v", appErr)
	}

	items := data["items"].([]map[string]any)
	if len(items) != 2 {
		t.Fatalf("unexpected items length: %d", len(items))
	}
	if items[0]["title"] != "项目周报" {
		t.Fatalf("unexpected first item title: %+v", items[0]["title"])
	}
	if items[1]["title"] != "项目库" {
		t.Fatalf("unexpected second item title: %+v", items[1]["title"])
	}
	if data["next_page_token"] != "cursor_next" {
		t.Fatalf("unexpected next_page_token: %+v", data["next_page_token"])
	}
}

func TestBuildSearchPayloadRejectsInvalidFilter(t *testing.T) {
	_, appErr := buildSearchPayload(map[string]any{
		"filter": "page",
	})
	if appErr == nil {
		t.Fatal("expected buildSearchPayload to reject invalid filter")
	}
	if appErr.Code != "INVALID_INPUT" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}
