package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	feishuadapter "github.com/clawrise/clawrise-cli/internal/adapter/feishu"
	notionadapter "github.com/clawrise/clawrise-cli/internal/adapter/notion"
	"github.com/clawrise/clawrise-cli/internal/config"
)

func TestExecutorDryRunSuccess(t *testing.T) {
	t.Setenv("FEISHU_BOT_OPS_APP_ID", "app-id")
	t.Setenv("FEISHU_BOT_OPS_APP_SECRET", "app-secret")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "feishu",
			Profile:  "feishu_bot_ops",
		},
		Profiles: map[string]config.Profile{
			"feishu_bot_ops": {
				Platform: "feishu",
				Subject:  "bot",
				Grant: config.Grant{
					Type:      "client_credentials",
					AppID:     "env:FEISHU_BOT_OPS_APP_ID",
					AppSecret: "env:FEISHU_BOT_OPS_APP_SECRET",
				},
			},
		},
	})

	executor := NewExecutor(store, adapter.NewRegistry())
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
	if envelope.Context.Profile != "feishu_bot_ops" {
		t.Fatalf("unexpected context profile: %s", envelope.Context.Profile)
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
			Profile:  "notion_team_docs",
		},
		Profiles: map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		},
	})

	executor := NewExecutor(store, adapter.NewRegistry())
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

func TestExecutorRejectsSubjectMismatch(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Profile:  "notion_team_docs",
		},
		Profiles: map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		},
	})

	executor := NewExecutor(store, adapter.NewRegistry())
	envelope, err := executor.ExecuteContext(context.Background(), ExecuteOptions{
		OperationInput: "feishu.calendar.event.list",
		ProfileName:    "notion_team_docs",
		DryRun:         true,
	})
	if err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if envelope.OK {
		t.Fatal("expected execution to fail because of subject mismatch")
	}
	if envelope.Error == nil || envelope.Error.Code != "PROFILE_PLATFORM_MISMATCH" {
		t.Fatalf("unexpected error payload: %+v", envelope.Error)
	}
}

func TestExecutorExecutesNotionPageGet(t *testing.T) {
	t.Setenv("NOTION_ACCESS_TOKEN", "notion-token")

	store := newTestStore(t, &config.Config{
		Defaults: config.Defaults{
			Platform: "notion",
			Profile:  "notion_team_docs",
		},
		Profiles: map[string]config.Profile{
			"notion_team_docs": {
				Platform: "notion",
				Subject:  "integration",
				Grant: config.Grant{
					Type:  "static_token",
					Token: "env:NOTION_ACCESS_TOKEN",
				},
			},
		},
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
		registry: adapter.NewRegistry(),
		feishu:   feishuClient,
		notion:   notionClient,
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
