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
	Accounts map[string]Account `yaml:"accounts,omitempty"`
	// Legacy-only input shim. New code should use Accounts.
	Connections map[string]Connection `yaml:"-"`
	// Legacy-only input shim. New code should use Accounts.
	Profiles map[string]Profile `yaml:"-"`
}

// Defaults stores the default platform and default execution connection.
type Defaults struct {
	Platform string `yaml:"platform,omitempty"`
	// Legacy-only input shim. New code should use PlatformAccounts.
	Connections      map[string]string `yaml:"-"`
	PlatformAccounts map[string]string `yaml:"platform_accounts,omitempty"`
	Account          string            `yaml:"account,omitempty"`

	// Subject is still user-visible, but Profile is a legacy input shim only.
	Subject string `yaml:"subject,omitempty"`
	Profile string `yaml:"-"`
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
}

// SessionStoreConfig describes storage settings for short-lived sessions.
type SessionStoreConfig struct {
	Backend string `yaml:"backend,omitempty"`
}

// AuthFlowStoreConfig describes storage settings for auth flow state.
type AuthFlowStoreConfig struct {
	Backend string `yaml:"backend,omitempty"`
}

// Connection describes one executable identity instance.
type Connection struct {
	Title    string           `yaml:"title,omitempty"`
	Platform string           `yaml:"platform"`
	Subject  string           `yaml:"subject"`
	Method   string           `yaml:"method,omitempty"`
	Params   ConnectionParams `yaml:"params,omitempty"`

	// This legacy grant stays only as an internal adapter bridge.
	Grant Grant `yaml:"grant,omitempty"`
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

// ConnectionParams describes non-secret fields for legacy adapter wiring.
type ConnectionParams struct {
	AppID         string   `yaml:"app_id,omitempty"`
	ClientID      string   `yaml:"client_id,omitempty"`
	NotionVersion string   `yaml:"notion_version,omitempty"`
	RedirectMode  string   `yaml:"redirect_mode,omitempty"`
	Scopes        []string `yaml:"scopes,omitempty"`
}

// Grant describes the legacy internal auth shape.
type Grant struct {
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
}

// Profile remains an internal alias for the resolved execution shape.
type Profile = Connection

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
	if len(c.Accounts) == 0 && len(c.Connections) > 0 {
		for name, connection := range c.Connections {
			c.Accounts[name] = buildAccountFromConnection(name, connection)
		}
	}
	if len(c.Accounts) == 0 && len(c.Profiles) > 0 {
		for name, profile := range c.Profiles {
			c.Accounts[name] = buildAccountFromConnection(name, profile)
		}
	}
	if c.Defaults.PlatformAccounts == nil {
		c.Defaults.PlatformAccounts = map[string]string{}
	}
	if c.Defaults.Account == "" && strings.TrimSpace(c.Defaults.Profile) != "" {
		c.Defaults.Account = strings.TrimSpace(c.Defaults.Profile)
	}
	if len(c.Defaults.PlatformAccounts) == 0 && len(c.Defaults.Connections) > 0 {
		for platform, accountName := range c.Defaults.Connections {
			c.Defaults.PlatformAccounts[platform] = accountName
		}
	}
}

// DefaultPath returns the default config file path.
func DefaultPath() (string, error) {
	return paths.ResolveConfigPath()
}

// Marshal encodes the config as YAML.
func (c *Config) Marshal() ([]byte, error) {
	c.Ensure()

	cloned := *c
	cloned.Profiles = nil
	cloned.Connections = nil
	cloned.Defaults.Connections = nil
	cloned.Defaults.Profile = ""
	return yaml.Marshal(&cloned)
}

func buildAccountFromConnection(connectionName string, connection Connection) Account {
	connection = normalizeConnection(connectionName, connection)
	account := Account{
		Title:    connection.Title,
		Platform: connection.Platform,
		Subject:  connection.Subject,
		Auth: AccountAuth{
			Method:     strings.TrimSpace(connection.Method),
			Public:     map[string]any{},
			SecretRefs: map[string]string{},
		},
	}

	setSecretRef := func(field string, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		account.Auth.SecretRefs[field] = value
	}

	switch strings.TrimSpace(connection.Method) {
	case "feishu.app_credentials":
		if value := strings.TrimSpace(connection.Params.AppID); value != "" {
			account.Auth.Public["app_id"] = value
		}
		setSecretRef("app_secret", connection.Grant.AppSecret)
	case "feishu.oauth_user":
		if value := strings.TrimSpace(connection.Params.ClientID); value != "" {
			account.Auth.Public["client_id"] = value
		}
		if value := strings.TrimSpace(connection.Params.RedirectMode); value != "" {
			account.Auth.Public["redirect_mode"] = value
		}
		if len(connection.Params.Scopes) > 0 {
			account.Auth.Public["scopes"] = append([]string(nil), connection.Params.Scopes...)
		}
		setSecretRef("client_secret", connection.Grant.ClientSecret)
		setSecretRef("access_token", connection.Grant.AccessToken)
		setSecretRef("refresh_token", connection.Grant.RefreshToken)
	case "notion.internal_token":
		if value := strings.TrimSpace(connection.Params.NotionVersion); value != "" {
			account.Auth.Public["notion_version"] = value
		}
		setSecretRef("token", connection.Grant.Token)
	case "notion.oauth_public":
		if value := strings.TrimSpace(connection.Params.ClientID); value != "" {
			account.Auth.Public["client_id"] = value
		}
		if value := strings.TrimSpace(connection.Params.NotionVersion); value != "" {
			account.Auth.Public["notion_version"] = value
		}
		if value := strings.TrimSpace(connection.Params.RedirectMode); value != "" {
			account.Auth.Public["redirect_mode"] = value
		}
		if len(connection.Params.Scopes) > 0 {
			account.Auth.Public["scopes"] = append([]string(nil), connection.Params.Scopes...)
		}
		setSecretRef("client_secret", connection.Grant.ClientSecret)
		setSecretRef("access_token", connection.Grant.AccessToken)
		setSecretRef("refresh_token", connection.Grant.RefreshToken)
	default:
		account.Auth.Method = strings.TrimSpace(connection.Method)
	}

	if len(account.Auth.Public) == 0 {
		account.Auth.Public = nil
	}
	if len(account.Auth.SecretRefs) == 0 {
		account.Auth.SecretRefs = nil
	}
	return account
}

func buildConnectionFromAccount(accountName string, account Account) Connection {
	connection := Connection{
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

	switch connection.Method {
	case "feishu.app_credentials":
		connection.Params.AppID = publicString("app_id")
		connection.Grant = Grant{
			Type:      "client_credentials",
			AppID:     connection.Params.AppID,
			AppSecret: firstNonEmpty(secretRef("app_secret"), SecretRef(accountName, "app_secret")),
		}
	case "feishu.oauth_user":
		connection.Params.ClientID = publicString("client_id")
		connection.Params.RedirectMode = publicString("redirect_mode")
		connection.Params.Scopes = publicStringSlice("scopes")
		connection.Grant = Grant{
			Type:         "oauth_user",
			ClientID:     connection.Params.ClientID,
			ClientSecret: firstNonEmpty(secretRef("client_secret"), SecretRef(accountName, "client_secret")),
			AccessToken:  secretRef("access_token"),
			RefreshToken: firstNonEmpty(secretRef("refresh_token"), SecretRef(accountName, "refresh_token")),
		}
	case "notion.internal_token":
		connection.Params.NotionVersion = publicString("notion_version")
		connection.Grant = Grant{
			Type:      "static_token",
			Token:     firstNonEmpty(secretRef("token"), SecretRef(accountName, "token")),
			NotionVer: connection.Params.NotionVersion,
		}
	case "notion.oauth_public":
		connection.Params.ClientID = publicString("client_id")
		connection.Params.NotionVersion = publicString("notion_version")
		connection.Params.RedirectMode = publicString("redirect_mode")
		connection.Params.Scopes = publicStringSlice("scopes")
		connection.Grant = Grant{
			Type:         "oauth_refreshable",
			ClientID:     connection.Params.ClientID,
			ClientSecret: firstNonEmpty(secretRef("client_secret"), SecretRef(accountName, "client_secret")),
			AccessToken:  secretRef("access_token"),
			RefreshToken: firstNonEmpty(secretRef("refresh_token"), SecretRef(accountName, "refresh_token")),
			NotionVer:    connection.Params.NotionVersion,
		}
	default:
		connection.Grant = Grant{Type: legacyGrantTypeForMethod(connection.Method)}
	}

	return connection
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

func normalizeConnection(name string, connection Connection) Connection {
	hasExplicitMethod := strings.TrimSpace(connection.Method) != ""
	if !hasExplicitMethod {
		connection.Method = inferMethodFromLegacyGrant(connection.Platform, connection.Grant.Type)
	}
	if strings.TrimSpace(connection.Method) != "" {
		applyConnectionDefaults(&connection)
		if hasPersistedLegacyGrant(connection.Grant) {
			if strings.TrimSpace(connection.Grant.Type) == "" {
				connection.Grant.Type = legacyGrantTypeForMethod(connection.Method)
			}
		} else if hasExplicitMethod || strings.TrimSpace(connection.Grant.Type) == "" {
			connection.Grant = buildLegacyGrant(name, connection)
		}
	}
	return connection
}

func applyConnectionDefaults(connection *Connection) {
	if connection == nil {
		return
	}

	switch connection.Method {
	case "feishu.app_credentials":
		if connection.Params.AppID == "" {
			connection.Params.AppID = connection.Grant.AppID
		}
	case "feishu.oauth_user":
		if connection.Params.ClientID == "" {
			connection.Params.ClientID = connection.Grant.ClientID
		}
	case "notion.internal_token":
		if connection.Params.NotionVersion == "" {
			connection.Params.NotionVersion = connection.Grant.NotionVer
		}
	case "notion.oauth_public":
		if connection.Params.ClientID == "" {
			connection.Params.ClientID = connection.Grant.ClientID
		}
		if connection.Params.NotionVersion == "" {
			connection.Params.NotionVersion = connection.Grant.NotionVer
		}
	}
}

func buildLegacyGrant(connectionName string, connection Connection) Grant {
	grant := Grant{
		Type: legacyGrantTypeForMethod(connection.Method),
	}

	switch connection.Method {
	case "feishu.app_credentials":
		grant.AppID = strings.TrimSpace(connection.Params.AppID)
		grant.AppSecret = SecretRef(connectionName, "app_secret")
	case "feishu.oauth_user":
		grant.ClientID = strings.TrimSpace(connection.Params.ClientID)
		grant.ClientSecret = SecretRef(connectionName, "client_secret")
		grant.RefreshToken = SecretRef(connectionName, "refresh_token")
	case "notion.internal_token":
		grant.Token = SecretRef(connectionName, "token")
		grant.NotionVer = strings.TrimSpace(connection.Params.NotionVersion)
	case "notion.oauth_public":
		grant.ClientID = strings.TrimSpace(connection.Params.ClientID)
		grant.ClientSecret = SecretRef(connectionName, "client_secret")
		grant.RefreshToken = SecretRef(connectionName, "refresh_token")
		grant.NotionVer = strings.TrimSpace(connection.Params.NotionVersion)
	default:
		grant = connection.Grant
	}

	// Tests may still build the legacy grant shape directly, so keep this fallback.
	if grant.AppID == "" {
		grant.AppID = connection.Grant.AppID
	}
	if grant.AppSecret == "" {
		grant.AppSecret = connection.Grant.AppSecret
	}
	if grant.Token == "" {
		grant.Token = connection.Grant.Token
	}
	if grant.ClientID == "" {
		grant.ClientID = connection.Grant.ClientID
	}
	if grant.ClientSecret == "" {
		grant.ClientSecret = connection.Grant.ClientSecret
	}
	if grant.AccessToken == "" {
		grant.AccessToken = connection.Grant.AccessToken
	}
	if grant.RefreshToken == "" {
		grant.RefreshToken = connection.Grant.RefreshToken
	}
	if grant.NotionVer == "" {
		grant.NotionVer = connection.Grant.NotionVer
	}
	if grant.Type == "" {
		grant.Type = connection.Grant.Type
	}
	return grant
}

func inferMethodFromLegacyGrant(platform string, grantType string) string {
	switch strings.TrimSpace(platform) + ":" + strings.TrimSpace(grantType) {
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

func legacyGrantTypeForMethod(method string) string {
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

func shouldOmitLegacyGrantOnMarshal(connection Connection) bool {
	if strings.TrimSpace(connection.Method) == "" {
		return false
	}

	secretFields := []string{
		connection.Grant.AppSecret,
		connection.Grant.Token,
		connection.Grant.ClientSecret,
		connection.Grant.AccessToken,
		connection.Grant.RefreshToken,
	}
	for _, value := range secretFields {
		value = strings.TrimSpace(value)
		if value != "" && !strings.HasPrefix(value, "secret:") {
			return false
		}
	}
	return true
}

func hasPersistedLegacyGrant(grant Grant) bool {
	if strings.TrimSpace(grant.Type) == "" {
		return false
	}

	values := []string{
		grant.AppID,
		grant.AppSecret,
		grant.Token,
		grant.ClientID,
		grant.ClientSecret,
		grant.AccessToken,
		grant.RefreshToken,
		grant.NotionVer,
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
