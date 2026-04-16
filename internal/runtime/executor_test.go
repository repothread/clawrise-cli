package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	feishuadapter "github.com/clawrise/clawrise-cli/internal/adapter/feishu"
	notionadapter "github.com/clawrise/clawrise-cli/internal/adapter/notion"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

// legacyTestAccount 仅用于构造运行时测试所需的旧授权桥接输入。
type legacyTestAccount struct {
	Title      string
	Platform   string
	Subject    string
	Method     string
	Params     legacyTestAuthParams
	LegacyAuth legacyTestAuth
}

type legacyTestAuthParams struct {
	AppID         string
	ClientID      string
	NotionVersion string
	RedirectMode  string
	Scopes        []string
}

type legacyTestAuth struct {
	Type         string
	AppID        string
	AppSecret    string
	Token        string
	ClientID     string
	ClientSecret string
	AccessToken  string
	RefreshToken string
	NotionVer    string
}

func TestExecutorDryRunSuccess(t *testing.T) {
	t.Setenv("FEISHU_BOT_OPS_APP_ID", "app-id")
	t.Setenv("FEISHU_BOT_OPS_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Account:  "feishu_bot_ops",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				LegacyAuth: legacyTestAuth{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_BOT_OPS_APP_ID",
					AppSecret: "env:FEISHU_BOT_OPS_APP_SECRET",
				},
			},
		}),
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "calendar.event.create",
		DryRun:         true,
		InputJSON:      `{"calendar_id":"cal_demo","summary":"Demo Event","start_at":"2026-03-30T10:00:00+08:00","end_at":"2026-03-30T11:00:00+08:00"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected dry-run success, got error: %+v", envelope.Error)
	}
	if envelope.Operation != "feishu.calendar.event.create" {
		t.Fatalf("unexpected operation: %s", envelope.Operation)
	}
	if envelope.Context == nil {
		t.Fatal("expected execution context to be present")
	}
	if envelope.Context.Platform != "feishu" {
		t.Fatalf("unexpected context platform: %s", envelope.Context.Platform)
	}
	if envelope.Context.Subject != "bot" {
		t.Fatalf("unexpected context subject: %s", envelope.Context.Subject)
	}
	if envelope.Context.Account != "feishu_bot_ops" {
		t.Fatalf("unexpected context account: %s", envelope.Context.Account)
	}
	if envelope.Idempotency == nil || envelope.Idempotency.Status != "dry_run" {
		t.Fatalf("unexpected idempotency state: %+v", envelope.Idempotency)
	}
}

func TestExecutorReadOperationOmitsIdempotency(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "page.get",
		DryRun:         true,
		InputJSON:      `{"page_id":"page_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected dry-run success, got error: %+v", envelope.Error)
	}
	if envelope.Idempotency != nil {
		t.Fatalf("expected no idempotency section for read operation, got: %+v", envelope.Idempotency)
	}
}

func TestExecutorDryRunWarnsWhenWriteEnhancementsAreSkipped(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput:       "page.create",
		DryRun:               true,
		DebugProviderPayload: true,
		VerifyAfterWrite:     true,
		InputJSON:            `{"title":"Dry Run","parent":{"type":"page_id","id":"page_demo"}}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected dry-run success, got error: %+v", envelope.Error)
	}
	joined := strings.Join(envelope.Warnings, " ")
	if !strings.Contains(joined, "skipped --debug-provider-payload") || !strings.Contains(joined, "skipped --verify") {
		t.Fatalf("expected dry-run enhancement warnings, got: %+v", envelope.Warnings)
	}
}

func TestExecutorRejectsSubjectMismatch(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "feishu.calendar.event.list",
		AccountName:    "notion_team_docs",
		DryRun:         true,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if envelope.OK {
		t.Fatal("expected execution to fail because of subject mismatch")
	}
	if envelope.Error == nil || envelope.Error.Code != "ACCOUNT_PLATFORM_MISMATCH" {
		t.Fatalf("unexpected error payload: %+v", envelope.Error)
	}
}

func TestExecutorUsesDefaultSubjectToSelectMatchingConnection(t *testing.T) {
	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Subject:  "user",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				LegacyAuth: legacyTestAuth{
					Type:  "client_credentials",
					AppID: "app-id",
				},
			},
			"feishu_user_alice": {
				Platform: "feishu",
				Subject:  "user",
				LegacyAuth: legacyTestAuth{
					Type:     "oauth_user",
					ClientID: "client-id",
				},
			},
		}),
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "docs.document.create",
		DryRun:         true,
		InputJSON:      `{}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected dry-run success, got error: %+v", envelope.Error)
	}
	if envelope.Context == nil {
		t.Fatal("expected execution context to be present")
	}
	if envelope.Context.Account != "feishu_user_alice" {
		t.Fatalf("unexpected selected account: %+v", envelope.Context)
	}
	if envelope.Context.Subject != "user" {
		t.Fatalf("unexpected selected subject: %+v", envelope.Context)
	}
}

func TestExecutorExplicitSubjectSkipsMismatchedDefaultConnection(t *testing.T) {
	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				LegacyAuth: legacyTestAuth{
					Type:  "client_credentials",
					AppID: "app-id",
				},
			},
			"feishu_user_alice": {
				Platform: "feishu",
				Subject:  "user",
				LegacyAuth: legacyTestAuth{
					Type:     "oauth_user",
					ClientID: "client-id",
				},
			},
		}),
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "docs.document.create",
		SubjectName:    "user",
		DryRun:         true,
		InputJSON:      `{}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected dry-run success, got error: %+v", envelope.Error)
	}
	if envelope.Context == nil {
		t.Fatal("expected execution context to be present")
	}
	if envelope.Context.Account != "feishu_user_alice" {
		t.Fatalf("unexpected selected account: %+v", envelope.Context)
	}
	if envelope.Context.Subject != "user" {
		t.Fatalf("unexpected selected subject: %+v", envelope.Context)
	}
}

func TestExecutorExplicitAccountOverridesDefaultSubject(t *testing.T) {
	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Subject:  "bot",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				LegacyAuth: legacyTestAuth{
					Type:  "client_credentials",
					AppID: "app-id",
				},
			},
			"feishu_user_alice": {
				Platform: "feishu",
				Subject:  "user",
				LegacyAuth: legacyTestAuth{
					Type:     "oauth_user",
					ClientID: "client-id",
				},
			},
		}),
	})

	executor := NewExecutor(store, newTestRegistry(t, nil, nil))
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "docs.document.create",
		AccountName:    "feishu_user_alice",
		DryRun:         true,
		InputJSON:      `{}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected explicit account to override default subject, got: %+v", envelope.Error)
	}
	if envelope.Context == nil {
		t.Fatal("expected execution context to be present")
	}
	if envelope.Context.Account != "feishu_user_alice" || envelope.Context.Subject != "user" {
		t.Fatalf("unexpected context: %+v", envelope.Context)
	}
}

func TestExecutorExecutesNotionPageGet(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path != "/v1/pages/page_demo" {
					t.Fatalf("unexpected request path: %s", request.URL.Path)
				}
				return jsonHTTPResponse(t, http.StatusOK, map[string]any{
					"id":       "page_demo",
					"url":      "https://www.notion.so/page_demo",
					"in_trash": false,
					"parent": map[string]any{
						"type":    "page_id",
						"page_id": "parent_demo",
					},
					"properties": map[string]any{
						"title": map[string]any{
							"title": []map[string]any{
								{
									"type":       "text",
									"plain_text": "执行器验证",
									"text": map[string]any{
										"content": "执行器验证",
									},
								},
							},
						},
					},
				}), nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, feishuClient, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "page.get",
		InputJSON:      `{"page_id":"page_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	if data["title"] != "执行器验证" {
		t.Fatalf("unexpected title: %+v", data["title"])
	}
}

func TestExecutorExecutesNotionPageUpdateWithInTrash(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	var requestBody map[string]any
	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path != "/v1/pages/page_demo" {
					t.Fatalf("unexpected request path: %s", request.URL.Path)
				}
				if request.Method != http.MethodPatch {
					t.Fatalf("unexpected method: %s", request.Method)
				}
				if err := json.NewDecoder(request.Body).Decode(&requestBody); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				return jsonHTTPResponse(t, http.StatusOK, map[string]any{
					"id":       "page_demo",
					"url":      "https://www.notion.so/page_demo",
					"in_trash": true,
				}), nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, feishuClient, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "page.update",
		InputJSON:      `{"page_id":"page_demo","in_trash":true}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion page update success, got error: %+v", envelope.Error)
	}

	if requestBody["in_trash"] != true {
		t.Fatalf("expected in_trash=true in request body, got: %+v", requestBody)
	}
	if _, exists := requestBody["archived"]; exists {
		t.Fatalf("expected archived to be omitted from request body, got: %+v", requestBody)
	}
}

func TestExecutorExecutesNotionPageUpdateWithArchivedAlias(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	var requestBody map[string]any
	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path != "/v1/pages/page_demo" {
					t.Fatalf("unexpected request path: %s", request.URL.Path)
				}
				if request.Method != http.MethodPatch {
					t.Fatalf("unexpected method: %s", request.Method)
				}
				if err := json.NewDecoder(request.Body).Decode(&requestBody); err != nil {
					t.Fatalf("failed to decode request body: %v", err)
				}
				return jsonHTTPResponse(t, http.StatusOK, map[string]any{
					"id":        "page_demo",
					"url":       "https://www.notion.so/page_demo",
					"archived":  true,
					"in_trash":  true,
					"is_locked": false,
				}), nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, feishuClient, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "page.update",
		InputJSON:      `{"page_id":"page_demo","archived":true}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion page update success, got error: %+v", envelope.Error)
	}

	if requestBody["in_trash"] != true {
		t.Fatalf("expected archived alias to map to in_trash=true, got: %+v", requestBody)
	}
	if _, exists := requestBody["archived"]; exists {
		t.Fatalf("expected archived alias to be normalized away before request, got: %+v", requestBody)
	}
}

func TestExecutorExportsProviderDebugForSupportedNotionWrite(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path != "/v1/pages" {
					t.Fatalf("unexpected request path: %s", request.URL.Path)
				}
				return jsonHTTPResponse(t, http.StatusOK, map[string]any{
					"id":       "page_debug_demo",
					"url":      "https://www.notion.so/page_debug_demo",
					"in_trash": false,
					"parent": map[string]any{
						"type":    "page_id",
						"page_id": "page_demo",
					},
					"properties": map[string]any{
						"title": map[string]any{
							"title": []map[string]any{
								{
									"type":       "text",
									"plain_text": "调试页面",
									"text": map[string]any{
										"content": "调试页面",
									},
								},
							},
						},
					},
				}), nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, feishuClient, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput:       "page.create",
		DebugProviderPayload: true,
		InputJSON:            `{"title":"调试页面","parent":{"type":"page_id","id":"page_demo"}}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion execution success, got error: %+v", envelope.Error)
	}
	if envelope.Debug == nil {
		t.Fatal("expected debug payload to be present")
	}
	requests := envelope.Debug["provider_requests"].([]map[string]any)
	if len(requests) != 1 {
		t.Fatalf("unexpected debug payload: %+v", envelope.Debug)
	}
	if requests[0]["path"] != "/v1/pages" {
		t.Fatalf("unexpected debug request entry: %+v", requests[0])
	}
}

func TestExecutorSupportsVerificationAndDebugForNotionPageUpdate(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				switch request.URL.Path {
				case "/v1/pages/page_demo":
					switch request.Method {
					case http.MethodPatch, http.MethodGet:
						return jsonHTTPResponse(t, http.StatusOK, map[string]any{
							"id":       "page_demo",
							"url":      "https://www.notion.so/page_demo",
							"in_trash": false,
							"parent": map[string]any{
								"type":    "page_id",
								"page_id": "parent_demo",
							},
							"properties": map[string]any{
								"title": map[string]any{
									"title": []map[string]any{
										{
											"type":       "text",
											"plain_text": "执行器更新验证",
											"text": map[string]any{
												"content": "执行器更新验证",
											},
										},
									},
								},
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
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, feishuClient, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput:       "page.update",
		DebugProviderPayload: true,
		VerifyAfterWrite:     true,
		InputJSON:            `{"page_id":"page_demo","title":"执行器更新验证"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion page update success, got error: %+v", envelope.Error)
	}
	if envelope.Debug == nil {
		t.Fatal("expected debug payload to be present")
	}
	data := envelope.Data.(map[string]any)
	verification := data["verification"].(map[string]any)
	if verification["ok"] != true {
		t.Fatalf("unexpected verification result: %+v", verification)
	}
	requests := envelope.Debug["provider_requests"].([]map[string]any)
	if len(requests) != 2 {
		t.Fatalf("unexpected debug payload: %+v", envelope.Debug)
	}
}

func TestExecutorExecutesNotionPageMarkdownGet(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path != "/v1/pages/page_demo/markdown" {
					t.Fatalf("unexpected request path: %s", request.URL.Path)
				}
				return jsonHTTPResponse(t, http.StatusOK, map[string]any{
					"object":            "page_markdown",
					"id":                "page_demo",
					"markdown":          "# 执行器验证",
					"truncated":         false,
					"unknown_block_ids": []string{},
				}), nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, nil, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "page.markdown.get",
		InputJSON:      `{"page_id":"page_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion markdown execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	if data["markdown"] != "# 执行器验证" {
		t.Fatalf("unexpected markdown: %+v", data["markdown"])
	}
}

func TestExecutorExecutesNotionSearch(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path != "/v1/search" {
					t.Fatalf("unexpected request path: %s", request.URL.Path)
				}
				return jsonHTTPResponse(t, http.StatusOK, map[string]any{
					"results": []map[string]any{
						{
							"object": "page",
							"id":     "page_demo",
							"properties": map[string]any{
								"title": map[string]any{
									"title": []map[string]any{
										{
											"type":       "text",
											"plain_text": "搜索命中",
											"text": map[string]any{
												"content": "搜索命中",
											},
										},
									},
								},
							},
						},
					},
					"has_more": false,
				}), nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, nil, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "search.query",
		InputJSON:      `{"query":"搜索"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion search execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	items := data["items"].([]map[string]any)
	if len(items) != 1 || items[0]["title"] != "搜索命中" {
		t.Fatalf("unexpected items: %+v", data["items"])
	}
}

func TestExecutorExecutesFeishuDocumentBlockGet(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Account:  "feishu_bot_ops",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				LegacyAuth: legacyTestAuth{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_APP_ID",
					AppSecret: "env:FEISHU_APP_SECRET",
				},
			},
		}),
	})

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				switch request.URL.Path {
				case "/open-apis/auth/v3/tenant_access_token/internal":
					return jsonHTTPResponse(t, http.StatusOK, map[string]any{
						"code":                0,
						"msg":                 "ok",
						"tenant_access_token": "tenant-token",
						"expire":              7200,
					}), nil
				case "/open-apis/docx/v1/documents/dox_123/blocks/blk_2":
					return jsonHTTPResponse(t, http.StatusOK, map[string]any{
						"code": 0,
						"msg":  "success",
						"data": map[string]any{
							"block": map[string]any{
								"block_id":   "blk_2",
								"parent_id":  "blk_1",
								"children":   []string{},
								"block_type": 2,
								"text": map[string]any{
									"elements": []map[string]any{
										{
											"text_run": map[string]any{
												"content": "执行器正文",
											},
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
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, feishuClient, nil),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "docs.block.get",
		InputJSON:      `{"document_id":"dox_123","block_id":"blk_2"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected feishu execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	if data["plain_text"] != "执行器正文" {
		t.Fatalf("unexpected plain_text: %+v", data["plain_text"])
	}
}

func TestExecutorExecutesFeishuDocumentBlockUpdate(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Account:  "feishu_bot_ops",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				LegacyAuth: legacyTestAuth{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_APP_ID",
					AppSecret: "env:FEISHU_APP_SECRET",
				},
			},
		}),
	})

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				switch request.URL.Path {
				case "/open-apis/auth/v3/tenant_access_token/internal":
					return jsonHTTPResponse(t, http.StatusOK, map[string]any{
						"code":                0,
						"msg":                 "ok",
						"tenant_access_token": "tenant-token",
						"expire":              7200,
					}), nil
				case "/open-apis/docx/v1/documents/dox_123/blocks/blk_2":
					return jsonHTTPResponse(t, http.StatusOK, map[string]any{
						"code": 0,
						"msg":  "success",
						"data": map[string]any{
							"block": map[string]any{
								"block_id":   "blk_2",
								"parent_id":  "blk_1",
								"children":   []string{},
								"block_type": 2,
								"text": map[string]any{
									"elements": []map[string]any{
										{
											"text_run": map[string]any{
												"content": "执行器更新正文",
											},
										},
									},
								},
							},
							"document_revision_id": 18,
						},
					}), nil
				default:
					t.Fatalf("unexpected request path: %s", request.URL.Path)
					return nil, nil
				}
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, feishuClient, nil),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "docs.block.update",
		InputJSON:      `{"document_id":"dox_123","block_id":"blk_2","text":"执行器更新正文"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected feishu execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	if data["plain_text"] != "执行器更新正文" {
		t.Fatalf("unexpected plain_text: %+v", data["plain_text"])
	}
}

func TestExecutorExecutesFeishuDocumentBlockDescendants(t *testing.T) {
	t.Setenv("FEISHU_APP_ID", "app-id")
	t.Setenv("FEISHU_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Account:  "feishu_bot_ops",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				LegacyAuth: legacyTestAuth{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_APP_ID",
					AppSecret: "env:FEISHU_APP_SECRET",
				},
			},
		}),
	})

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{
		BaseURL: "https://open.feishu.cn",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				switch request.URL.Path {
				case "/open-apis/auth/v3/tenant_access_token/internal":
					return jsonHTTPResponse(t, http.StatusOK, map[string]any{
						"code":                0,
						"msg":                 "ok",
						"tenant_access_token": "tenant-token",
						"expire":              7200,
					}), nil
				case "/open-apis/docx/v1/documents/dox_123/blocks/blk_root/children":
					if request.URL.Query().Get("with_descendants") != "true" {
						t.Fatalf("unexpected with_descendants: %s", request.URL.Query().Get("with_descendants"))
					}
					return jsonHTTPResponse(t, http.StatusOK, map[string]any{
						"code": 0,
						"msg":  "success",
						"data": map[string]any{
							"items": []map[string]any{
								{
									"block_id":   "blk_child",
									"parent_id":  "blk_root",
									"children":   []string{},
									"block_type": 2,
									"text": map[string]any{
										"elements": []map[string]any{
											{
												"text_run": map[string]any{
													"content": "后代正文",
												},
											},
										},
									},
								},
							},
							"has_more": false,
						},
					}), nil
				default:
					t.Fatalf("unexpected request path: %s", request.URL.Path)
					return nil, nil
				}
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, feishuClient, nil),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "docs.block.get_descendants",
		InputJSON:      `{"document_id":"dox_123","block_id":"blk_root"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected feishu execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	items := data["items"].([]map[string]any)
	if len(items) != 1 || items[0]["plain_text"] != "后代正文" {
		t.Fatalf("unexpected items: %+v", data["items"])
	}
}

func TestExecutorExecutesNotionDataSourceQuery(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path != "/v1/data_sources/ds_demo/query" {
					t.Fatalf("unexpected request path: %s", request.URL.Path)
				}
				return jsonHTTPResponse(t, http.StatusOK, map[string]any{
					"type":     "page_or_data_source",
					"has_more": false,
					"results": []map[string]any{
						{
							"object": "page",
							"id":     "page_demo",
							"properties": map[string]any{
								"title": map[string]any{
									"title": []map[string]any{
										{
											"type":       "text",
											"plain_text": "数据源命中",
											"text": map[string]any{
												"content": "数据源命中",
											},
										},
									},
								},
							},
						},
					},
				}), nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, nil, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "data_source.query",
		InputJSON:      `{"data_source_id":"ds_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	items := data["items"].([]map[string]any)
	if len(items) != 1 || items[0]["title"] != "数据源命中" {
		t.Fatalf("unexpected items: %+v", data["items"])
	}
}

func TestExecutorExecutesNotionBlockListChildren(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	notionClient, err := notionadapter.NewClient(notionadapter.Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.Path != "/v1/blocks/block_demo/children" {
					t.Fatalf("unexpected request path: %s", request.URL.Path)
				}
				return jsonHTTPResponse(t, http.StatusOK, map[string]any{
					"results": []map[string]any{
						{
							"id":           "blk_1",
							"type":         "paragraph",
							"has_children": false,
							"in_trash":     false,
							"paragraph": map[string]any{
								"rich_text": []map[string]any{
									{
										"type":       "text",
										"plain_text": "结构化正文",
										"text": map[string]any{
											"content": "结构化正文",
										},
									},
								},
							},
						},
					},
					"has_more": false,
				}), nil
			}),
		},
	})
	if err != nil {
		t.Fatalf("failed to construct notion client: %v", err)
	}

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu client: %v", err)
	}

	executor := &Executor{
		store:    store,
		registry: newTestRegistry(t, feishuClient, notionClient),
		now:      time.Now,
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "block.list_children",
		InputJSON:      `{"block_id":"block_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected notion execution success, got error: %+v", envelope.Error)
	}
	data := envelope.Data.(map[string]any)
	items := data["items"].([]map[string]any)
	if len(items) != 1 {
		t.Fatalf("unexpected items length: %d", len(items))
	}
	if items[0]["plain_text"] != "结构化正文" {
		t.Fatalf("unexpected plain_text: %+v", items[0]["plain_text"])
	}
}

func TestExecutorPersistsIdempotencyAndReplaysWrite(t *testing.T) {
	t.Setenv("FEISHU_BOT_OPS_APP_ID", "app-id")
	t.Setenv("FEISHU_BOT_OPS_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Account:  "feishu_bot_ops",
		},
		Runtime: config.RuntimeConfig{
			Retry: config.RetryConfig{
				MaxAttempts: 0,
			},
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				LegacyAuth: legacyTestAuth{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_BOT_OPS_APP_ID",
					AppSecret: "env:FEISHU_BOT_OPS_APP_SECRET",
				},
			},
		}),
	})

	registry := adapter.NewRegistry()
	callCount := 0
	registry.Register(adapter.Definition{
		Operation:       "feishu.demo.write",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"bot"},
		Spec: adapter.OperationSpec{
			Summary: "测试写操作。",
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			callCount++
			return map[string]any{
				"message": call.Input["message"],
				"id":      "demo_1",
			}, nil
		},
	})

	executor := NewExecutor(store, registry)
	firstEnvelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.write",
		InputJSON:      `{"message":"hello"}`,
	})
	if err != nil {
		t.Fatalf("first ExecuteContext returned error: %v", err)
	}
	if !firstEnvelope.OK {
		t.Fatalf("expected first execution to succeed, got: %+v", firstEnvelope.Error)
	}
	if firstEnvelope.Idempotency == nil || firstEnvelope.Idempotency.Status != "executed" || !firstEnvelope.Idempotency.Persisted {
		t.Fatalf("unexpected first idempotency state: %+v", firstEnvelope.Idempotency)
	}

	secondEnvelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.write",
		InputJSON:      `{"message":"hello"}`,
	})
	if err != nil {
		t.Fatalf("second ExecuteContext returned error: %v", err)
	}
	if !secondEnvelope.OK {
		t.Fatalf("expected second execution to succeed, got: %+v", secondEnvelope.Error)
	}
	if secondEnvelope.Idempotency == nil || secondEnvelope.Idempotency.Status != "replayed" || !secondEnvelope.Idempotency.Persisted {
		t.Fatalf("unexpected replay idempotency state: %+v", secondEnvelope.Idempotency)
	}
	if callCount != 1 {
		t.Fatalf("expected handler to run once, got %d", callCount)
	}

	idempotencyDir := filepath.Join(filepath.Dir(store.Path()), "runtime", "idempotency")
	entries, err := os.ReadDir(idempotencyDir)
	if err != nil {
		t.Fatalf("failed to read idempotency dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one idempotency record, got %d", len(entries))
	}
}

func TestExecutorRejectsIdempotencyConflict(t *testing.T) {
	t.Setenv("FEISHU_BOT_OPS_APP_ID", "app-id")
	t.Setenv("FEISHU_BOT_OPS_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Account:  "feishu_bot_ops",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				LegacyAuth: legacyTestAuth{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_BOT_OPS_APP_ID",
					AppSecret: "env:FEISHU_BOT_OPS_APP_SECRET",
				},
			},
		}),
	})

	registry := adapter.NewRegistry()
	callCount := 0
	registry.Register(adapter.Definition{
		Operation:       "feishu.demo.write",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"bot"},
		Spec: adapter.OperationSpec{
			Summary: "测试写操作。",
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			callCount++
			return map[string]any{
				"message": call.Input["message"],
			}, nil
		},
	})

	executor := NewExecutor(store, registry)
	if _, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.write",
		IdempotencyKey: "idem-fixed",
		InputJSON:      `{"message":"hello"}`,
	}); err != nil {
		t.Fatalf("first ExecuteContext returned error: %v", err)
	}

	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.write",
		IdempotencyKey: "idem-fixed",
		InputJSON:      `{"message":"world"}`,
	})
	if err != nil {
		t.Fatalf("second ExecuteContext returned error: %v", err)
	}
	if envelope.OK {
		t.Fatal("expected idempotency conflict to fail")
	}
	if envelope.Error == nil || envelope.Error.Code != "IDEMPOTENCY_KEY_CONFLICT" {
		t.Fatalf("unexpected conflict error: %+v", envelope.Error)
	}
	if callCount != 1 {
		t.Fatalf("expected handler to run once, got %d", callCount)
	}
}

func TestExecutorRestartsRejectedInvalidInputIdempotencyRecord(t *testing.T) {
	t.Setenv("FEISHU_BOT_OPS_APP_ID", "app-id")
	t.Setenv("FEISHU_BOT_OPS_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Account:  "feishu_bot_ops",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				LegacyAuth: legacyTestAuth{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_BOT_OPS_APP_ID",
					AppSecret: "env:FEISHU_BOT_OPS_APP_SECRET",
				},
			},
		}),
	})

	registry := adapter.NewRegistry()
	callCount := 0
	shouldReject := true
	registry.Register(adapter.Definition{
		Operation:       "feishu.demo.write",
		Platform:        "feishu",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"bot"},
		Spec: adapter.OperationSpec{
			Summary: "测试写操作。",
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			callCount++
			if shouldReject {
				return nil, apperr.New("INVALID_INPUT", "old validation rejected this request")
			}
			return map[string]any{
				"message": call.Input["message"],
				"id":      "demo_1",
			}, nil
		},
	})

	executor := NewExecutor(store, registry)
	firstEnvelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.write",
		InputJSON:      `{"message":"hello"}`,
	})
	if err != nil {
		t.Fatalf("first ExecuteContext returned error: %v", err)
	}
	if firstEnvelope.OK {
		t.Fatal("expected first execution to fail with INVALID_INPUT")
	}
	if firstEnvelope.Error == nil || firstEnvelope.Error.Code != "INVALID_INPUT" {
		t.Fatalf("unexpected first error: %+v", firstEnvelope.Error)
	}
	if firstEnvelope.Idempotency == nil || firstEnvelope.Idempotency.Status != "rejected" || !firstEnvelope.Idempotency.Persisted {
		t.Fatalf("unexpected first idempotency state: %+v", firstEnvelope.Idempotency)
	}

	shouldReject = false
	secondEnvelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.write",
		InputJSON:      `{"message":"hello"}`,
	})
	if err != nil {
		t.Fatalf("second ExecuteContext returned error: %v", err)
	}
	if !secondEnvelope.OK {
		t.Fatalf("expected second execution to rerun instead of replaying stale INVALID_INPUT, got error: %+v", secondEnvelope.Error)
	}
	if secondEnvelope.Idempotency == nil || secondEnvelope.Idempotency.Status != "executed" || !secondEnvelope.Idempotency.Persisted {
		t.Fatalf("unexpected second idempotency state: %+v", secondEnvelope.Idempotency)
	}
	if callCount != 2 {
		t.Fatalf("expected handler to run twice, got %d", callCount)
	}
}

func TestExecutorRetriesRetryableReadByConfig(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Runtime: config.RuntimeConfig{
			Retry: config.RetryConfig{
				MaxAttempts: 1,
				BaseDelayMS: 1,
				MaxDelayMS:  1,
			},
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	registry := adapter.NewRegistry()
	callCount := 0
	registry.Register(adapter.Definition{
		Operation:       "notion.demo.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "测试读操作。",
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			callCount++
			if callCount == 1 {
				return nil, apperr.New("TEMPORARY", "temporary upstream error").WithRetryable(true)
			}
			return map[string]any{
				"id": "ok",
			}, nil
		},
	})

	executor := NewExecutor(store, registry)
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.get",
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected retry to succeed, got: %+v", envelope.Error)
	}
	if envelope.Meta.RetryCount != 1 {
		t.Fatalf("expected retry_count=1, got %+v", envelope.Meta)
	}
	if callCount != 2 {
		t.Fatalf("expected handler to run twice, got %d", callCount)
	}
}

func TestExecutorWritesRedactedAuditLog(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Account:  "notion_team_docs",
		},
		Accounts: accountsFromLegacyAccounts(map[string]legacyTestAccount{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				LegacyAuth: legacyTestAuth{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		}),
	})

	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "notion.demo.get",
		Platform:        "notion",
		Mutating:        false,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "测试审计。",
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{
				"access_token":  "secret-token",
				"authorization": "Bearer top-secret",
				"message":       "safe",
			}, nil
		},
	})

	executor := NewExecutor(store, registry)
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.get",
		InputJSON:      `{"token":"very-secret","note":"visible"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected audit test execution to succeed, got: %+v", envelope.Error)
	}

	auditFile := filepath.Join(filepath.Dir(store.Path()), "runtime", "audit", time.Now().UTC().Format("2006-01-02")+".jsonl")
	data, err := os.ReadFile(auditFile)
	if err != nil {
		t.Fatalf("failed to read audit file: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "very-secret") || strings.Contains(content, "secret-token") || strings.Contains(content, "top-secret") {
		t.Fatalf("expected audit log to redact secrets, got: %s", content)
	}
	if !strings.Contains(content, `"note":"visible"`) {
		t.Fatalf("expected audit log to keep non-sensitive fields, got: %s", content)
	}
	if !strings.Contains(content, `"authorization":"***"`) {
		t.Fatalf("expected audit log to redact authorization field, got: %s", content)
	}
}

func TestExecutorLocalPolicyRequiresApproval(t *testing.T) {
	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "demo",
			Account:  "demo_operator",
		},
		Runtime: config.RuntimeConfig{
			Policy: config.PolicyConfig{
				RequireApprovalOperations: []string{"demo.page.update"},
			},
		},
		Accounts: map[string]config.Account{
			"demo_operator": {
				Platform: "demo",
				Subject:  "integration",
				Auth: config.AccountAuth{
					Method: "demo.token",
				},
			},
		},
	})

	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "demo.page.update",
		Platform:        "demo",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "Test local policy approval.",
			Idempotency: adapter.IdempotencySpec{
				Required: true,
			},
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{"ok": true}, nil
		},
	})

	executor := NewExecutor(store, registry)
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.page.update",
		DryRun:         true,
		InputJSON:      `{"id":"page_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if envelope.OK {
		t.Fatalf("expected approval-required policy to block execution: %+v", envelope)
	}
	if envelope.Error == nil || envelope.Error.Code != "POLICY_APPROVAL_REQUIRED" {
		t.Fatalf("unexpected policy error: %+v", envelope.Error)
	}
	if envelope.Policy == nil {
		t.Fatalf("expected structured policy result, got: %+v", envelope)
	}
	if envelope.Policy.FinalDecision != "require_approval" {
		t.Fatalf("unexpected final policy decision: %+v", envelope.Policy)
	}
	if len(envelope.Policy.Hits) != 1 {
		t.Fatalf("expected one policy hit, got: %+v", envelope.Policy)
	}
	if envelope.Policy.Hits[0].SourceType != "local" || envelope.Policy.Hits[0].SourceName != "runtime.policy.require_approval_operations" {
		t.Fatalf("unexpected policy hit source: %+v", envelope.Policy.Hits[0])
	}
	if envelope.Policy.Hits[0].MatchedRule != "demo.page.update" {
		t.Fatalf("unexpected matched rule: %+v", envelope.Policy.Hits[0])
	}
}

func TestExecutorLocalPolicyDenyProducesStructuredResult(t *testing.T) {
	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "demo",
			Account:  "demo_operator",
		},
		Runtime: config.RuntimeConfig{
			Policy: config.PolicyConfig{
				DenyOperations: []string{"demo.page.delete"},
			},
		},
		Accounts: map[string]config.Account{
			"demo_operator": {
				Platform: "demo",
				Subject:  "integration",
				Auth: config.AccountAuth{
					Method: "demo.token",
				},
			},
		},
	})

	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "demo.page.delete",
		Platform:        "demo",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "Test local policy deny.",
			Idempotency: adapter.IdempotencySpec{
				Required: true,
			},
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{"ok": true}, nil
		},
	})

	executor := NewExecutor(store, registry)
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.page.delete",
		DryRun:         true,
		InputJSON:      `{"id":"page_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if envelope.OK {
		t.Fatalf("expected deny policy to block execution: %+v", envelope)
	}
	if envelope.Error == nil || envelope.Error.Code != "POLICY_DENIED" {
		t.Fatalf("unexpected policy error: %+v", envelope.Error)
	}
	if envelope.Policy == nil || envelope.Policy.FinalDecision != "deny" {
		t.Fatalf("expected deny decision in structured policy result, got: %+v", envelope.Policy)
	}
	if len(envelope.Policy.Hits) != 1 || envelope.Policy.Hits[0].MatchedRule != "demo.page.delete" {
		t.Fatalf("unexpected policy hits: %+v", envelope.Policy)
	}
}

func TestExecutorPluginPolicyAnnotatesExecution(t *testing.T) {
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())

	pluginDir := filepath.Join(pluginRoot, "policy-demo", "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create policy plugin dir: %v", err)
	}
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.policy.evaluate"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"decision":"annotate","message":"manual review of the output is recommended","annotations":{"reviewer":"ops","severity":"medium"}}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(filepath.Join(pluginDir, "policy-demo.sh"), []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write policy plugin executable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{
  "schema_version": 2,
  "name": "policy-demo",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "policy",
      "id": "review",
      "priority": 90,
      "platforms": ["demo"]
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./policy-demo.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write policy plugin manifest: %v", err)
	}

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "demo",
			Account:  "demo_operator",
		},
		Accounts: map[string]config.Account{
			"demo_operator": {
				Platform: "demo",
				Subject:  "integration",
				Auth: config.AccountAuth{
					Method: "demo.token",
				},
			},
		},
	})

	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "demo.page.update",
		Platform:        "demo",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "Test policy plugin annotation.",
			Idempotency: adapter.IdempotencySpec{
				Required: true,
			},
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{"ok": true}, nil
		},
	})

	executor := NewExecutor(store, registry)
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.page.update",
		InputJSON:      `{"id":"page_demo"}`,
		IdempotencyKey: "idem_demo",
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected policy annotation to keep execution successful, got: %+v", envelope.Error)
	}
	if len(envelope.Warnings) == 0 || !strings.Contains(envelope.Warnings[0], "manual review of the output is recommended") {
		t.Fatalf("expected policy warning to be surfaced, got: %+v", envelope.Warnings)
	}
	if envelope.Policy == nil {
		t.Fatalf("expected structured policy result, got: %+v", envelope)
	}
	if envelope.Policy.FinalDecision != "allow" {
		t.Fatalf("unexpected final policy decision: %+v", envelope.Policy)
	}
	if len(envelope.Policy.Hits) != 1 {
		t.Fatalf("expected one policy hit, got: %+v", envelope.Policy)
	}
	if envelope.Policy.Hits[0].SourceType != "plugin" || envelope.Policy.Hits[0].SourceName != "policy-demo" || envelope.Policy.Hits[0].MatchedRule != "review" {
		t.Fatalf("unexpected policy hit source: %+v", envelope.Policy.Hits[0])
	}
	if envelope.Policy.Hits[0].Annotations["reviewer"] != "ops" {
		t.Fatalf("expected policy annotations to be preserved, got: %+v", envelope.Policy.Hits[0].Annotations)
	}
}

func TestExecutorPolicyManualSelectionSkipsUnconfiguredPlugins(t *testing.T) {
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())

	annotateDir := filepath.Join(pluginRoot, "policy-annotate", "0.1.0")
	if err := os.MkdirAll(annotateDir, 0o755); err != nil {
		t.Fatalf("failed to create annotate policy plugin dir: %v", err)
	}
	annotateScript := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.policy.evaluate"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"decision":"annotate","message":"selected policy executed"}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(filepath.Join(annotateDir, "policy-annotate.sh"), []byte(annotateScript), 0o755); err != nil {
		t.Fatalf("failed to write annotate policy executable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(annotateDir, "plugin.json"), []byte(`{
  "schema_version": 2,
  "name": "policy-annotate",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "policy",
      "id": "review",
      "priority": 90,
      "platforms": ["demo"]
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./policy-annotate.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write annotate policy manifest: %v", err)
	}

	denyDir := filepath.Join(pluginRoot, "policy-deny", "0.1.0")
	if err := os.MkdirAll(denyDir, 0o755); err != nil {
		t.Fatalf("failed to create deny policy plugin dir: %v", err)
	}
	denyScript := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.policy.evaluate"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"decision":"deny","message":"this plugin should not run"}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(filepath.Join(denyDir, "policy-deny.sh"), []byte(denyScript), 0o755); err != nil {
		t.Fatalf("failed to write deny policy executable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(denyDir, "plugin.json"), []byte(`{
  "schema_version": 2,
  "name": "policy-deny",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "policy",
      "id": "blocker",
      "priority": 100,
      "platforms": ["demo"]
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./policy-deny.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write deny policy manifest: %v", err)
	}

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "demo",
			Account:  "demo_operator",
		},
		Runtime: config.RuntimeConfig{
			Policy: config.PolicyConfig{
				Mode: "manual",
				Plugins: []config.PolicyPluginBinding{
					{Plugin: "policy-annotate", PolicyID: "review"},
				},
			},
		},
		Accounts: map[string]config.Account{
			"demo_operator": {
				Platform: "demo",
				Subject:  "integration",
				Auth: config.AccountAuth{
					Method: "demo.token",
				},
			},
		},
	})

	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "demo.page.update",
		Platform:        "demo",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "Test manual policy selection.",
			Idempotency: adapter.IdempotencySpec{
				Required: true,
			},
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{"ok": true}, nil
		},
	})

	executor := NewExecutor(store, registry)
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.page.update",
		DryRun:         true,
		InputJSON:      `{"id":"page_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected manual policy selection to skip unconfigured deny plugin, got: %+v", envelope.Error)
	}
	if len(envelope.Warnings) == 0 || !strings.Contains(strings.Join(envelope.Warnings, " "), "selected policy executed") {
		t.Fatalf("expected selected policy warning to be surfaced, got: %+v", envelope.Warnings)
	}
	if envelope.Policy == nil || len(envelope.Policy.Hits) != 1 || envelope.Policy.Hits[0].MatchedRule != "review" {
		t.Fatalf("unexpected structured policy result: %+v", envelope.Policy)
	}
}

func TestExecutorPolicyHitOrderKeepsLocalBeforePlugin(t *testing.T) {
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())

	pluginDir := filepath.Join(pluginRoot, "policy-demo", "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create policy plugin dir: %v", err)
	}
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.policy.evaluate"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"decision":"annotate","message":"plugin review is recommended"}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(filepath.Join(pluginDir, "policy-demo.sh"), []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write policy plugin executable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{
  "schema_version": 2,
  "name": "policy-demo",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "policy",
      "id": "review",
      "priority": 90,
      "platforms": ["demo"]
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./policy-demo.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write policy plugin manifest: %v", err)
	}

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "demo",
			Account:  "demo_operator",
		},
		Runtime: config.RuntimeConfig{
			Policy: config.PolicyConfig{
				AnnotateOperations: map[string]string{
					"demo.page.update": "local review is recommended",
				},
			},
		},
		Accounts: map[string]config.Account{
			"demo_operator": {
				Platform: "demo",
				Subject:  "integration",
				Auth: config.AccountAuth{
					Method: "demo.token",
				},
			},
		},
	})

	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "demo.page.update",
		Platform:        "demo",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "Test policy hit order.",
			Idempotency: adapter.IdempotencySpec{
				Required: true,
			},
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{"ok": true}, nil
		},
	})

	executor := NewExecutor(store, registry)
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.page.update",
		DryRun:         true,
		InputJSON:      `{"id":"page_demo"}`,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected annotated execution to succeed, got: %+v", envelope.Error)
	}
	if envelope.Policy == nil || len(envelope.Policy.Hits) != 2 {
		t.Fatalf("expected two ordered policy hits, got: %+v", envelope.Policy)
	}
	if envelope.Policy.Hits[0].SourceType != "local" || envelope.Policy.Hits[1].SourceType != "plugin" {
		t.Fatalf("expected local hit before plugin hit, got: %+v", envelope.Policy.Hits)
	}
	if envelope.Policy.Hits[0].Message != "local review is recommended" || envelope.Policy.Hits[1].Message != "plugin review is recommended" {
		t.Fatalf("unexpected policy hit messages: %+v", envelope.Policy.Hits)
	}
}

func TestExecutorEmitsAuditSinkPlugin(t *testing.T) {
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())

	markerPath := filepath.Join(t.TempDir(), "audit-sink.jsonl")
	pluginDir := filepath.Join(pluginRoot, "audit-demo", "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create audit sink plugin dir: %v", err)
	}
	script := fmt.Sprintf(`#!/bin/sh
marker=%q
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.audit.emit"'*)
      printf '%%s\n' "$line" >> "$marker"
      printf '{"jsonrpc":"2.0","id":"1","result":{}}'"\n"
      ;;
  esac
done
`, markerPath)
	if err := os.WriteFile(filepath.Join(pluginDir, "audit-demo.sh"), []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write audit sink plugin executable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{
  "schema_version": 2,
  "name": "audit-demo",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "audit_sink",
      "id": "capture",
      "priority": 50
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./audit-demo.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write audit sink plugin manifest: %v", err)
	}

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "demo",
			Account:  "demo_operator",
		},
		Accounts: map[string]config.Account{
			"demo_operator": {
				Platform: "demo",
				Subject:  "integration",
				Auth: config.AccountAuth{
					Method: "demo.token",
				},
			},
		},
	})

	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "demo.page.update",
		Platform:        "demo",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "Test audit sink fan-out.",
			Idempotency: adapter.IdempotencySpec{
				Required: true,
			},
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{"ok": true}, nil
		},
	})

	executor := NewExecutor(store, registry)
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.page.update",
		InputJSON:      `{"id":"page_demo"}`,
		IdempotencyKey: "idem_audit_demo",
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected audit sink execution to succeed, got: %+v", envelope.Error)
	}

	data, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("failed to read audit sink marker: %v", err)
	}
	if !strings.Contains(string(data), `"method":"clawrise.audit.emit"`) {
		t.Fatalf("expected audit sink plugin to receive emit request, got: %s", string(data))
	}
}

func TestExecutorEmitsBuiltinStdoutAuditSink(t *testing.T) {
	var sinkOutput bytes.Buffer
	previousWriter := builtinAuditStdoutWriter
	builtinAuditStdoutWriter = &sinkOutput
	t.Cleanup(func() {
		builtinAuditStdoutWriter = previousWriter
	})

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "demo",
			Account:  "demo_operator",
		},
		Runtime: config.RuntimeConfig{
			Audit: config.AuditConfig{
				Mode: "manual",
				Sinks: []config.AuditSinkConfig{
					{Type: "stdout"},
				},
			},
		},
		Accounts: map[string]config.Account{
			"demo_operator": {
				Platform: "demo",
				Subject:  "integration",
				Auth: config.AccountAuth{
					Method: "demo.token",
				},
			},
		},
	})

	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "demo.page.update",
		Platform:        "demo",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "Test builtin stdout audit sink.",
			Idempotency: adapter.IdempotencySpec{
				Required: true,
			},
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{"ok": true}, nil
		},
	})

	executor := NewExecutor(store, registry)
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.page.update",
		InputJSON:      `{"id":"page_demo"}`,
		IdempotencyKey: "idem_stdout_audit",
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected builtin stdout audit sink execution to succeed, got: %+v", envelope.Error)
	}
	if !strings.Contains(sinkOutput.String(), `"operation":"demo.page.update"`) {
		t.Fatalf("expected builtin stdout audit sink to write audit json, got: %s", sinkOutput.String())
	}
}

func TestExecutorEmitsBuiltinWebhookAuditSink(t *testing.T) {
	var (
		receivedBody   []byte
		receivedHeader string
	)
	server := newIPv4TestServer(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedHeader = request.Header.Get("Authorization")
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("failed to read webhook body: %v", err)
		}
		receivedBody = append([]byte(nil), body...)
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	t.Setenv("CLAWRISE_AUDIT_WEBHOOK_TOKEN", "Bearer audit-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "demo",
			Account:  "demo_operator",
		},
		Runtime: config.RuntimeConfig{
			Audit: config.AuditConfig{
				Mode: "manual",
				Sinks: []config.AuditSinkConfig{
					{
						Type:      "webhook",
						URL:       server.URL,
						Headers:   map[string]string{"Authorization": "env:CLAWRISE_AUDIT_WEBHOOK_TOKEN"},
						TimeoutMS: 3000,
					},
				},
			},
		},
		Accounts: map[string]config.Account{
			"demo_operator": {
				Platform: "demo",
				Subject:  "integration",
				Auth: config.AccountAuth{
					Method: "demo.token",
				},
			},
		},
	})

	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "demo.page.update",
		Platform:        "demo",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "Test builtin webhook audit sink.",
			Idempotency: adapter.IdempotencySpec{
				Required: true,
			},
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{"ok": true}, nil
		},
	})

	executor := NewExecutor(store, registry)
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.page.update",
		InputJSON:      `{"id":"page_demo"}`,
		IdempotencyKey: "idem_webhook_audit",
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if !envelope.OK {
		t.Fatalf("expected builtin webhook audit sink execution to succeed, got: %+v", envelope.Error)
	}
	if receivedHeader != "Bearer audit-token" {
		t.Fatalf("expected webhook header to be resolved from env, got: %s", receivedHeader)
	}
	if !strings.Contains(string(receivedBody), `"operation":"demo.page.update"`) {
		t.Fatalf("expected webhook body to carry audit record, got: %s", string(receivedBody))
	}
}

func newIPv4TestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	// 显式绑定到 127.0.0.1，避免在 IPv6 受限环境下默认监听 [::1] 造成测试不稳定。
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on ipv4 loopback: %v", err)
	}

	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	return server
}

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func jsonHTTPResponse(t *testing.T, statusCode int, value any) *http.Response {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal response body: %v", err)
	}

	return &http.Response{
		StatusCode: statusCode,
		Header: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
		},
		Body:          io.NopCloser(bytes.NewReader(data)),
		ContentLength: int64(len(data)),
		Request: &http.Request{
			Header: http.Header{},
		},
	}
}

func newTestStore(t *testing.T, cfg *config.Config) *config.Store {
	t.Helper()

	store := config.NewStore(t.TempDir() + "/config.yaml")
	if err := store.Save(cfg); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}
	return store
}

func accountsFromLegacyAccounts(profiles map[string]legacyTestAccount) map[string]config.Account {
	accounts := make(map[string]config.Account, len(profiles))
	for name, profile := range profiles {
		method := strings.TrimSpace(profile.Method)
		if method == "" {
			switch strings.TrimSpace(profile.LegacyAuth.Type) {
			case "client_credentials":
				method = "feishu.app_credentials"
			case "oauth_user":
				method = "feishu.oauth_user"
			case "static_token":
				method = "notion.internal_token"
			case "oauth_refreshable":
				method = "notion.oauth_public"
			}
		}

		account := config.Account{
			Title:    profile.Title,
			Platform: profile.Platform,
			Subject:  profile.Subject,
			Auth: config.AccountAuth{
				Method:     method,
				Public:     map[string]any{},
				SecretRefs: map[string]string{},
			},
		}

		switch method {
		case "feishu.app_credentials":
			if strings.TrimSpace(profile.Params.AppID) != "" {
				account.Auth.Public["app_id"] = strings.TrimSpace(profile.Params.AppID)
			} else if strings.TrimSpace(profile.LegacyAuth.AppID) != "" {
				account.Auth.Public["app_id"] = strings.TrimSpace(profile.LegacyAuth.AppID)
			}
			if strings.TrimSpace(profile.LegacyAuth.AppSecret) != "" {
				account.Auth.SecretRefs["app_secret"] = strings.TrimSpace(profile.LegacyAuth.AppSecret)
			}
		case "feishu.oauth_user":
			if strings.TrimSpace(profile.Params.ClientID) != "" {
				account.Auth.Public["client_id"] = strings.TrimSpace(profile.Params.ClientID)
			} else if strings.TrimSpace(profile.LegacyAuth.ClientID) != "" {
				account.Auth.Public["client_id"] = strings.TrimSpace(profile.LegacyAuth.ClientID)
			}
			if strings.TrimSpace(profile.Params.RedirectMode) != "" {
				account.Auth.Public["redirect_mode"] = strings.TrimSpace(profile.Params.RedirectMode)
			}
			if len(profile.Params.Scopes) > 0 {
				account.Auth.Public["scopes"] = append([]string(nil), profile.Params.Scopes...)
			}
			if strings.TrimSpace(profile.LegacyAuth.ClientSecret) != "" {
				account.Auth.SecretRefs["client_secret"] = strings.TrimSpace(profile.LegacyAuth.ClientSecret)
			}
			if strings.TrimSpace(profile.LegacyAuth.AccessToken) != "" {
				account.Auth.SecretRefs["access_token"] = strings.TrimSpace(profile.LegacyAuth.AccessToken)
			}
			if strings.TrimSpace(profile.LegacyAuth.RefreshToken) != "" {
				account.Auth.SecretRefs["refresh_token"] = strings.TrimSpace(profile.LegacyAuth.RefreshToken)
			}
		case "notion.internal_token":
			if strings.TrimSpace(profile.Params.NotionVersion) != "" {
				account.Auth.Public["notion_version"] = strings.TrimSpace(profile.Params.NotionVersion)
			} else if strings.TrimSpace(profile.LegacyAuth.NotionVer) != "" {
				account.Auth.Public["notion_version"] = strings.TrimSpace(profile.LegacyAuth.NotionVer)
			}
			if strings.TrimSpace(profile.LegacyAuth.Token) != "" {
				account.Auth.SecretRefs["token"] = strings.TrimSpace(profile.LegacyAuth.Token)
			}
		case "notion.oauth_public":
			if strings.TrimSpace(profile.Params.ClientID) != "" {
				account.Auth.Public["client_id"] = strings.TrimSpace(profile.Params.ClientID)
			} else if strings.TrimSpace(profile.LegacyAuth.ClientID) != "" {
				account.Auth.Public["client_id"] = strings.TrimSpace(profile.LegacyAuth.ClientID)
			}
			if strings.TrimSpace(profile.Params.NotionVersion) != "" {
				account.Auth.Public["notion_version"] = strings.TrimSpace(profile.Params.NotionVersion)
			} else if strings.TrimSpace(profile.LegacyAuth.NotionVer) != "" {
				account.Auth.Public["notion_version"] = strings.TrimSpace(profile.LegacyAuth.NotionVer)
			}
			if strings.TrimSpace(profile.Params.RedirectMode) != "" {
				account.Auth.Public["redirect_mode"] = strings.TrimSpace(profile.Params.RedirectMode)
			}
			if len(profile.Params.Scopes) > 0 {
				account.Auth.Public["scopes"] = append([]string(nil), profile.Params.Scopes...)
			}
			if strings.TrimSpace(profile.LegacyAuth.ClientSecret) != "" {
				account.Auth.SecretRefs["client_secret"] = strings.TrimSpace(profile.LegacyAuth.ClientSecret)
			}
			if strings.TrimSpace(profile.LegacyAuth.AccessToken) != "" {
				account.Auth.SecretRefs["access_token"] = strings.TrimSpace(profile.LegacyAuth.AccessToken)
			}
			if strings.TrimSpace(profile.LegacyAuth.RefreshToken) != "" {
				account.Auth.SecretRefs["refresh_token"] = strings.TrimSpace(profile.LegacyAuth.RefreshToken)
			}
		}

		if len(account.Auth.Public) == 0 {
			account.Auth.Public = nil
		}
		if len(account.Auth.SecretRefs) == 0 {
			account.Auth.SecretRefs = nil
		}
		accounts[name] = account
	}
	return accounts
}

// TestExecutorPluginAuditSinkSupportsMultipleEmitsInOneCall 验证 BUG-02 修复：
// 同一 CLI 调用中多次审计事件写入不会因插件进程被关闭而失败。
func TestExecutorPluginAuditSinkSupportsMultipleEmitsInOneCall(t *testing.T) {
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())

	markerPath := filepath.Join(t.TempDir(), "audit-sink-multi.jsonl")
	pluginDir := filepath.Join(pluginRoot, "audit-multi", "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create audit sink plugin dir: %v", err)
	}
	script := fmt.Sprintf(`#!/bin/sh
marker=%q
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.audit.emit"'*)
      printf '%%s\n' "$line" >> "$marker"
      printf '{"jsonrpc":"2.0","id":"1","result":{}}'"\n"
      ;;
  esac
done
`, markerPath)
	if err := os.WriteFile(filepath.Join(pluginDir, "audit-multi.sh"), []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write audit sink plugin executable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{
  "schema_version": 2,
  "name": "audit-multi",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "audit_sink",
      "id": "capture",
      "priority": 50
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./audit-multi.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write audit sink plugin manifest: %v", err)
	}

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "demo",
			Account:  "demo_operator",
		},
		Accounts: map[string]config.Account{
			"demo_operator": {
				Platform: "demo",
				Subject:  "integration",
				Auth: config.AccountAuth{
					Method: "demo.token",
				},
			},
		},
	})

	registry := adapter.NewRegistry()
	registry.Register(adapter.Definition{
		Operation:       "demo.page.update",
		Platform:        "demo",
		Mutating:        true,
		DefaultTimeout:  time.Second,
		AllowedSubjects: []string{"integration"},
		Spec: adapter.OperationSpec{
			Summary: "测试审计 sink 多次写入。",
			Idempotency: adapter.IdempotencySpec{
				Required: true,
			},
		},
		Handler: func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
			return map[string]any{"ok": true}, nil
		},
	})

	executor := NewExecutor(store, registry)

	// 第一次执行
	firstEnvelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.page.update",
		InputJSON:      `{"id":"page_1"}`,
		IdempotencyKey: "idem_multi_first",
	})
	if err != nil {
		t.Fatalf("first ExecuteContext returned error: %v", err)
	}
	if !firstEnvelope.OK {
		t.Fatalf("expected first execution to succeed, got: %+v", firstEnvelope.Error)
	}

	// 第二次执行 - 使用不同的 idempotency key
	secondEnvelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "demo.page.update",
		InputJSON:      `{"id":"page_2"}`,
		IdempotencyKey: "idem_multi_second",
	})
	if err != nil {
		t.Fatalf("second ExecuteContext returned error: %v", err)
	}
	if !secondEnvelope.OK {
		t.Fatalf("expected second execution to succeed, got: %+v", secondEnvelope.Error)
	}

	// 验证插件收到两次 emit
	data, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("failed to read audit sink marker: %v", err)
	}
	lines := strings.Count(string(data), `"method":"clawrise.audit.emit"`)
	if lines != 2 {
		t.Fatalf("expected 2 audit emit calls to plugin, got %d; marker content:\n%s", lines, string(data))
	}
}

// TestPluginAuditSinkCloseLifecycle 验证 closePluginAuditSinks 的行为正确性：
// 仅关闭 plugin 类型的 sink，不影响 stdout/webhook sink。
func TestPluginAuditSinkCloseLifecycle(t *testing.T) {
	t.Run("closes_only_plugin_sinks", func(t *testing.T) {
		var stdoutBuf bytes.Buffer
		sinks := []auditSink{
			&stdoutAuditSink{writer: &stdoutBuf},
			&pluginAuditSink{runtime: nil}, // nil runtime 不应 panic
		}
		// 调用 closePluginAuditSinks 不应 panic
		closePluginAuditSinks(sinks)

		// stdout sink 仍然可用
		record := auditRecord{Operation: "test.op", OK: true}
		if err := sinks[0].Emit(context.Background(), record); err != nil {
			t.Fatalf("stdout sink should still work after closePluginAuditSinks: %v", err)
		}
		if !strings.Contains(stdoutBuf.String(), "test.op") {
			t.Fatalf("expected stdout output to contain operation, got: %s", stdoutBuf.String())
		}
	})
}

func newTestRegistry(t *testing.T, feishuClient *feishuadapter.Client, notionClient *notionadapter.Client) *adapter.Registry {
	t.Helper()

	registry := adapter.NewRegistry()

	if feishuClient == nil {
		client, err := feishuadapter.NewClient(feishuadapter.Options{})
		if err != nil {
			t.Fatalf("failed to construct feishu client: %v", err)
		}
		feishuClient = client
	}
	if notionClient == nil {
		client, err := notionadapter.NewClient(notionadapter.Options{})
		if err != nil {
			t.Fatalf("failed to construct notion client: %v", err)
		}
		notionClient = client
	}

	feishuadapter.RegisterOperations(registry, feishuClient)
	notionadapter.RegisterOperations(registry, notionClient)
	return registry
}
