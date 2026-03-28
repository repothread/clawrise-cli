package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	feishuadapter "github.com/clawrise/clawrise-cli/internal/adapter/feishu"
	notionadapter "github.com/clawrise/clawrise-cli/internal/adapter/notion"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

func TestRunRootHelpFlag(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"--help"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("Usage:")) {
		t.Fatalf("expected root help output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("clawrise spec [list|get|status|export]")) {
		t.Fatalf("expected spec usage in root help, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("clawrise auth [list|inspect|check|begin|connect|status|continue|session|secret]")) {
		t.Fatalf("expected auth usage in root help, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunOperationHelpFlag(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"feishu.calendar.event.create", "--help"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got: %s", stdout.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Usage of clawrise:")) {
		t.Fatalf("expected operation flag help, got: %s", stderr.String())
	}
}

func TestRunOperationDryRun(t *testing.T) {
	t.Setenv("FEISHU_BOT_OPS_APP_ID", "app-id")
	t.Setenv("FEISHU_BOT_OPS_APP_SECRET", "app-secret")
	t.Setenv("CLAWRISE_CONFIG", "../../examples/config.example.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{
		"feishu.calendar.event.create",
		"--dry-run",
		"--json",
		`{"calendar_id":"cal_demo","summary":"Demo Event","start_at":"2026-03-30T10:00:00+08:00","end_at":"2026-03-30T11:00:00+08:00"}`,
	}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"ok": true`)) {
		t.Fatalf("expected success output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"context"`)) {
		t.Fatalf("expected context output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"subject": "bot"`)) {
		t.Fatalf("expected bot subject in output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunSubjectUse(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"subject", "use", "bot"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"subject": "bot"`)) {
		t.Fatalf("expected subject output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunProfileUseSynchronizesPlatformForBareOperation(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("NOTION_TEAM_DOCS_TOKEN", "notion-token")

	configBytes, err := os.ReadFile("../../examples/config.example.yaml")
	if err != nil {
		t.Fatalf("failed to read example config: %v", err)
	}
	configBytes = bytes.Replace(configBytes, []byte("backend: auto"), []byte("backend: encrypted_file"), 1)
	if err := os.WriteFile(configPath, configBytes, 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err = Run([]string{"profile", "use", "notion_team_docs"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	stdout.Reset()
	stderr.Reset()

	err = Run([]string{"page.get", "--dry-run", "--json", `{}`}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"normalized": "notion.page.get"`)) {
		t.Fatalf("expected bare operation to resolve with notion platform, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"profile": "notion_team_docs"`)) {
		t.Fatalf("expected notion profile in output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"subject": "integration"`)) {
		t.Fatalf("expected integration subject in output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunSpecList(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"spec", "list"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"full_path": "feishu"`)) {
		t.Fatalf("expected feishu in spec list output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"full_path": "notion"`)) {
		t.Fatalf("expected notion in spec list output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunSpecGet(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"spec", "get", "notion.page.create"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"operation": "notion.page.create"`)) {
		t.Fatalf("expected operation output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"implemented": true`)) {
		t.Fatalf("expected implemented flag, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunSpecGetFeishuPriorityOperations(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"spec", "get", "feishu.bitable.field.list"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"operation": "feishu.bitable.field.list"`)) {
		t.Fatalf("expected operation output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"required"`)) {
		t.Fatalf("expected required field metadata, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"optional"`)) {
		t.Fatalf("expected optional field metadata, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"sample"`)) {
		t.Fatalf("expected sample metadata, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunSpecStatus(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"spec", "status"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"summary"`)) {
		t.Fatalf("expected summary output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"registered_count"`)) {
		t.Fatalf("expected runtime counts in status output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"declared_count"`)) {
		t.Fatalf("expected catalog counts in status output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunSpecExportJSON(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"spec", "export", "notion.page"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"exported_operation_count"`)) {
		t.Fatalf("expected export summary output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"operation": "notion.page.create"`)) {
		t.Fatalf("expected notion.page.create in export output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunSpecExportMarkdown(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"spec", "export", "notion.page.create", "--format", "markdown"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("# Clawrise Spec Export")) {
		t.Fatalf("expected markdown export title, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("## `notion.page.create`")) {
		t.Fatalf("expected markdown operation heading, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunCompletionBash(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"completion", "bash"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("_clawrise_completion")) {
		t.Fatalf("expected bash completion function, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("feishu.calendar.event.create")) {
		t.Fatalf("expected operation completion entry, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("notion.page.markdown")) {
		t.Fatalf("expected spec path completion entry, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunSpecHelpFlag(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"spec", "--help"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("Usage: clawrise spec [list|get|status|export]")) {
		t.Fatalf("expected spec help output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunPlatformHelpFlag(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"platform", "--help"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("Usage: clawrise platform [use|current|unset]")) {
		t.Fatalf("expected platform help output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunPluginHelpFlag(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"plugin", "--help"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("Usage: clawrise plugin [list|install|info|remove|verify]")) {
		t.Fatalf("expected plugin help output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunCompletionHelpFlag(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"completion", "--help"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("Usage: clawrise completion <bash|zsh|fish>")) {
		t.Fatalf("expected completion help output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunConfigInit(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{
		"config",
		"init",
		"--platform", "notion",
		"--subject", "integration",
		"--profile", "notion_team_docs",
	}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"profile_name": "notion_team_docs"`)) {
		t.Fatalf("expected profile name in output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"grant_type": "static_token"`)) {
		t.Fatalf("expected grant type in output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunAuthCheck(t *testing.T) {
	copyExampleConfig(t)
	runSecretSet(t, "feishu_bot_ops", "app_secret", "app-secret")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"auth", "check", "feishu_bot_ops"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"resolved_valid": true`)) {
		t.Fatalf("expected valid auth check output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"subject_allowed_operation_count"`)) {
		t.Fatalf("expected operation summary in auth output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunAuthSessionInspectAndClear(t *testing.T) {
	configPath := copyExampleConfig(t)

	sessionStore := authcache.NewFileStore(configPath)
	expiresAt := time.Now().UTC().Add(30 * time.Minute)
	if err := sessionStore.Save(authcache.Session{
		ProfileName:  "notion_public_workspace_a",
		Platform:     "notion",
		Subject:      "integration",
		GrantType:    "oauth_refreshable",
		AccessToken:  "fresh-token",
		RefreshToken: "refresh-token-2",
		TokenType:    "Bearer",
		ExpiresAt:    &expiresAt,
	}); err != nil {
		t.Fatalf("failed to seed session cache: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"auth", "session", "inspect", "notion_public_workspace_a"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"exists": true`)) {
		t.Fatalf("expected existing session output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"matches": true`)) {
		t.Fatalf("expected matching profile output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"access_token": "fr***en"`)) {
		t.Fatalf("expected redacted access token output, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()

	err = Run([]string{"auth", "session", "clear", "notion_public_workspace_a"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"deleted": true`)) {
		t.Fatalf("expected deleted session output, got: %s", stdout.String())
	}
	if _, err := os.Stat(sessionStore.Path("notion_public_workspace_a")); !os.IsNotExist(err) {
		t.Fatalf("expected session file to be removed, got: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunAuthSessionRefresh(t *testing.T) {
	configPath := copyExampleConfig(t)
	runSecretSet(t, "notion_public_workspace_a", "client_secret", "client-secret")
	runSecretSet(t, "notion_public_workspace_a", "refresh_token", "refresh-token")

	previousFactory := newNotionAuthSessionClient
	t.Cleanup(func() {
		newNotionAuthSessionClient = previousFactory
	})

	newNotionAuthSessionClient = func(sessionStore authcache.Store) (*notionadapter.Client, error) {
		return notionadapter.NewClient(notionadapter.Options{
			BaseURL: "https://api.notion.com",
			HTTPClient: &http.Client{
				Transport: &testRoundTripFunc{
					handler: func(request *http.Request) (*http.Response, error) {
						if request.URL.Path != "/v1/oauth/token" {
							t.Fatalf("unexpected request path: %s", request.URL.Path)
						}
						var payload map[string]any
						if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
							t.Fatalf("failed to decode refresh payload: %v", err)
						}
						if payload["refresh_token"] != "refresh-token" {
							t.Fatalf("unexpected refresh token: %+v", payload["refresh_token"])
						}
						return testJSONResponse(t, http.StatusOK, map[string]any{
							"access_token":  "fresh-token",
							"token_type":    "bearer",
							"refresh_token": "refresh-token-2",
							"expires_in":    3600,
						}), nil
					},
				},
			},
			SessionStore: sessionStore,
		})
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"auth", "session", "refresh", "notion_public_workspace_a"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"exists": true`)) {
		t.Fatalf("expected refreshed session output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"access_token": "fr***en"`)) {
		t.Fatalf("expected redacted cached access token output, got: %s", stdout.String())
	}

	sessionStore := authcache.NewFileStore(configPath)
	session, err := sessionStore.Load("notion_public_workspace_a")
	if err != nil {
		t.Fatalf("failed to load refreshed session: %v", err)
	}
	if session.AccessToken != "fresh-token" {
		t.Fatalf("unexpected cached access token: %s", session.AccessToken)
	}
	if session.RefreshToken != "refresh-token-2" {
		t.Fatalf("unexpected cached refresh token: %s", session.RefreshToken)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunAuthBeginNotionPublic(t *testing.T) {
	copyExampleConfig(t)

	result := runAuthBeginForTest(t, []string{
		"auth", "begin", "notion_public_workspace_a",
		"--mode", "manual_code",
		"--redirect-uri", "https://example.com/callback",
	})

	authURL := nestedString(result, "data", "flow", "authorization_url")
	if authURL == "" {
		t.Fatalf("expected authorization_url in output, got: %+v", result)
	}
	if !strings.HasPrefix(authURL, "https://api.notion.com/v1/oauth/authorize") {
		t.Fatalf("unexpected notion authorize url: %s", authURL)
	}
}

func TestRunAuthConnectOpensAuthorizationURL(t *testing.T) {
	copyExampleConfig(t)

	previousOpenAuthURL := openAuthURL
	t.Cleanup(func() {
		openAuthURL = previousOpenAuthURL
	})

	openedURL := ""
	openAuthURL = func(rawURL string) error {
		openedURL = rawURL
		return nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run([]string{
		"auth", "connect", "notion_public_workspace_a",
		"--mode", "manual_code",
		"--redirect-uri", "https://example.com/callback",
	}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	if openedURL == "" {
		t.Fatalf("expected auth connect to open authorization url, stdout=%s", stdout.String())
	}
	if !strings.HasPrefix(openedURL, "https://api.notion.com/v1/oauth/authorize") {
		t.Fatalf("unexpected opened url: %s", openedURL)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"opened": true`)) {
		t.Fatalf("expected browser opened result, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunAuthContinueNotionPublicWithCallbackURL(t *testing.T) {
	configPath := copyExampleConfig(t)
	runSecretSet(t, "notion_public_workspace_a", "client_secret", "client-secret")

	previousFactory := newNotionAuthFlowClient
	t.Cleanup(func() {
		newNotionAuthFlowClient = previousFactory
	})

	newNotionAuthFlowClient = func(sessionStore authcache.Store) (*notionadapter.Client, error) {
		return notionadapter.NewClient(notionadapter.Options{
			BaseURL: "https://api.notion.com",
			HTTPClient: &http.Client{
				Transport: &testRoundTripFunc{
					handler: func(request *http.Request) (*http.Response, error) {
						if request.URL.Path != "/v1/oauth/token" {
							t.Fatalf("unexpected request path: %s", request.URL.Path)
						}
						var payload map[string]any
						if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
							t.Fatalf("failed to decode auth code payload: %v", err)
						}
						if payload["grant_type"] != "authorization_code" {
							t.Fatalf("unexpected grant_type: %+v", payload["grant_type"])
						}
						if payload["code"] != "demo-code" {
							t.Fatalf("unexpected auth code: %+v", payload["code"])
						}
						return testJSONResponse(t, http.StatusOK, map[string]any{
							"access_token":   "fresh-token",
							"token_type":     "bearer",
							"refresh_token":  "refresh-token-2",
							"expires_in":     3600,
							"workspace_id":   "workspace_demo",
							"workspace_name": "Workspace Demo",
							"bot_id":         "bot_demo",
						}), nil
					},
				},
			},
			SessionStore: sessionStore,
		})
	}

	beginResult := runAuthBeginForTest(t, []string{
		"auth", "begin", "notion_public_workspace_a",
		"--mode", "manual_code",
		"--redirect-uri", "https://example.com/callback",
	})

	flowID := nestedString(beginResult, "data", "flow", "id")
	authURL := nestedString(beginResult, "data", "flow", "authorization_url")
	if flowID == "" || authURL == "" {
		t.Fatalf("expected flow id and authorization_url, got: %+v", beginResult)
	}

	parsedURL, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse authorization url: %v", err)
	}
	stateToken := parsedURL.Query().Get("state")
	if stateToken == "" {
		t.Fatalf("expected state token in authorization url: %s", authURL)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	callbackURL := "https://example.com/callback?code=demo-code&state=" + stateToken

	err = Run([]string{"auth", "continue", flowID, "--callback-url", callbackURL}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"state": "completed"`)) {
		t.Fatalf("expected completed flow output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"workspace_name": "Workspace Demo"`)) {
		t.Fatalf("expected workspace metadata in output, got: %s", stdout.String())
	}

	sessionStore := authcache.NewFileStore(configPath)
	session, err := sessionStore.Load("notion_public_workspace_a")
	if err != nil {
		t.Fatalf("failed to load exchanged session: %v", err)
	}
	if session.AccessToken != "fresh-token" {
		t.Fatalf("unexpected access token: %s", session.AccessToken)
	}
	if session.RefreshToken != "refresh-token-2" {
		t.Fatalf("unexpected refresh token in session: %s", session.RefreshToken)
	}

	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunDoctor(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("HOME", t.TempDir())

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"doctor"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"checks"`)) {
		t.Fatalf("expected checks in doctor output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"next_steps"`)) {
		t.Fatalf("expected next steps in doctor output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func newTestPluginManager(t *testing.T) *pluginruntime.Manager {
	t.Helper()

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu test client: %v", err)
	}
	notionClient, err := notionadapter.NewClient(notionadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct notion test client: %v", err)
	}

	feishuRegistry := adapter.NewRegistry()
	feishuadapter.RegisterOperations(feishuRegistry, feishuClient)

	notionRegistry := adapter.NewRegistry()
	notionadapter.RegisterOperations(notionRegistry, notionClient)

	manager, err := pluginruntime.NewManager(context.Background(), []pluginruntime.Runtime{
		pluginruntime.NewRegistryRuntime("feishu", "test", []string{"feishu"}, feishuRegistry, pluginruntime.FilterCatalogByPrefix(speccatalog.All(), "feishu.")),
		pluginruntime.NewRegistryRuntime("notion", "test", []string{"notion"}, notionRegistry, pluginruntime.FilterCatalogByPrefix(speccatalog.All(), "notion.")),
	})
	if err != nil {
		t.Fatalf("failed to construct test plugin manager: %v", err)
	}
	return manager
}

type testRoundTripFunc struct {
	handler func(request *http.Request) (*http.Response, error)
}

func (f *testRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f.handler(request)
}

func testJSONResponse(t *testing.T, statusCode int, value any) *http.Response {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal JSON response: %v", err)
	}
	return &http.Response{
		StatusCode: statusCode,
		Header: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
		},
		Body:          io.NopCloser(bytes.NewReader(data)),
		ContentLength: int64(len(data)),
		Request: &http.Request{
			Header: http.Header{},
		},
	}
}

func copyExampleConfig(t *testing.T) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("CLAWRISE_CONFIG", configPath)

	configBytes, err := os.ReadFile("../../examples/config.example.yaml")
	if err != nil {
		t.Fatalf("failed to read example config: %v", err)
	}
	if err := os.WriteFile(configPath, configBytes, 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	return configPath
}

func runSecretSet(t *testing.T, connectionName string, fieldName string, value string) {
	t.Helper()

	t.Setenv("CLAWRISE_MASTER_KEY", "test-master-key")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run([]string{"auth", "secret", "set", connectionName, fieldName, "--value", value}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("failed to seed secret via CLI: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}
}

func runAuthBeginForTest(t *testing.T, args []string) map[string]any {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run(args, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	result := map[string]any{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode begin result: %v; output=%s", err, stdout.String())
	}
	return result
}

func nestedString(data map[string]any, keys ...string) string {
	current := any(data)
	for _, key := range keys {
		record, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = record[key]
	}
	text, _ := current.(string)
	return text
}
