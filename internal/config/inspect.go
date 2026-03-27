package config

import (
	"fmt"
	"sort"
	"strings"
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

// ValidateGrantShape 只校验结构和必填字段，不解析环境变量。
func ValidateGrantShape(profile Profile) error {
	if profile.Platform == "notion" && profile.Subject != "integration" {
		return fmt.Errorf("notion profiles must use subject integration")
	}

	requiredFields, err := requiredGrantFieldSpecs(profile)
	if err != nil {
		return err
	}

	for _, field := range requiredFields {
		raw := strings.TrimSpace(field.Value(profile.Grant))
		if field.PlainValue != nil {
			raw = strings.TrimSpace(field.PlainValue(profile.Grant))
		}
		if raw == "" {
			return fmt.Errorf("missing %s", field.Name)
		}
	}
	return nil
}

// InspectProfile 生成一个适合 `doctor` 和 `auth` 复用的检查视图。
func InspectProfile(name string, profile Profile) ProfileInspection {
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
				} else if item.Source == "env" {
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

	if err := ValidateGrantShape(profile); err != nil {
		inspection.ShapeError = err.Error()
	} else {
		inspection.ShapeValid = true
	}

	if err := ValidateGrant(profile); err != nil {
		inspection.ResolvedError = err.Error()
	} else {
		inspection.ResolvedValid = true
	}
	return inspection
}

// SortedProfileInspections 按名称稳定排序，方便命令和测试复用。
func SortedProfileInspections(cfg *Config) []ProfileInspection {
	cfg.Ensure()

	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]ProfileInspection, 0, len(names))
	for _, name := range names {
		items = append(items, InspectProfile(name, cfg.Profiles[name]))
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
			{Name: "app_id", Secret: true, Required: true, Value: func(g Grant) string { return g.AppID }},
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
			{Name: "client_id", Secret: true, Required: true, Value: func(g Grant) string { return g.ClientID }},
			{Name: "client_secret", Secret: true, Required: true, Value: func(g Grant) string { return g.ClientSecret }},
			{Name: "refresh_token", Secret: true, Required: true, Value: func(g Grant) string { return g.RefreshToken }},
		}, nil
	case "oauth_refreshable":
		return []grantFieldSpec{
			{Name: "client_id", Secret: true, Required: true, Value: func(g Grant) string { return g.ClientID }},
			{Name: "client_secret", Secret: true, Required: true, Value: func(g Grant) string { return g.ClientSecret }},
			{Name: "refresh_token", Secret: true, Required: true, Value: func(g Grant) string { return g.RefreshToken }},
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
		{Name: "app_id", Secret: true, Value: func(g Grant) string { return g.AppID }},
		{Name: "app_secret", Secret: true, Value: func(g Grant) string { return g.AppSecret }},
		{Name: "token", Secret: true, Value: func(g Grant) string { return g.Token }},
		{Name: "client_id", Secret: true, Value: func(g Grant) string { return g.ClientID }},
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
