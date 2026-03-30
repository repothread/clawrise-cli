package notion

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
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

type authProvider struct {
	client *Client
}

// NewAuthProvider creates the Notion auth capability provider.
func NewAuthProvider(client *Client) pluginruntime.AuthProvider {
	return &authProvider{client: client}
}

func (p *authProvider) ListMethods(ctx context.Context) ([]pluginruntime.AuthMethodDescriptor, error) {
	return []pluginruntime.AuthMethodDescriptor{
		{
			ID:          "notion.internal_token",
			Platform:    "notion",
			DisplayName: "Notion Internal Integration Token",
			Subjects:    []string{"integration"},
			Kind:        "machine",
			PublicFields: []pluginruntime.AuthFieldDescriptor{
				{Name: "notion_version", Required: false, Type: "string"},
			},
			SecretFields: []pluginruntime.AuthFieldDescriptor{
				{Name: "token", Required: true, Type: "string"},
			},
		},
		{
			ID:               "notion.oauth_public",
			Platform:         "notion",
			DisplayName:      "Notion Public OAuth",
			Subjects:         []string{"integration"},
			Kind:             "interactive",
			Interactive:      true,
			InteractiveModes: []string{"local_browser", "manual_code"},
			PublicFields: []pluginruntime.AuthFieldDescriptor{
				{Name: "client_id", Required: true, Type: "string"},
				{Name: "notion_version", Required: false, Type: "string"},
				{Name: "redirect_mode", Required: false, Type: "string"},
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
			ID:                 "internal_token",
			Platform:           "notion",
			DisplayName:        "Notion Internal Token",
			Description:        "Use a workspace integration token to call the Notion API directly.",
			Subject:            "integration",
			AuthMethod:         "notion.internal_token",
			DefaultAccountName: "notion_integration_default",
			Public: map[string]any{
				"notion_version": "2026-03-11",
			},
			SecretFields: []string{"token"},
		},
		{
			ID:                 "public_oauth",
			Platform:           "notion",
			DisplayName:        "Notion Public OAuth",
			Description:        "Complete Notion public integration authorization in a browser.",
			Subject:            "integration",
			AuthMethod:         "notion.oauth_public",
			DefaultAccountName: "notion_public_default",
			Public: map[string]any{
				"client_id":      "",
				"notion_version": "2026-03-11",
				"redirect_mode":  "loopback",
			},
			SecretFields: []string{"client_secret", "refresh_token"},
		},
	}, nil
}

func (p *authProvider) Inspect(ctx context.Context, params pluginruntime.AuthInspectParams) (pluginruntime.AuthInspectResult, error) {
	account := params.Account
	switch account.AuthMethod {
	case "notion.internal_token":
		missingSecrets := missingNotionSecretFields(account, "token")
		if len(missingSecrets) > 0 {
			return pluginruntime.AuthInspectResult{
				Ready:               false,
				Status:              "invalid_auth_config",
				Message:             "The account configuration is missing required fields.",
				MissingSecretFields: missingSecrets,
			}, nil
		}
		return pluginruntime.AuthInspectResult{
			Ready:  true,
			Status: "ready",
		}, nil
	case "notion.oauth_public":
		missingPublic := missingNotionPublicFields(account, "client_id")
		missingSecrets := missingNotionSecretFields(account, "client_secret")
		if len(missingPublic) > 0 || len(missingSecrets) > 0 {
			return pluginruntime.AuthInspectResult{
				Ready:               false,
				Status:              "invalid_auth_config",
				Message:             "The account configuration is missing required fields.",
				MissingPublicFields: missingPublic,
				MissingSecretFields: missingSecrets,
			}, nil
		}

		session := notionSessionFromAccount(account)
		if session != nil && session.UsableAt(time.Now().UTC(), authcache.DefaultRefreshSkew) {
			return pluginruntime.AuthInspectResult{
				Ready:         true,
				Status:        "ready",
				SessionStatus: "session_valid",
			}, nil
		}
		if canRefreshNotionAccount(account, session) {
			return pluginruntime.AuthInspectResult{
				Ready:         true,
				Status:        "ready",
				SessionStatus: "refresh_required",
			}, nil
		}
		return pluginruntime.AuthInspectResult{
			Ready:             false,
			Status:            "authorization_required",
			Message:           "Interactive Notion authorization is required before execution.",
			SessionStatus:     "missing",
			HumanRequired:     true,
			RecommendedAction: "auth.login",
			NextActions: []pluginruntime.AuthAction{
				{Type: "auth_login", Message: "Run `clawrise auth login <account>` to complete authorization."},
			},
		}, nil
	default:
		return pluginruntime.AuthInspectResult{
			Ready:   false,
			Status:  "unsupported_auth_method",
			Message: fmt.Sprintf("unsupported Notion auth method: %s", account.AuthMethod),
		}, nil
	}
}

func (p *authProvider) Begin(ctx context.Context, params pluginruntime.AuthBeginParams) (pluginruntime.AuthBeginResult, error) {
	account := params.Account
	if account.AuthMethod != "notion.oauth_public" {
		return pluginruntime.AuthBeginResult{}, fmt.Errorf("auth method %s does not support interactive login", account.AuthMethod)
	}
	if items := missingNotionPublicFields(account, "client_id"); len(items) > 0 {
		return pluginruntime.AuthBeginResult{}, fmt.Errorf("missing public field: %s", items[0])
	}
	if items := missingNotionSecretFields(account, "client_secret"); len(items) > 0 {
		return pluginruntime.AuthBeginResult{}, fmt.Errorf("missing secret field: %s", items[0])
	}

	mode := strings.TrimSpace(params.Mode)
	if mode == "" {
		mode = "local_browser"
	}
	redirectURI := strings.TrimSpace(params.RedirectURI)
	if redirectURI == "" && strings.EqualFold(notionPublicString(account, "redirect_mode"), "loopback") {
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
		return pluginruntime.AuthBeginResult{}, fmt.Errorf("redirect_uri is required for notion.oauth_public")
	}

	endpoint, _ := url.Parse("https://api.notion.com/v1/oauth/authorize")
	query := endpoint.Query()
	query.Set("client_id", notionPublicString(account, "client_id"))
	query.Set("response_type", "code")
	query.Set("owner", "user")
	query.Set("redirect_uri", redirectURI)
	query.Set("state", randomNotionToken(16))
	endpoint.RawQuery = query.Encode()

	return pluginruntime.AuthBeginResult{
		HumanRequired: true,
		Flow: pluginruntime.AuthFlowPayload{
			ID:               "flow_" + randomNotionToken(8),
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
	}, nil
}

func (p *authProvider) Complete(ctx context.Context, params pluginruntime.AuthCompleteParams) (pluginruntime.AuthCompleteResult, error) {
	account := params.Account
	if account.AuthMethod != "notion.oauth_public" {
		return pluginruntime.AuthCompleteResult{}, fmt.Errorf("auth method %s does not support interactive completion", account.AuthMethod)
	}

	code, err := extractNotionOAuthCode(params.Flow, params.Code, params.CallbackURL)
	if err != nil {
		return pluginruntime.AuthCompleteResult{}, err
	}

	profile := buildNotionProfile(account)
	session, appErr := p.client.exchangeAuthorizationCode(adapter.WithAccountName(ctx, account.Name), profile, code, params.Flow.RedirectURI)
	if appErr != nil {
		return pluginruntime.AuthCompleteResult{}, fmt.Errorf("%s", appErr.Message)
	}

	result := pluginruntime.AuthCompleteResult{
		Ready:         true,
		Status:        "ready",
		ExecutionAuth: buildNotionExecutionAuth(strings.TrimSpace(session.AccessToken), resolveNotionVersion(profile)),
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
	case "notion.internal_token":
		token := strings.TrimSpace(account.Secrets["token"])
		if token == "" {
			return pluginruntime.AuthResolveResult{
				Ready:   false,
				Status:  "invalid_auth_config",
				Message: "missing token",
			}, nil
		}
		return pluginruntime.AuthResolveResult{
			Ready:         true,
			Status:        "ready",
			ExecutionAuth: buildNotionExecutionAuth(token, notionPublicString(account, "notion_version")),
		}, nil
	case "notion.oauth_public":
		profile := buildNotionProfile(account)
		session := notionSessionFromAccount(account)
		if session != nil && session.UsableAt(time.Now().UTC(), authcache.DefaultRefreshSkew) {
			return pluginruntime.AuthResolveResult{
				Ready:         true,
				Status:        "ready",
				ExecutionAuth: buildNotionExecutionAuth(strings.TrimSpace(session.AccessToken), resolveNotionVersion(profile)),
			}, nil
		}

		ctx = adapter.WithAccountName(ctx, account.Name)
		refreshedSession, appErr := p.client.refreshAccessToken(ctx, profile, session)
		if appErr == nil {
			result := pluginruntime.AuthResolveResult{
				Ready:         true,
				Status:        "ready",
				ExecutionAuth: buildNotionExecutionAuth(strings.TrimSpace(refreshedSession.AccessToken), resolveNotionVersion(profile)),
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
			Message: fmt.Sprintf("unsupported Notion auth method: %s", account.AuthMethod),
		}, nil
	}
}

func buildNotionProfile(account pluginruntime.AuthAccount) ExecutionProfile {
	profile := ExecutionProfile{
		Platform: "notion",
		Subject:  account.Subject,
		Method:   account.AuthMethod,
	}
	switch account.AuthMethod {
	case "notion.internal_token":
		profile.Params.NotionVersion = notionPublicString(account, "notion_version")
		profile.Grant = ExecutionGrant{
			Type:      "static_token",
			Token:     strings.TrimSpace(account.Secrets["token"]),
			NotionVer: profile.Params.NotionVersion,
		}
	case "notion.oauth_public":
		profile.Params.ClientID = notionPublicString(account, "client_id")
		profile.Params.NotionVersion = notionPublicString(account, "notion_version")
		profile.Params.RedirectMode = notionPublicString(account, "redirect_mode")
		profile.Grant = ExecutionGrant{
			Type:         "oauth_refreshable",
			ClientID:     profile.Params.ClientID,
			ClientSecret: strings.TrimSpace(account.Secrets["client_secret"]),
			AccessToken:  strings.TrimSpace(account.Secrets["access_token"]),
			RefreshToken: strings.TrimSpace(account.Secrets["refresh_token"]),
			NotionVer:    profile.Params.NotionVersion,
		}
	}
	return profile
}

func executionProfileFromCall(call adapter.Call) ExecutionProfile {
	return buildNotionExecutionProfileFromIdentity(call.Identity)
}

func buildNotionExecutionProfileFromIdentity(identity adapter.Identity) ExecutionProfile {
	profile := ExecutionProfile{
		Platform: strings.TrimSpace(identity.Platform),
		Subject:  strings.TrimSpace(identity.Subject),
		Method:   strings.TrimSpace(identity.AuthMethod),
	}

	if accessToken := notionExecutionAuthString(identity.ExecutionAuth, "access_token"); accessToken != "" {
		profile.Grant = ExecutionGrant{
			Type:        notionFirstNonEmptyExecutionAuthType(identity.ExecutionAuth, "resolved_access_token"),
			AccessToken: accessToken,
			NotionVer:   notionExecutionAuthString(identity.ExecutionAuth, "notion_version"),
		}
		return profile
	}

	switch profile.Method {
	case "notion.internal_token":
		profile.Params.NotionVersion = notionIdentityPublicString(identity, "notion_version")
		profile.Grant = ExecutionGrant{
			Type:      "static_token",
			Token:     notionIdentitySecretString(identity, "token"),
			NotionVer: profile.Params.NotionVersion,
		}
	case "notion.oauth_public":
		profile.Params.ClientID = notionIdentityPublicString(identity, "client_id")
		profile.Params.NotionVersion = notionIdentityPublicString(identity, "notion_version")
		profile.Params.RedirectMode = notionIdentityPublicString(identity, "redirect_mode")
		profile.Grant = ExecutionGrant{
			Type:         "oauth_refreshable",
			ClientID:     profile.Params.ClientID,
			ClientSecret: notionIdentitySecretString(identity, "client_secret"),
			AccessToken:  notionIdentitySessionString(identity, "access_token"),
			RefreshToken: notionFirstNonEmpty(notionIdentitySessionString(identity, "refresh_token"), notionIdentitySecretString(identity, "refresh_token")),
			NotionVer:    profile.Params.NotionVersion,
		}
	default:
		profile.Grant.Type = notionGrantTypeForMethod(profile.Method)
	}
	return profile
}

func notionSessionFromAccount(account pluginruntime.AuthAccount) *authcache.Session {
	if account.Session == nil {
		return nil
	}
	session := account.Session.ToSession()
	session.AccountName = account.Name
	session.Platform = account.Platform
	session.Subject = account.Subject
	session.GrantType = account.AuthMethod
	return &session
}

func buildNotionExecutionAuth(accessToken string, notionVersion string) map[string]any {
	return map[string]any{
		"type":           "resolved_access_token",
		"access_token":   strings.TrimSpace(accessToken),
		"notion_version": strings.TrimSpace(notionVersion),
	}
}

func notionPublicString(account pluginruntime.AuthAccount, field string) string {
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

func missingNotionPublicFields(account pluginruntime.AuthAccount, fields ...string) []string {
	items := make([]string, 0)
	for _, field := range fields {
		if notionPublicString(account, field) == "" {
			items = append(items, field)
		}
	}
	return items
}

func missingNotionSecretFields(account pluginruntime.AuthAccount, fields ...string) []string {
	items := make([]string, 0)
	for _, field := range fields {
		if strings.TrimSpace(account.Secrets[field]) == "" {
			items = append(items, field)
		}
	}
	return items
}

func notionIdentityPublicString(identity adapter.Identity, field string) string {
	if identity.Public == nil {
		return ""
	}
	value, ok := identity.Public[field]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func notionIdentitySecretString(identity adapter.Identity, field string) string {
	if identity.Secrets == nil {
		return ""
	}
	return strings.TrimSpace(identity.Secrets[field])
}

func notionIdentitySessionString(identity adapter.Identity, field string) string {
	if identity.Session == nil {
		return ""
	}
	switch strings.TrimSpace(field) {
	case "access_token":
		return strings.TrimSpace(identity.Session.AccessToken)
	case "refresh_token":
		return strings.TrimSpace(identity.Session.RefreshToken)
	default:
		return ""
	}
}

func notionExecutionAuthString(values map[string]any, field string) string {
	text, _ := values[field].(string)
	return strings.TrimSpace(text)
}

func notionFirstNonEmptyExecutionAuthType(values map[string]any, fallback string) string {
	if value := notionExecutionAuthString(values, "type"); value != "" {
		return value
	}
	return strings.TrimSpace(fallback)
}

func notionFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func notionGrantTypeForMethod(method string) string {
	switch strings.TrimSpace(method) {
	case "notion.internal_token":
		return "static_token"
	case "notion.oauth_public":
		return "oauth_refreshable"
	default:
		return ""
	}
}

func canRefreshNotionAccount(account pluginruntime.AuthAccount, session *authcache.Session) bool {
	if session != nil && strings.TrimSpace(session.RefreshToken) != "" {
		return true
	}
	return strings.TrimSpace(account.Secrets["refresh_token"]) != ""
}

func extractNotionOAuthCode(flow pluginruntime.AuthFlowPayload, code string, callbackURL string) (string, error) {
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

func randomNotionToken(size int) string {
	if size <= 0 {
		size = 8
	}
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
