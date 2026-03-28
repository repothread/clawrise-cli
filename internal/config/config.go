package config

import (
	"sort"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/paths"
	"gopkg.in/yaml.v3"
)

// Config is the top-level structure of the main Clawrise config file.
type Config struct {
	Defaults    Defaults              `yaml:"defaults"`
	Paths       PathsConfig           `yaml:"paths,omitempty"`
	Auth        AuthConfig            `yaml:"auth,omitempty"`
	Runtime     RuntimeConfig         `yaml:"runtime,omitempty"`
	Connections map[string]Connection `yaml:"connections"`
	Profiles    map[string]Profile    `yaml:"-"`
}

// Defaults stores the default platform and default execution connection.
type Defaults struct {
	Platform    string            `yaml:"platform,omitempty"`
	Connections map[string]string `yaml:"connections,omitempty"`

	// 这些字段仅保留给当前仓库的内部兼容层，不再作为推荐模型。
	Subject string `yaml:"subject,omitempty"`
	Profile string `yaml:"profile,omitempty"`
}

// PathsConfig 描述配置目录和状态目录覆盖项。
type PathsConfig struct {
	ConfigDir string `yaml:"config_dir,omitempty"`
	StateDir  string `yaml:"state_dir,omitempty"`
}

// AuthConfig 描述授权相关的底层存储策略。
type AuthConfig struct {
	SecretStore  SecretStoreConfig  `yaml:"secret_store,omitempty"`
	SessionStore SessionStoreConfig `yaml:"session_store,omitempty"`
}

// SecretStoreConfig 描述长期敏感信息的存储策略。
type SecretStoreConfig struct {
	Backend         string `yaml:"backend,omitempty"`
	FallbackBackend string `yaml:"fallback_backend,omitempty"`
}

// SessionStoreConfig 描述短期 session 的存储策略。
type SessionStoreConfig struct {
	Backend string `yaml:"backend,omitempty"`
}

// Connection describes one executable identity instance.
type Connection struct {
	Title    string           `yaml:"title,omitempty"`
	Platform string           `yaml:"platform"`
	Subject  string           `yaml:"subject"`
	Method   string           `yaml:"method,omitempty"`
	Params   ConnectionParams `yaml:"params,omitempty"`

	// 这一组字段只用于当前代码库的适配层，避免一次性改动全部 adapter。
	Grant Grant `yaml:"grant,omitempty"`
}

// ConnectionParams 描述会落入主配置文件的非敏感参数。
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

// RuntimeConfig 描述运行时治理相关配置。
type RuntimeConfig struct {
	Retry RetryConfig `yaml:"retry,omitempty"`
}

// RetryConfig 描述自动重试策略。
type RetryConfig struct {
	MaxAttempts int `yaml:"max_attempts,omitempty"`
	BaseDelayMS int `yaml:"base_delay_ms,omitempty"`
	MaxDelayMS  int `yaml:"max_delay_ms,omitempty"`
}

// Profile 是 Connection 的内部别名，便于平滑迁移已有实现。
type Profile = Connection

// NamedProfile is a connection value paired with its config key.
type NamedProfile struct {
	Name    string
	Profile Profile
}

// New returns an empty config.
func New() *Config {
	return &Config{
		Connections: map[string]Connection{},
		Profiles:    map[string]Profile{},
	}
}

// Ensure initializes nil maps so later writes remain safe.
func (c *Config) Ensure() {
	if c.Connections == nil {
		c.Connections = map[string]Connection{}
	}
	if len(c.Connections) == 0 && len(c.Profiles) > 0 {
		for name, profile := range c.Profiles {
			c.Connections[name] = profile
		}
	}
	if c.Defaults.Connections == nil {
		c.Defaults.Connections = map[string]string{}
	}
	for name, connection := range c.Connections {
		connection = normalizeConnection(name, connection)
		c.Connections[name] = connection
	}
	c.Profiles = c.Connections
}

// CandidateProfiles returns candidate profiles for a platform in a stable order.
func (c *Config) CandidateProfiles(platform string) []NamedProfile {
	return c.CandidateProfilesBySubject(platform, "")
}

// CandidateProfilesBySubject returns candidate profiles for a platform and,
// when provided, filters them by subject.
func (c *Config) CandidateProfilesBySubject(platform, subject string) []NamedProfile {
	c.Ensure()

	names := make([]string, 0, len(c.Connections))
	for name, connection := range c.Connections {
		if connection.Platform == platform && (subject == "" || connection.Subject == subject) {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	result := make([]NamedProfile, 0, len(names))
	for _, name := range names {
		result = append(result, NamedProfile{
			Name:    name,
			Profile: c.Connections[name],
		})
	}
	return result
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
	cloned.Connections = map[string]Connection{}
	for name, connection := range c.Connections {
		connection = normalizeConnection(name, connection)
		if shouldOmitLegacyGrantOnMarshal(connection) {
			connection.Grant = Grant{}
		}
		cloned.Connections[name] = connection
	}
	return yaml.Marshal(&cloned)
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

	// 测试里仍然可能直接构造 legacy Grant，这里保留回退路径。
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

// LegacyGrantTypeForMethod 返回当前 method 对应的 legacy grant type。
func LegacyGrantTypeForMethod(method string) string {
	return legacyGrantTypeForMethod(method)
}

// SecretRef 把 secret store 中的字段编码成统一引用。
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
