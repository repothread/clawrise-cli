package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestEnsureDataSourceSchemaAddsMissingPropertiesAndOptions(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/data_sources/ds_demo":
				switch request.Method {
				case http.MethodGet:
					return jsonResponse(t, http.StatusOK, map[string]any{
						"object": "data_source",
						"id":     "ds_demo",
						"title": []map[string]any{
							{
								"type":       "text",
								"plain_text": "CRM",
								"text": map[string]any{
									"content": "CRM",
								},
							},
						},
						"properties": map[string]any{
							"Name": map[string]any{
								"type": "title",
							},
							"Status": map[string]any{
								"type": "select",
								"select": map[string]any{
									"options": []map[string]any{
										{
											"name":  "Active",
											"color": "green",
										},
									},
								},
							},
						},
						"parent": map[string]any{
							"type":        "database_id",
							"database_id": "db_demo",
						},
					}), nil
				case http.MethodPatch:
					var payload map[string]any
					if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
						t.Fatalf("failed to decode schema ensure payload: %v", err)
					}
					properties := payload["properties"].(map[string]any)
					if _, ok := properties["Owner"]; !ok {
						t.Fatalf("expected Owner property in update payload: %+v", properties)
					}
					status := properties["Status"].(map[string]any)
					options := status["select"].(map[string]any)["options"].([]any)
					if len(options) != 2 {
						t.Fatalf("expected merged select options: %+v", options)
					}

					return jsonResponse(t, http.StatusOK, map[string]any{
						"object": "data_source",
						"id":     "ds_demo",
						"title": []map[string]any{
							{
								"type":       "text",
								"plain_text": "CRM",
								"text": map[string]any{
									"content": "CRM",
								},
							},
						},
						"properties": map[string]any{
							"Name": map[string]any{
								"type": "title",
							},
							"Status": map[string]any{
								"type": "select",
							},
							"Owner": map[string]any{
								"type": "people",
							},
						},
						"parent": map[string]any{
							"type":        "database_id",
							"database_id": "db_demo",
						},
					}), nil
				default:
					t.Fatalf("unexpected method: %s", request.Method)
					return nil, nil
				}
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.EnsureDataSourceSchema(context.Background(), testStaticProfile(), map[string]any{
		"data_source_id": "ds_demo",
		"properties": map[string]any{
			"Status": map[string]any{
				"select": map[string]any{
					"options": []any{
						map[string]any{
							"name": "Active",
						},
						map[string]any{
							"name": "Pending",
						},
					},
				},
			},
			"Owner": map[string]any{
				"people": map[string]any{},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("EnsureDataSourceSchema returned error: %+v", appErr)
	}
	if data["action"] != "updated" {
		t.Fatalf("unexpected result: %+v", data)
	}
}

func TestEnsurePageSectionsCreatesOnlyMissingSections(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/pages/page_demo/markdown":
				switch request.Method {
				case http.MethodGet:
					return jsonResponse(t, http.StatusOK, map[string]any{
						"object":            "page_markdown",
						"id":                "page_demo",
						"markdown":          "# Weekly Review\n\n## Notes\n\nHello",
						"truncated":         false,
						"unknown_block_ids": []string{},
					}), nil
				case http.MethodPatch:
					var payload map[string]any
					if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
						t.Fatalf("failed to decode ensure sections payload: %v", err)
					}
					replaceContent := payload["replace_content"].(map[string]any)
					expected := "# Weekly Review\n\n## Notes\n\nHello\n\n## Summary"
					if replaceContent["new_str"] != expected {
						t.Fatalf("unexpected updated markdown: %+v", replaceContent["new_str"])
					}
					return jsonResponse(t, http.StatusOK, map[string]any{
						"object":            "page_markdown",
						"id":                "page_demo",
						"markdown":          expected,
						"truncated":         false,
						"unknown_block_ids": []string{},
					}), nil
				default:
					t.Fatalf("unexpected method: %s", request.Method)
					return nil, nil
				}
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.EnsurePageSections(context.Background(), testStaticProfile(), map[string]any{
		"page_id": "page_demo",
		"sections": []any{
			map[string]any{
				"heading":       "Notes",
				"heading_level": 2,
			},
			map[string]any{
				"heading":       "Summary",
				"heading_level": 2,
			},
		},
	})
	if appErr != nil {
		t.Fatalf("EnsurePageSections returned error: %+v", appErr)
	}
	if data["created_count"] != 1 || data["existing_count"] != 1 {
		t.Fatalf("unexpected ensure sections result: %+v", data)
	}
}

func TestAppendUnderHeadingAppendsMarkdown(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/pages/page_demo/markdown":
				switch request.Method {
				case http.MethodGet:
					return jsonResponse(t, http.StatusOK, map[string]any{
						"object":            "page_markdown",
						"id":                "page_demo",
						"markdown":          "# Weekly Review\n\n## Notes\n\nHello",
						"truncated":         false,
						"unknown_block_ids": []string{},
					}), nil
				case http.MethodPatch:
					var payload map[string]any
					if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
						t.Fatalf("failed to decode append payload: %v", err)
					}
					replaceContent := payload["replace_content"].(map[string]any)
					expected := "# Weekly Review\n\n## Notes\n\nHello\n\nWorld"
					if replaceContent["new_str"] != expected {
						t.Fatalf("unexpected appended markdown: %+v", replaceContent["new_str"])
					}
					return jsonResponse(t, http.StatusOK, map[string]any{
						"object":            "page_markdown",
						"id":                "page_demo",
						"markdown":          expected,
						"truncated":         false,
						"unknown_block_ids": []string{},
					}), nil
				default:
					t.Fatalf("unexpected method: %s", request.Method)
					return nil, nil
				}
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.AppendUnderHeading(context.Background(), testStaticProfile(), map[string]any{
		"page_id":       "page_demo",
		"heading":       "Notes",
		"heading_level": 2,
		"markdown":      "World",
	})
	if appErr != nil {
		t.Fatalf("AppendUnderHeading returned error: %+v", appErr)
	}
	if data["action"] != "appended" {
		t.Fatalf("unexpected append result: %+v", data)
	}
}

func TestFindOrCreatePageByPathCreatesMissingChainFromDatabaseRoot(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	createCount := 0
	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/databases/db_demo":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object": "database",
					"id":     "db_demo",
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
							"id":   "ds_demo",
							"name": "All Projects",
						},
					},
				}), nil
			case "/v1/data_sources/ds_demo":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object": "data_source",
					"id":     "ds_demo",
					"title": []map[string]any{
						{
							"type":       "text",
							"plain_text": "All Projects",
							"text": map[string]any{
								"content": "All Projects",
							},
						},
					},
					"properties": map[string]any{
						"Name": map[string]any{
							"type": "title",
						},
					},
					"parent": map[string]any{
						"type":        "database_id",
						"database_id": "db_demo",
					},
				}), nil
			case "/v1/data_sources/ds_demo/query":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"results":  []map[string]any{},
					"has_more": false,
				}), nil
			case "/v1/search":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"results":  []map[string]any{},
					"has_more": false,
				}), nil
			case "/v1/pages":
				createCount++
				var payload map[string]any
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatalf("failed to decode create page payload: %v", err)
				}

				if createCount == 1 {
					parent := payload["parent"].(map[string]any)
					if parent["data_source_id"] != "ds_demo" {
						t.Fatalf("unexpected first create parent: %+v", parent)
					}
					properties := payload["properties"].(map[string]any)
					if _, ok := properties["Name"]; !ok {
						t.Fatalf("expected inferred Name title property: %+v", payload)
					}
					return jsonResponse(t, http.StatusOK, map[string]any{
						"id":       "page_project_a",
						"url":      "https://www.notion.so/page_project_a",
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
										"plain_text": "Project A",
										"text": map[string]any{
											"content": "Project A",
										},
									},
								},
							},
						},
					}), nil
				}

				if payload["markdown"] != "# Weekly\n\nCreated by test." {
					t.Fatalf("unexpected leaf markdown: %+v", payload)
				}
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":       "page_weekly",
					"url":      "https://www.notion.so/page_weekly",
					"in_trash": false,
					"parent": map[string]any{
						"type":    "page_id",
						"page_id": "page_project_a",
					},
					"properties": map[string]any{
						"title": map[string]any{
							"title": []map[string]any{
								{
									"type":       "text",
									"plain_text": "Weekly",
									"text": map[string]any{
										"content": "Weekly",
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

	data, appErr := client.FindOrCreatePageByPath(context.Background(), testStaticProfile(), map[string]any{
		"database_id": "db_demo",
		"path":        []string{"Project A", "Weekly"},
		"markdown":    "# Weekly\n\nCreated by test.",
	})
	if appErr != nil {
		t.Fatalf("FindOrCreatePageByPath returned error: %+v", appErr)
	}
	if data["leaf_page_id"] != "page_weekly" || data["created_count"] != 2 {
		t.Fatalf("unexpected find or create result: %+v", data)
	}
}

func TestReadPageGraphFollowsRelationProperties(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	client := newTestClient(t, &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/v1/pages/page_root":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":       "page_root",
					"url":      "https://www.notion.so/page_root",
					"in_trash": false,
					"parent": map[string]any{
						"type":    "page_id",
						"page_id": "page_parent",
					},
					"properties": map[string]any{
						"Related": map[string]any{
							"id":   "rel_prop",
							"type": "relation",
						},
					},
				}), nil
			case "/v1/pages/page_root/properties/rel_prop":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object": "list",
					"results": []map[string]any{
						{
							"object": "property_item",
							"type":   "relation",
							"relation": map[string]any{
								"id": "page_related",
							},
						},
					},
					"has_more": false,
				}), nil
			case "/v1/pages/page_related":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id":       "page_related",
					"url":      "https://www.notion.so/page_related",
					"in_trash": false,
					"parent": map[string]any{
						"type":    "page_id",
						"page_id": "page_parent",
					},
					"properties": map[string]any{
						"Related": map[string]any{
							"id":   "rel_prop",
							"type": "relation",
						},
					},
				}), nil
			case "/v1/pages/page_related/properties/rel_prop":
				return jsonResponse(t, http.StatusOK, map[string]any{
					"object":   "list",
					"results":  []map[string]any{},
					"has_more": false,
				}), nil
			default:
				t.Fatalf("unexpected request path: %s", request.URL.Path)
				return nil, nil
			}
		},
	})

	data, appErr := client.ReadPageGraph(context.Background(), testStaticProfile(), map[string]any{
		"page_id":             "page_root",
		"relation_properties": []string{"Related"},
		"filter_properties":   []string{"Related"},
		"include_markdown":    false,
		"max_depth":           1,
		"max_nodes":           10,
	})
	if appErr != nil {
		t.Fatalf("ReadPageGraph returned error: %+v", appErr)
	}
	if data["node_count"] != 2 || data["edge_count"] != 1 {
		t.Fatalf("unexpected page graph result: %+v", data)
	}
}
