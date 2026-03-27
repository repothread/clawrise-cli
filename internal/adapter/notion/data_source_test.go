package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestQueryDataSourceSuccess(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/data_sources/ds_123/query" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", request.Method)
			}
			if request.URL.Query()["filter_properties[]"][0] != "title" {
				t.Fatalf("unexpected filter_properties: %+v", request.URL.Query()["filter_properties[]"])
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode query data source request: %v", err)
			}
			if payload["page_size"] != float64(20) {
				t.Fatalf("unexpected page_size: %+v", payload["page_size"])
			}
			sorts := payload["sorts"].([]any)
			if len(sorts) != 1 {
				t.Fatalf("unexpected sorts length: %d", len(sorts))
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"type":        "page_or_data_source",
				"object":      "list",
				"has_more":    true,
				"next_cursor": "cursor_next",
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
										"plain_text": "任务 A",
										"text": map[string]any{
											"content": "任务 A",
										},
									},
								},
							},
						},
					},
				},
			}), nil
		},
	}

	client := newTestClient(t, transport)
	data, appErr := client.QueryDataSource(context.Background(), testStaticProfile(), map[string]any{
		"data_source_id": "ds_123",
		"page_size":      20,
		"filter_properties": []any{
			"title",
		},
		"sorts": []any{
			map[string]any{
				"property":  "created_time",
				"direction": "descending",
			},
		},
	})
	if appErr != nil {
		t.Fatalf("QueryDataSource returned error: %+v", appErr)
	}
	if data["type"] != "page_or_data_source" {
		t.Fatalf("unexpected type: %+v", data["type"])
	}
	items := data["items"].([]map[string]any)
	if len(items) != 1 || items[0]["title"] != "任务 A" {
		t.Fatalf("unexpected items: %+v", data["items"])
	}
}

func TestBuildQueryDataSourcePayloadRejectsInvalidSorts(t *testing.T) {
	_, appErr := buildQueryDataSourcePayload(map[string]any{
		"sorts": "created_time",
	})
	if appErr == nil {
		t.Fatal("expected buildQueryDataSourcePayload to reject invalid sorts")
	}
	if appErr.Code != "INVALID_INPUT" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}
