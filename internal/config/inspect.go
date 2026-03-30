package config

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	authcache "github.com/clawrise/clawrise-cli/internal/auth"
)

// AuthFieldInspection 描述一个授权字段的检查结果。
type AuthFieldInspection struct {
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

// AccountInspection 描述一个账号配置的整体检查结果。
type AccountInspection struct {
	Name          string                `json:"name"`
	Platform      string                `json:"platform"`
	Subject       string                `json:"subject"`
	AuthType      string                `json:"auth_type"`
	ShapeValid    bool                  `json:"shape_valid"`
	ResolvedValid bool                  `json:"resolved_valid"`
	Ready         bool                  `json:"ready"`
	AuthStatus    string                `json:"auth_status,omitempty"`
	AuthMessage   string                `json:"auth_message,omitempty"`
	ShapeError    string                `json:"shape_error,omitempty"`
	ResolvedError string                `json:"resolved_error,omitempty"`
	Fields        []AuthFieldInspection `json:"fields"`
}

type authFieldSpec struct {
	Name       string
	Secret     bool
	Required   bool
	Value      func(Grant) string
	PlainValue func(Grant) string
}

// ValidateResolvedAuthShape 只校验归一化后的账号授权结构和主配置里的必填字段。
func ValidateResolvedAuthShape(account Connection) error {
	if account.Platform == "notion" && account.Subject != "integration" {
		return fmt.Errorf("notion accounts must use subject integration")
	}
	if strings.TrimSpace(account.Method) == "" && strings.TrimSpace(account.Grant.Type) == "" {
		return fmt.Errorf("missing auth method")
	}

	switch strings.TrimSpace(account.Method) {
	case "":
		// 兼容执行期仍在使用的 legacy 授权桥接结构，继续走旧字段校验。
	case "feishu.app_credentials":
		if strings.TrimSpace(account.Params.AppID) == "" {
			return fmt.Errorf("missing app_id")
		}
		return nil
	case "feishu.oauth_user":
		if strings.TrimSpace(account.Params.ClientID) == "" {
			return fmt.Errorf("missing client_id")
		}
		return nil
	case "notion.internal_token":
		return nil
	case "notion.oauth_public":
		if strings.TrimSpace(account.Params.ClientID) == "" {
			return fmt.Errorf("missing client_id")
		}
		return nil
	default:
		return fmt.Errorf("unsupported method: %s", account.Method)
	}

	// 这条 legacy 路径仅保留给执行期桥接对象和单元测试。
	requiredFields, err := requiredAuthFieldSpecs(account)
	if err != nil {
		return err
	}
	for _, field := range requiredFields {
		raw := strings.TrimSpace(field.Value(account.Grant))
		if field.PlainValue != nil {
			raw = strings.TrimSpace(field.PlainValue(account.Grant))
		}
		if raw == "" {
			return fmt.Errorf("missing %s", field.Name)
		}
	}
	return nil
}

// InspectAccount 生成一个适合诊断和测试复用的账号检查视图。
func InspectAccount(name string, account Account) AccountInspection {
	resolvedAccount := normalizeConnection(name, buildConnectionFromAccount(name, account))
	inspection := AccountInspection{
		Name:     name,
		Platform: resolvedAccount.Platform,
		Subject:  resolvedAccount.Subject,
		AuthType: resolvedAccount.Grant.Type,
	}

	fieldSpecs := allAuthFieldSpecs(resolvedAccount)
	inspection.Fields = make([]AuthFieldInspection, 0, len(fieldSpecs))
	for _, field := range fieldSpecs {
		item := AuthFieldInspection{
			Field:    field.Name,
			Required: field.Required,
			Secret:   field.Secret,
			Source:   "empty",
		}

		raw := ""
		if field.PlainValue != nil {
			raw = strings.TrimSpace(field.PlainValue(resolvedAccount.Grant))
		} else {
			raw = strings.TrimSpace(field.Value(resolvedAccount.Grant))
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

	if err := ValidateResolvedAuthShape(resolvedAccount); err != nil {
		inspection.ShapeError = err.Error()
	} else {
		inspection.ShapeValid = true
	}

	if err := ValidateGrant(resolvedAccount); err != nil {
		inspection.ResolvedError = err.Error()
	} else {
		inspection.ResolvedValid = true
	}

	authState := inspectAccountAuthState(name, resolvedAccount, inspection.ShapeValid && inspection.ResolvedValid)
	inspection.Ready = authState.Ready
	inspection.AuthStatus = authState.Status
	inspection.AuthMessage = authState.Message
	return inspection
}

// SortedAccountInspections 按名称稳定排序，方便命令和测试复用。
func SortedAccountInspections(cfg *Config) []AccountInspection {
	cfg.Ensure()

	names := make([]string, 0, len(cfg.Accounts))
	for name := range cfg.Accounts {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]AccountInspection, 0, len(names))
	for _, name := range names {
		items = append(items, InspectAccount(name, cfg.Accounts[name]))
	}
	return items
}

func requiredAuthFieldSpecs(account Connection) ([]authFieldSpec, error) {
	switch account.Grant.Type {
	case "client_credentials":
		if account.Platform == "notion" {
			return nil, fmt.Errorf("notion does not support grant type: %s", account.Grant.Type)
		}
		return []authFieldSpec{
			{Name: "app_id", Secret: false, Required: true, Value: func(g Grant) string { return g.AppID }},
			{Name: "app_secret", Secret: true, Required: true, Value: func(g Grant) string { return g.AppSecret }},
		}, nil
	case "static_token":
		return []authFieldSpec{
			{Name: "token", Secret: true, Required: true, Value: func(g Grant) string { return g.Token }},
		}, nil
	case "oauth_user":
		if account.Platform == "notion" {
			return nil, fmt.Errorf("notion does not support grant type: %s", account.Grant.Type)
		}
		return []authFieldSpec{
			{Name: "client_id", Secret: false, Required: true, Value: func(g Grant) string { return g.ClientID }},
			{Name: "client_secret", Secret: true, Required: true, Value: func(g Grant) string { return g.ClientSecret }},
		}, nil
	case "oauth_refreshable":
		return []authFieldSpec{
			{Name: "client_id", Secret: false, Required: true, Value: func(g Grant) string { return g.ClientID }},
			{Name: "client_secret", Secret: true, Required: true, Value: func(g Grant) string { return g.ClientSecret }},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported grant type: %s", account.Grant.Type)
	}
}

func allAuthFieldSpecs(account Connection) []authFieldSpec {
	requiredMap := map[string]authFieldSpec{}
	if required, err := requiredAuthFieldSpecs(account); err == nil {
		for _, field := range required {
			requiredMap[field.Name] = field
		}
	}

	// 这里固定字段顺序，保证命令输出稳定且容易阅读。
	ordered := []authFieldSpec{
		{Name: "app_id", Secret: false, Value: func(g Grant) string { return g.AppID }},
		{Name: "app_secret", Secret: true, Value: func(g Grant) string { return g.AppSecret }},
		{Name: "token", Secret: true, Value: func(g Grant) string { return g.Token }},
		{Name: "client_id", Secret: false, Value: func(g Grant) string { return g.ClientID }},
		{Name: "client_secret", Secret: true, Value: func(g Grant) string { return g.ClientSecret }},
		{Name: "access_token", Secret: true, Value: func(g Grant) string { return g.AccessToken }},
		{Name: "refresh_token", Secret: true, Value: func(g Grant) string { return g.RefreshToken }},
		{Name: "notion_version", Secret: false, PlainValue: func(g Grant) string { return g.NotionVer }},
	}

	items := make([]authFieldSpec, 0, len(ordered))
	for _, field := range ordered {
		raw := ""
		if field.PlainValue != nil {
			raw = strings.TrimSpace(field.PlainValue(account.Grant))
		} else {
			raw = strings.TrimSpace(field.Value(account.Grant))
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

type accountAuthState struct {
	Ready   bool
	Status  string
	Message string
}

func inspectAccountAuthState(name string, account Connection, configValid bool) accountAuthState {
	if !configValid {
		return accountAuthState{
			Ready:  false,
			Status: "invalid_config",
		}
	}

	switch account.Grant.Type {
	case "oauth_user", "oauth_refreshable":
		if session, ok := loadMatchingSession(name, account); ok {
			now := time.Now().UTC()
			if session.UsableAt(now, authcache.DefaultRefreshSkew) {
				return accountAuthState{
					Ready:  true,
					Status: "session_valid",
				}
			}
			if session.CanRefresh() {
				return accountAuthState{
					Ready:  true,
					Status: "refreshable",
				}
			}
		}

		if hasResolvedSecretValue(account.Grant.AccessToken) {
			return accountAuthState{
				Ready:  true,
				Status: "access_token_configured",
			}
		}
		if hasResolvedSecretValue(account.Grant.RefreshToken) {
			return accountAuthState{
				Ready:  true,
				Status: "refresh_token_configured",
			}
		}
		return accountAuthState{
			Ready:   false,
			Status:  "authorization_required",
			Message: "interactive authorization has not been completed yet",
		}
	default:
		return accountAuthState{
			Ready:  true,
			Status: "configured",
		}
	}
}

func loadMatchingSession(name string, account Connection) (*authcache.Session, bool) {
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
	if strings.TrimSpace(session.Platform) != strings.TrimSpace(account.Platform) {
		return nil, false
	}
	if strings.TrimSpace(session.Subject) != strings.TrimSpace(account.Subject) {
		return nil, false
	}
	if strings.TrimSpace(session.GrantType) != strings.TrimSpace(account.Grant.Type) {
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
