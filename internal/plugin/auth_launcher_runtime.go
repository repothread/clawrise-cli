package plugin

import (
	"context"
	"fmt"
	"os/exec"
	goRuntime "runtime"
	"strings"
)

// SystemAuthLauncherRuntime provides the default system-level auth launcher.
// It exposes launcher behavior through a runtime interface so core no longer
// depends on platform-specific shell commands directly.
type SystemAuthLauncherRuntime struct{}

// NewSystemAuthLauncherRuntime creates the default system launcher.
func NewSystemAuthLauncherRuntime() AuthLauncherRuntime {
	return &SystemAuthLauncherRuntime{}
}

func (r *SystemAuthLauncherRuntime) Name() string {
	return "system_auth_launcher"
}

func (r *SystemAuthLauncherRuntime) Handshake(ctx context.Context) (HandshakeResult, error) {
	return HandshakeResult{
		ProtocolVersion: ProtocolVersion,
		Name:            r.Name(),
		Version:         "builtin",
	}, nil
}

func (r *SystemAuthLauncherRuntime) DescribeAuthLauncher(ctx context.Context) (AuthLauncherDescriptor, error) {
	return AuthLauncherDescriptor{
		ID:          "system_browser",
		DisplayName: "System Browser Auth Launcher",
		Description: "Open authorization URLs or device-code verification pages with the default system browser.",
		ActionTypes: []string{"open_url", "device_code"},
		Priority:    10,
	}, nil
}

func (r *SystemAuthLauncherRuntime) LaunchAuth(ctx context.Context, params AuthLaunchParams) (AuthLaunchResult, error) {
	rawURL, err := resolveLaunchURL(params)
	if err != nil {
		return AuthLaunchResult{}, err
	}
	if err := openURLWithSystemBrowser(rawURL); err != nil {
		return AuthLaunchResult{}, err
	}

	result := AuthLaunchResult{
		Handled:    true,
		Status:     "launched",
		LauncherID: "system_browser",
		Metadata: map[string]any{
			"url":         rawURL,
			"action_type": strings.TrimSpace(params.Action.Type),
		},
	}
	if code := firstNonEmpty(params.Action.UserCode, params.Flow.UserCode); code != "" {
		result.Metadata["user_code"] = code
	}
	return result, nil
}

func resolveLaunchURL(params AuthLaunchParams) (string, error) {
	for _, candidate := range []string{
		strings.TrimSpace(params.Action.URL),
		strings.TrimSpace(params.Action.VerificationURL),
		strings.TrimSpace(params.Flow.AuthorizationURL),
		strings.TrimSpace(params.Flow.VerificationURL),
	} {
		if candidate != "" {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no launchable url is available for auth action %s", strings.TrimSpace(params.Action.Type))
}

func openURLWithSystemBrowser(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("authorization url is empty")
	}

	var command *exec.Cmd
	switch goRuntime.GOOS {
	case "darwin":
		command = exec.Command("open", rawURL)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		command = exec.Command("xdg-open", rawURL)
	}
	if output, err := command.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("failed to open authorization url: %s", message)
	}
	return nil
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
