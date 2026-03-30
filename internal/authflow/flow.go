package authflow

import "time"

const (
	// DefaultFlowTTL keeps abandoned auth flows from lingering too long.
	DefaultFlowTTL = 10 * time.Minute
)

// Flow describes one in-progress local auth flow.
type Flow struct {
	ID               string            `json:"id"`
	ConnectionName   string            `json:"connection_name"`
	Platform         string            `json:"platform"`
	Method           string            `json:"method"`
	Mode             string            `json:"mode"`
	State            string            `json:"state"`
	RedirectURI      string            `json:"redirect_uri,omitempty"`
	CallbackHost     string            `json:"callback_host,omitempty"`
	CallbackPort     int               `json:"callback_port,omitempty"`
	CallbackPath     string            `json:"callback_path,omitempty"`
	AuthorizationURL string            `json:"authorization_url,omitempty"`
	DeviceCode       string            `json:"device_code,omitempty"`
	UserCode         string            `json:"user_code,omitempty"`
	VerificationURL  string            `json:"verification_url,omitempty"`
	IntervalSec      int               `json:"interval_sec,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	ExpiresAt        time.Time         `json:"expires_at"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	Result           map[string]any    `json:"result,omitempty"`
	ErrorCode        string            `json:"error_code,omitempty"`
	ErrorMessage     string            `json:"error_message,omitempty"`
}

// Action describes one next step required by the current auth flow.
type Action struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
	Host        string `json:"host,omitempty"`
	Port        int    `json:"port,omitempty"`
	Path        string `json:"path,omitempty"`
	TimeoutSec  int    `json:"timeout_sec,omitempty"`
	Field       string `json:"field,omitempty"`
}

// MethodSpec describes the interactive capability surface of one auth method.
type MethodSpec struct {
	Name             string   `json:"name"`
	Interactive      bool     `json:"interactive"`
	SupportsCodeFlow bool     `json:"supports_code_flow"`
	DefaultMode      string   `json:"default_mode,omitempty"`
	Modes            []string `json:"modes,omitempty"`
}

// LookupMethodSpec returns the auth capability descriptor for one method.
func LookupMethodSpec(method string) (MethodSpec, bool) {
	switch method {
	case "feishu.oauth_user":
		return MethodSpec{
			Name:             method,
			Interactive:      true,
			SupportsCodeFlow: true,
			DefaultMode:      "local_browser",
			Modes:            []string{"local_browser", "manual_url", "manual_code"},
		}, true
	case "notion.oauth_public":
		return MethodSpec{
			Name:             method,
			Interactive:      true,
			SupportsCodeFlow: true,
			DefaultMode:      "local_browser",
			Modes:            []string{"local_browser", "manual_url", "manual_code"},
		}, true
	case "feishu.app_credentials", "notion.internal_token":
		return MethodSpec{
			Name:             method,
			Interactive:      false,
			SupportsCodeFlow: false,
		}, true
	default:
		return MethodSpec{}, false
	}
}

// BuildActions returns structured next-step actions for one flow state.
func BuildActions(flow Flow) []Action {
	if flow.State == "completed" || flow.State == "failed" {
		return nil
	}

	actions := []Action{}
	if flow.Mode == "device_code" && flow.VerificationURL != "" {
		actions = append(actions, Action{
			Type:        "device_code",
			URL:         flow.VerificationURL,
			Field:       "device_code",
			Description: "Open the verification page in a browser and enter the user code.",
		})
	}
	if flow.AuthorizationURL != "" {
		actions = append(actions, Action{
			Type:        "open_url",
			URL:         flow.AuthorizationURL,
			Description: "Open the authorization URL on a user-accessible device.",
		})
	}
	if flow.Mode == "local_browser" && flow.CallbackHost != "" && flow.CallbackPort > 0 {
		actions = append(actions, Action{
			Type:        "wait_callback",
			Host:        flow.CallbackHost,
			Port:        flow.CallbackPort,
			Path:        flow.CallbackPath,
			TimeoutSec:  int(DefaultFlowTTL.Seconds()),
			Description: "Loopback callback details are preserved for future automation; for now you can still continue with callback_url or code.",
		})
	}
	actions = append(actions, Action{
		Type:        "submit_callback_url",
		Field:       "callback_url",
		Description: "After authorization, pass the final callback URL to `clawrise auth complete`.",
	})
	actions = append(actions, Action{
		Type:        "submit_code",
		Field:       "code",
		Description: "If a full callback URL is not available, pass the code directly to `clawrise auth complete`.",
	})
	return actions
}
