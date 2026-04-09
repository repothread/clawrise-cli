package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestUpsertDataSourceRowCreatesMissingRow(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/data_sources/ds_demo/query":
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode data source query request: %v", err)
				}
				filter := payload["filter"].(map[string]any)
				if filter["property"] != "External ID" {
					t.Fatalf("unexpected match filter payload: %+v", filter)
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"results":  []map[string]any{},
					"has_more": false,
				}), nil
			case "/v1/pages":
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode create page request: %v", err)
				}
				parent := payload["parent"].(map[string]any)
				if parent["data_source_id"] != "ds_demo" {
					t.Fatalf("unexpected create parent payload: %+v", parent)
				}
				if payload["markdown"] != "# Acme\n\n客户跟进记录" {
					t.Fatalf("unexpected create markdown payload: %+v", payload)
				}
				properties := payload["properties"].(map[string]any)
				if _, ok := properties["Name"]; !ok {
					t.Fatalf("expected Name title property in create payload: %+v", properties)
				}
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
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.UpsertDataSourceRow(context.Background(), testStaticProfile(), map[string]any{
		"data_source_id": "ds_demo",
		"match": map[string]any{
			"property": "External ID",
			"rich_text": map[string]any{
				"equals": "crm_123",
			},
		},
		"title":          "Acme",
		"title_property": "Name",
		"markdown":       "# Acme\n\n客户跟进记录",
	})
	if appErr != nil {
		t.Fatalf("UpsertDataSourceRow returned error: %+v", appErr)
	}
	if data["action"] != "created" || data["page_id"] != "page_created_123" {
		t.Fatalf("unexpected create result: %+v", data)
	}
}

func TestUpsertDataSourceRowUpdatesMatchedRow(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/data_sources/ds_demo/query":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"results": []map[string]any{
						{
							"object": "page",
							"id":     "page_existing_123",
							"url":    "https://www.notion.so/page_existing_123",
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
						},
					},
					"has_more": false,
				}), nil
			case "/v1/pages/page_existing_123":
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode page update request: %v", err)
				}
				properties := payload["properties"].(map[string]any)
				if _, ok := properties["Status"]; !ok {
					t.Fatalf("expected Status property update payload: %+v", properties)
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":       "page_existing_123",
					"url":      "https://www.notion.so/page_existing_123",
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
			case "/v1/pages/page_existing_123/markdown":
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode markdown update request: %v", err)
				}
				replaceContent := payload["replace_content"].(map[string]any)
				if replaceContent["new_str"] != "# Acme\n\n已同步最新跟进" {
					t.Fatalf("unexpected replace_content payload: %+v", replaceContent)
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object":            "page_markdown",
					"id":                "page_existing_123",
					"markdown":          "# Acme\n\n已同步最新跟进",
					"truncated":         false,
					"unknown_block_ids": []string{},
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.UpsertDataSourceRow(context.Background(), testStaticProfile(), map[string]any{
		"data_source_id": "ds_demo",
		"match": map[string]any{
			"property": "External ID",
			"rich_text": map[string]any{
				"equals": "crm_123",
			},
		},
		"properties": map[string]any{
			"Status": map[string]any{
				"select": map[string]any{
					"name": "Active",
				},
			},
		},
		"markdown": "# Acme\n\n已同步最新跟进",
	})
	if appErr != nil {
		t.Fatalf("UpsertDataSourceRow returned error: %+v", appErr)
	}
	if data["action"] != "updated" || data["page_id"] != "page_existing_123" {
		t.Fatalf("unexpected update result: %+v", data)
	}
}

func TestUpsertDataSourceRowRejectsAmbiguousMatches(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/data_sources/ds_demo/query" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			return jsonResponse(t, http.StatusOK, map[string]any{
				"results": []map[string]any{
					{"object": "page", "id": "page_dup_1"},
					{"object": "page", "id": "page_dup_2"},
				},
				"has_more": false,
			}), nil
		},
	})

	_, appErr := client.UpsertDataSourceRow(context.Background(), testStaticProfile(), map[string]any{
		"data_source_id": "ds_demo",
		"match": map[string]any{
			"property": "External ID",
			"rich_text": map[string]any{
				"equals": "crm_123",
			},
		},
		"properties": map[string]any{
			"Status": map[string]any{
				"select": map[string]any{
					"name": "Active",
				},
			},
		},
	})
	if appErr == nil {
		t.Fatal("expected UpsertDataSourceRow to reject ambiguous matches")
	}
	if appErr.Code != "AMBIGUOUS_TARGET" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}
