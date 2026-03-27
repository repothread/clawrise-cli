package config

import (
	"fmt"
	"regexp"
	"strings"
)

const defaultNotionVersion = "2026-03-11"

var nonAlphaNumericPattern = regexp.MustCompile(`[^A-Za-z0-9]+`)

// InitOptions 描述 `config init` 需要的参数。
type InitOptions struct {
	Platform  string
	Subject   string
	Profile   string
	GrantType string
}

// InitResult 描述初始化后生成的配置与提示信息。
type InitResult struct {
	Config      *Config           `json:"-"`
	ProfileName string            `json:"profile_name"`
	Platform    string            `json:"platform"`
	Subject     string            `json:"subject"`
	GrantType   string            `json:"grant_type"`
	EnvTemplate map[string]string `json:"env_template"`
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

	grantType := strings.TrimSpace(opts.GrantType)
	if grantType == "" {
		grantType = defaultGrantType(platform, subject)
	}
	if grantType == "" {
		return InitResult{}, fmt.Errorf("unsupported platform and subject combination: %s/%s", platform, subject)
	}

	profileName := strings.TrimSpace(opts.Profile)
	if profileName == "" {
		profileName = defaultProfileName(platform, subject)
	}

	profile, envTemplate, err := buildProfileTemplate(profileName, platform, subject, grantType)
	if err != nil {
		return InitResult{}, err
	}

	cfg := New()
	cfg.Defaults.Platform = platform
	cfg.Defaults.Subject = subject
	cfg.Defaults.Profile = profileName
	cfg.Runtime = RuntimeConfig{
		Retry: RetryConfig{
			MaxAttempts: 1,
			BaseDelayMS: 200,
			MaxDelayMS:  1000,
		},
	}
	cfg.Profiles[profileName] = profile

	return InitResult{
		Config:      cfg,
		ProfileName: profileName,
		Platform:    platform,
		Subject:     subject,
		GrantType:   grantType,
		EnvTemplate: envTemplate,
	}, nil
}

func buildProfileTemplate(profileName, platform, subject, grantType string) (Profile, map[string]string, error) {
	profile := Profile{
		Platform: platform,
		Subject:  subject,
		Grant: Grant{
			Type: grantType,
		},
	}

	prefix := envPrefixForProfile(profileName)
	envTemplate := map[string]string{}

	switch grantType {
	case "client_credentials":
		profile.Grant.AppID = "env:" + prefix + "_APP_ID"
		profile.Grant.AppSecret = "env:" + prefix + "_APP_SECRET"
		envTemplate["APP_ID"] = prefix + "_APP_ID"
		envTemplate["APP_SECRET"] = prefix + "_APP_SECRET"
	case "static_token":
		profile.Grant.Token = "env:" + prefix + "_TOKEN"
		envTemplate["TOKEN"] = prefix + "_TOKEN"
		if platform == "notion" {
			profile.Grant.NotionVer = defaultNotionVersion
		}
	case "oauth_user":
		profile.Grant.ClientID = "env:" + prefix + "_CLIENT_ID"
		profile.Grant.ClientSecret = "env:" + prefix + "_CLIENT_SECRET"
		profile.Grant.AccessToken = "env:" + prefix + "_ACCESS_TOKEN"
		profile.Grant.RefreshToken = "env:" + prefix + "_REFRESH_TOKEN"
		envTemplate["CLIENT_ID"] = prefix + "_CLIENT_ID"
		envTemplate["CLIENT_SECRET"] = prefix + "_CLIENT_SECRET"
		envTemplate["ACCESS_TOKEN"] = prefix + "_ACCESS_TOKEN"
		envTemplate["REFRESH_TOKEN"] = prefix + "_REFRESH_TOKEN"
	case "oauth_refreshable":
		profile.Grant.ClientID = "env:" + prefix + "_CLIENT_ID"
		profile.Grant.ClientSecret = "env:" + prefix + "_CLIENT_SECRET"
		profile.Grant.AccessToken = "env:" + prefix + "_ACCESS_TOKEN"
		profile.Grant.RefreshToken = "env:" + prefix + "_REFRESH_TOKEN"
		envTemplate["CLIENT_ID"] = prefix + "_CLIENT_ID"
		envTemplate["CLIENT_SECRET"] = prefix + "_CLIENT_SECRET"
		envTemplate["ACCESS_TOKEN"] = prefix + "_ACCESS_TOKEN"
		envTemplate["REFRESH_TOKEN"] = prefix + "_REFRESH_TOKEN"
		if platform == "notion" {
			profile.Grant.NotionVer = defaultNotionVersion
		}
	default:
		return Profile{}, nil, fmt.Errorf("unsupported grant type: %s", grantType)
	}

	if err := ValidateGrantShape(profile); err != nil {
		return Profile{}, nil, err
	}
	return profile, envTemplate, nil
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

func defaultGrantType(platform, subject string) string {
	switch {
	case platform == "feishu" && subject == "bot":
		return "client_credentials"
	case platform == "feishu" && subject == "user":
		return "oauth_user"
	case platform == "notion" && subject == "integration":
		return "static_token"
	default:
		return ""
	}
}

func defaultProfileName(platform, subject string) string {
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

func envPrefixForProfile(profileName string) string {
	profileName = strings.TrimSpace(profileName)
	normalized := nonAlphaNumericPattern.ReplaceAllString(strings.ToUpper(profileName), "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return "CLAWRISE_PROFILE"
	}
	return normalized
}
