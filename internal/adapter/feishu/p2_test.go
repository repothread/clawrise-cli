package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestFeishuP2OperationsSuccess(t *testing.T) {
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
					case "/open-apis/calendar/v4/calendars":
						if request.Method != http.MethodGet {
							t.Fatalf("unexpected calendars method: %s", request.Method)
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"calendar_list": []map[string]any{
									{
										"calendar_id": "cal_1",
										"summary":     "Team Calendar",
										"type":        "shared",
										"role":        "owner",
									},
								},
								"sync_token": "sync_demo",
								"has_more":   false,
							},
						}), nil
					case "/open-apis/contact/v3/departments/0/children":
						if request.Method != http.MethodGet {
							t.Fatalf("unexpected department method: %s", request.Method)
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"items": []map[string]any{
									{
										"open_department_id": "od_root_child",
										"name":               "Engineering",
										"member_count":       12,
									},
								},
								"has_more": false,
							},
						}), nil
					case "/open-apis/contact/v3/users/find_by_department":
						if request.Method != http.MethodGet {
							t.Fatalf("unexpected department users method: %s", request.Method)
						}
						if request.URL.Query().Get("department_id") != "od_root_child" {
							t.Fatalf("unexpected department_id: %s", request.URL.Query().Get("department_id"))
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"items": []map[string]any{
									{
										"user_id":        "ou_user_1",
										"name":           "Alice",
										"department_ids": []string{"od_root_child"},
									},
								},
								"has_more": false,
							},
						}), nil
					case "/open-apis/drive/v1/permissions/dox_123/members":
						if request.Method != http.MethodPost {
							t.Fatalf("unexpected share method: %s", request.Method)
						}
						if request.URL.Query().Get("type") != "docx" {
							t.Fatalf("unexpected permission type: %s", request.URL.Query().Get("type"))
						}
						var payload map[string]any
						if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
							t.Fatalf("failed to decode share payload: %v", err)
						}
						if payload["member_id"] != "ou_share_demo" {
							t.Fatalf("unexpected member_id: %+v", payload["member_id"])
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"member": map[string]any{
									"member_type": "openid",
									"member_id":   "ou_share_demo",
									"perm":        "edit",
									"type":        "user",
								},
							},
						}), nil
					case "/open-apis/bitable/v1/apps/app_demo/tables/tbl_demo/records/batch_create":
						if request.Method != http.MethodPost {
							t.Fatalf("unexpected batch create method: %s", request.Method)
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"records": []map[string]any{
									{
										"record_id": "rec_1",
										"fields":    map[string]any{"Title": "Task A"},
									},
								},
							},
						}), nil
					case "/open-apis/bitable/v1/apps/app_demo/tables/tbl_demo/records/batch_update":
						if request.Method != http.MethodPost {
							t.Fatalf("unexpected batch update method: %s", request.Method)
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"records": []map[string]any{
									{
										"record_id": "rec_1",
										"fields":    map[string]any{"Status": "Done"},
									},
								},
							},
						}), nil
					case "/open-apis/bitable/v1/apps/app_demo/tables/tbl_demo/records/batch_delete":
						if request.Method != http.MethodPost {
							t.Fatalf("unexpected batch delete method: %s", request.Method)
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"records": []map[string]any{
									{
										"record_id": "rec_1",
										"deleted":   true,
									},
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

	calendars, appErr := client.ListCalendars(context.Background(), testBotProfile(), map[string]any{
		"page_size": 50,
	})
	if appErr != nil {
		t.Fatalf("ListCalendars returned error: %+v", appErr)
	}
	calendarItems := calendars["items"].([]map[string]any)
	if len(calendarItems) != 1 || calendarItems[0]["calendar_id"] != "cal_1" {
		t.Fatalf("unexpected calendars: %+v", calendars["items"])
	}

	departments, appErr := client.ListDepartments(context.Background(), testBotProfile(), map[string]any{})
	if appErr != nil {
		t.Fatalf("ListDepartments returned error: %+v", appErr)
	}
	departmentItems := departments["items"].([]map[string]any)
	if len(departmentItems) != 1 || departmentItems[0]["name"] != "Engineering" {
		t.Fatalf("unexpected departments: %+v", departments["items"])
	}

	users, appErr := client.ListDepartmentUsers(context.Background(), testBotProfile(), map[string]any{
		"department_id": "od_root_child",
	})
	if appErr != nil {
		t.Fatalf("ListDepartmentUsers returned error: %+v", appErr)
	}
	userItems := users["items"].([]map[string]any)
	if len(userItems) != 1 || userItems[0]["user_id"] != "ou_user_1" {
		t.Fatalf("unexpected department users: %+v", users["items"])
	}

	shared, appErr := client.ShareDocument(context.Background(), testBotProfile(), map[string]any{
		"document_id": "dox_123",
		"member_type": "openid",
		"member_id":   "ou_share_demo",
		"perm":        "edit",
		"type":        "user",
	})
	if appErr != nil {
		t.Fatalf("ShareDocument returned error: %+v", appErr)
	}
	if shared["member_id"] != "ou_share_demo" {
		t.Fatalf("unexpected share result: %+v", shared)
	}

	batchCreated, appErr := client.BatchCreateBitableRecords(context.Background(), testBotProfile(), map[string]any{
		"app_token": "app_demo",
		"table_id":  "tbl_demo",
		"records": []any{
			map[string]any{
				"fields": map[string]any{
					"Title": "Task A",
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("BatchCreateBitableRecords returned error: %+v", appErr)
	}
	if batchCreated["created_count"] != 1 {
		t.Fatalf("unexpected batch create result: %+v", batchCreated)
	}

	batchUpdated, appErr := client.BatchUpdateBitableRecords(context.Background(), testBotProfile(), map[string]any{
		"app_token": "app_demo",
		"table_id":  "tbl_demo",
		"records": []any{
			map[string]any{
				"record_id": "rec_1",
				"fields": map[string]any{
					"Status": "Done",
				},
			},
		},
	})
	if appErr != nil {
		t.Fatalf("BatchUpdateBitableRecords returned error: %+v", appErr)
	}
	if batchUpdated["updated_count"] != 1 {
		t.Fatalf("unexpected batch update result: %+v", batchUpdated)
	}

	batchDeleted, appErr := client.BatchDeleteBitableRecords(context.Background(), testBotProfile(), map[string]any{
		"app_token": "app_demo",
		"table_id":  "tbl_demo",
		"records":   []any{"rec_1"},
	})
	if appErr != nil {
		t.Fatalf("BatchDeleteBitableRecords returned error: %+v", appErr)
	}
	if batchDeleted["deleted_count"] != 1 {
		t.Fatalf("unexpected batch delete result: %+v", batchDeleted)
	}
}
