package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestBulkUpsertDataSourceRowsAggregatesResults(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	queryCount := 0
	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/data_sources/ds_demo/query":
				queryCount++

				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode data source query payload: %v", err)
				}
				filter := payload["filter"].(map[string]any)
				matchValue := filter["rich_text"].(map[string]any)["equals"]
				switch matchValue {
				case "crm_123":
					return jsonResponse(t, http.StatusOK, map[string]any{
						"results":  []map[string]any{},
						"has_more": false,
					}), nil
				case "crm_456":
					return jsonResponse(t, http.StatusOK, map[string]any{
						"results": []map[string]any{
							{
								"object": "page",
								"id":     "page_existing_456",
								"parent": map[string]any{
									"type":           "data_source_id",
									"data_source_id": "ds_demo",
								},
								"properties": map[string]any{
									"Name": map[string]any{
										"title": []map[string]any{
											{
												"type":       "text",
												"plain_text": "Globex",
												"text": map[string]any{
													"content": "Globex",
												},
											},
										},
									},
								},
							},
						},
						"has_more": false,
					}), nil
				default:
					t.Fatalf("unexpected filter payload: %+v", payload)
					return nil, nil
				}
			case "/v1/pages":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":       "page_created_123",
					"url":      "https://www.notion.so/page_created_123",
					"in_trash": false,
					"parent": map[string]any{
						"type":           "data_source_id",
						"data_source_id": "ds_demo",
					},
					"properties": map[string]any{
						"Name": map[string]any{
							"title": []map[string]any{
								{
									"type":       "text",
									"plain_text": "Acme",
									"text": map[string]any{
										"content": "Acme",
									},
								},
							},
						},
					},
				}), nil
			case "/v1/pages/page_existing_456":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":       "page_existing_456",
					"url":      "https://www.notion.so/page_existing_456",
					"in_trash": false,
					"parent": map[string]any{
						"type":           "data_source_id",
						"data_source_id": "ds_demo",
					},
					"properties": map[string]any{
						"Name": map[string]any{
							"title": []map[string]any{
								{
									"type":       "text",
									"plain_text": "Globex",
									"text": map[string]any{
										"content": "Globex",
									},
								},
							},
						},
					},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.BulkUpsertDataSourceRows(context.Background(), testStaticProfile(), map[string]any{
		"data_source_id": "ds_demo",
		"title_property": "Name",
		"items": []any{
			map[string]any{
				"match": map[string]any{
					"property": "External ID",
					"rich_text": map[string]any{
						"equals": "crm_123",
					},
				},
				"title": "Acme",
			},
			map[string]any{
				"match": map[string]any{
					"property": "External ID",
					"rich_text": map[string]any{
						"equals": "crm_456",
					},
				},
				"properties": map[string]any{
					"Status": map[string]any{
						"select": map[string]any{
							"name": "Active",
						},
					},
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("BulkUpsertDataSourceRows returned error: %+v", appErr)
	}
	if queryCount != 2 {
		t.Fatalf("unexpected query count: %d", queryCount)
	}
	if data["created_count"] != 1 || data["updated_count"] != 1 || data["failed_count"] != 0 {
		t.Fatalf("unexpected bulk upsert summary: %+v", data)
	}
}
