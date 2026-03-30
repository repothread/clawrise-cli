package authflow

import "time"

const (
	// 默认授权 flow 过期时间，避免旧 flow 长时间残留。
	DefaultFlowTTL = 10 * time.Minute
)

// Flow 表示一次进行中的本地授权流程。
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
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	ExpiresAt        time.Time         `json:"expires_at"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	Result           map[string]any    `json:"result,omitempty"`
	ErrorCode        string            `json:"error_code,omitempty"`
	ErrorMessage     string            `json:"error_message,omitempty"`
}

// Action 描述 flow 当前需要调用方执行的动作。
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

// MethodSpec 描述授权接入方式的交互能力。
type MethodSpec struct {
	Name             string   `json:"name"`
	Interactive      bool     `json:"interactive"`
	SupportsCodeFlow bool     `json:"supports_code_flow"`
	DefaultMode      string   `json:"default_mode,omitempty"`
	Modes            []string `json:"modes,omitempty"`
}

// LookupMethodSpec 返回 method 的授权能力描述。
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

// BuildActions 基于 flow 当前状态返回结构化动作。
func BuildActions(flow Flow) []Action {
	if flow.State == "completed" || flow.State == "failed" {
		return nil
	}

	actions := []Action{}
	if flow.AuthorizationURL != "" {
		actions = append(actions, Action{
			Type:        "open_url",
			URL:         flow.AuthorizationURL,
			Description: "在用户可交互的设备上打开授权链接",
		})
	}
	if flow.Mode == "local_browser" && flow.CallbackHost != "" && flow.CallbackPort > 0 {
		actions = append(actions, Action{
			Type:        "wait_callback",
			Host:        flow.CallbackHost,
			Port:        flow.CallbackPort,
			Path:        flow.CallbackPath,
			TimeoutSec:  int(DefaultFlowTTL.Seconds()),
			Description: "当前版本保留本地回调信息，后续可接自动监听；现在仍可用 callback_url 或 code 手动继续",
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
