package config

import (
	"fmt"
	"sort"
	"strings"
)

const defaultNotionVersion = "2026-03-11"

// InitOptions describes the inputs accepted by `config init`.
type InitOptions struct {
	Platform string
	Subject  string
	Account  string
	Method   string
	Scopes   []string
}

// InitResult describes the generated config and setup hints.
type InitResult struct {
	Config         *Config  `json:"-"`
	AccountName    string   `json:"account_name"`
	Platform       string   `json:"platform"`
	Subject        string   `json:"subject"`
	Method         string   `json:"method"`
	SecretFields   []string `json:"secret_fields"`
	SessionBackend string   `json:"session_backend"`
	SecretBackend  string   `json:"secret_backend"`
}

// BuildInitConfig creates the minimal initial config skeleton.
func BuildInitConfig(opts InitOptions) (InitResult, error) {
	platform := strings.TrimSpace(opts.Platform)
	if platform == "" {
		platform = "feishu"
	}

	subject := strings.TrimSpace(opts.Subject)
	if subject == "" {
		subject = defaultSubjectForPlatform(platform)
	}

	method := strings.TrimSpace(opts.Method)
	if method == "" {
		method = defaultMethod(platform, subject)
	}
	if method == "" {
		return InitResult{}, fmt.Errorf("unsupported platform and subject combination: %s/%s", platform, subject)
	}

	accountName := strings.TrimSpace(opts.Account)
	if accountName == "" {
		accountName = defaultAccountName(platform, subject)
	}

	connection, secretFields, err := buildConnectionTemplate(platform, subject, method, opts.Scopes)
	if err != nil {
		return InitResult{}, err
	}

	cfg := New()
	cfg.Ensure()
	cfg.Defaults.Platform = platform
	cfg.Defaults.Account = accountName
	cfg.Defaults.PlatformAccounts[platform] = accountName
	cfg.Auth = AuthConfig{
		SecretStore: SecretStoreConfig{
			Backend:         "auto",
			FallbackBackend: "encrypted_file",
		},
		SessionStore: SessionStoreConfig{
			Backend: "file",
		},
	}
	cfg.Runtime = RuntimeConfig{
		Retry: RetryConfig{
			MaxAttempts: 1,
			BaseDelayMS: 200,
			MaxDelayMS:  1000,
		},
	}
	cfg.Accounts[accountName] = buildAccountFromConnection(accountName, connection)
	cfg.Ensure()

	return InitResult{
		Config:         cfg,
		AccountName:    accountName,
		Platform:       platform,
		Subject:        subject,
		Method:         method,
		SecretFields:   secretFields,
		SessionBackend: cfg.Auth.SessionStore.Backend,
		SecretBackend:  cfg.Auth.SecretStore.Backend,
	}, nil
}

func buildConnectionTemplate(platform, subject, method string, scopes []string) (Connection, []string, error) {
	connection := Connection{
		Platform: platform,
		Subject:  subject,
		Method:   method,
	}

	secretFields := []string{}

	switch method {
	case "feishu.app_credentials":
		connection.Params.AppID = "<fill_app_id>"
		secretFields = []string{"app_secret"}
	case "feishu.oauth_user":
		connection.Params.ClientID = "<fill_client_id>"
		connection.Params.RedirectMode = "loopback"
		connection.Params.Scopes = normalizeInitScopes(scopes, []string{"offline_access"})
		// Before the first interactive authorization only client_secret is required.
		secretFields = []string{"client_secret"}
	case "notion.internal_token":
		connection.Params.NotionVersion = defaultNotionVersion
		secretFields = []string{"token"}
	case "notion.oauth_public":
		connection.Params.ClientID = "<fill_client_id>"
		connection.Params.NotionVersion = defaultNotionVersion
		connection.Params.RedirectMode = "loopback"
		// Before the first interactive authorization only client_secret is required.
		secretFields = []string{"client_secret"}
	default:
		return Connection{}, nil, fmt.Errorf("unsupported method: %s", method)
	}

	if err := ValidateConnectionShape(connection); err != nil {
		return Connection{}, nil, err
	}
	sort.Strings(secretFields)
	return connection, secretFields, nil
}

func normalizeInitScopes(scopes []string, defaults []string) []string {
	items := make([]string, 0, len(scopes))
	seen := map[string]struct{}{}

	appendScope := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		items = append(items, value)
	}

	for _, value := range scopes {
		appendScope(value)
	}
	if len(items) > 0 {
		return items
	}
	for _, value := range defaults {
		appendScope(value)
	}
	return items
}

func defaultSubjectForPlatform(platform string) string {
	switch platform {
	case "notion":
		return "integration"
	case "feishu":
		return "bot"
	default:
		return "integration"
	}
}

func defaultMethod(platform, subject string) string {
	switch {
	case platform == "feishu" && subject == "bot":
		return "feishu.app_credentials"
	case platform == "feishu" && subject == "user":
		return "feishu.oauth_user"
	case platform == "notion" && subject == "integration":
		return "notion.internal_token"
	default:
		return ""
	}
}

func defaultAccountName(platform, subject string) string {
	switch {
	case platform == "feishu" && subject == "bot":
		return "feishu_bot_default"
	case platform == "feishu" && subject == "user":
		return "feishu_user_default"
	case platform == "notion" && subject == "integration":
		return "notion_integration_default"
	default:
		return platform + "_" + subject + "_default"
	}
}
