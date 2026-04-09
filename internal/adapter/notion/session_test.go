package notion

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	authcache "github.com/clawrise/clawrise-cli/internal/auth"
)

func TestResolveNotionRefreshTokenPrefersCachedSession(t *testing.T) {
	// 优先使用已缓存 session 里的 refresh token，避免账号配置里的旧 token 覆盖最新轮转结果。
	t.Setenv("NOTION_REFRESH_TOKEN", "fallback-refresh-token")

	token, err := resolveNotionRefreshToken(
		ExecutionProfile{
			Platform: "notion",
			Subject:  "integration",
			Method:   "notion.oauth_public",
			Grant: ExecutionGrant{
				Type:         "oauth_refreshable",
				RefreshToken: "env:NOTION_REFRESH_TOKEN",
			},
		},
		&authcache.Session{
			RefreshToken: "cached-refresh-token",
		},
	)
	if err != nil {
		t.Fatalf("resolveNotionRefreshToken returned error: %v", err)
	}
	if token != "cached-refresh-token" {
		t.Fatalf("unexpected refresh token: %s", token)
	}
}

func TestResolveNotionRefreshTokenTreatsPendingSecretsAsAuthorizationRequired(t *testing.T) {
	// 对 env/secret 引用类“待注入”场景，要统一回落到 authorization_required，而不是暴露底层取值细节。
	cases := []struct {
		name         string
		refreshToken string
	}{
		{
			name:         "missing_env",
			refreshToken: "env:CLAWRISE_TEST_NOTION_REFRESH_TOKEN_MISSING",
		},
		{
			name:         "empty_env",
			refreshToken: "env:CLAWRISE_TEST_NOTION_REFRESH_TOKEN_EMPTY",
		},
	}

	t.Setenv("CLAWRISE_TEST_NOTION_REFRESH_TOKEN_EMPTY", "")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveNotionRefreshToken(ExecutionProfile{
				Platform: "notion",
				Subject:  "integration",
				Method:   "notion.oauth_public",
				Grant: ExecutionGrant{
					Type:         "oauth_refreshable",
					RefreshToken: tc.refreshToken,
				},
			}, nil)
			if !errors.Is(err, errNotionAuthorizationRequired) {
				t.Fatalf("expected authorization required error, got: %v", err)
			}
		})
	}
}

func TestShouldTreatOAuthSecretAsPending(t *testing.T) {
	// 这个判断直接影响 auth inspect / resolve 的用户提示，必须把常见错误分支钉死。
	cases := []struct {
		name string
		raw  string
		err  error
		want bool
	}{
		{
			name: "missing_env",
			raw:  "env:NOTION_REFRESH_TOKEN",
			err:  errors.New("environment variable NOTION_REFRESH_TOKEN is not set"),
			want: true,
		},
		{
			name: "empty_secret_ref",
			raw:  "secret:notion_bot:refresh_token",
			err:  errors.New("secret notion_bot.refresh_token is empty"),
			want: true,
		},
		{
			name: "plain_value_error",
			raw:  "plain-refresh-token",
			err:  errors.New("unexpected failure"),
			want: false,
		},
		{
			name: "nil_error",
			raw:  "env:NOTION_REFRESH_TOKEN",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldTreatOAuthSecretAsPending(tc.raw, tc.err)
			if got != tc.want {
				t.Fatalf("unexpected pending result: got=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestRefreshSessionSavesSession(t *testing.T) {
	// 覆盖 RefreshSession 这个对外 wrapper，确保它会读取旧 session、刷新 token，并把新结果写回 cache。
	t.Setenv("NOTION_CLIENT_ID", "client-id")
	t.Setenv("NOTION_CLIENT_SECRET", "client-secret")
	t.Setenv("NOTION_REFRESH_TOKEN", "fallback-refresh-token")

	sessionStore := authcache.NewFileStore(filepath.Join(t.TempDir(), "config.yaml"))
	if err := sessionStore.Save(authcache.Session{
		AccountName:  "notion_public_workspace_a",
		Platform:     "notion",
		Subject:      "integration",
		GrantType:    "notion.oauth_public",
		RefreshToken: "cached-refresh-token",
		TokenType:    "Bearer",
	}); err != nil {
		t.Fatalf("failed to seed session store: %v", err)
	}

	client, err := NewClient(Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: &roundTripFunc{
				handler: func(request *http.Request) (*http.Response, error) {
					if request.URL.Path != "/v1/oauth/token" {
						t.Fatalf("unexpected request path: %s", request.URL.Path)
					}
					if request.Method != http.MethodPost {
						t.Fatalf("unexpected method: %s", request.Method)
					}

					expectedAuthorization := "Basic " + base64.StdEncoding.EncodeToString([]byte("client-id:client-secret"))
					if got := request.Header.Get("Authorization"); got != expectedAuthorization {
						t.Fatalf("unexpected authorization header: %s", got)
					}

					var payload map[string]any
					if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
						t.Fatalf("failed to decode refresh payload: %v", err)
					}
					if payload["grant_type"] != "refresh_token" {
						t.Fatalf("unexpected grant_type: %+v", payload["grant_type"])
					}
					if payload["refresh_token"] != "cached-refresh-token" {
						t.Fatalf("expected cached refresh token to be used, got: %+v", payload["refresh_token"])
					}

					return jsonResponse(t, http.StatusOK, map[string]any{
						"access_token":  "fresh-access-token",
						"refresh_token": "rotated-refresh-token",
						"token_type":    "bearer",
						"expires_in":    3600,
					}), nil
				},
			},
		},
		SessionStore: sessionStore,
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	now := time.Date(2026, 4, 3, 4, 0, 0, 0, time.UTC)
	client.now = func() time.Time {
		return now
	}

	session, appErr := client.RefreshSession(context.Background(), "notion_public_workspace_a", ExecutionProfile{
		Platform: "notion",
		Subject:  "integration",
		Method:   "notion.oauth_public",
		Grant: ExecutionGrant{
			Type:         "oauth_refreshable",
			ClientID:     "env:NOTION_CLIENT_ID",
			ClientSecret: "env:NOTION_CLIENT_SECRET",
			RefreshToken: "env:NOTION_REFRESH_TOKEN",
		},
	})
	if appErr != nil {
		t.Fatalf("RefreshSession returned error: %+v", appErr)
	}
	if session.AccessToken != "fresh-access-token" {
		t.Fatalf("unexpected access token: %s", session.AccessToken)
	}
	if session.RefreshToken != "rotated-refresh-token" {
		t.Fatalf("unexpected refresh token: %s", session.RefreshToken)
	}

	saved, err := sessionStore.Load("notion_public_workspace_a")
	if err != nil {
		t.Fatalf("failed to load saved session: %v", err)
	}
	if saved.AccessToken != "fresh-access-token" {
		t.Fatalf("unexpected saved access token: %s", saved.AccessToken)
	}
	if saved.RefreshToken != "rotated-refresh-token" {
		t.Fatalf("unexpected saved refresh token: %s", saved.RefreshToken)
	}
	if saved.GrantType != "notion.oauth_public" {
		t.Fatalf("unexpected saved grant type: %s", saved.GrantType)
	}
}

func TestRefreshSessionRejectsUnsupportedProfiles(t *testing.T) {
	// wrapper 自身也要兜住非法 subject / method，避免直接把错误带到更深层网络调用。
	client := newTestClient(t, nil)

	_, appErr := client.RefreshSession(context.Background(), "notion_public_workspace_a", ExecutionProfile{
		Platform: "notion",
		Subject:  "user",
		Method:   "notion.oauth_public",
		Grant: ExecutionGrant{
			Type: "oauth_refreshable",
		},
	})
	if appErr == nil || appErr.Code != "SUBJECT_NOT_ALLOWED" {
		t.Fatalf("expected SUBJECT_NOT_ALLOWED, got: %+v", appErr)
	}

	_, appErr = client.RefreshSession(context.Background(), "notion_public_workspace_a", ExecutionProfile{
		Platform: "notion",
		Subject:  "integration",
		Method:   "notion.internal_token",
		Grant: ExecutionGrant{
			Type:  "static_token",
			Token: "env:NOTION_ACCESS_TOKEN",
		},
	})
	if appErr == nil || appErr.Code != "UNSUPPORTED_GRANT" {
		t.Fatalf("expected UNSUPPORTED_GRANT, got: %+v", appErr)
	}
}

func TestExchangeAuthorizationCodeSavesSessionAndMetadata(t *testing.T) {
	// 覆盖授权码换 token 的外层入口，确保 session 与 workspace 元数据都会落盘。
	t.Setenv("NOTION_CLIENT_ID", "client-id")
	t.Setenv("NOTION_CLIENT_SECRET", "client-secret")

	sessionStore := authcache.NewFileStore(filepath.Join(t.TempDir(), "config.yaml"))
	client, err := NewClient(Options{
		BaseURL: "https://api.notion.com",
		HTTPClient: &http.Client{
			Transport: &roundTripFunc{
				handler: func(request *http.Request) (*http.Response, error) {
					if request.URL.Path != "/v1/oauth/token" {
						t.Fatalf("unexpected request path: %s", request.URL.Path)
					}
					if request.Method != http.MethodPost {
						t.Fatalf("unexpected method: %s", request.Method)
					}

					expectedAuthorization := "Basic " + base64.StdEncoding.EncodeToString([]byte("client-id:client-secret"))
					if got := request.Header.Get("Authorization"); got != expectedAuthorization {
						t.Fatalf("unexpected authorization header: %s", got)
					}

					var payload map[string]any
					if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
						t.Fatalf("failed to decode auth code payload: %v", err)
					}
					if payload["grant_type"] != "authorization_code" {
						t.Fatalf("unexpected grant_type: %+v", payload["grant_type"])
					}
					if payload["code"] != "oauth-code" {
						t.Fatalf("unexpected code: %+v", payload["code"])
					}
					if payload["redirect_uri"] != "http://localhost:3333/callback" {
						t.Fatalf("unexpected redirect_uri: %+v", payload["redirect_uri"])
					}

					return jsonResponse(t, http.StatusOK, map[string]any{
						"access_token":   "fresh-access-token",
						"refresh_token":  "fresh-refresh-token",
						"token_type":     "bearer",
						"expires_in":     3600,
						"workspace_id":   "workspace_123",
						"workspace_name": "Workspace Name",
						"bot_id":         "bot_123",
					}), nil
				},
			},
		},
		SessionStore: sessionStore,
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	now := time.Date(2026, 4, 3, 4, 30, 0, 0, time.UTC)
	client.now = func() time.Time {
		return now
	}

	session, appErr := client.ExchangeAuthorizationCode(context.Background(), "notion_public_workspace_a", ExecutionProfile{
		Platform: "notion",
		Subject:  "integration",
		Method:   "notion.oauth_public",
		Grant: ExecutionGrant{
			Type:         "oauth_refreshable",
			ClientID:     "env:NOTION_CLIENT_ID",
			ClientSecret: "env:NOTION_CLIENT_SECRET",
		},
	}, "oauth-code", "http://localhost:3333/callback")
	if appErr != nil {
		t.Fatalf("ExchangeAuthorizationCode returned error: %+v", appErr)
	}
	if session.AccessToken != "fresh-access-token" {
		t.Fatalf("unexpected access token: %s", session.AccessToken)
	}
	if session.Metadata["workspace_id"] != "workspace_123" {
		t.Fatalf("unexpected workspace metadata: %+v", session.Metadata)
	}
	if session.Metadata["bot_id"] != "bot_123" {
		t.Fatalf("unexpected bot metadata: %+v", session.Metadata)
	}

	saved, err := sessionStore.Load("notion_public_workspace_a")
	if err != nil {
		t.Fatalf("failed to load saved session: %v", err)
	}
	if saved.AccessToken != "fresh-access-token" {
		t.Fatalf("unexpected saved access token: %s", saved.AccessToken)
	}
	if saved.Metadata["workspace_name"] != "Workspace Name" {
		t.Fatalf("unexpected saved workspace metadata: %+v", saved.Metadata)
	}
}

func TestExchangeAuthorizationCodeRejectsUnsupportedProfiles(t *testing.T) {
	// 授权码入口和刷新入口一样，需要在 wrapper 层直接拦住非法 profile。
	client := newTestClient(t, nil)

	_, appErr := client.ExchangeAuthorizationCode(context.Background(), "notion_public_workspace_a", ExecutionProfile{
		Platform: "notion",
		Subject:  "user",
		Method:   "notion.oauth_public",
		Grant: ExecutionGrant{
			Type: "oauth_refreshable",
		},
	}, "oauth-code", "http://localhost:3333/callback")
	if appErr == nil || appErr.Code != "SUBJECT_NOT_ALLOWED" {
		t.Fatalf("expected SUBJECT_NOT_ALLOWED, got: %+v", appErr)
	}

	_, appErr = client.ExchangeAuthorizationCode(context.Background(), "notion_public_workspace_a", ExecutionProfile{
		Platform: "notion",
		Subject:  "integration",
		Method:   "notion.internal_token",
		Grant: ExecutionGrant{
			Type:  "static_token",
			Token: "env:NOTION_ACCESS_TOKEN",
		},
	}, "oauth-code", "http://localhost:3333/callback")
	if appErr == nil || appErr.Code != "UNSUPPORTED_GRANT" {
		t.Fatalf("expected UNSUPPORTED_GRANT, got: %+v", appErr)
	}
}
