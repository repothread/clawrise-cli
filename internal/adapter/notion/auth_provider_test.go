package notion

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

func TestAuthProviderListMethodsAndPresets(t *testing.T) {
	// 先验证对外暴露的授权方法和预设是否完整，避免 capability 元数据回归。
	provider := NewAuthProvider(newTestClient(t, nil))

	methods, err := provider.ListMethods(context.Background())
	if err != nil {
		t.Fatalf("ListMethods returned error: %v", err)
	}
	if len(methods) != 2 {
		t.Fatalf("unexpected method count: %d", len(methods))
	}
	if methods[0].ID != "notion.internal_token" || methods[1].ID != "notion.oauth_public" {
		t.Fatalf("unexpected methods: %+v", methods)
	}

	presets, err := provider.ListPresets(context.Background())
	if err != nil {
		t.Fatalf("ListPresets returned error: %v", err)
	}
	if len(presets) != 2 {
		t.Fatalf("unexpected preset count: %d", len(presets))
	}
	if presets[0].ID != "internal_token" || presets[1].ID != "public_oauth" {
		t.Fatalf("unexpected presets: %+v", presets)
	}
}

func TestAuthProviderInspectInternalTokenRequiresToken(t *testing.T) {
	// 内部 token 模式最基础的失败路径必须被覆盖，否则配置错误会被误判为可执行。
	provider := NewAuthProvider(newTestClient(t, nil))

	result, err := provider.Inspect(context.Background(), pluginruntime.AuthInspectParams{
		Account: pluginruntime.AuthAccount{
			Name:       "notion_ci",
			Platform:   "notion",
			Subject:    "integration",
			AuthMethod: "notion.internal_token",
		},
	})
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if result.Ready {
		t.Fatalf("expected account to be not ready: %+v", result)
	}
	if result.Status != "invalid_auth_config" {
		t.Fatalf("unexpected status: %+v", result)
	}
	if len(result.MissingSecretFields) != 1 || result.MissingSecretFields[0] != "token" {
		t.Fatalf("unexpected missing secrets: %+v", result.MissingSecretFields)
	}
}

func TestAuthProviderInspectOAuthStates(t *testing.T) {
	// 这里一次覆盖三种状态：已有可用 session、可刷新、以及必须人工授权。
	provider := NewAuthProvider(newTestClient(t, nil))

	validSessionResult, err := provider.Inspect(context.Background(), pluginruntime.AuthInspectParams{
		Account: pluginruntime.AuthAccount{
			Name:       "notion_public_valid",
			Platform:   "notion",
			Subject:    "integration",
			AuthMethod: "notion.oauth_public",
			Public: map[string]any{
				"client_id":      "client-id",
				"notion_version": "2026-03-11",
			},
			Secrets: map[string]string{
				"client_secret": "client-secret",
			},
			Session: &pluginruntime.AuthSessionPayload{
				AccessToken: "session-token",
				ExpiresAt:   time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
			},
		},
	})
	if err != nil {
		t.Fatalf("Inspect returned error for valid session: %v", err)
	}
	if !validSessionResult.Ready || validSessionResult.SessionStatus != "session_valid" {
		t.Fatalf("unexpected valid session inspection result: %+v", validSessionResult)
	}

	refreshRequiredResult, err := provider.Inspect(context.Background(), pluginruntime.AuthInspectParams{
		Account: pluginruntime.AuthAccount{
			Name:       "notion_public_refresh",
			Platform:   "notion",
			Subject:    "integration",
			AuthMethod: "notion.oauth_public",
			Public: map[string]any{
				"client_id":      "client-id",
				"notion_version": "2026-03-11",
			},
			Secrets: map[string]string{
				"client_secret": "client-secret",
				"refresh_token": "refresh-token",
			},
		},
	})
	if err != nil {
		t.Fatalf("Inspect returned error for refreshable account: %v", err)
	}
	if !refreshRequiredResult.Ready || refreshRequiredResult.SessionStatus != "refresh_required" {
		t.Fatalf("unexpected refresh inspection result: %+v", refreshRequiredResult)
	}

	authorizationRequiredResult, err := provider.Inspect(context.Background(), pluginruntime.AuthInspectParams{
		Account: pluginruntime.AuthAccount{
			Name:       "notion_public_missing",
			Platform:   "notion",
			Subject:    "integration",
			AuthMethod: "notion.oauth_public",
			Public: map[string]any{
				"client_id":      "client-id",
				"notion_version": "2026-03-11",
			},
			Secrets: map[string]string{
				"client_secret": "client-secret",
			},
		},
	})
	if err != nil {
		t.Fatalf("Inspect returned error for missing authorization: %v", err)
	}
	if authorizationRequiredResult.Ready || authorizationRequiredResult.Status != "authorization_required" {
		t.Fatalf("unexpected authorization-required result: %+v", authorizationRequiredResult)
	}
	if authorizationRequiredResult.RecommendedAction != "auth.login" {
		t.Fatalf("unexpected recommended action: %+v", authorizationRequiredResult)
	}
}

func TestAuthProviderBeginBuildsLoopbackAuthorizationURL(t *testing.T) {
	// Begin 阶段要确保默认回调地址和 OAuth 参数正确拼装，否则 GitHub Action 里的真实登录无法复用。
	provider := NewAuthProvider(newTestClient(t, nil))
	account := testOAuthAccount()

	result, err := provider.Begin(context.Background(), pluginruntime.AuthBeginParams{
		Account:      account,
		CallbackHost: "127.0.0.1",
		CallbackPath: "/done",
	})
	if err != nil {
		t.Fatalf("Begin returned error: %v", err)
	}
	if !result.HumanRequired {
		t.Fatalf("expected Begin to require user action: %+v", result)
	}
	if result.Flow.RedirectURI != "http://127.0.0.1:3333/done" {
		t.Fatalf("unexpected redirect uri: %s", result.Flow.RedirectURI)
	}

	parsed, err := url.Parse(result.Flow.AuthorizationURL)
	if err != nil {
		t.Fatalf("failed to parse authorization url: %v", err)
	}
	query := parsed.Query()
	if query.Get("client_id") != "client-id" {
		t.Fatalf("unexpected client_id in authorization url: %s", result.Flow.AuthorizationURL)
	}
	if query.Get("redirect_uri") != result.Flow.RedirectURI {
		t.Fatalf("unexpected redirect_uri in authorization url: %s", result.Flow.AuthorizationURL)
	}
	if strings.TrimSpace(result.Flow.Metadata["oauth_state"]) == "" {
		t.Fatalf("expected oauth_state to be present: %+v", result.Flow)
	}
}

func TestAuthProviderBeginRequiresRedirectURIOutsideLoopbackMode(t *testing.T) {
	provider := NewAuthProvider(newTestClient(t, nil))
	account := testOAuthAccount()
	account.Public["redirect_mode"] = "manual_code"

	_, err := provider.Begin(context.Background(), pluginruntime.AuthBeginParams{
		Account: account,
	})
	if err == nil {
		t.Fatal("expected Begin to require redirect_uri outside loopback mode")
	}
	if !strings.Contains(err.Error(), "redirect_uri is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthProviderCompleteExchangesAuthorizationCodeFromCallbackURL(t *testing.T) {
	transport := &roundTripFunc{
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
				t.Fatalf("failed to decode oauth completion payload: %v", err)
			}
			if payload["grant_type"] != "authorization_code" {
				t.Fatalf("unexpected grant_type: %+v", payload["grant_type"])
			}
			if payload["code"] != "oauth-code" {
				t.Fatalf("unexpected authorization code: %+v", payload["code"])
			}
			if payload["redirect_uri"] != "http://localhost:3333/callback" {
				t.Fatalf("unexpected redirect_uri: %+v", payload["redirect_uri"])
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"access_token":   "fresh-access-token",
				"refresh_token":  "fresh-refresh-token",
				"token_type":     "bearer",
				"workspace_id":   "workspace_123",
				"workspace_name": "Workspace Name",
				"bot_id":         "bot_123",
				"expires_in":     3600,
			}), nil
		},
	}

	provider := NewAuthProvider(newTestClient(t, transport))
	account := testOAuthAccount()
	account.Name = "notion_public_demo"

	result, err := provider.Complete(context.Background(), pluginruntime.AuthCompleteParams{
		Account: account,
		Flow: pluginruntime.AuthFlowPayload{
			RedirectURI: "http://localhost:3333/callback",
			Metadata: map[string]string{
				"oauth_state": "state_demo",
			},
		},
		CallbackURL: "http://localhost:3333/callback?code=oauth-code&state=state_demo",
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if !result.Ready || result.Status != "ready" {
		t.Fatalf("unexpected complete result: %+v", result)
	}
	if result.ExecutionAuth["access_token"] != "fresh-access-token" {
		t.Fatalf("unexpected execution auth: %+v", result.ExecutionAuth)
	}
	if result.SecretPatches["refresh_token"] != "fresh-refresh-token" {
		t.Fatalf("unexpected secret patches: %+v", result.SecretPatches)
	}
	if result.SessionPatch == nil || result.SessionPatch.AccessToken != "fresh-access-token" {
		t.Fatalf("unexpected session patch: %+v", result.SessionPatch)
	}
}

func TestAuthProviderResolveOAuthRefreshesSession(t *testing.T) {
	transport := &roundTripFunc{
		handler: func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/v1/oauth/token" {
				t.Fatalf("unexpected request path: %s", request.URL.Path)
			}

			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("failed to decode refresh payload: %v", err)
			}
			if payload["grant_type"] != "refresh_token" || payload["refresh_token"] != "refresh-token" {
				t.Fatalf("unexpected refresh payload: %+v", payload)
			}

			return jsonResponse(t, http.StatusOK, map[string]any{
				"access_token":  "resolved-access-token",
				"refresh_token": "resolved-refresh-token",
				"token_type":    "bearer",
				"expires_in":    3600,
			}), nil
		},
	}

	provider := NewAuthProvider(newTestClient(t, transport))
	account := testOAuthAccount()
	account.Name = "notion_public_refreshable"
	account.Secrets["refresh_token"] = "refresh-token"

	result, err := provider.Resolve(context.Background(), pluginruntime.AuthResolveParams{
		Account: account,
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if !result.Ready || result.Status != "ready" {
		t.Fatalf("unexpected resolve result: %+v", result)
	}
	if result.ExecutionAuth["access_token"] != "resolved-access-token" {
		t.Fatalf("unexpected execution auth: %+v", result.ExecutionAuth)
	}
	if result.SessionPatch == nil || result.SessionPatch.RefreshToken != "resolved-refresh-token" {
		t.Fatalf("unexpected session patch: %+v", result.SessionPatch)
	}
	if result.SecretPatches["refresh_token"] != "resolved-refresh-token" {
		t.Fatalf("unexpected secret patches: %+v", result.SecretPatches)
	}
}

func TestBuildNotionExecutionProfileFromIdentity(t *testing.T) {
	// 先覆盖 execution_auth 直传，再覆盖 session/secret 混合回退，避免 runtime 桥接时丢字段。
	resolvedProfile := buildNotionExecutionProfileFromIdentity(adapter.Identity{
		Platform:   "notion",
		Subject:    "integration",
		AuthMethod: "notion.oauth_public",
		ExecutionAuth: map[string]any{
			"access_token":   "resolved-token",
			"notion_version": "2026-03-11",
		},
	})
	if resolvedProfile.Grant.Type != "resolved_access_token" {
		t.Fatalf("unexpected grant type for resolved profile: %+v", resolvedProfile)
	}
	if resolvedProfile.Grant.AccessToken != "resolved-token" {
		t.Fatalf("unexpected resolved access token: %+v", resolvedProfile)
	}

	oauthProfile := buildNotionExecutionProfileFromIdentity(adapter.Identity{
		Platform:   "notion",
		Subject:    "integration",
		AuthMethod: "notion.oauth_public",
		Public: map[string]any{
			"client_id":      "client-id",
			"notion_version": "2026-03-11",
			"redirect_mode":  "loopback",
		},
		Secrets: map[string]string{
			"client_secret": "client-secret",
			"refresh_token": "secret-refresh-token",
		},
		Session: &authcache.Session{
			AccessToken: "session-access-token",
		},
	})
	if oauthProfile.Grant.Type != "oauth_refreshable" {
		t.Fatalf("unexpected grant type for oauth profile: %+v", oauthProfile)
	}
	if oauthProfile.Grant.AccessToken != "session-access-token" {
		t.Fatalf("unexpected access token from session: %+v", oauthProfile)
	}
	if oauthProfile.Grant.RefreshToken != "secret-refresh-token" {
		t.Fatalf("unexpected refresh token fallback: %+v", oauthProfile)
	}
}

func TestExtractNotionOAuthCodeValidation(t *testing.T) {
	flow := pluginruntime.AuthFlowPayload{
		Metadata: map[string]string{
			"oauth_state": "state_ok",
		},
	}

	if _, err := extractNotionOAuthCode(flow, "", "http://localhost/callback?error=access_denied"); err == nil {
		t.Fatal("expected callback error to be surfaced")
	}
	if _, err := extractNotionOAuthCode(flow, "", "http://localhost/callback?state=wrong&code=oauth-code"); err == nil {
		t.Fatal("expected state mismatch to be rejected")
	}

	code, err := extractNotionOAuthCode(flow, "", "http://localhost/callback?state=state_ok&code=oauth-code")
	if err != nil {
		t.Fatalf("extractNotionOAuthCode returned error: %v", err)
	}
	if code != "oauth-code" {
		t.Fatalf("unexpected code: %s", code)
	}
}

func testOAuthAccount() pluginruntime.AuthAccount {
	return pluginruntime.AuthAccount{
		Name:       "notion_public_default",
		Platform:   "notion",
		Subject:    "integration",
		AuthMethod: "notion.oauth_public",
		Public: map[string]any{
			"client_id":      "client-id",
			"notion_version": "2026-03-11",
			"redirect_mode":  "loopback",
		},
		Secrets: map[string]string{
			"client_secret": "client-secret",
		},
	}
}
