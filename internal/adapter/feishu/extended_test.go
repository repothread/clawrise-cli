package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestListCalendarEventsSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: &roundTripFunc{
				handler: func(request *http.Request) (*http.Response, error) {
					switch request.URL.Path {
					case "/open-apis/auth/v3/tenant_access_token/internal":
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code":                0,
							"tenant_access_token": "tenant-token",
						}), nil
					case "/open-apis/calendar/v4/calendars/cal_demo/events":
						if request.URL.Query().Get("page_size") != "20" {
							t.Fatalf("unexpected page_size: %s", request.URL.Query().Get("page_size"))
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"items": []map[string]any{
									{
										"event_id":              "evt_123",
										"organizer_calendar_id": "cal_demo",
										"summary":               "Demo Event",
										"start_time": map[string]any{
											"timestamp": "1711764000",
											"timezone":  "Asia/Shanghai",
										},
										"end_time": map[string]any{
											"timestamp": "1711767600",
											"timezone":  "Asia/Shanghai",
										},
									},
								},
								"page_token": "next_token",
								"has_more":   true,
							},
						}), nil
					default:
						t.Fatalf("unexpected request path: %s", request.URL.Path)
						return nil, nil
					}
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.ListCalendarEvents(context.Background(), testBotProfile(), map[string]any{
		"calendar_id": "cal_demo",
		"page_size":   20,
	})
	if appErr != nil {
		t.Fatalf("ListCalendarEvents returned error: %+v", appErr)
	}
	if data["next_page_token"] != "next_token" {
		t.Fatalf("unexpected next_page_token: %+v", data["next_page_token"])
	}
	items := data["items"].([]map[string]any)
	if len(items) != 1 || items[0]["event_id"] != "evt_123" {
		t.Fatalf("unexpected items: %+v", data["items"])
	}
}

func TestGetUpdateDeleteCalendarEventSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: &roundTripFunc{
				handler: func(request *http.Request) (*http.Response, error) {
					switch request.URL.Path {
					case "/open-apis/auth/v3/tenant_access_token/internal":
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code":                0,
							"tenant_access_token": "tenant-token",
						}), nil
					case "/open-apis/calendar/v4/calendars/cal_demo/events/evt_123":
						switch request.Method {
						case http.MethodGet:
							return jsonResponse(t, http.StatusOK, map[string]any{
								"code": 0,
								"data": map[string]any{
									"event": map[string]any{
										"event_id":              "evt_123",
										"organizer_calendar_id": "cal_demo",
										"summary":               "Demo Event",
									},
								},
							}), nil
						case http.MethodPatch:
							var payload map[string]any
							if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
								t.Fatalf("failed to decode update payload: %v", err)
							}
							if payload["summary"] != "Updated Event" {
								t.Fatalf("unexpected summary: %+v", payload["summary"])
							}
							return jsonResponse(t, http.StatusOK, map[string]any{
								"code": 0,
								"data": map[string]any{
									"event": map[string]any{
										"event_id":              "evt_123",
										"organizer_calendar_id": "cal_demo",
										"summary":               "Updated Event",
									},
								},
							}), nil
						case http.MethodDelete:
							return jsonResponse(t, http.StatusOK, map[string]any{
								"code": 0,
								"data": map[string]any{},
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
			},
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	got, appErr := client.GetCalendarEvent(context.Background(), testBotProfile(), map[string]any{
		"calendar_id": "cal_demo",
		"event_id":    "evt_123",
	})
	if appErr != nil {
		t.Fatalf("GetCalendarEvent returned error: %+v", appErr)
	}
	if got["summary"] != "Demo Event" {
		t.Fatalf("unexpected get summary: %+v", got["summary"])
	}

	updated, appErr := client.UpdateCalendarEvent(context.Background(), testBotProfile(), map[string]any{
		"calendar_id": "cal_demo",
		"event_id":    "evt_123",
		"summary":     "Updated Event",
	})
	if appErr != nil {
		t.Fatalf("UpdateCalendarEvent returned error: %+v", appErr)
	}
	if updated["summary"] != "Updated Event" {
		t.Fatalf("unexpected update summary: %+v", updated["summary"])
	}

	deleted, appErr := client.DeleteCalendarEvent(context.Background(), testBotProfile(), map[string]any{
		"calendar_id": "cal_demo",
		"event_id":    "evt_123",
	})
	if appErr != nil {
		t.Fatalf("DeleteCalendarEvent returned error: %+v", appErr)
	}
	if deleted["deleted"] != true {
		t.Fatalf("unexpected delete result: %+v", deleted)
	}
}

func TestCreateDocumentAndEditReplaceAllSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: &roundTripFunc{
				handler: func(request *http.Request) (*http.Response, error) {
					switch request.URL.Path {
					case "/open-apis/auth/v3/tenant_access_token/internal":
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code":                0,
							"tenant_access_token": "tenant-token",
						}), nil
					case "/open-apis/docx/v1/documents":
						var payload map[string]any
						if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
							t.Fatalf("failed to decode create document payload: %v", err)
						}
						if payload["title"] != "Project Notes" {
							t.Fatalf("unexpected title: %+v", payload["title"])
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"document": map[string]any{
									"document_id": "dox_123",
									"title":       "Project Notes",
									"revision_id": 1,
								},
							},
						}), nil
					case "/open-apis/docx/v1/documents/dox_123/blocks/dox_123/children":
						switch request.Method {
						case http.MethodGet:
							return jsonResponse(t, http.StatusOK, map[string]any{
								"code": 0,
								"data": map[string]any{
									"items": []map[string]any{
										{"block_id": "blk_old", "block_type": 2},
									},
								},
							}), nil
						case http.MethodPost:
							var payload map[string]any
							if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
								t.Fatalf("failed to decode append payload: %v", err)
							}
							children := payload["children"].([]any)
							if len(children) != 1 {
								t.Fatalf("unexpected children count: %d", len(children))
							}
							return jsonResponse(t, http.StatusOK, map[string]any{
								"code": 0,
								"data": map[string]any{
									"children": []map[string]any{
										{"block_id": "blk_new", "block_type": 2},
									},
								},
							}), nil
						default:
							t.Fatalf("unexpected method for children endpoint: %s", request.Method)
							return nil, nil
						}
					case "/open-apis/docx/v1/documents/dox_123/blocks/dox_123/children/batch_delete":
						if request.Method != http.MethodDelete {
							t.Fatalf("unexpected delete method: %s", request.Method)
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"document_revision_id": 2,
							},
						}), nil
					default:
						t.Fatalf("unexpected request path: %s", request.URL.Path)
						return nil, nil
					}
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	created, appErr := client.CreateDocument(context.Background(), testBotProfile(), map[string]any{
		"title": "Project Notes",
	})
	if appErr != nil {
		t.Fatalf("CreateDocument returned error: %+v", appErr)
	}
	if created["document_id"] != "dox_123" {
		t.Fatalf("unexpected document_id: %+v", created["document_id"])
	}

	edited, appErr := client.EditDocument(context.Background(), testBotProfile(), map[string]any{
		"document_id": "dox_123",
		"mode":        "replace_all",
		"text":        "Regenerated content",
	}, "idem-demo")
	if appErr != nil {
		t.Fatalf("EditDocument returned error: %+v", appErr)
	}
	if edited["deleted_count"] != 1 || edited["appended_count"] != 1 {
		t.Fatalf("unexpected edit result: %+v", edited)
	}
}

func TestBitableOperationsSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: &roundTripFunc{
				handler: func(request *http.Request) (*http.Response, error) {
					switch request.URL.Path {
					case "/open-apis/auth/v3/tenant_access_token/internal":
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code":                0,
							"tenant_access_token": "tenant-token",
						}), nil
					case "/open-apis/bitable/v1/apps/app_demo/tables":
						if request.Method != http.MethodGet {
							t.Fatalf("unexpected tables method: %s", request.Method)
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"items": []map[string]any{
									{"table_id": "tbl_demo", "name": "Tasks", "revision": 3},
								},
								"page_token": "tbl_cursor_next",
								"has_more":   true,
							},
						}), nil
					case "/open-apis/bitable/v1/apps/app_demo/tables/tbl_demo/fields":
						if request.Method != http.MethodGet {
							t.Fatalf("unexpected fields method: %s", request.Method)
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"items": []map[string]any{
									{
										"field_id":   "fld_title",
										"field_name": "Title",
										"type":       1,
										"property": map[string]any{
											"formatter":  "text",
											"is_primary": true,
										},
									},
								},
							},
						}), nil
					case "/open-apis/bitable/v1/apps/app_demo/tables/tbl_demo/records":
						switch request.Method {
						case http.MethodGet:
							return jsonResponse(t, http.StatusOK, map[string]any{
								"code": 0,
								"data": map[string]any{
									"items": []map[string]any{
										{"record_id": "rec_1", "fields": map[string]any{"Title": "Task A"}},
									},
									"page_token": "cursor_next",
									"has_more":   true,
								},
							}), nil
						case http.MethodPost:
							return jsonResponse(t, http.StatusOK, map[string]any{
								"code": 0,
								"data": map[string]any{
									"record": map[string]any{
										"record_id": "rec_2",
										"fields":    map[string]any{"Title": "Task B"},
									},
								},
							}), nil
						default:
							t.Fatalf("unexpected records method: %s", request.Method)
							return nil, nil
						}
					case "/open-apis/bitable/v1/apps/app_demo/tables/tbl_demo/records/rec_1":
						switch request.Method {
						case http.MethodGet:
							return jsonResponse(t, http.StatusOK, map[string]any{
								"code": 0,
								"data": map[string]any{
									"record": map[string]any{
										"record_id": "rec_1",
										"fields":    map[string]any{"Title": "Task A"},
									},
								},
							}), nil
						case http.MethodPut:
							return jsonResponse(t, http.StatusOK, map[string]any{
								"code": 0,
								"data": map[string]any{
									"record": map[string]any{
										"record_id": "rec_1",
										"fields":    map[string]any{"Status": "Done"},
									},
								},
							}), nil
						case http.MethodDelete:
							return jsonResponse(t, http.StatusOK, map[string]any{
								"code": 0,
								"data": map[string]any{},
							}), nil
						default:
							t.Fatalf("unexpected record method: %s", request.Method)
							return nil, nil
						}
					default:
						t.Fatalf("unexpected request path: %s", request.URL.Path)
						return nil, nil
					}
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	tables, appErr := client.ListBitableTables(context.Background(), testBotProfile(), map[string]any{
		"app_token": "app_demo",
	})
	if appErr != nil {
		t.Fatalf("ListBitableTables returned error: %+v", appErr)
	}
	if tables["next_page_token"] != "tbl_cursor_next" {
		t.Fatalf("unexpected table next_page_token: %+v", tables["next_page_token"])
	}
	tableItems := tables["items"].([]map[string]any)
	if len(tableItems) != 1 || tableItems[0]["name"] != "Tasks" {
		t.Fatalf("unexpected table items: %+v", tables["items"])
	}

	fields, appErr := client.ListBitableFields(context.Background(), testBotProfile(), map[string]any{
		"app_token": "app_demo",
		"table_id":  "tbl_demo",
	})
	if appErr != nil {
		t.Fatalf("ListBitableFields returned error: %+v", appErr)
	}
	fieldItems := fields["items"].([]map[string]any)
	if len(fieldItems) != 1 || fieldItems[0]["field_id"] != "fld_title" {
		t.Fatalf("unexpected field items: %+v", fields["items"])
	}
	if fieldItems[0]["is_primary"] != true {
		t.Fatalf("unexpected field primary flag: %+v", fieldItems[0]["is_primary"])
	}

	listed, appErr := client.ListBitableRecords(context.Background(), testBotProfile(), map[string]any{
		"app_token": "app_demo",
		"table_id":  "tbl_demo",
	})
	if appErr != nil {
		t.Fatalf("ListBitableRecords returned error: %+v", appErr)
	}
	if listed["next_page_token"] != "cursor_next" {
		t.Fatalf("unexpected next_page_token: %+v", listed["next_page_token"])
	}

	got, appErr := client.GetBitableRecord(context.Background(), testBotProfile(), map[string]any{
		"app_token": "app_demo",
		"table_id":  "tbl_demo",
		"record_id": "rec_1",
	})
	if appErr != nil {
		t.Fatalf("GetBitableRecord returned error: %+v", appErr)
	}
	if got["record_id"] != "rec_1" {
		t.Fatalf("unexpected record_id: %+v", got["record_id"])
	}

	created, appErr := client.CreateBitableRecord(context.Background(), testBotProfile(), map[string]any{
		"app_token": "app_demo",
		"table_id":  "tbl_demo",
		"fields": map[string]any{
			"Title": "Task B",
		},
	})
	if appErr != nil {
		t.Fatalf("CreateBitableRecord returned error: %+v", appErr)
	}
	if created["record_id"] != "rec_2" {
		t.Fatalf("unexpected created record_id: %+v", created["record_id"])
	}

	updated, appErr := client.UpdateBitableRecord(context.Background(), testBotProfile(), map[string]any{
		"app_token": "app_demo",
		"table_id":  "tbl_demo",
		"record_id": "rec_1",
		"fields": map[string]any{
			"Status": "Done",
		},
	})
	if appErr != nil {
		t.Fatalf("UpdateBitableRecord returned error: %+v", appErr)
	}
	if updated["record_id"] != "rec_1" {
		t.Fatalf("unexpected updated record_id: %+v", updated["record_id"])
	}

	deleted, appErr := client.DeleteBitableRecord(context.Background(), testBotProfile(), map[string]any{
		"app_token": "app_demo",
		"table_id":  "tbl_demo",
		"record_id": "rec_1",
	})
	if appErr != nil {
		t.Fatalf("DeleteBitableRecord returned error: %+v", appErr)
	}
	if deleted["deleted"] != true {
		t.Fatalf("unexpected delete result: %+v", deleted)
	}
}

func TestGetUserSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: &roundTripFunc{
				handler: func(request *http.Request) (*http.Response, error) {
					switch request.URL.Path {
					case "/open-apis/auth/v3/tenant_access_token/internal":
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code":                0,
							"tenant_access_token": "tenant-token",
						}), nil
					case "/open-apis/contact/v3/users/ou_demo":
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"user": map[string]any{
									"user_id": "ou_demo",
									"name":    "Demo User",
									"email":   "demo@example.com",
								},
							},
						}), nil
					default:
						t.Fatalf("unexpected request path: %s", request.URL.Path)
						return nil, nil
					}
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	data, appErr := client.GetUser(context.Background(), testBotProfile(), map[string]any{
		"user_id": "ou_demo",
	})
	if appErr != nil {
		t.Fatalf("GetUser returned error: %+v", appErr)
	}
	if data["email"] != "demo@example.com" {
		t.Fatalf("unexpected email: %+v", data["email"])
	}
}

func TestSearchUsersSuccess(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: &roundTripFunc{
				handler: func(request *http.Request) (*http.Response, error) {
					switch request.URL.Path {
					case "/open-apis/auth/v3/tenant_access_token/internal":
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code":                0,
							"tenant_access_token": "tenant-token",
						}), nil
					case "/open-apis/contact/v3/users":
						if request.URL.Query().Get("page_size") != "50" {
							t.Fatalf("unexpected page_size: %s", request.URL.Query().Get("page_size"))
						}
						switch request.URL.Query().Get("page_token") {
						case "":
							return jsonResponse(t, http.StatusOK, map[string]any{
								"code": 0,
								"data": map[string]any{
									"items": []map[string]any{
										{
											"user_id": "ou_demo_1",
											"name":    "Demo User One",
											"email":   "demo.one@example.com",
										},
										{
											"user_id": "ou_demo_2",
											"name":    "Demo User Two",
											"email":   "demo.two@example.com",
										},
									},
									"page_token": "cursor_2",
									"has_more":   true,
								},
							}), nil
						case "cursor_2":
							return jsonResponse(t, http.StatusOK, map[string]any{
								"code": 0,
								"data": map[string]any{
									"items": []map[string]any{
										{
											"user_id": "ou_demo_3",
											"name":    "Another User",
											"email":   "another@example.com",
										},
									},
									"has_more": false,
								},
							}), nil
						default:
							t.Fatalf("unexpected contact page_token: %s", request.URL.Query().Get("page_token"))
							return nil, nil
						}
					default:
						t.Fatalf("unexpected request path: %s", request.URL.Path)
						return nil, nil
					}
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	firstPage, appErr := client.SearchUsers(context.Background(), testBotProfile(), map[string]any{
		"query":     "demo",
		"page_size": 1,
	})
	if appErr != nil {
		t.Fatalf("SearchUsers first page returned error: %+v", appErr)
	}
	firstItems := firstPage["items"].([]map[string]any)
	if len(firstItems) != 1 || firstItems[0]["user_id"] != "ou_demo_1" {
		t.Fatalf("unexpected first page items: %+v", firstPage["items"])
	}
	if firstPage["has_more"] != true {
		t.Fatalf("expected first page has_more, got: %+v", firstPage["has_more"])
	}

	secondPage, appErr := client.SearchUsers(context.Background(), testBotProfile(), map[string]any{
		"query":      "demo",
		"page_size":  1,
		"page_token": firstPage["next_page_token"],
	})
	if appErr != nil {
		t.Fatalf("SearchUsers second page returned error: %+v", appErr)
	}
	secondItems := secondPage["items"].([]map[string]any)
	if len(secondItems) != 1 || secondItems[0]["user_id"] != "ou_demo_2" {
		t.Fatalf("unexpected second page items: %+v", secondPage["items"])
	}
	if matchedFields := secondItems[0]["matched_fields"].([]string); len(matchedFields) == 0 {
		t.Fatalf("expected matched_fields to be populated: %+v", secondItems[0])
	}
}
