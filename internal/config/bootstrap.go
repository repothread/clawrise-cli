package config

import (
	"fmt"
	"sort"
	"strings"
)

const defaultNotionVersion = "2026-03-11"

// InitOptions 描述 `config init` 需要的参数。
type InitOptions struct {
	Platform   string
	Subject    string
	Connection string
	Method     string
	Scopes     []string
}

// InitResult 描述初始化后生成的配置与提示信息。
type InitResult struct {
	Config         *Config  `json:"-"`
	ConnectionName string   `json:"connection_name"`
	Platform       string   `json:"platform"`
	Subject        string   `json:"subject"`
	Method         string   `json:"method"`
	SecretFields   []string `json:"secret_fields"`
	SessionBackend string   `json:"session_backend"`
	SecretBackend  string   `json:"secret_backend"`
}

// BuildInitConfig 生成最小可用的配置骨架。
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

	connectionName := strings.TrimSpace(opts.Connection)
	if connectionName == "" {
		connectionName = defaultConnectionName(platform, subject)
	}

	connection, secretFields, err := buildConnectionTemplate(platform, subject, method, opts.Scopes)
	if err != nil {
		return InitResult{}, err
	}

	cfg := New()
	cfg.Ensure()
	cfg.Defaults.Platform = platform
	cfg.Defaults.Profile = connectionName
	cfg.Defaults.Connections[platform] = connectionName
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
	cfg.Connections[connectionName] = connection
	cfg.Ensure()

	return InitResult{
		Config:         cfg,
		ConnectionName: connectionName,
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
		connection.Params.AppID = "<请填写 app_id>"
		secretFields = []string{"app_secret"}
	case "feishu.oauth_user":
		connection.Params.ClientID = "<请填写 client_id>"
		connection.Params.RedirectMode = "loopback"
		connection.Params.Scopes = normalizeInitScopes(scopes, []string{"offline_access"})
		// 首次交互式授权前只需要 client_secret，refresh_token 会在授权完成后自动写回。
		secretFields = []string{"client_secret"}
	case "notion.internal_token":
		connection.Params.NotionVersion = defaultNotionVersion
		secretFields = []string{"token"}
	case "notion.oauth_public":
		connection.Params.ClientID = "<请填写 client_id>"
		connection.Params.NotionVersion = defaultNotionVersion
		connection.Params.RedirectMode = "loopback"
		// 首次交互式授权前只需要 client_secret，refresh_token 会在授权完成后自动写回。
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

func defaultConnectionName(platform, subject string) string {
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
