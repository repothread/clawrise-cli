package notion

import "strings"

// ExecutionProfile describes the Notion-specific execution auth context.
type ExecutionProfile struct {
	Platform string
	Subject  string
	Method   string
	Params   ExecutionParams
	Grant    ExecutionGrant
}

// ExecutionParams describes Notion public auth parameters.
type ExecutionParams struct {
	ClientID      string
	NotionVersion string
	RedirectMode  string
}

// ExecutionGrant describes Notion secrets and resolved auth state.
type ExecutionGrant struct {
	Type         string
	Token        string
	ClientID     string
	ClientSecret string
	AccessToken  string
	RefreshToken string
	NotionVer    string
}

func normalizeExecutionProfile(profile ExecutionProfile) ExecutionProfile {
	if strings.TrimSpace(profile.Method) == "" {
		profile.Method = notionMethodForGrantType(profile.Grant.Type)
	}
	if strings.TrimSpace(profile.Grant.Type) == "" {
		profile.Grant.Type = notionGrantTypeForMethod(profile.Method)
	}
	if strings.TrimSpace(profile.Params.ClientID) == "" {
		profile.Params.ClientID = strings.TrimSpace(profile.Grant.ClientID)
	}
	if strings.TrimSpace(profile.Params.NotionVersion) == "" {
		profile.Params.NotionVersion = strings.TrimSpace(profile.Grant.NotionVer)
	}
	return profile
}

func notionMethodForGrantType(grantType string) string {
	switch strings.TrimSpace(grantType) {
	case "static_token":
		return "notion.internal_token"
	case "oauth_refreshable":
		return "notion.oauth_public"
	default:
		return ""
	}
}
