package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/config"
)

func TestRegisterOperationsAllowUserSubject(t *testing.T) {
	registry := adapter.NewRegistry()

	client, err := NewClient(Options{
		BaseURL: "https://open.feishu.cn",
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	RegisterOperations(registry, client)

	expectedUserOperations := map[string]bool{
		"feishu.calendar.calendar.list":        true,
		"feishu.calendar.event.create":         true,
		"feishu.calendar.event.list":           true,
		"feishu.calendar.event.get":            true,
		"feishu.calendar.event.update":         true,
		"feishu.calendar.event.delete":         true,
		"feishu.docs.document.create":          true,
		"feishu.docs.document.get":             true,
		"feishu.docs.document.list_blocks":     true,
		"feishu.docs.block.get":                true,
		"feishu.docs.block.list_children":      true,
		"feishu.docs.block.update":             true,
		"feishu.docs.block.batch_delete":       true,
		"feishu.docs.document.append_blocks":   true,
		"feishu.docs.document.edit":            true,
		"feishu.docs.document.get_raw_content": true,
		"feishu.docs.document.share":           true,
		"feishu.contact.user.get":              true,
		"feishu.contact.user.search":           true,
		"feishu.contact.department.list":       true,
		"feishu.department.user.list":          true,
		"feishu.bitable.table.list":            true,
		"feishu.bitable.field.list":            true,
		"feishu.bitable.record.list":           true,
		"feishu.bitable.record.get":            true,
		"feishu.bitable.record.create":         true,
		"feishu.bitable.record.batch_create":   true,
		"feishu.bitable.record.update":         true,
		"feishu.bitable.record.batch_update":   true,
		"feishu.bitable.record.delete":         true,
		"feishu.bitable.record.batch_delete":   true,
	}

	for _, definition := range registry.Definitions() {
		if definition.Platform != "feishu" {
			continue
		}
		if !containsSubject(definition.AllowedSubjects, "bot") {
			t.Fatalf("operation %s no longer allows bot subject", definition.Operation)
		}
		if expectedUserOperations[definition.Operation] && !containsSubject(definition.AllowedSubjects, "user") {
			t.Fatalf("operation %s should allow user subject", definition.Operation)
		}
		if !expectedUserOperations[definition.Operation] && containsSubject(definition.AllowedSubjects, "user") {
			t.Fatalf("operation %s should not allow user subject", definition.Operation)
		}
	}
}

func TestUserProfileCanCallFormerBotOnlyOperations(t *testing.T) {
	sessionStore := authcache.NewFileStore(filepath.Join(t.TempDir(), "config.yaml"))
	profileName := "feishu_user_test"

	if err := sessionStore.Save(authcache.Session{
		Version:     authcache.SessionVersion,
		ProfileName: profileName,
		Platform:    "feishu",
		Subject:     "user",
		GrantType:   "oauth_user",
		AccessToken: "user-token",
		TokenType:   "Bearer",
	}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	assertUserToken := func(request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer user-token" {
			t.Fatalf("unexpected authorization header for %s: %s", request.URL.Path, got)
		}
	}

	client, err := NewClient(Options{
		BaseURL:      "https://open.feishu.cn",
		SessionStore: sessionStore,
		HTTPClient: &http.Client{
			Transport: &roundTripFunc{
				handler: func(request *http.Request) (*http.Response, error) {
					switch request.URL.Path {
					case "/open-apis/auth/v3/tenant_access_token/internal":
						t.Fatalf("unexpected tenant access token request for user profile")
						return nil, nil
					case "/open-apis/authen/v2/oauth/token":
						t.Fatalf("unexpected oauth token refresh request for cached user session")
						return nil, nil
					case "/open-apis/calendar/v4/calendars/cal_demo/events":
						assertUserToken(request)
						if request.Method != http.MethodPost {
							t.Fatalf("unexpected create event method: %s", request.Method)
						}
						var payload map[string]any
						if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
							t.Fatalf("failed to decode create event payload: %v", err)
						}
						if payload["summary"] != "User Event" {
							t.Fatalf("unexpected create event summary: %+v", payload["summary"])
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"event": map[string]any{
									"event_id":              "evt_user_1",
									"organizer_calendar_id": "cal_demo",
									"summary":               "User Event",
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
						}), nil
					case "/open-apis/calendar/v4/calendars":
						assertUserToken(request)
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"calendar_list": []map[string]any{
									{
										"calendar_id": "cal_demo",
										"summary":     "User Calendar",
									},
								},
								"has_more": false,
							},
						}), nil
					case "/open-apis/docx/v1/documents/dox_123":
						assertUserToken(request)
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"document": map[string]any{
									"document_id": "dox_123",
									"title":       "User Document",
									"revision_id": 7,
								},
							},
						}), nil
					case "/open-apis/docx/v1/documents/dox_123/blocks":
						assertUserToken(request)
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"items": []map[string]any{
									{
										"block_id":   "blk_1",
										"block_type": 2,
									},
								},
								"has_more": false,
							},
						}), nil
					case "/open-apis/docx/v1/documents/dox_123/raw_content":
						assertUserToken(request)
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"content": "user raw content",
							},
						}), nil
					case "/open-apis/bitable/v1/apps/app_demo/tables":
						assertUserToken(request)
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"items": []map[string]any{
									{
										"table_id":    "tbl_demo",
										"name":        "User Table",
										"revision":    1,
										"field_count": 3,
									},
								},
							},
						}), nil
					case "/open-apis/contact/v3/users/ou_user_1":
						assertUserToken(request)
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"user": map[string]any{
									"user_id": "ou_user_1",
									"name":    "Alice",
								},
							},
						}), nil
					case "/open-apis/contact/v3/departments":
						assertUserToken(request)
						if request.URL.Query().Get("parent_department_id") != "0" {
							t.Fatalf("unexpected parent_department_id: %s", request.URL.Query().Get("parent_department_id"))
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"items": []map[string]any{
									{
										"open_department_id": "od_user_root",
										"name":               "User Dept",
									},
								},
								"has_more": false,
							},
						}), nil
					case "/open-apis/contact/v3/users":
						assertUserToken(request)
						if request.URL.Query().Get("department_id") != "od_user_root" {
							t.Fatalf("unexpected department_id: %s", request.URL.Query().Get("department_id"))
						}
						return jsonResponse(t, http.StatusOK, map[string]any{
							"code": 0,
							"data": map[string]any{
								"items": []map[string]any{
									{
										"user_id": "ou_dept_user_1",
										"name":    "Bob",
									},
								},
								"has_more": false,
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

	ctx := adapter.WithProfileName(context.Background(), profileName)
	profile := testUserProfile()

	createdEvent, appErr := client.CreateCalendarEvent(ctx, profile, map[string]any{
		"calendar_id": "cal_demo",
		"summary":     "User Event",
		"start_at":    "2024-03-30T10:00:00+08:00",
		"end_at":      "2024-03-30T11:00:00+08:00",
	}, "idem-user")
	if appErr != nil {
		t.Fatalf("CreateCalendarEvent returned error: %+v", appErr)
	}
	if createdEvent["event_id"] != "evt_user_1" {
		t.Fatalf("unexpected create event result: %+v", createdEvent)
	}

	calendars, appErr := client.ListCalendars(ctx, profile, map[string]any{})
	if appErr != nil {
		t.Fatalf("ListCalendars returned error: %+v", appErr)
	}
	if len(calendars["items"].([]map[string]any)) != 1 {
		t.Fatalf("unexpected calendars: %+v", calendars["items"])
	}

	document, appErr := client.GetDocument(ctx, profile, map[string]any{
		"document_id": "dox_123",
	})
	if appErr != nil {
		t.Fatalf("GetDocument returned error: %+v", appErr)
	}
	if document["title"] != "User Document" {
		t.Fatalf("unexpected document: %+v", document)
	}

	blocks, appErr := client.ListDocumentBlocks(ctx, profile, map[string]any{
		"document_id": "dox_123",
	})
	if appErr != nil {
		t.Fatalf("ListDocumentBlocks returned error: %+v", appErr)
	}
	if len(blocks["items"].([]map[string]any)) != 1 {
		t.Fatalf("unexpected blocks: %+v", blocks["items"])
	}

	rawContent, appErr := client.GetDocumentRawContent(ctx, profile, map[string]any{
		"document_id": "dox_123",
	})
	if appErr != nil {
		t.Fatalf("GetDocumentRawContent returned error: %+v", appErr)
	}
	if rawContent["content"] != "user raw content" {
		t.Fatalf("unexpected raw content: %+v", rawContent)
	}

	tables, appErr := client.ListBitableTables(ctx, profile, map[string]any{
		"app_token": "app_demo",
	})
	if appErr != nil {
		t.Fatalf("ListBitableTables returned error: %+v", appErr)
	}
	if len(tables["items"].([]map[string]any)) != 1 {
		t.Fatalf("unexpected tables: %+v", tables["items"])
	}

	user, appErr := client.GetUser(ctx, profile, map[string]any{
		"user_id": "ou_user_1",
	})
	if appErr != nil {
		t.Fatalf("GetUser returned error: %+v", appErr)
	}
	if user["name"] != "Alice" {
		t.Fatalf("unexpected user: %+v", user)
	}

	departments, appErr := client.ListDepartments(ctx, profile, map[string]any{})
	if appErr != nil {
		t.Fatalf("ListDepartments returned error: %+v", appErr)
	}
	if len(departments["items"].([]map[string]any)) != 1 {
		t.Fatalf("unexpected departments: %+v", departments["items"])
	}

	departmentUsers, appErr := client.ListDepartmentUsers(ctx, profile, map[string]any{
		"department_id": "od_user_root",
	})
	if appErr != nil {
		t.Fatalf("ListDepartmentUsers returned error: %+v", appErr)
	}
	if len(departmentUsers["items"].([]map[string]any)) != 1 {
		t.Fatalf("unexpected department users: %+v", departmentUsers["items"])
	}
}

func testUserProfile() config.Profile {
	return config.Profile{
		Platform: "feishu",
		Subject:  "user",
		Grant: config.Grant{
			Type: "oauth_user",
		},
	}
}

func containsSubject(subjects []string, target string) bool {
	for _, subject := range subjects {
		if subject == target {
			return true
		}
	}
	return false
}
