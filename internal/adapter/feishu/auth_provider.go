package feishu

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/config"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

type authProvider struct {
	client *Client
}

// NewAuthProvider creates the Feishu auth capability provider.
func NewAuthProvider(client *Client) pluginruntime.AuthProvider {
	return &authProvider{client: client}
}

func (p *authProvider) ListMethods(ctx context.Context) ([]pluginruntime.AuthMethodDescriptor, error) {
	return []pluginruntime.AuthMethodDescriptor{
		{
			ID:          "feishu.app_credentials",
			Platform:    "feishu",
			DisplayName: "Feishu Bot App Credentials",
			Subjects:    []string{"bot"},
			Kind:        "machine",
			PublicFields: []pluginruntime.AuthFieldDescriptor{
				{Name: "app_id", Required: true, Type: "string"},
			},
			SecretFields: []pluginruntime.AuthFieldDescriptor{
				{Name: "app_secret", Required: true, Type: "string"},
			},
		},
		{
			ID:               "feishu.oauth_user",
			Platform:         "feishu",
			DisplayName:      "Feishu User OAuth",
			Subjects:         []string{"user"},
			Kind:             "interactive",
			Interactive:      true,
			InteractiveModes: []string{"local_browser", "manual_code"},
			PublicFields: []pluginruntime.AuthFieldDescriptor{
				{Name: "client_id", Required: true, Type: "string"},
				{Name: "redirect_mode", Required: false, Type: "string"},
				{Name: "scopes", Required: false, Type: "string_list"},
			},
			SecretFields: []pluginruntime.AuthFieldDescriptor{
				{Name: "client_secret", Required: true, Type: "string"},
			},
		},
	}, nil
}

func (p *authProvider) ListPresets(ctx context.Context) ([]pluginruntime.AuthPresetDescriptor, error) {
	return []pluginruntime.AuthPresetDescriptor{
		{
			ID:                 "bot",
			Platform:           "feishu",
			DisplayName:        "Feishu Bot",
			Description:        "Use app_id and app_secret to obtain a tenant_access_token.",
			Subject:            "bot",
			AuthMethod:         "feishu.app_credentials",
			DefaultAccountName: "feishu_bot_default",
			Public:             map[string]any{"app_id": ""},
			SecretFields:       []string{"app_secret"},
		},
		{
			ID:                 "user",
			Platform:           "feishu",
			DisplayName:        "Feishu User OAuth",
			Description:        "Complete user authorization in a browser and store the session locally.",
			Subject:            "user",
			AuthMethod:         "feishu.oauth_user",
			DefaultAccountName: "feishu_user_default",
			Public: map[string]any{
				"client_id":     "",
				"redirect_mode": "loopback",
				"scopes":        []string{"offline_access"},
			},
			SecretFields: []string{"client_secret", "refresh_token"},
		},
	}, nil
}

func (p *authProvider) Inspect(ctx context.Context, params pluginruntime.AuthInspectParams) (pluginruntime.AuthInspectResult, error) {
	account := params.Account
	switch account.AuthMethod {
	case "feishu.app_credentials":
		missingPublic := missingPublicFields(account, "app_id")
		missingSecrets := missingSecretFields(account, "app_secret")
		return buildInspectResult(missingPublic, missingSecrets, "", false), nil
	case "feishu.oauth_user":
		missingPublic := missingPublicFields(account, "client_id")
		missingSecrets := missingSecretFields(account, "client_secret")
		if len(missingPublic) > 0 || len(missingSecrets) > 0 {
			return buildInspectResult(missingPublic, missingSecrets, "", false), nil
		}

		session := authSessionFromAccount(account)
		if session != nil && session.UsableAt(time.Now().UTC(), authcache.DefaultRefreshSkew) {
			return pluginruntime.AuthInspectResult{
				Ready:         true,
				Status:        "ready",
				SessionStatus: "session_valid",
			}, nil
		}
		if canRefreshFeishuAccount(account, session) {
			return pluginruntime.AuthInspectResult{
				Ready:         true,
				Status:        "ready",
				SessionStatus: "refresh_required",
			}, nil
		}
		return pluginruntime.AuthInspectResult{
			Ready:             false,
			Status:            "authorization_required",
			Message:           "Interactive Feishu user authorization is required before execution.",
			SessionStatus:     "missing",
			HumanRequired:     true,
			RecommendedAction: "auth.login",
			NextActions: []pluginruntime.AuthAction{
				{Type: "auth_login", Message: "Run `clawrise auth login <account>` to complete user authorization."},
			},
		}, nil
	default:
		return pluginruntime.AuthInspectResult{
			Ready:   false,
			Status:  "unsupported_auth_method",
			Message: fmt.Sprintf("unsupported Feishu auth method: %s", account.AuthMethod),
		}, nil
	}
}

func (p *authProvider) Begin(ctx context.Context, params pluginruntime.AuthBeginParams) (pluginruntime.AuthBeginResult, error) {
	account := params.Account
	if account.AuthMethod != "feishu.oauth_user" {
		return pluginruntime.AuthBeginResult{}, fmt.Errorf("auth method %s does not support interactive login", account.AuthMethod)
	}
	if items := missingPublicFields(account, "client_id"); len(items) > 0 {
		return pluginruntime.AuthBeginResult{}, fmt.Errorf("missing public field: %s", items[0])
	}
	if items := missingSecretFields(account, "client_secret"); len(items) > 0 {
		return pluginruntime.AuthBeginResult{}, fmt.Errorf("missing secret field: %s", items[0])
	}

	mode := strings.TrimSpace(params.Mode)
	if mode == "" {
		mode = "local_browser"
	}

	redirectURI := strings.TrimSpace(params.RedirectURI)
	if redirectURI == "" && strings.EqualFold(accountPublicString(account, "redirect_mode"), "loopback") {
		host := strings.TrimSpace(params.CallbackHost)
		if host == "" {
			host = "localhost"
		}
		path := strings.TrimSpace(params.CallbackPath)
		if path == "" {
			path = "/callback"
		}
		redirectURI = fmt.Sprintf("http://%s:3333%s", host, path)
	}
	if redirectURI == "" {
		return pluginruntime.AuthBeginResult{}, fmt.Errorf("redirect_uri is required for feishu.oauth_user")
	}

	endpoint, _ := url.Parse("https://accounts.feishu.cn/open-apis/authen/v1/authorize")
	query := endpoint.Query()
	query.Set("client_id", accountPublicString(account, "client_id"))
	query.Set("response_type", "code")
	query.Set("redirect_uri", redirectURI)
	query.Set("state", randomToken(16))
	query.Set("prompt", "consent")
	if scopes := accountPublicStringSlice(account, "scopes"); len(scopes) > 0 {
		query.Set("scope", strings.Join(scopes, " "))
	}
	endpoint.RawQuery = query.Encode()

	result := pluginruntime.AuthBeginResult{
		HumanRequired: true,
		Flow: pluginruntime.AuthFlowPayload{
			ID:               "flow_" + randomToken(8),
			Method:           account.AuthMethod,
			Mode:             mode,
			State:            "awaiting_user_action",
			RedirectURI:      redirectURI,
			AuthorizationURL: endpoint.String(),
			Metadata: map[string]string{
				"oauth_state": query.Get("state"),
			},
			ExpiresAt: time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339),
		},
		NextActions: []pluginruntime.AuthAction{
			{Type: "open_url", URL: endpoint.String(), Message: "Open the authorization URL in a browser."},
			{Type: "submit_callback_url", Field: "callback_url", Message: "Submit the full callback URL to `clawrise auth complete` after authorization."},
			{Type: "submit_code", Field: "code", Message: "If a callback URL is not available, submit the code directly."},
		},
	}
	return result, nil
}

func (p *authProvider) Complete(ctx context.Context, params pluginruntime.AuthCompleteParams) (pluginruntime.AuthCompleteResult, error) {
	account := params.Account
	if account.AuthMethod != "feishu.oauth_user" {
		return pluginruntime.AuthCompleteResult{}, fmt.Errorf("auth method %s does not support interactive completion", account.AuthMethod)
	}

	code, err := extractOAuthCode(params.Flow, params.Code, params.CallbackURL)
	if err != nil {
		return pluginruntime.AuthCompleteResult{}, err
	}

	profile := buildFeishuProfile(account)
	session, appErr := p.client.exchangeAuthorizationCode(adapter.WithProfileName(ctx, account.Name), profile, code, params.Flow.RedirectURI)
	if appErr != nil {
		return pluginruntime.AuthCompleteResult{}, fmt.Errorf("%s", appErr.Message)
	}

	result := pluginruntime.AuthCompleteResult{
		Ready:         true,
		Status:        "ready",
		ExecutionAuth: buildResolvedExecutionAuth(strings.TrimSpace(session.AccessToken)),
		SessionPatch:  pluginruntime.AuthSessionPayloadFromSession(session),
	}
	if strings.TrimSpace(session.RefreshToken) != "" {
		result.SecretPatches = map[string]string{
			"refresh_token": strings.TrimSpace(session.RefreshToken),
		}
	}
	return result, nil
}

func (p *authProvider) Resolve(ctx context.Context, params pluginruntime.AuthResolveParams) (pluginruntime.AuthResolveResult, error) {
	account := params.Account
	switch account.AuthMethod {
	case "feishu.app_credentials":
		profile := buildFeishuProfile(account)
		accessToken, appErr := p.client.fetchTenantAccessToken(ctx, profile)
		if appErr != nil {
			return pluginruntime.AuthResolveResult{
				Ready:   false,
				Status:  "invalid_auth_config",
				Message: appErr.Message,
			}, nil
		}
		return pluginruntime.AuthResolveResult{
			Ready:         true,
			Status:        "ready",
			ExecutionAuth: buildResolvedExecutionAuth(accessToken),
		}, nil
	case "feishu.oauth_user":
		profile := buildFeishuProfile(account)
		session := authSessionFromAccount(account)
		if session != nil && session.UsableAt(time.Now().UTC(), authcache.DefaultRefreshSkew) {
			return pluginruntime.AuthResolveResult{
				Ready:         true,
				Status:        "ready",
				ExecutionAuth: buildResolvedExecutionAuth(strings.TrimSpace(session.AccessToken)),
			}, nil
		}

		ctx = adapter.WithProfileName(ctx, account.Name)
		refreshedSession, appErr := p.client.refreshUserAccessToken(ctx, profile, session)
		if appErr == nil {
			result := pluginruntime.AuthResolveResult{
				Ready:         true,
				Status:        "ready",
				ExecutionAuth: buildResolvedExecutionAuth(strings.TrimSpace(refreshedSession.AccessToken)),
				SessionPatch:  pluginruntime.AuthSessionPayloadFromSession(refreshedSession),
			}
			if strings.TrimSpace(refreshedSession.RefreshToken) != "" {
				result.SecretPatches = map[string]string{
					"refresh_token": strings.TrimSpace(refreshedSession.RefreshToken),
				}
			}
			return result, nil
		}

		inspection, _ := p.Inspect(ctx, pluginruntime.AuthInspectParams{Account: account})
		return pluginruntime.AuthResolveResult{
			Ready:             false,
			Status:            inspection.Status,
			Message:           appErr.Message,
			HumanRequired:     inspection.HumanRequired,
			RecommendedAction: inspection.RecommendedAction,
			NextActions:       inspection.NextActions,
		}, nil
	default:
		return pluginruntime.AuthResolveResult{
			Ready:   false,
			Status:  "unsupported_auth_method",
			Message: fmt.Sprintf("unsupported Feishu auth method: %s", account.AuthMethod),
		}, nil
	}
}

func buildFeishuProfile(account pluginruntime.AuthAccount) config.Profile {
	profile := config.Profile{
		Platform: "feishu",
		Subject:  account.Subject,
		Method:   account.AuthMethod,
	}
	switch account.AuthMethod {
	case "feishu.app_credentials":
		profile.Params.AppID = accountPublicString(account, "app_id")
		profile.Grant = config.Grant{
			Type:      "client_credentials",
			AppID:     profile.Params.AppID,
			AppSecret: strings.TrimSpace(account.Secrets["app_secret"]),
		}
	case "feishu.oauth_user":
		profile.Params.ClientID = accountPublicString(account, "client_id")
		profile.Params.RedirectMode = accountPublicString(account, "redirect_mode")
		profile.Params.Scopes = accountPublicStringSlice(account, "scopes")
		profile.Grant = config.Grant{
			Type:         "oauth_user",
			ClientID:     profile.Params.ClientID,
			ClientSecret: strings.TrimSpace(account.Secrets["client_secret"]),
			AccessToken:  strings.TrimSpace(account.Secrets["access_token"]),
			RefreshToken: strings.TrimSpace(account.Secrets["refresh_token"]),
		}
	}
	return profile
}

func buildResolvedExecutionAuth(accessToken string) map[string]any {
	return map[string]any{
		"type":         "resolved_access_token",
		"access_token": strings.TrimSpace(accessToken),
	}
}

func authSessionFromAccount(account pluginruntime.AuthAccount) *authcache.Session {
	if account.Session == nil {
		return nil
	}
	session := account.Session.ToSession()
	session.ProfileName = account.Name
	session.Platform = account.Platform
	session.Subject = account.Subject
	session.GrantType = account.AuthMethod
	return &session
}

func accountPublicString(account pluginruntime.AuthAccount, field string) string {
	value, ok := account.Public[field]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func accountPublicStringSlice(account pluginruntime.AuthAccount, field string) []string {
	value, ok := account.Public[field]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		items := make([]string, 0, len(typed))
		for _, raw := range typed {
			text, ok := raw.(string)
			if !ok {
				continue
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			items = append(items, text)
		}
		return items
	default:
		return nil
	}
}

func missingPublicFields(account pluginruntime.AuthAccount, fields ...string) []string {
	items := make([]string, 0)
	for _, field := range fields {
		if accountPublicString(account, field) == "" {
			items = append(items, field)
		}
	}
	return items
}

func missingSecretFields(account pluginruntime.AuthAccount, fields ...string) []string {
	items := make([]string, 0)
	for _, field := range fields {
		if strings.TrimSpace(account.Secrets[field]) == "" {
			items = append(items, field)
		}
	}
	return items
}

func buildInspectResult(missingPublic []string, missingSecrets []string, sessionStatus string, humanRequired bool) pluginruntime.AuthInspectResult {
	if len(missingPublic) == 0 && len(missingSecrets) == 0 {
		return pluginruntime.AuthInspectResult{
			Ready:         true,
			Status:        "ready",
			SessionStatus: sessionStatus,
		}
	}
	return pluginruntime.AuthInspectResult{
		Ready:               false,
		Status:              "invalid_auth_config",
		Message:             "The account configuration is missing required fields.",
		MissingPublicFields: missingPublic,
		MissingSecretFields: missingSecrets,
		HumanRequired:       humanRequired,
	}
}

func canRefreshFeishuAccount(account pluginruntime.AuthAccount, session *authcache.Session) bool {
	if session != nil && strings.TrimSpace(session.RefreshToken) != "" {
		return true
	}
	return strings.TrimSpace(account.Secrets["refresh_token"]) != ""
}

func extractOAuthCode(flow pluginruntime.AuthFlowPayload, code string, callbackURL string) (string, error) {
	code = strings.TrimSpace(code)
	callbackURL = strings.TrimSpace(callbackURL)
	if callbackURL != "" {
		parsed, err := url.Parse(callbackURL)
		if err != nil {
			return "", fmt.Errorf("failed to parse callback_url: %w", err)
		}
		query := parsed.Query()
		if returnedError := strings.TrimSpace(query.Get("error")); returnedError != "" {
			return "", fmt.Errorf("authorization callback returned error: %s", returnedError)
		}
		expectedState := strings.TrimSpace(flow.Metadata["oauth_state"])
		if expectedState != "" && strings.TrimSpace(query.Get("state")) != expectedState {
			return "", fmt.Errorf("callback state does not match the current auth flow")
		}
		code = strings.TrimSpace(query.Get("code"))
	}
	if code == "" {
		return "", fmt.Errorf("authorization code is required")
	}
	return code, nil
}

func randomToken(size int) string {
	if size <= 0 {
		size = 8
	}
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
