package config

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	authcache "github.com/clawrise/clawrise-cli/internal/auth"
)

// GrantFieldInspection 描述一个授权字段的检查结果。
type GrantFieldInspection struct {
	Field         string `json:"field"`
	Required      bool   `json:"required"`
	Secret        bool   `json:"secret"`
	Source        string `json:"source"`
	Reference     string `json:"reference,omitempty"`
	Value         string `json:"value,omitempty"`
	Resolved      bool   `json:"resolved"`
	ResolvedValue string `json:"resolved_value,omitempty"`
	Error         string `json:"error,omitempty"`
}

// ProfileInspection 描述一个 profile 的整体检查结果。
type ProfileInspection struct {
	Name          string                 `json:"name"`
	Platform      string                 `json:"platform"`
	Subject       string                 `json:"subject"`
	GrantType     string                 `json:"grant_type"`
	ShapeValid    bool                   `json:"shape_valid"`
	ResolvedValid bool                   `json:"resolved_valid"`
	Ready         bool                   `json:"ready"`
	AuthStatus    string                 `json:"auth_status,omitempty"`
	AuthMessage   string                 `json:"auth_message,omitempty"`
	ShapeError    string                 `json:"shape_error,omitempty"`
	ResolvedError string                 `json:"resolved_error,omitempty"`
	Fields        []GrantFieldInspection `json:"fields"`
}

type grantFieldSpec struct {
	Name       string
	Secret     bool
	Required   bool
	Value      func(Grant) string
	PlainValue func(Grant) string
}

// ValidateConnectionShape 只校验连接结构和主配置里的必填字段。
func ValidateConnectionShape(connection Connection) error {
	if connection.Platform == "notion" && connection.Subject != "integration" {
		return fmt.Errorf("notion profiles must use subject integration")
	}
	if strings.TrimSpace(connection.Method) == "" && strings.TrimSpace(connection.Grant.Type) == "" {
		return fmt.Errorf("missing auth method")
	}

	switch strings.TrimSpace(connection.Method) {
	case "":
		// 兼容测试里的 legacy 构造方式，继续走旧字段校验。
	case "feishu.app_credentials":
		if strings.TrimSpace(connection.Params.AppID) == "" {
			return fmt.Errorf("missing app_id")
		}
		return nil
	case "feishu.oauth_user":
		if strings.TrimSpace(connection.Params.ClientID) == "" {
			return fmt.Errorf("missing client_id")
		}
		return nil
	case "notion.internal_token":
		return nil
	case "notion.oauth_public":
		if strings.TrimSpace(connection.Params.ClientID) == "" {
			return fmt.Errorf("missing client_id")
		}
		return nil
	default:
		return fmt.Errorf("unsupported method: %s", connection.Method)
	}

	// legacy 路径仅保留给单元测试和 plugin 进程内部对象。
	requiredFields, err := requiredGrantFieldSpecs(connection)
	if err != nil {
		return err
	}
	for _, field := range requiredFields {
		raw := strings.TrimSpace(field.Value(connection.Grant))
		if field.PlainValue != nil {
			raw = strings.TrimSpace(field.PlainValue(connection.Grant))
		}
		if raw == "" {
			return fmt.Errorf("missing %s", field.Name)
		}
	}
	return nil
}

// ValidateGrantShape 保留给当前仓库中的 legacy 调用点。
func ValidateGrantShape(profile Profile) error {
	return ValidateConnectionShape(profile)
}

// InspectProfile 生成一个适合 `doctor` 和 `auth` 复用的检查视图。
func InspectProfile(name string, profile Profile) ProfileInspection {
	profile = normalizeConnection(name, profile)
	inspection := ProfileInspection{
		Name:      name,
		Platform:  profile.Platform,
		Subject:   profile.Subject,
		GrantType: profile.Grant.Type,
	}

	fieldSpecs := allGrantFieldSpecs(profile)
	inspection.Fields = make([]GrantFieldInspection, 0, len(fieldSpecs))
	for _, field := range fieldSpecs {
		item := GrantFieldInspection{
			Field:    field.Name,
			Required: field.Required,
			Secret:   field.Secret,
			Source:   "empty",
		}

		raw := ""
		if field.PlainValue != nil {
			raw = strings.TrimSpace(field.PlainValue(profile.Grant))
		} else {
			raw = strings.TrimSpace(field.Value(profile.Grant))
		}

		if raw != "" {
			if field.Secret {
				item.Source, item.Reference = describeSecretSource(raw)
				if item.Source == "literal" {
					item.Resolved = true
					item.ResolvedValue = redactSecret(raw)
				} else if item.Source == "env" || item.Source == "secret_store" {
					resolved, err := ResolveSecret(raw)
					if err != nil {
						item.Error = err.Error()
					} else {
						item.Resolved = true
						item.ResolvedValue = redactSecret(resolved)
					}
				}
			} else {
				item.Source = "literal"
				item.Value = raw
				item.Resolved = true
			}
		}

		if item.Required && raw == "" {
			item.Error = fmt.Sprintf("missing %s", field.Name)
		}
		inspection.Fields = append(inspection.Fields, item)
	}

	if err := ValidateConnectionShape(profile); err != nil {
		inspection.ShapeError = err.Error()
	} else {
		inspection.ShapeValid = true
	}

	if err := ValidateGrant(profile); err != nil {
		inspection.ResolvedError = err.Error()
	} else {
		inspection.ResolvedValid = true
	}

	authState := inspectProfileAuthState(name, profile, inspection.ShapeValid && inspection.ResolvedValid)
	inspection.Ready = authState.Ready
	inspection.AuthStatus = authState.Status
	inspection.AuthMessage = authState.Message
	return inspection
}

// SortedProfileInspections 按名称稳定排序，方便命令和测试复用。
func SortedProfileInspections(cfg *Config) []ProfileInspection {
	cfg.Ensure()

	names := make([]string, 0, len(cfg.Connections))
	for name := range cfg.Connections {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]ProfileInspection, 0, len(names))
	for _, name := range names {
		items = append(items, InspectProfile(name, cfg.Connections[name]))
	}
	return items
}

func requiredGrantFieldSpecs(profile Profile) ([]grantFieldSpec, error) {
	switch profile.Grant.Type {
	case "client_credentials":
		if profile.Platform == "notion" {
			return nil, fmt.Errorf("notion does not support grant type: %s", profile.Grant.Type)
		}
		return []grantFieldSpec{
			{Name: "app_id", Secret: false, Required: true, Value: func(g Grant) string { return g.AppID }},
			{Name: "app_secret", Secret: true, Required: true, Value: func(g Grant) string { return g.AppSecret }},
		}, nil
	case "static_token":
		return []grantFieldSpec{
			{Name: "token", Secret: true, Required: true, Value: func(g Grant) string { return g.Token }},
		}, nil
	case "oauth_user":
		if profile.Platform == "notion" {
			return nil, fmt.Errorf("notion does not support grant type: %s", profile.Grant.Type)
		}
		return []grantFieldSpec{
			{Name: "client_id", Secret: false, Required: true, Value: func(g Grant) string { return g.ClientID }},
			{Name: "client_secret", Secret: true, Required: true, Value: func(g Grant) string { return g.ClientSecret }},
		}, nil
	case "oauth_refreshable":
		return []grantFieldSpec{
			{Name: "client_id", Secret: false, Required: true, Value: func(g Grant) string { return g.ClientID }},
			{Name: "client_secret", Secret: true, Required: true, Value: func(g Grant) string { return g.ClientSecret }},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported grant type: %s", profile.Grant.Type)
	}
}

func allGrantFieldSpecs(profile Profile) []grantFieldSpec {
	requiredMap := map[string]grantFieldSpec{}
	if required, err := requiredGrantFieldSpecs(profile); err == nil {
		for _, field := range required {
			requiredMap[field.Name] = field
		}
	}

	// 这里固定字段顺序，保证命令输出稳定且容易阅读。
	ordered := []grantFieldSpec{
		{Name: "app_id", Secret: false, Value: func(g Grant) string { return g.AppID }},
		{Name: "app_secret", Secret: true, Value: func(g Grant) string { return g.AppSecret }},
		{Name: "token", Secret: true, Value: func(g Grant) string { return g.Token }},
		{Name: "client_id", Secret: false, Value: func(g Grant) string { return g.ClientID }},
		{Name: "client_secret", Secret: true, Value: func(g Grant) string { return g.ClientSecret }},
		{Name: "access_token", Secret: true, Value: func(g Grant) string { return g.AccessToken }},
		{Name: "refresh_token", Secret: true, Value: func(g Grant) string { return g.RefreshToken }},
		{Name: "notion_version", Secret: false, PlainValue: func(g Grant) string { return g.NotionVer }},
	}

	items := make([]grantFieldSpec, 0, len(ordered))
	for _, field := range ordered {
		raw := ""
		if field.PlainValue != nil {
			raw = strings.TrimSpace(field.PlainValue(profile.Grant))
		} else {
			raw = strings.TrimSpace(field.Value(profile.Grant))
		}

		if required, ok := requiredMap[field.Name]; ok {
			field.Required = true
			field.Secret = required.Secret
			items = append(items, field)
			continue
		}
		if raw != "" {
			items = append(items, field)
		}
	}
	return items
}

func describeSecretSource(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "empty", ""
	}
	if strings.HasPrefix(raw, "env:") {
		return "env", strings.TrimSpace(strings.TrimPrefix(raw, "env:"))
	}
	if strings.HasPrefix(raw, "secret:") {
		return "secret_store", strings.TrimSpace(strings.TrimPrefix(raw, "secret:"))
	}
	return "literal", ""
}

func redactSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "***"
	}
	return value[:2] + "***" + value[len(value)-2:]
}

type profileAuthState struct {
	Ready   bool
	Status  string
	Message string
}

func inspectProfileAuthState(name string, profile Profile, configValid bool) profileAuthState {
	if !configValid {
		return profileAuthState{
			Ready:  false,
			Status: "invalid_config",
		}
	}

	switch profile.Grant.Type {
	case "oauth_user", "oauth_refreshable":
		if session, ok := loadMatchingSession(name, profile); ok {
			now := time.Now().UTC()
			if session.UsableAt(now, authcache.DefaultRefreshSkew) {
				return profileAuthState{
					Ready:  true,
					Status: "session_valid",
				}
			}
			if session.CanRefresh() {
				return profileAuthState{
					Ready:  true,
					Status: "refreshable",
				}
			}
		}

		if hasResolvedSecretValue(profile.Grant.AccessToken) {
			return profileAuthState{
				Ready:  true,
				Status: "access_token_configured",
			}
		}
		if hasResolvedSecretValue(profile.Grant.RefreshToken) {
			return profileAuthState{
				Ready:  true,
				Status: "refresh_token_configured",
			}
		}
		return profileAuthState{
			Ready:   false,
			Status:  "authorization_required",
			Message: "interactive authorization has not been completed yet",
		}
	default:
		return profileAuthState{
			Ready:  true,
			Status: "configured",
		}
	}
}

func loadMatchingSession(name string, profile Profile) (*authcache.Session, bool) {
	configPath, err := DefaultPath()
	if err != nil {
		return nil, false
	}

	sessionStore := authcache.NewFileStore(configPath)
	session, err := sessionStore.Load(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false
		}
		return nil, false
	}
	if session == nil {
		return nil, false
	}
	if strings.TrimSpace(session.Platform) != strings.TrimSpace(profile.Platform) {
		return nil, false
	}
	if strings.TrimSpace(session.Subject) != strings.TrimSpace(profile.Subject) {
		return nil, false
	}
	if strings.TrimSpace(session.GrantType) != strings.TrimSpace(profile.Grant.Type) {
		return nil, false
	}
	return session, true
}

func hasResolvedSecretValue(raw string) bool {
	value, err := ResolveSecret(raw)
	if err != nil {
		return false
	}
	return strings.TrimSpace(value) != ""
}
