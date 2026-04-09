package config

import (
	"strings"

	"github.com/clawrise/clawrise-cli/internal/paths"
	"gopkg.in/yaml.v3"
)

// Config is the top-level structure of the main Clawrise config file.
type Config struct {
	Defaults Defaults           `yaml:"defaults"`
	Auth     AuthConfig         `yaml:"auth,omitempty"`
	Runtime  RuntimeConfig      `yaml:"runtime,omitempty"`
	Plugins  PluginsConfig      `yaml:"plugins,omitempty"`
	Accounts map[string]Account `yaml:"accounts,omitempty"`
}

// Defaults stores the default platform and default execution account.
type Defaults struct {
	Platform         string            `yaml:"platform,omitempty"`
	PlatformAccounts map[string]string `yaml:"platform_accounts,omitempty"`
	Account          string            `yaml:"account,omitempty"`

	Subject string `yaml:"subject,omitempty"`
}

// AuthConfig describes low-level auth storage settings.
type AuthConfig struct {
	SecretStore   SecretStoreConfig   `yaml:"secret_store,omitempty"`
	SessionStore  SessionStoreConfig  `yaml:"session_store,omitempty"`
	AuthFlowStore AuthFlowStoreConfig `yaml:"authflow_store,omitempty"`
}

// SecretStoreConfig describes storage settings for long-lived secrets.
type SecretStoreConfig struct {
	Backend         string `yaml:"backend,omitempty"`
	FallbackBackend string `yaml:"fallback_backend,omitempty"`
	Plugin          string `yaml:"plugin,omitempty"`
}

// SessionStoreConfig describes storage settings for short-lived sessions.
type SessionStoreConfig struct {
	Backend string `yaml:"backend,omitempty"`
	Plugin  string `yaml:"plugin,omitempty"`
}

// AuthFlowStoreConfig describes storage settings for auth flow state.
type AuthFlowStoreConfig struct {
	Backend string `yaml:"backend,omitempty"`
	Plugin  string `yaml:"plugin,omitempty"`
}

// accountAuthBridge 描述从 account 配置派生出的内部授权桥接对象。
type accountAuthBridge struct {
	Title    string           `yaml:"title,omitempty"`
	Platform string           `yaml:"platform"`
	Subject  string           `yaml:"subject"`
	Method   string           `yaml:"method,omitempty"`
	Params   legacyAuthParams `yaml:"params,omitempty"`

	// 旧授权字段仅保留给执行期桥接与少量内部测试。
	LegacyAuth legacyAuthConfig `yaml:"grant,omitempty"`
}

// Account is the persisted account configuration shape.
type Account struct {
	Title    string      `yaml:"title,omitempty"`
	Platform string      `yaml:"platform"`
	Subject  string      `yaml:"subject"`
	Auth     AccountAuth `yaml:"auth,omitempty"`
}

// AccountAuth describes the auth method and fields used by one account.
type AccountAuth struct {
	Method     string            `yaml:"method,omitempty"`
	Public     map[string]any    `yaml:"public,omitempty"`
	SecretRefs map[string]string `yaml:"secret_refs,omitempty"`
}

// legacyAuthParams 描述桥接期仍需保留的非 secret 授权字段。
type legacyAuthParams struct {
	AppID         string   `yaml:"app_id,omitempty"`
	ClientID      string   `yaml:"client_id,omitempty"`
	NotionVersion string   `yaml:"notion_version,omitempty"`
	RedirectMode  string   `yaml:"redirect_mode,omitempty"`
	Scopes        []string `yaml:"scopes,omitempty"`
}

// legacyAuthConfig 描述桥接期保留的旧授权字段集合。
type legacyAuthConfig struct {
	Type         string `yaml:"type"`
	AppID        string `yaml:"app_id,omitempty"`
	AppSecret    string `yaml:"app_secret,omitempty"`
	Token        string `yaml:"token,omitempty"`
	ClientID     string `yaml:"client_id,omitempty"`
	ClientSecret string `yaml:"client_secret,omitempty"`
	AccessToken  string `yaml:"access_token,omitempty"`
	RefreshToken string `yaml:"refresh_token,omitempty"`
	NotionVer    string `yaml:"notion_version,omitempty"`
}

// RuntimeConfig describes runtime governance settings.
type RuntimeConfig struct {
	Retry      RetryConfig      `yaml:"retry,omitempty"`
	Governance GovernanceConfig `yaml:"governance,omitempty"`
	Policy     PolicyConfig     `yaml:"policy,omitempty"`
	Audit      AuditConfig      `yaml:"audit,omitempty"`
}

// RetryConfig describes automatic retry settings.
type RetryConfig struct {
	MaxAttempts int `yaml:"max_attempts,omitempty"`
	BaseDelayMS int `yaml:"base_delay_ms,omitempty"`
	MaxDelayMS  int `yaml:"max_delay_ms,omitempty"`
}

// GovernanceConfig describes runtime governance storage settings.
type GovernanceConfig struct {
	Backend string `yaml:"backend,omitempty"`
	Plugin  string `yaml:"plugin,omitempty"`
}

// PolicyConfig describes the base local policy chain settings.
type PolicyConfig struct {
	Mode                      string                `yaml:"mode,omitempty"`
	Plugins                   []PolicyPluginBinding `yaml:"plugins,omitempty"`
	DenyOperations            []string              `yaml:"deny_operations,omitempty"`
	RequireApprovalOperations []string              `yaml:"require_approval_operations,omitempty"`
	AnnotateOperations        map[string]string     `yaml:"annotate_operations,omitempty"`
}

// PolicyPluginBinding 描述一个外部 policy capability 的选择器。
type PolicyPluginBinding struct {
	Plugin   string `yaml:"plugin,omitempty"`
	PolicyID string `yaml:"policy_id,omitempty"`
}

// AuditConfig 描述审计扇出链的显式配置。
type AuditConfig struct {
	Mode  string            `yaml:"mode,omitempty"`
	Sinks []AuditSinkConfig `yaml:"sinks,omitempty"`
}

// AuditSinkConfig 描述一个内建或外部审计 sink 的配置项。
type AuditSinkConfig struct {
	Type      string            `yaml:"type,omitempty"`
	Plugin    string            `yaml:"plugin,omitempty"`
	SinkID    string            `yaml:"sink_id,omitempty"`
	URL       string            `yaml:"url,omitempty"`
	Headers   map[string]string `yaml:"headers,omitempty"`
	TimeoutMS int               `yaml:"timeout_ms,omitempty"`
}

// New returns an empty config.
func New() *Config {
	return &Config{
		Accounts: map[string]Account{},
	}
}

// Ensure initializes nil maps so later writes remain safe.
func (c *Config) Ensure() {
	if c.Accounts == nil {
		c.Accounts = map[string]Account{}
	}
	if c.Defaults.PlatformAccounts == nil {
		c.Defaults.PlatformAccounts = map[string]string{}
	}
	if c.Plugins.Enabled == nil {
		c.Plugins.Enabled = map[string]string{}
	}
	if c.Plugins.Bindings.Providers == nil {
		c.Plugins.Bindings.Providers = map[string]ProviderPluginBinding{}
	}
	if c.Plugins.Bindings.AuthLaunchers == nil {
		c.Plugins.Bindings.AuthLaunchers = map[string][]string{}
	}
	if c.Plugins.PluginConfig == nil {
		c.Plugins.PluginConfig = map[string]map[string]any{}
	}
}

// DefaultPath returns the default config file path.
func DefaultPath() (string, error) {
	return paths.ResolveConfigPath()
}

// Marshal encodes the config as YAML.
func (c *Config) Marshal() ([]byte, error) {
	c.Ensure()
	return yaml.Marshal(c)
}

func buildAccountAuthBridgeFromAccount(accountName string, account Account) accountAuthBridge {
	bridge := accountAuthBridge{
		Title:    account.Title,
		Platform: account.Platform,
		Subject:  account.Subject,
		Method:   strings.TrimSpace(account.Auth.Method),
	}

	publicString := func(field string) string {
		if account.Auth.Public == nil {
			return ""
		}
		value, ok := account.Auth.Public[field]
		if !ok {
			return ""
		}
		text, ok := value.(string)
		if !ok {
			return ""
		}
		return strings.TrimSpace(text)
	}
	publicStringSlice := func(field string) []string {
		if account.Auth.Public == nil {
			return nil
		}
		value, ok := account.Auth.Public[field]
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
	secretRef := func(field string) string {
		if account.Auth.SecretRefs == nil {
			return ""
		}
		return strings.TrimSpace(account.Auth.SecretRefs[field])
	}

	switch bridge.Method {
	case "feishu.app_credentials":
		bridge.Params.AppID = publicString("app_id")
		bridge.LegacyAuth = legacyAuthConfig{
			Type:      "client_credentials",
			AppID:     bridge.Params.AppID,
			AppSecret: firstNonEmpty(secretRef("app_secret"), SecretRef(accountName, "app_secret")),
		}
	case "feishu.oauth_user":
		bridge.Params.ClientID = publicString("client_id")
		bridge.Params.RedirectMode = publicString("redirect_mode")
		bridge.Params.Scopes = publicStringSlice("scopes")
		bridge.LegacyAuth = legacyAuthConfig{
			Type:         "oauth_user",
			ClientID:     bridge.Params.ClientID,
			ClientSecret: firstNonEmpty(secretRef("client_secret"), SecretRef(accountName, "client_secret")),
			AccessToken:  secretRef("access_token"),
			RefreshToken: firstNonEmpty(secretRef("refresh_token"), SecretRef(accountName, "refresh_token")),
		}
	case "notion.internal_token":
		bridge.Params.NotionVersion = publicString("notion_version")
		bridge.LegacyAuth = legacyAuthConfig{
			Type:      "static_token",
			Token:     firstNonEmpty(secretRef("token"), SecretRef(accountName, "token")),
			NotionVer: bridge.Params.NotionVersion,
		}
	case "notion.oauth_public":
		bridge.Params.ClientID = publicString("client_id")
		bridge.Params.NotionVersion = publicString("notion_version")
		bridge.Params.RedirectMode = publicString("redirect_mode")
		bridge.Params.Scopes = publicStringSlice("scopes")
		bridge.LegacyAuth = legacyAuthConfig{
			Type:         "oauth_refreshable",
			ClientID:     bridge.Params.ClientID,
			ClientSecret: firstNonEmpty(secretRef("client_secret"), SecretRef(accountName, "client_secret")),
			AccessToken:  secretRef("access_token"),
			RefreshToken: firstNonEmpty(secretRef("refresh_token"), SecretRef(accountName, "refresh_token")),
			NotionVer:    bridge.Params.NotionVersion,
		}
	default:
		bridge.LegacyAuth = legacyAuthConfig{Type: legacyAuthTypeForMethod(bridge.Method)}
	}

	return bridge
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeAccountAuthBridge(name string, bridge accountAuthBridge) accountAuthBridge {
	hasExplicitMethod := strings.TrimSpace(bridge.Method) != ""
	if !hasExplicitMethod {
		bridge.Method = inferMethodFromLegacyAuthType(bridge.Platform, bridge.LegacyAuth.Type)
	}
	if strings.TrimSpace(bridge.Method) != "" {
		applyAccountAuthBridgeDefaults(&bridge)
		if hasPersistedLegacyAuthConfig(bridge.LegacyAuth) {
			if strings.TrimSpace(bridge.LegacyAuth.Type) == "" {
				bridge.LegacyAuth.Type = legacyAuthTypeForMethod(bridge.Method)
			}
		} else if hasExplicitMethod || strings.TrimSpace(bridge.LegacyAuth.Type) == "" {
			bridge.LegacyAuth = buildLegacyAuthConfig(name, bridge)
		}
	}
	return bridge
}

func applyAccountAuthBridgeDefaults(bridge *accountAuthBridge) {
	if bridge == nil {
		return
	}

	switch bridge.Method {
	case "feishu.app_credentials":
		if bridge.Params.AppID == "" {
			bridge.Params.AppID = bridge.LegacyAuth.AppID
		}
	case "feishu.oauth_user":
		if bridge.Params.ClientID == "" {
			bridge.Params.ClientID = bridge.LegacyAuth.ClientID
		}
	case "notion.internal_token":
		if bridge.Params.NotionVersion == "" {
			bridge.Params.NotionVersion = bridge.LegacyAuth.NotionVer
		}
	case "notion.oauth_public":
		if bridge.Params.ClientID == "" {
			bridge.Params.ClientID = bridge.LegacyAuth.ClientID
		}
		if bridge.Params.NotionVersion == "" {
			bridge.Params.NotionVersion = bridge.LegacyAuth.NotionVer
		}
	}
}

func buildLegacyAuthConfig(accountName string, bridge accountAuthBridge) legacyAuthConfig {
	legacy := legacyAuthConfig{
		Type: legacyAuthTypeForMethod(bridge.Method),
	}

	switch bridge.Method {
	case "feishu.app_credentials":
		legacy.AppID = strings.TrimSpace(bridge.Params.AppID)
		legacy.AppSecret = SecretRef(accountName, "app_secret")
	case "feishu.oauth_user":
		legacy.ClientID = strings.TrimSpace(bridge.Params.ClientID)
		legacy.ClientSecret = SecretRef(accountName, "client_secret")
		legacy.RefreshToken = SecretRef(accountName, "refresh_token")
	case "notion.internal_token":
		legacy.Token = SecretRef(accountName, "token")
		legacy.NotionVer = strings.TrimSpace(bridge.Params.NotionVersion)
	case "notion.oauth_public":
		legacy.ClientID = strings.TrimSpace(bridge.Params.ClientID)
		legacy.ClientSecret = SecretRef(accountName, "client_secret")
		legacy.RefreshToken = SecretRef(accountName, "refresh_token")
		legacy.NotionVer = strings.TrimSpace(bridge.Params.NotionVersion)
	default:
		legacy = bridge.LegacyAuth
	}

	// 单元测试仍可能直接构造旧授权字段，因此这里保留字段级回填。
	if legacy.AppID == "" {
		legacy.AppID = bridge.LegacyAuth.AppID
	}
	if legacy.AppSecret == "" {
		legacy.AppSecret = bridge.LegacyAuth.AppSecret
	}
	if legacy.Token == "" {
		legacy.Token = bridge.LegacyAuth.Token
	}
	if legacy.ClientID == "" {
		legacy.ClientID = bridge.LegacyAuth.ClientID
	}
	if legacy.ClientSecret == "" {
		legacy.ClientSecret = bridge.LegacyAuth.ClientSecret
	}
	if legacy.AccessToken == "" {
		legacy.AccessToken = bridge.LegacyAuth.AccessToken
	}
	if legacy.RefreshToken == "" {
		legacy.RefreshToken = bridge.LegacyAuth.RefreshToken
	}
	if legacy.NotionVer == "" {
		legacy.NotionVer = bridge.LegacyAuth.NotionVer
	}
	if legacy.Type == "" {
		legacy.Type = bridge.LegacyAuth.Type
	}
	return legacy
}

func inferMethodFromLegacyAuthType(platform string, authType string) string {
	switch strings.TrimSpace(platform) + ":" + strings.TrimSpace(authType) {
	case "feishu:client_credentials":
		return "feishu.app_credentials"
	case "feishu:oauth_user":
		return "feishu.oauth_user"
	case "notion:static_token":
		return "notion.internal_token"
	case "notion:oauth_refreshable":
		return "notion.oauth_public"
	default:
		return ""
	}
}

func legacyAuthTypeForMethod(method string) string {
	switch strings.TrimSpace(method) {
	case "feishu.app_credentials":
		return "client_credentials"
	case "feishu.oauth_user":
		return "oauth_user"
	case "notion.internal_token":
		return "static_token"
	case "notion.oauth_public":
		return "oauth_refreshable"
	default:
		return ""
	}
}

// SecretRef encodes one secret-store field reference in the shared format.
func SecretRef(connectionName string, field string) string {
	return "secret:" + strings.TrimSpace(connectionName) + ":" + strings.TrimSpace(field)
}

func hasPersistedLegacyAuthConfig(legacy legacyAuthConfig) bool {
	if strings.TrimSpace(legacy.Type) == "" {
		return false
	}

	values := []string{
		legacy.AppID,
		legacy.AppSecret,
		legacy.Token,
		legacy.ClientID,
		legacy.ClientSecret,
		legacy.AccessToken,
		legacy.RefreshToken,
		legacy.NotionVer,
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !strings.HasPrefix(value, "secret:") {
			return true
		}
	}
	return false
}
