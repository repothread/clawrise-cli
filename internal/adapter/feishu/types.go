package feishu

import "strings"

// ExecutionProfile describes the Feishu-specific execution auth context.
type ExecutionProfile struct {
	Platform string
	Subject  string
	Method   string
	Params   ExecutionParams
	Grant    ExecutionGrant
}

// ExecutionParams describes Feishu public auth parameters.
type ExecutionParams struct {
	AppID        string
	ClientID     string
	RedirectMode string
	Scopes       []string
}

// ExecutionGrant describes Feishu secrets and resolved auth state.
type ExecutionGrant struct {
	Type         string
	AppID        string
	AppSecret    string
	ClientID     string
	ClientSecret string
	AccessToken  string
	RefreshToken string
}

func normalizeExecutionProfile(profile ExecutionProfile) ExecutionProfile {
	if strings.TrimSpace(profile.Method) == "" {
		profile.Method = feishuMethodForGrantType(profile.Grant.Type)
	}
	if strings.TrimSpace(profile.Grant.Type) == "" {
		profile.Grant.Type = feishuGrantTypeForMethod(profile.Method)
	}
	if strings.TrimSpace(profile.Params.AppID) == "" {
		profile.Params.AppID = strings.TrimSpace(profile.Grant.AppID)
	}
	if strings.TrimSpace(profile.Params.ClientID) == "" {
		profile.Params.ClientID = strings.TrimSpace(profile.Grant.ClientID)
	}
	return profile
}

func feishuMethodForGrantType(grantType string) string {
	switch strings.TrimSpace(grantType) {
	case "client_credentials":
		return "feishu.app_credentials"
	case "oauth_user":
		return "feishu.oauth_user"
	default:
		return ""
	}
}
