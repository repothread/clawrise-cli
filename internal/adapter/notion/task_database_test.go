package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateDatabaseSupportsShorthandFields(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/databases" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}
			if request.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", request.Method)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode create database request: %v", err)
			}
			parent := payload["parent"].(map[string]any)
			if parent["page_id"] != "page_demo" {
				t.Fatalf("unexpected parent payload: %+v", parent)
			}
			titleItems := payload["title"].([]any)
			titleText := titleItems[0].(map[string]any)["text"].(map[string]any)["content"]
			if titleText != "Project Hub" {
				t.Fatalf("unexpected title payload: %+v", payload["title"])
			}
			initialDataSource := payload["initial_data_source"].(map[string]any)
			if initialDataSource["name"] != "All Projects" {
				t.Fatalf("unexpected initial_data_source payload: %+v", initialDataSource)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"object": "database",
				"id":     "db_123",
				"url":    "https://www.notion.so/db_123",
				"title": []map[string]any{
					{
						"type":       "text",
						"plain_text": "Project Hub",
						"text": map[string]any{
							"content": "Project Hub",
						},
					},
				},
				"data_sources": []map[string]any{
					{
						"id":   "ds_123",
						"name": "All Projects",
					},
				},
			}), nil
		},
	})

	data, appErr := client.CreateDatabase(context.Background(), testStaticProfile(), map[string]any{
		"parent": map[string]any{
			"type": "page_id",
			"id":   "page_demo",
		},
		"title": "Project Hub",
		"initial_data_source": map[string]any{
			"name": "All Projects",
			"properties": map[string]any{
				"Name": map[string]any{
					"title": map[string]any{},
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("CreateDatabase returned error: %+v", appErr)
	}
	if data["database_id"] != "db_123" {
		t.Fatalf("unexpected database_id: %+v", data)
	}
}

func TestResolveDatabaseTargetFromURLReturnsDatabaseAndDefaultDataSource(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/databases/01234567-89ab-cdef-0123-456789abcdef":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object": "database",
					"id":     "01234567-89ab-cdef-0123-456789abcdef",
					"url":    "https://www.notion.so/db_123",
					"title": []map[string]any{
						{
							"type":       "text",
							"plain_text": "Project Hub",
							"text": map[string]any{
								"content": "Project Hub",
							},
						},
					},
					"data_sources": []map[string]any{
						{
							"id":   "ds_123",
							"name": "All Projects",
						},
					},
				}), nil
			case "/v1/data_sources/ds_123":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object": "data_source",
					"id":     "ds_123",
					"url":    "https://www.notion.so/ds_123",
					"parent": map[string]any{
						"type":        "database_id",
						"database_id": "01234567-89ab-cdef-0123-456789abcdef",
					},
					"title": []map[string]any{
						{
							"type":       "text",
							"plain_text": "All Projects",
							"text": map[string]any{
								"content": "All Projects",
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

	data, appErr := client.ResolveDatabaseTarget(context.Background(), testStaticProfile(), map[string]any{
		"target": "https://www.notion.so/workspace/Project-Hub-0123456789abcdef0123456789abcdef",
	})
	if appErr != nil {
		t.Fatalf("ResolveDatabaseTarget returned error: %+v", appErr)
	}
	if data["database_id"] != "01234567-89ab-cdef-0123-456789abcdef" {
		t.Fatalf("unexpected database resolution: %+v", data)
	}
	if data["data_source_id"] != "ds_123" {
		t.Fatalf("unexpected default data source resolution: %+v", data)
	}
}
