package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	feishuadapter "github.com/clawrise/clawrise-cli/internal/adapter/feishu"
	notionadapter "github.com/clawrise/clawrise-cli/internal/adapter/notion"
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"github.com/clawrise/clawrise-cli/internal/config"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
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
	if !bytes.Contains(stdout.Bytes(), []byte("clawrise docs generate [path] [--out-dir <dir>]")) {
		t.Fatalf("expected docs usage in root help, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("clawrise account [list|inspect|use|current|add|ensure|remove]")) {
		t.Fatalf("expected account ensure in root help, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("clawrise auth [list|methods|presets|inspect|check|login|complete|logout|secret]")) {
		t.Fatalf("expected auth usage in root help, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("clawrise account [list|inspect|use|current|add|ensure|remove]")) {
		t.Fatalf("expected account ensure usage in root help, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("clawrise secret [set|put|delete]")) {
		t.Fatalf("expected root secret usage in root help, got: %s", stdout.String())
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

func TestRunVersion(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"version"}, Dependencies{
		Version:       "test-version",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"version": "test-version"`)) {
		t.Fatalf("expected version output, got: %s", stdout.String())
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

func TestRunPlatformCurrentAndUnset(t *testing.T) {
	copyExampleConfig(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"platform", "current"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"platform": "feishu"`)) {
		t.Fatalf("expected default platform in output, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"platform", "unset"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"ok": true`)) {
		t.Fatalf("expected platform unset success, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"platform", "current"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"platform": null`)) {
		t.Fatalf("expected cleared platform, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunSubjectCurrentListAndUnset(t *testing.T) {
	copyExampleConfig(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"subject", "list"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"subjects": [`)) ||
		!bytes.Contains(stdout.Bytes(), []byte(`"bot"`)) ||
		!bytes.Contains(stdout.Bytes(), []byte(`"integration"`)) ||
		!bytes.Contains(stdout.Bytes(), []byte(`"user"`)) {
		t.Fatalf("expected sorted subject list, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"subject", "use", "integration"}, Dependencies{
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
	err = Run([]string{"subject", "current"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"subject": "integration"`)) {
		t.Fatalf("expected current subject output, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"subject", "unset"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"ok": true`)) {
		t.Fatalf("expected subject unset success, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"subject", "current"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"subject": null`)) {
		t.Fatalf("expected cleared subject, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunConfigInitSelectsMachinePresetByDefault(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"config", "init", "--platform", "notion"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"method": "notion.internal_token"`)) {
		t.Fatalf("expected config init to choose the machine preset, got: %s", stdout.String())
	}

	cfg, err := config.NewStore(configPath).Load()
	if err != nil {
		t.Fatalf("failed to load generated config: %v", err)
	}
	account, ok := cfg.Accounts["notion_integration_default"]
	if !ok {
		t.Fatalf("expected notion_integration_default account, got: %+v", cfg.Accounts)
	}
	if account.Auth.Method != "notion.internal_token" {
		t.Fatalf("unexpected auth method: %+v", account)
	}
	if cfg.Auth.SecretStore.Backend != "encrypted_file" {
		t.Fatalf("unexpected default secret backend: %+v", cfg.Auth.SecretStore)
	}
	if cfg.Plugins.Bindings.Storage.SecretStore.Backend != "encrypted_file" || cfg.Plugins.Bindings.Storage.SecretStore.Plugin != "builtin" {
		t.Fatalf("expected secret store binding to use new plugins model, got: %+v", cfg.Plugins.Bindings.Storage.SecretStore)
	}
	if cfg.Plugins.Bindings.Storage.SessionStore.Backend != "file" || cfg.Plugins.Bindings.Storage.SessionStore.Plugin != "builtin" {
		t.Fatalf("expected session store binding to use new plugins model, got: %+v", cfg.Plugins.Bindings.Storage.SessionStore)
	}
	if cfg.Defaults.Account != "notion_integration_default" || cfg.Defaults.Platform != "notion" {
		t.Fatalf("unexpected defaults: %+v", cfg.Defaults)
	}
}

func TestRunConfigInitUsesInteractivePresetScopes(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{
		"config", "init",
		"--platform", "feishu",
		"--subject", "user",
		"--scope", "offline_access",
		"--scope", "docx:document",
	}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"method": "feishu.oauth_user"`)) {
		t.Fatalf("expected config init to choose feishu.oauth_user, got: %s", stdout.String())
	}

	cfg, err := config.NewStore(configPath).Load()
	if err != nil {
		t.Fatalf("failed to load generated config: %v", err)
	}
	account, ok := cfg.Accounts["feishu_user_default"]
	if !ok {
		t.Fatalf("expected feishu_user_default account, got: %+v", cfg.Accounts)
	}
	if account.Auth.Method != "feishu.oauth_user" {
		t.Fatalf("unexpected auth method: %+v", account)
	}
	rawScopes, ok := account.Auth.Public["scopes"].([]any)
	if !ok {
		t.Fatalf("expected scopes to be stored as a list, got: %+v", account.Auth.Public["scopes"])
	}
	if len(rawScopes) != 2 || rawScopes[0] != "offline_access" || rawScopes[1] != "docx:document" {
		t.Fatalf("unexpected scopes: %+v", rawScopes)
	}
}

func TestRunConfigSecretStoreUseUpdatesBackend(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"config", "secret-store", "use", "keychain"}, Dependencies{
		Version: "test",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	cfg, err := config.NewStore(configPath).Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.Auth.SecretStore.Backend != "keychain" {
		t.Fatalf("unexpected secret backend: %+v", cfg.Auth.SecretStore)
	}
	if cfg.Plugins.Bindings.Storage.SecretStore.Backend != "keychain" || cfg.Plugins.Bindings.Storage.SecretStore.Plugin != "builtin" {
		t.Fatalf("expected plugins storage binding to be updated, got: %+v", cfg.Plugins.Bindings.Storage.SecretStore)
	}
	if cfg.Auth.SecretStore.FallbackBackend != "" {
		t.Fatalf("unexpected fallback backend: %+v", cfg.Auth.SecretStore)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"secret_backend": "keychain"`)) {
		t.Fatalf("expected config command output to include keychain backend, got: %s", stdout.String())
	}
}

func TestRunConfigProviderUseUpdatesBinding(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)

	pluginDir := filepath.Join(pluginRoot, "demo-provider", "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{
  "schema_version": 1,
  "name": "demo-provider",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./demo-provider"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write plugin manifest: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"config", "provider", "use", "demo", "demo-provider"}, Dependencies{
		Version: "test",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	cfg, err := config.NewStore(configPath).Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if got := cfg.Plugins.Bindings.Providers["demo"].Plugin; got != "demo-provider" {
		t.Fatalf("unexpected provider binding: %+v", cfg.Plugins.Bindings.Providers)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"plugin": "demo-provider"`)) {
		t.Fatalf("expected output to include provider plugin, got: %s", stdout.String())
	}
}

func TestRunConfigProviderUseRejectsDisabledPlugin(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)

	cfg := config.New()
	cfg.Ensure()
	cfg.Plugins.Enabled["demo-provider"] = "disabled"
	if err := config.NewStore(configPath).Save(cfg); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	pluginDir := filepath.Join(pluginRoot, "demo-provider", "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{
  "schema_version": 1,
  "name": "demo-provider",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./demo-provider"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write plugin manifest: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"config", "provider", "use", "demo", "demo-provider"}, Dependencies{
		Version: "test",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err == nil {
		t.Fatalf("expected disabled provider use command to fail, stdout=%s", stdout.String())
	}
	if err.Error() != "plugin demo-provider supports platform demo, but it is disabled by plugins.enabled" {
		t.Fatalf("unexpected error: %v", err)
	}

	updatedCfg, loadErr := config.NewStore(configPath).Load()
	if loadErr != nil {
		t.Fatalf("failed to reload config: %v", loadErr)
	}
	if _, exists := updatedCfg.Plugins.Bindings.Providers["demo"]; exists {
		t.Fatalf("expected provider binding to remain unset, got: %+v", updatedCfg.Plugins.Bindings.Providers)
	}
}

func TestRunConfigProviderUnsetRemovesBinding(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)

	cfg := config.New()
	cfg.Ensure()
	cfg.Plugins.Bindings.Providers["demo"] = config.ProviderPluginBinding{
		Plugin: "demo-provider",
	}
	if err := config.NewStore(configPath).Save(cfg); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"config", "provider", "unset", "demo"}, Dependencies{
		Version: "test",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	updatedCfg, err := config.NewStore(configPath).Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if _, exists := updatedCfg.Plugins.Bindings.Providers["demo"]; exists {
		t.Fatalf("expected provider binding to be removed, got: %+v", updatedCfg.Plugins.Bindings.Providers)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"unset": true`)) {
		t.Fatalf("expected unset marker in output, got: %s", stdout.String())
	}
}

func TestRunConfigAuthLauncherPreferUpdatesPreference(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)

	manager := newTestPluginManagerWithManagerOptions(t, []pluginruntime.AuthLauncherRuntime{
		&testAuthLauncherRuntime{
			descriptor: pluginruntime.AuthLauncherDescriptor{
				ID:          "browser",
				DisplayName: "Browser",
				ActionTypes: []string{"open_url"},
			},
		},
		&testAuthLauncherRuntime{
			descriptor: pluginruntime.AuthLauncherDescriptor{
				ID:          "device",
				DisplayName: "Device",
				ActionTypes: []string{"open_url"},
			},
		},
	}, map[string][]string{
		"open_url": {"browser"},
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"config", "auth-launcher", "prefer", "open_url", "device"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: manager,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	cfg, err := config.NewStore(configPath).Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	preferences := cfg.Plugins.Bindings.AuthLaunchers["open_url"]
	if len(preferences) != 1 || preferences[0] != "device" {
		t.Fatalf("unexpected auth launcher preferences: %+v", cfg.Plugins.Bindings.AuthLaunchers)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"launcher_id": "device"`)) {
		t.Fatalf("expected output to include launcher id, got: %s", stdout.String())
	}
}

func TestRunConfigAuthLauncherUnsetRemovesLauncherPreference(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)

	cfg := config.New()
	cfg.Ensure()
	cfg.Plugins.Bindings.AuthLaunchers["open_url"] = []string{"browser", "device"}
	if err := config.NewStore(configPath).Save(cfg); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"config", "auth-launcher", "unset", "open_url", "browser"}, Dependencies{
		Version: "test",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	updatedCfg, err := config.NewStore(configPath).Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	preferences := updatedCfg.Plugins.Bindings.AuthLaunchers["open_url"]
	if len(preferences) != 1 || preferences[0] != "device" {
		t.Fatalf("unexpected auth launcher preferences after unset: %+v", updatedCfg.Plugins.Bindings.AuthLaunchers)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"unset": true`)) {
		t.Fatalf("expected unset marker in output, got: %s", stdout.String())
	}
}

func TestRunAccountAddBootstrapsConfigByDefaultForNotion(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"account", "add", "notion_docs", "--platform", "notion"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	cfg, err := config.NewStore(configPath).Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	account, ok := cfg.Accounts["notion_docs"]
	if !ok {
		t.Fatalf("expected notion_docs account, got: %+v", cfg.Accounts)
	}
	if account.Auth.Method != "notion.internal_token" {
		t.Fatalf("unexpected auth method: %+v", account)
	}
	if cfg.Auth.SecretStore.Backend != "encrypted_file" {
		t.Fatalf("unexpected secret backend: %+v", cfg.Auth.SecretStore)
	}
	if cfg.Defaults.Account != "notion_docs" || cfg.Defaults.Platform != "notion" || cfg.Defaults.Subject != "integration" {
		t.Fatalf("unexpected defaults: %+v", cfg.Defaults)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"used_as_default": true`)) {
		t.Fatalf("expected account add output to mark first account as default, got: %s", stdout.String())
	}
}

func TestRunAccountAddChoosesBotPresetForFeishuByDefault(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"account", "add", "feishu_bot", "--platform", "feishu"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	cfg, err := config.NewStore(configPath).Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	account, ok := cfg.Accounts["feishu_bot"]
	if !ok {
		t.Fatalf("expected feishu_bot account, got: %+v", cfg.Accounts)
	}
	if account.Auth.Method != "feishu.app_credentials" {
		t.Fatalf("unexpected auth method: %+v", account)
	}
	if account.Subject != "bot" {
		t.Fatalf("unexpected subject: %+v", account)
	}
}

func TestRunAccountEnsureCreatesNotionAccount(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"account", "ensure", "notion_bot", "--platform", "notion", "--preset", "internal_token", "--use"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	cfg, err := config.NewStore(configPath).Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	account, ok := cfg.Accounts["notion_bot"]
	if !ok {
		t.Fatalf("expected notion_bot account, got: %+v", cfg.Accounts)
	}
	if account.Auth.Method != "notion.internal_token" {
		t.Fatalf("unexpected auth method: %+v", account)
	}
	if got := account.Auth.SecretRefs["token"]; got != "secret:notion_bot:token" {
		t.Fatalf("unexpected token secret ref: %q", got)
	}
	if cfg.Defaults.Account != "notion_bot" || cfg.Defaults.Platform != "notion" {
		t.Fatalf("unexpected defaults: %+v", cfg.Defaults)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"action": "created"`)) {
		t.Fatalf("expected created action in output, got: %s", stdout.String())
	}
}

func TestRunAccountEnsureUpdatesExistingFeishuAccount(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)

	cfg := config.New()
	cfg.Accounts["feishu_bot"] = config.Account{
		Title:    "旧账号标题",
		Platform: "feishu",
		Subject:  "bot",
		Auth: config.AccountAuth{
			Method: "feishu.app_credentials",
			Public: map[string]any{
				"app_id": "old-app-id",
			},
			SecretRefs: map[string]string{
				"app_secret": "secret:feishu_bot:app_secret",
			},
		},
	}
	if err := config.NewStore(configPath).Save(cfg); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"account", "ensure", "feishu_bot", "--platform", "feishu", "--preset", "bot", "--public", "app_id=new-app-id", "--use"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	updatedCfg, err := config.NewStore(configPath).Load()
	if err != nil {
		t.Fatalf("failed to load updated config: %v", err)
	}
	account := updatedCfg.Accounts["feishu_bot"]
	if got := account.Auth.Public["app_id"]; got != "new-app-id" {
		t.Fatalf("unexpected app_id after ensure: %+v", account.Auth.Public)
	}
	if account.Title != "旧账号标题" {
		t.Fatalf("expected title to be preserved, got: %+v", account)
	}
	if got := account.Auth.SecretRefs["app_secret"]; got != "secret:feishu_bot:app_secret" {
		t.Fatalf("unexpected app_secret ref: %q", got)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"action": "updated"`)) {
		t.Fatalf("expected updated action in output, got: %s", stdout.String())
	}
}

func TestRunAccountUseSynchronizesSubject(t *testing.T) {
	copyExampleConfig(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"account", "use", "feishu_user_alice"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"subject": "user"`)) {
		t.Fatalf("expected account use to expose user subject, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"subject", "current"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"subject": "user"`)) {
		t.Fatalf("expected account use to synchronize default subject, got: %s", stdout.String())
	}
}

func TestRunAccountUseUsesAccountCommandBehavior(t *testing.T) {
	copyExampleConfig(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"account", "use", "notion_bot"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"name": "notion_bot"`)) {
		t.Fatalf("expected account response payload, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"account"`)) {
		t.Fatalf("expected account payload shape, got: %s", stdout.String())
	}
}

func TestRunAccountUseSynchronizesPlatformForBareOperation(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("NOTION_BOT_TOKEN", "notion-token")

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

	err = Run([]string{"account", "use", "notion_bot"}, Dependencies{
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
	if !bytes.Contains(stdout.Bytes(), []byte(`"account": "notion_bot"`)) {
		t.Fatalf("expected notion account in output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"subject": "integration"`)) {
		t.Fatalf("expected integration subject in output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunAccountListInspectCurrentAndRemove(t *testing.T) {
	configPath := copyExampleConfig(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"account", "list"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"name": "feishu_bot"`)) || !bytes.Contains(stdout.Bytes(), []byte(`"name": "notion_bot"`)) {
		t.Fatalf("expected account list to include example accounts, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"account", "current"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"name": "feishu_bot"`)) {
		t.Fatalf("expected current account output, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"account", "inspect", "notion_bot"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"platform": "notion"`)) || !bytes.Contains(stdout.Bytes(), []byte(`"method": "notion.internal_token"`)) {
		t.Fatalf("expected notion account inspection payload, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"account", "remove", "notion_bot"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"deleted": true`)) {
		t.Fatalf("expected account remove success, got: %s", stdout.String())
	}

	cfg, err := config.NewStore(configPath).Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	if _, ok := cfg.Accounts["notion_bot"]; ok {
		t.Fatalf("expected notion_bot to be removed from config: %+v", cfg.Accounts)
	}
	if _, ok := cfg.Defaults.PlatformAccounts["notion"]; ok {
		t.Fatalf("expected notion platform binding to be cleared after remove: %+v", cfg.Defaults.PlatformAccounts)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunSubjectUseInfluencesBareOperationResolution(t *testing.T) {
	configPath := copyExampleConfig(t)
	t.Setenv("CLAWRISE_MASTER_KEY", "test-master-key")

	cfgStore := config.NewStore(configPath)
	cfg, err := cfgStore.Load()
	if err != nil {
		t.Fatalf("failed to load example config: %v", err)
	}
	cfg.Auth.SecretStore.Backend = "encrypted_file"
	cfg.Auth.SecretStore.FallbackBackend = "encrypted_file"
	if err := cfgStore.Save(cfg); err != nil {
		t.Fatalf("failed to persist encrypted secret backend: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err = Run([]string{"subject", "use", "user"}, Dependencies{
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
	err = Run([]string{"docs.document.create", "--dry-run", "--json", `{}`}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"account": "feishu_user_alice"`)) {
		t.Fatalf("expected user account to be selected after subject switch, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"subject": "user"`)) {
		t.Fatalf("expected user subject after subject switch, got: %s", stdout.String())
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

func TestRunSpecExportMarkdownToDirectory(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	outputDir := filepath.Join(t.TempDir(), "generated-spec")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"spec", "export", "notion.page", "--format", "markdown", "--out-dir", outputDir}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"written_files"`)) {
		t.Fatalf("expected written_files in export output, got: %s", stdout.String())
	}
	indexPath := filepath.Join(outputDir, "index.md")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("expected markdown index to be written: %v", err)
	}
	operationPath := filepath.Join(outputDir, "operations", "notion", "page", "create.md")
	if _, err := os.Stat(operationPath); err != nil {
		t.Fatalf("expected markdown operation file to be written: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunBatchDryRun(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", "../../examples/config.example.yaml")
	t.Setenv("FEISHU_BOT_OPS_APP_ID", "app-id")
	t.Setenv("FEISHU_BOT_OPS_APP_SECRET", "app-secret")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{
		"batch",
		"--json",
		`{
		  "requests": [
		    {
		      "operation": "feishu.calendar.event.create",
		      "dry_run": true,
		      "input": {
		        "calendar_id": "cal_demo",
		        "summary": "Batch Demo",
		        "start_at": "2026-03-30T10:00:00+08:00",
		        "end_at": "2026-03-30T11:00:00+08:00"
		      }
		    },
		    {
		      "operation": "feishu.calendar.event.create",
		      "dry_run": true,
		      "input": {
		        "calendar_id": "cal_demo",
		        "summary": "Batch Demo 2",
		        "start_at": "2026-03-30T12:00:00+08:00",
		        "end_at": "2026-03-30T13:00:00+08:00"
		      }
		    }
		  ]
		}`,
	}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"success_count": 2`)) {
		t.Fatalf("expected two successful batch requests, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"operation": "feishu.calendar.event.create"`)) {
		t.Fatalf("expected feishu operation in batch output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"Batch Demo 2"`)) {
		t.Fatalf("expected second batch payload in output, got: %s", stdout.String())
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
	if !bytes.Contains(stdout.Bytes(), []byte("docs")) {
		t.Fatalf("expected docs command completion entry, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("platform account subject auth secret config")) {
		t.Fatalf("expected root secret command in bash completion, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("list inspect use current add ensure remove")) {
		t.Fatalf("expected account ensure command in bash completion, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("init secret-store provider auth-launcher policy audit")) {
		t.Fatalf("expected full config command set in bash completion, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunCompletionZsh(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"completion", "zsh"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`#compdef clawrise`)) {
		t.Fatalf("expected zsh completion header, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`root_commands=('platform' 'account' 'subject' 'auth' 'secret' 'config' 'plugin' 'spec' 'docs' 'completion' 'doctor' 'version' 'batch'`)) {
		t.Fatalf("expected root secret command in zsh completion, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`account_commands=('list' 'inspect' 'use' 'current' 'add' 'ensure' 'remove')`)) {
		t.Fatalf("expected account ensure command in zsh completion, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`config_commands=('init' 'secret-store' 'provider' 'auth-launcher' 'policy' 'audit')`)) {
		t.Fatalf("expected full config command set in zsh completion, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`completion_shells=('bash' 'zsh' 'fish')`)) {
		t.Fatalf("expected shell variants in zsh completion, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`operation_flags=('--account' '--subject' '--json' '--input' '--timeout' '--dry-run' '--debug-provider-payload' '--verify'`)) {
		t.Fatalf("expected operation flags in zsh completion, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunCompletionFish(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"completion", "fish"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`complete -c clawrise -f`)) {
		t.Fatalf("expected fish completion prefix, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`complete -c clawrise -n '__fish_use_subcommand' -a 'platform'`)) {
		t.Fatalf("expected root command in fish completion, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`complete -c clawrise -n '__fish_use_subcommand' -a 'secret'`)) {
		t.Fatalf("expected root secret command in fish completion, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`complete -c clawrise -n '__fish_seen_subcommand_from account' -a 'ensure'`)) {
		t.Fatalf("expected account ensure command in fish completion, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`complete -c clawrise -n '__fish_seen_subcommand_from config' -a 'auth-launcher'`)) {
		t.Fatalf("expected config auth-launcher command in fish completion, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`complete -c clawrise -n '__fish_seen_subcommand_from config' -a 'audit'`)) {
		t.Fatalf("expected config audit command in fish completion, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`complete -c clawrise -n '__fish_seen_subcommand_from completion' -a 'bash'`)) {
		t.Fatalf("expected shell completion choices in fish completion, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunCompletionRejectsUnsupportedShell(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"completion", "powershell"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported completion shell") {
		t.Fatalf("expected unsupported shell error, got: %v", err)
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

func TestRunDocsHelpFlag(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"docs", "--help"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("Usage: clawrise docs generate [path] [--out-dir <dir>]")) {
		t.Fatalf("expected docs help output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunDocsGenerate(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	outputDir := filepath.Join(t.TempDir(), "generated")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"docs", "generate", "notion.page", "--out-dir", outputDir}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	if _, err := os.Stat(filepath.Join(outputDir, "index.md")); err != nil {
		t.Fatalf("expected generated index markdown, got error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "operations", "notion", "page", "create.md")); err != nil {
		t.Fatalf("expected generated operation markdown, got error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"output_dir"`)) {
		t.Fatalf("expected docs generate output to include output_dir, got: %s", stdout.String())
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

func TestRunAccountHelpFlag(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"account", "--help"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte("Usage: clawrise account [list|inspect|use|current|add|ensure|remove]")) {
		t.Fatalf("expected account help output, got: %s", stdout.String())
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

	if !bytes.Contains(stdout.Bytes(), []byte("Usage: clawrise plugin [list|install|info|remove|verify|upgrade]")) {
		t.Fatalf("expected plugin help output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunPluginUpgradeCommand(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	homeDir := t.TempDir()
	sourceDir := filepath.Join(t.TempDir(), "plugin-src")
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("HOME", homeDir)

	if err := os.MkdirAll(filepath.Join(sourceDir, "bin"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	writeManifest := func(version string) {
		t.Helper()
		manifest := `{
  "schema_version": 1,
  "name": "demo",
  "version": "` + version + `",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./bin/demo-plugin"]
  }
}`
		if err := os.WriteFile(filepath.Join(sourceDir, "plugin.json"), []byte(manifest), 0o644); err != nil {
			t.Fatalf("failed to write manifest: %v", err)
		}
		if err := os.WriteFile(filepath.Join(sourceDir, "bin", "demo-plugin"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("failed to write plugin binary: %v", err)
		}
	}

	writeManifest("0.1.0")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run([]string{"plugin", "install", sourceDir}, Dependencies{
		Version: "0.2.0",
		Stdout:  &stdout,
		Stderr:  &stderr,
	}); err != nil {
		t.Fatalf("plugin install returned error: %v", err)
	}

	writeManifest("0.2.0")
	stdout.Reset()
	stderr.Reset()

	if err := Run([]string{"plugin", "upgrade", "demo", "0.1.0"}, Dependencies{
		Version: "0.2.0",
		Stdout:  &stdout,
		Stderr:  &stderr,
	}); err != nil {
		t.Fatalf("plugin upgrade returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"to_version": "0.2.0"`)) {
		t.Fatalf("expected plugin upgrade output to include upgraded version, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunPluginUpgradeAllCommand(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	homeDir := t.TempDir()
	sourceDirA := filepath.Join(t.TempDir(), "plugin-src-a")
	sourceDirB := filepath.Join(t.TempDir(), "plugin-src-b")
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("HOME", homeDir)

	writeManifest := func(dir string, name string, version string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(dir, "bin"), 0o755); err != nil {
			t.Fatalf("failed to create source dir: %v", err)
		}
		manifest := `{
  "schema_version": 1,
  "name": "` + name + `",
  "version": "` + version + `",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./bin/demo-plugin"]
  }
}`
		if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(manifest), 0o644); err != nil {
			t.Fatalf("failed to write manifest: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "bin", "demo-plugin"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatalf("failed to write plugin binary: %v", err)
		}
	}

	writeManifest(sourceDirA, "demo-a", "0.1.0")
	writeManifest(sourceDirB, "demo-b", "0.1.0")

	if _, err := pluginruntime.InstallLocal(sourceDirA); err != nil {
		t.Fatalf("failed to install plugin A: %v", err)
	}
	if _, err := pluginruntime.InstallLocal(sourceDirB); err != nil {
		t.Fatalf("failed to install plugin B: %v", err)
	}

	writeManifest(sourceDirA, "demo-a", "0.2.0")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Run([]string{"plugin", "upgrade", "--all"}, Dependencies{
		Version: "0.2.0",
		Stdout:  &stdout,
		Stderr:  &stderr,
	}); err != nil {
		t.Fatalf("plugin upgrade --all returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"upgraded": 1`)) {
		t.Fatalf("expected plugin upgrade --all output to include one upgraded plugin, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"name": "demo-b"`)) {
		t.Fatalf("expected plugin upgrade --all output to include unchanged plugin results, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunPluginVerifyUsesConfiguredTrustPolicy(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	homeDir := t.TempDir()
	sourceDir := filepath.Join(t.TempDir(), "plugin-src")
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("HOME", homeDir)

	cfg := config.New()
	cfg.Plugins.Install.AllowedSources = []string{"https"}
	if err := config.NewStore(configPath).Save(cfg); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(sourceDir, "bin"), 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "plugin.json"), []byte(`{
  "schema_version": 1,
  "name": "demo",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./bin/demo-plugin"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "bin", "demo-plugin"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to write plugin binary: %v", err)
	}

	if _, err := pluginruntime.InstallLocal(sourceDir); err != nil {
		t.Fatalf("failed to install plugin: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run([]string{"plugin", "verify", "demo", "0.1.0"}, Dependencies{
		Version: "0.2.0",
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	if err == nil {
		t.Fatal("expected plugin verify to fail when current trust policy rejects the recorded source")
	}
	if exitErr, ok := err.(ExitError); !ok || exitErr.Code != 1 {
		t.Fatalf("expected verify command to exit with code 1, got: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"allowed": false`)) {
		t.Fatalf("expected verify output to expose trust rejection, got: %s", stdout.String())
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
		"--account", "notion_bot",
	}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"account": "notion_bot"`)) {
		t.Fatalf("expected account name in output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"method": "notion.internal_token"`)) {
		t.Fatalf("expected auth method in output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunAuthCheck(t *testing.T) {
	copyExampleConfig(t)
	runSecretSet(t, "feishu_bot", "app_secret", "app-secret")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"auth", "check", "feishu_bot"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"status": "ready"`)) {
		t.Fatalf("expected ready auth check output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"ready": true`)) {
		t.Fatalf("expected ready=true in auth output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunAuthCheckReportsAuthorizationPending(t *testing.T) {
	copyExampleConfig(t)
	runSecretSet(t, "notion_public_workspace_a", "client_secret", "client-secret")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"auth", "check", "notion_public_workspace_a"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err == nil {
		t.Fatalf("expected auth check to fail before interactive authorization, stdout=%s", stdout.String())
	}

	var exitErr ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("expected exit code 1, got: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"status": "authorization_required"`)) {
		t.Fatalf("expected authorization_required status, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"ready": false`)) {
		t.Fatalf("expected ready=false in auth check output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunAuthInspectUsesStableOutputShape(t *testing.T) {
	copyExampleConfig(t)
	runSecretSet(t, "notion_public_workspace_a", "client_secret", "client-secret")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"auth", "inspect", "notion_public_workspace_a"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("auth inspect returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	var payload map[string]any
	if decodeErr := json.Unmarshal(stdout.Bytes(), &payload); decodeErr != nil {
		t.Fatalf("failed to decode auth inspect output: %v", decodeErr)
	}
	data := payload["data"].(map[string]any)
	if _, ok := data["inspection"]; ok {
		t.Fatalf("expected flattened auth inspect payload, got nested inspection: %+v", data)
	}
	if data["status"] != "authorization_required" {
		t.Fatalf("unexpected status: %+v", data)
	}
	if data["recommended_action"] != "auth.login" {
		t.Fatalf("unexpected recommended_action: %+v", data)
	}
	if _, ok := data["next_actions"]; !ok {
		t.Fatalf("expected next_actions in auth inspect output: %+v", data)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunAuthMethodsAndPresets(t *testing.T) {
	copyExampleConfig(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"auth", "methods", "--platform", "notion"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"id": "notion.internal_token"`)) || !bytes.Contains(stdout.Bytes(), []byte(`"id": "notion.oauth_public"`)) {
		t.Fatalf("expected notion auth methods, got: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"auth", "presets", "--platform", "notion"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"id": "internal_token"`)) || !bytes.Contains(stdout.Bytes(), []byte(`"id": "public_oauth"`)) {
		t.Fatalf("expected notion auth presets, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunAuthLogoutClearsSessionAndTokenSecrets(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("CLAWRISE_STATE_DIR", filepath.Join(t.TempDir(), "state"))
	t.Setenv("CLAWRISE_MASTER_KEY", "test-master-key")

	cfg := config.New()
	cfg.Ensure()
	cfg.Auth.SecretStore.Backend = "encrypted_file"
	cfg.Auth.SecretStore.FallbackBackend = "encrypted_file"
	cfg.Auth.SessionStore.Backend = "file"
	cfg.Defaults.Account = "notion_oauth_live"
	cfg.Defaults.Platform = "notion"
	cfg.Defaults.Subject = "integration"
	cfg.Defaults.PlatformAccounts["notion"] = "notion_oauth_live"
	cfg.Accounts["notion_oauth_live"] = config.Account{
		Title:    "Notion OAuth Live",
		Platform: "notion",
		Subject:  "integration",
		Auth: config.AccountAuth{
			Method: "notion.oauth_public",
			Public: map[string]any{
				"client_id":      "demo-client",
				"notion_version": "2026-03-11",
				"redirect_mode":  "loopback",
			},
			SecretRefs: map[string]string{
				"client_secret": "secret:notion_oauth_live:client_secret",
				"access_token":  "secret:notion_oauth_live:access_token",
				"refresh_token": "secret:notion_oauth_live:refresh_token",
			},
		},
	}
	store := config.NewStore(configPath)
	if err := store.Save(cfg); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	secretStore, err := openCLISecretStore(cfg, store)
	if err != nil {
		t.Fatalf("failed to open secret store: %v", err)
	}
	for field, value := range map[string]string{
		"client_secret": "client-secret",
		"access_token":  "access-token",
		"refresh_token": "refresh-token",
	} {
		if err := secretStore.Set("notion_oauth_live", field, value); err != nil {
			t.Fatalf("failed to seed secret %s: %v", field, err)
		}
	}

	sessionStore, err := openCLISessionStore(cfg, store)
	if err != nil {
		t.Fatalf("failed to open session store: %v", err)
	}
	if err := sessionStore.Save(authcache.Session{
		AccountName:  "notion_oauth_live",
		Platform:     "notion",
		Subject:      "integration",
		GrantType:    "notion.oauth_public",
		AccessToken:  "session-access-token",
		RefreshToken: "session-refresh-token",
	}); err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = Run([]string{"auth", "logout", "notion_oauth_live"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"logged_out": true`)) {
		t.Fatalf("expected logout success output, got: %s", stdout.String())
	}

	if _, err := sessionStore.Load("notion_oauth_live"); err == nil {
		t.Fatal("expected auth logout to remove stored session")
	}
	if _, err := secretStore.Get("notion_oauth_live", "access_token"); err == nil {
		t.Fatal("expected auth logout to delete access_token secret")
	}
	if _, err := secretStore.Get("notion_oauth_live", "refresh_token"); err == nil {
		t.Fatal("expected auth logout to delete refresh_token secret")
	}
	if value, err := secretStore.Get("notion_oauth_live", "client_secret"); err != nil || value != "client-secret" {
		t.Fatalf("expected client_secret to be preserved, got value=%q err=%v", value, err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunAuthBeginNotionPublic(t *testing.T) {
	copyExampleConfig(t)
	runSecretSet(t, "notion_public_workspace_a", "client_secret", "client-secret")

	result := runAuthLoginForTest(t, []string{
		"auth", "login", "notion_public_workspace_a",
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

func TestRunAuthLoginUsesLoopbackRedirectModeDefault(t *testing.T) {
	copyExampleConfig(t)
	runSecretSet(t, "notion_public_workspace_a", "client_secret", "client-secret")

	result := runAuthLoginForTest(t, []string{
		"auth", "login", "notion_public_workspace_a",
		"--mode", "manual_code",
	})

	if got := nestedString(result, "data", "flow", "redirect_uri"); got != "http://localhost:3333/callback" {
		t.Fatalf("unexpected default redirect_uri: %+v", result)
	}
	authURL := nestedString(result, "data", "flow", "authorization_url")
	if !strings.Contains(authURL, "redirect_uri=http%3A%2F%2Flocalhost%3A3333%2Fcallback") {
		t.Fatalf("expected loopback redirect uri in authorization url, got: %s", authURL)
	}
}

func TestRunAuthLoginOpensAuthorizationURL(t *testing.T) {
	copyExampleConfig(t)
	runSecretSet(t, "notion_public_workspace_a", "client_secret", "client-secret")
	launcher := &testAuthLauncherRuntime{
		descriptor: pluginruntime.AuthLauncherDescriptor{
			ID:          "test_launcher",
			DisplayName: "test launcher",
			ActionTypes: []string{"open_url"},
			Priority:    100,
		},
		launch: func(params pluginruntime.AuthLaunchParams) (pluginruntime.AuthLaunchResult, error) {
			return pluginruntime.AuthLaunchResult{
				Handled:    true,
				Status:     "launched",
				LauncherID: "test_launcher",
				Metadata: map[string]any{
					"url": params.Action.URL,
				},
			}, nil
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run([]string{
		"auth", "login", "notion_public_workspace_a",
		"--mode", "manual_code",
		"--redirect-uri", "https://example.com/callback",
	}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManagerWithLaunchers(t, []pluginruntime.AuthLauncherRuntime{launcher}),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	openedURL := nestedStringFromBytes(t, stdout.Bytes(), "data", "launcher", "metadata", "url")
	if !strings.HasPrefix(openedURL, "https://api.notion.com/v1/oauth/authorize") {
		t.Fatalf("unexpected opened url: %s", openedURL)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"opened": true`)) {
		t.Fatalf("expected browser opened result, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"launcher_id": "test_launcher"`)) {
		t.Fatalf("expected launcher result in output, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunAuthLoginAndCompleteWithDeviceCode(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("CLAWRISE_CONFIG", configPath)
	configYAML := `defaults:
  platform: device
  account: device_user
  platform_accounts:
    device: device_user

auth:
  secret_store:
    backend: encrypted_file
    fallback_backend: encrypted_file
  session_store:
    backend: file

accounts:
  device_user:
    title: 设备码测试账号
    platform: device
    subject: user
    auth:
      method: device.oauth
      public:
        client_id: demo-client
`
	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		t.Fatalf("failed to write device test config: %v", err)
	}

	var launchedAction pluginruntime.AuthLaunchParams
	launcher := &testAuthLauncherRuntime{
		descriptor: pluginruntime.AuthLauncherDescriptor{
			ID:          "device_launcher",
			DisplayName: "device launcher",
			ActionTypes: []string{"device_code"},
			Priority:    100,
		},
		launch: func(params pluginruntime.AuthLaunchParams) (pluginruntime.AuthLaunchResult, error) {
			launchedAction = params
			return pluginruntime.AuthLaunchResult{
				Handled:    true,
				Status:     "launched",
				LauncherID: "device_launcher",
				Metadata: map[string]any{
					"verification_url": params.Action.VerificationURL,
					"user_code":        params.Action.UserCode,
				},
			}, nil
		},
	}

	provider := &testAuthProvider{
		listMethods: func(ctx context.Context) ([]pluginruntime.AuthMethodDescriptor, error) {
			return []pluginruntime.AuthMethodDescriptor{
				{
					ID:               "device.oauth",
					Platform:         "device",
					DisplayName:      "Device OAuth",
					Subjects:         []string{"user"},
					Kind:             "interactive",
					Interactive:      true,
					InteractiveModes: []string{"device_code"},
				},
			}, nil
		},
		begin: func(ctx context.Context, params pluginruntime.AuthBeginParams) (pluginruntime.AuthBeginResult, error) {
			if params.Mode != "device_code" {
				t.Fatalf("expected auth login to prefer device_code mode, got: %+v", params)
			}
			return pluginruntime.AuthBeginResult{
				HumanRequired: true,
				Flow: pluginruntime.AuthFlowPayload{
					ID:              "flow_device_code_demo",
					Method:          "device.oauth",
					Mode:            "device_code",
					State:           "awaiting_user_action",
					DeviceCode:      "device-code-demo",
					UserCode:        "ABCD-EFGH",
					VerificationURL: "https://auth.example.com/verify",
					IntervalSec:     5,
					ExpiresAt:       "2026-03-30T10:00:00Z",
				},
				NextActions: []pluginruntime.AuthAction{
					{
						Type:            "device_code",
						Message:         "请在浏览器中输入用户码完成授权",
						DeviceCode:      "device-code-demo",
						UserCode:        "ABCD-EFGH",
						VerificationURL: "https://auth.example.com/verify",
						IntervalSec:     5,
					},
				},
			}, nil
		},
		complete: func(ctx context.Context, params pluginruntime.AuthCompleteParams) (pluginruntime.AuthCompleteResult, error) {
			if params.Flow.Mode != "device_code" {
				t.Fatalf("expected device_code mode in complete params, got: %+v", params.Flow)
			}
			if params.Flow.DeviceCode != "device-code-demo" {
				t.Fatalf("expected persisted device_code in complete params, got: %+v", params.Flow)
			}
			if params.Flow.UserCode != "ABCD-EFGH" || params.Flow.VerificationURL != "https://auth.example.com/verify" {
				t.Fatalf("expected persisted device code fields in complete params, got: %+v", params.Flow)
			}
			return pluginruntime.AuthCompleteResult{
				Ready:  true,
				Status: "ready",
				ExecutionAuth: map[string]any{
					"type":         "resolved_access_token",
					"access_token": "device-token",
				},
			}, nil
		},
		resolve: func(ctx context.Context, params pluginruntime.AuthResolveParams) (pluginruntime.AuthResolveResult, error) {
			return pluginruntime.AuthResolveResult{
				Ready:             false,
				Status:            "authorization_required",
				HumanRequired:     true,
				RecommendedAction: "auth.login",
			}, nil
		},
	}

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu test client: %v", err)
	}
	deviceRuntime := pluginruntime.NewRegistryRuntimeWithOptions(
		"device",
		"test",
		[]string{"device"},
		adapter.NewRegistry(),
		nil,
		pluginruntime.RegistryRuntimeOptions{
			AuthProvider: provider,
		},
	)
	manager := newTestPluginManagerWithOptions(t, feishuClient, nil, []pluginruntime.AuthLauncherRuntime{launcher}, []pluginruntime.Runtime{deviceRuntime}, nil)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = Run([]string{"auth", "login", "device_user"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: manager,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	if launchedAction.Action.Type != "device_code" {
		t.Fatalf("expected launcher to receive device_code action, got: %+v", launchedAction)
	}
	if launchedAction.Action.UserCode != "ABCD-EFGH" {
		t.Fatalf("expected launcher to receive user_code, got: %+v", launchedAction)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"mode": "device_code"`)) {
		t.Fatalf("expected device_code mode in auth login output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"user_code": "ABCD-EFGH"`)) {
		t.Fatalf("expected user_code in auth login output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"launcher_id": "device_launcher"`)) {
		t.Fatalf("expected launcher result in auth login output, got: %s", stdout.String())
	}

	var loginResult map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &loginResult); err != nil {
		t.Fatalf("failed to decode login result: %v", err)
	}
	flowID := nestedString(loginResult, "data", "flow", "id")
	if flowID != "flow_device_code_demo" {
		t.Fatalf("unexpected flow id: %s", flowID)
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"auth", "complete", flowID}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: manager,
	})
	if err != nil {
		t.Fatalf("auth complete returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"status": "ready"`)) {
		t.Fatalf("expected ready status in auth complete output, got: %s", stdout.String())
	}
}

func TestSelectPreferredInteractiveMode(t *testing.T) {
	if mode := selectPreferredInteractiveMode([]string{"manual_code", "device_code", "local_browser"}); mode != "device_code" {
		t.Fatalf("expected device_code to win, got: %s", mode)
	}
	if mode := selectPreferredInteractiveMode([]string{"manual_code", "local_browser"}); mode != "local_browser" {
		t.Fatalf("expected local_browser to win when device_code is unavailable, got: %s", mode)
	}
	if mode := selectPreferredInteractiveMode([]string{"manual_url", "custom_mode"}); mode != "manual_url" {
		t.Fatalf("expected manual_url to win over unknown modes, got: %s", mode)
	}
	if mode := selectPreferredInteractiveMode([]string{"custom_mode"}); mode != "custom_mode" {
		t.Fatalf("expected first unknown mode to be preserved, got: %s", mode)
	}
	if mode := selectPreferredInteractiveMode(nil); mode != "" {
		t.Fatalf("expected empty mode for empty input, got: %s", mode)
	}
}

func TestRunAuthCompleteNotionPublicWithCallbackURL(t *testing.T) {
	configPath := copyExampleConfig(t)
	runSecretSet(t, "notion_public_workspace_a", "client_secret", "client-secret")

	beginResult := runAuthLoginForTest(t, []string{
		"auth", "login", "notion_public_workspace_a",
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

	manager := newTestPluginManagerWithNotionClient(t, func(sessionStore authcache.Store) (*notionadapter.Client, error) {
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
	})

	err = Run([]string{"auth", "complete", flowID, "--callback-url", callbackURL}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: manager,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"status": "ready"`)) {
		t.Fatalf("expected ready result output, got: %s", stdout.String())
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

func TestRunAuthSecretSetFromEnv(t *testing.T) {
	configPath := copyExampleConfig(t)
	t.Setenv("CLAWRISE_MASTER_KEY", "test-master-key")
	t.Setenv("TEST_SECRET_VALUE", "secret-from-env")

	store := config.NewStore(configPath)
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	cfg.Auth.SecretStore.Backend = "encrypted_file"
	cfg.Auth.SecretStore.FallbackBackend = "encrypted_file"
	if err := store.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = Run([]string{"auth", "secret", "set", "notion_bot", "token", "--from-env", "TEST_SECRET_VALUE"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	cfg, err = store.Load()
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}
	secretStore, err := openCLISecretStore(cfg, store)
	if err != nil {
		t.Fatalf("failed to open secret store: %v", err)
	}
	value, err := secretStore.Get("notion_bot", "token")
	if err != nil {
		t.Fatalf("failed to read stored secret: %v", err)
	}
	if value != "secret-from-env" {
		t.Fatalf("unexpected secret value: %s", value)
	}
}

func TestRunAuthSecretSetValueRequiresExplicitInsecureFlag(t *testing.T) {
	copyExampleConfig(t)
	t.Setenv("CLAWRISE_MASTER_KEY", "test-master-key")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run([]string{"auth", "secret", "set", "notion_bot", "token", "--value", "plain-secret"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err == nil {
		t.Fatalf("expected command to fail without explicit insecure flag, stdout=%s", stdout.String())
	}
	if !strings.Contains(err.Error(), "--value requires --allow-insecure-cli-secret") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunAuthSecretDelete(t *testing.T) {
	configPath := copyExampleConfig(t)
	t.Setenv("CLAWRISE_MASTER_KEY", "test-master-key")

	store := config.NewStore(configPath)
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	secretStore, err := openCLISecretStore(cfg, store)
	if err != nil {
		t.Fatalf("failed to open secret store: %v", err)
	}
	if err := secretStore.Set("notion_bot", "token", "secret-to-delete"); err != nil {
		t.Fatalf("failed to seed secret: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = Run([]string{"auth", "secret", "delete", "notion_bot", "token"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"deleted": true`)) {
		t.Fatalf("expected delete success output, got: %s", stdout.String())
	}
	if _, err := secretStore.Get("notion_bot", "token"); err == nil {
		t.Fatal("expected auth secret delete to remove stored secret")
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
	if bytes.Contains(stdout.Bytes(), []byte(`"auth":`)) {
		t.Fatalf("expected doctor account items to expose flattened auth fields, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunDoctorShowsResolvedAuthLauncherPreferences(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("HOME", t.TempDir())

	cfg := config.New()
	cfg.Ensure()
	cfg.Defaults.Account = "demo_account"
	cfg.Accounts["demo_account"] = config.Account{
		Platform: "notion",
		Subject:  "integration",
		Auth: config.AccountAuth{
			Method:     "notion.internal_token",
			SecretRefs: map[string]string{"token": "secret://demo_account/token"},
		},
	}
	cfg.Plugins.Bindings.AuthLaunchers["open_url"] = []string{"device"}
	if err := config.NewStore(configPath).Save(cfg); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	manager := newTestPluginManagerWithManagerOptions(t, []pluginruntime.AuthLauncherRuntime{
		&testAuthLauncherRuntime{
			descriptor: pluginruntime.AuthLauncherDescriptor{
				ID:          "browser",
				DisplayName: "Browser",
				ActionTypes: []string{"open_url"},
				Priority:    100,
			},
		},
		&testAuthLauncherRuntime{
			descriptor: pluginruntime.AuthLauncherDescriptor{
				ID:          "device",
				DisplayName: "Device",
				ActionTypes: []string{"open_url"},
				Priority:    1,
			},
		},
	}, map[string][]string{
		"open_url": {"device"},
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run([]string{"doctor"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: manager,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	var payload map[string]any
	if decodeErr := json.Unmarshal(stdout.Bytes(), &payload); decodeErr != nil {
		t.Fatalf("failed to decode doctor output: %v", decodeErr)
	}

	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected doctor payload: %+v", payload)
	}
	runtimeData, ok := data["runtime"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected runtime payload: %+v", data)
	}
	preferences, ok := runtimeData["auth_launcher_preferences"].(map[string]any)
	if !ok {
		t.Fatalf("expected auth launcher preferences in doctor output, got: %+v", runtimeData)
	}
	openURL, ok := preferences["open_url"].(map[string]any)
	if !ok {
		t.Fatalf("expected open_url preference summary, got: %+v", preferences)
	}
	resolved, ok := openURL["resolved"].([]any)
	if !ok || len(resolved) < 2 {
		t.Fatalf("expected resolved launcher order, got: %+v", openURL)
	}
	first, ok := resolved[0].(map[string]any)
	if !ok || first["id"] != "device" {
		t.Fatalf("expected configured launcher to be resolved first, got: %+v", resolved)
	}
}

func TestRunDoctorShowsPolicyAndAuditChains(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAWRISE_TEST_NOTION_TOKEN", "notion-token")

	cfg := config.New()
	cfg.Ensure()
	cfg.Defaults.Account = "demo_account"
	cfg.Runtime.Policy = config.PolicyConfig{
		Mode:           "manual",
		DenyOperations: []string{"demo.page.delete"},
		Plugins: []config.PolicyPluginBinding{
			{Plugin: "policy-demo", PolicyID: "review"},
		},
	}
	cfg.Runtime.Audit = config.AuditConfig{
		Mode: "manual",
		Sinks: []config.AuditSinkConfig{
			{Type: "stdout"},
			{Plugin: "audit-demo", SinkID: "capture"},
		},
	}
	cfg.Accounts["demo_account"] = config.Account{
		Platform: "notion",
		Subject:  "integration",
		Auth: config.AccountAuth{
			Method:     "notion.internal_token",
			SecretRefs: map[string]string{"token": "env:CLAWRISE_TEST_NOTION_TOKEN"},
		},
	}
	if err := config.NewStore(configPath).Save(cfg); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	policyDir := filepath.Join(pluginRoot, "policy-demo", "0.1.0")
	if err := os.MkdirAll(policyDir, 0o755); err != nil {
		t.Fatalf("failed to create policy plugin dir: %v", err)
	}
	policyScript := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.handshake"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"protocol_version":1,"name":"policy-demo","version":"0.1.0","platforms":["demo"]}}'"\n"
      ;;
    *'"method":"clawrise.capabilities.list"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"capabilities":[{"type":"policy","id":"review","priority":80,"platforms":["demo"]}]}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(filepath.Join(policyDir, "policy-demo.sh"), []byte(policyScript), 0o755); err != nil {
		t.Fatalf("failed to write policy plugin executable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "plugin.json"), []byte(`{
  "schema_version": 2,
  "name": "policy-demo",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "policy",
      "id": "review",
      "priority": 80,
      "platforms": ["demo"]
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./policy-demo.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write policy plugin manifest: %v", err)
	}

	auditDir := filepath.Join(pluginRoot, "audit-demo", "0.1.0")
	if err := os.MkdirAll(auditDir, 0o755); err != nil {
		t.Fatalf("failed to create audit plugin dir: %v", err)
	}
	auditScript := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.handshake"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"protocol_version":1,"name":"audit-demo","version":"0.1.0","platforms":[]}}'"\n"
      ;;
    *'"method":"clawrise.capabilities.list"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"capabilities":[{"type":"audit_sink","id":"capture","priority":50}]}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(filepath.Join(auditDir, "audit-demo.sh"), []byte(auditScript), 0o755); err != nil {
		t.Fatalf("failed to write audit plugin executable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(auditDir, "plugin.json"), []byte(`{
  "schema_version": 2,
  "name": "audit-demo",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "audit_sink",
      "id": "capture",
      "priority": 50
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./audit-demo.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write audit plugin manifest: %v", err)
	}

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

	var payload map[string]any
	if decodeErr := json.Unmarshal(stdout.Bytes(), &payload); decodeErr != nil {
		t.Fatalf("failed to decode doctor output: %v", decodeErr)
	}
	data := payload["data"].(map[string]any)
	runtimeData := data["runtime"].(map[string]any)

	policyData, ok := runtimeData["policy"].(map[string]any)
	if !ok {
		t.Fatalf("expected policy summary in doctor output, got: %+v", runtimeData)
	}
	activePolicy, ok := policyData["active_chain"].([]any)
	if !ok || len(activePolicy) != 1 {
		t.Fatalf("expected one active policy runtime, got: %+v", policyData)
	}
	firstPolicy := activePolicy[0].(map[string]any)
	if firstPolicy["plugin"] != "policy-demo" || firstPolicy["policy_id"] != "review" {
		t.Fatalf("unexpected policy chain item: %+v", firstPolicy)
	}
	localPolicy := policyData["local"].(map[string]any)
	if localPolicy["rule_count"] != float64(1) {
		t.Fatalf("expected local policy summary to report one rule, got: %+v", localPolicy)
	}

	auditData, ok := runtimeData["audit"].(map[string]any)
	if !ok {
		t.Fatalf("expected audit summary in doctor output, got: %+v", runtimeData)
	}
	activeSinks, ok := auditData["active_sinks"].([]any)
	if !ok || len(activeSinks) != 2 {
		t.Fatalf("expected two active audit sinks, got: %+v", auditData)
	}
	firstSink := activeSinks[0].(map[string]any)
	secondSink := activeSinks[1].(map[string]any)
	if firstSink["type"] != "stdout" {
		t.Fatalf("expected builtin stdout sink to appear first, got: %+v", firstSink)
	}
	if secondSink["plugin"] != "audit-demo" || secondSink["sink_id"] != "capture" {
		t.Fatalf("unexpected plugin audit sink summary: %+v", secondSink)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunDoctorReportsDisabledPluginBinding(t *testing.T) {
	configPath := t.TempDir() + "/config.yaml"
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())

	cfg := config.New()
	cfg.Ensure()
	cfg.Plugins.Enabled["demo-provider"] = "disabled"
	cfg.Plugins.Bindings.Providers["demo"] = config.ProviderPluginBinding{
		Plugin: "demo-provider",
	}
	if err := config.NewStore(configPath).Save(cfg); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	pluginDir := filepath.Join(pluginRoot, "demo-provider", "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "demo-provider.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to write plugin executable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{
  "schema_version": 1,
  "name": "demo-provider",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./demo-provider.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write plugin manifest: %v", err)
	}

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

	if !bytes.Contains(stdout.Bytes(), []byte(`"enabled": false`)) {
		t.Fatalf("expected doctor output to mark plugin as disabled, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"selected": false`)) {
		t.Fatalf("expected doctor output to mark plugin as unselected, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`disabled by plugins.enabled`)) {
		t.Fatalf("expected disabled binding warning, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunConfigPolicyCommandsWriteBack(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())

	cfg := config.New()
	if err := config.NewStore(configPath).Save(cfg); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}
	writePolicyCapabilityFixture(t, pluginRoot, "policy-demo", "review")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run([]string{"config", "policy", "mode", "manual"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("config policy mode returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"config", "policy", "use", "policy-demo", "--policy-id", "review"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("config policy use returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	store := config.NewStore(configPath)
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("failed to load written config: %v", err)
	}
	if loaded.Runtime.Policy.Mode != "manual" {
		t.Fatalf("expected manual policy mode, got: %+v", loaded.Runtime.Policy)
	}
	if len(loaded.Runtime.Policy.Plugins) != 1 || loaded.Runtime.Policy.Plugins[0].Plugin != "policy-demo" || loaded.Runtime.Policy.Plugins[0].PolicyID != "review" {
		t.Fatalf("unexpected policy plugin bindings: %+v", loaded.Runtime.Policy.Plugins)
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"config", "policy", "remove", "policy-demo", "--policy-id", "review"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("config policy remove returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	loaded, err = store.Load()
	if err != nil {
		t.Fatalf("failed to reload written config: %v", err)
	}
	if len(loaded.Runtime.Policy.Plugins) != 0 {
		t.Fatalf("expected policy selectors to be removed, got: %+v", loaded.Runtime.Policy.Plugins)
	}
}

func TestRunConfigAuditCommandsWriteBack(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())

	cfg := config.New()
	if err := config.NewStore(configPath).Save(cfg); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}
	writeAuditSinkCapabilityFixture(t, pluginRoot, "audit-demo", "capture")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run([]string{"config", "audit", "mode", "manual"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("config audit mode returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"config", "audit", "add", "stdout"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("config audit add stdout returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"config", "audit", "add", "webhook", "env:CLAWRISE_AUDIT_WEBHOOK_URL", "--header", "Authorization=env:CLAWRISE_AUDIT_TOKEN", "--timeout-ms", "3000"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("config audit add webhook returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"config", "audit", "add", "plugin", "audit-demo", "--sink-id", "capture"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("config audit add plugin returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	store := config.NewStore(configPath)
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("failed to load written config: %v", err)
	}
	if loaded.Runtime.Audit.Mode != "manual" {
		t.Fatalf("expected manual audit mode, got: %+v", loaded.Runtime.Audit)
	}
	if len(loaded.Runtime.Audit.Sinks) != 3 {
		t.Fatalf("expected three audit sinks, got: %+v", loaded.Runtime.Audit.Sinks)
	}
	if loaded.Runtime.Audit.Sinks[0].Type != "stdout" {
		t.Fatalf("unexpected first audit sink: %+v", loaded.Runtime.Audit.Sinks[0])
	}
	if loaded.Runtime.Audit.Sinks[1].Type != "webhook" || loaded.Runtime.Audit.Sinks[1].URL != "env:CLAWRISE_AUDIT_WEBHOOK_URL" {
		t.Fatalf("unexpected webhook sink: %+v", loaded.Runtime.Audit.Sinks[1])
	}
	if loaded.Runtime.Audit.Sinks[2].Plugin != "audit-demo" || loaded.Runtime.Audit.Sinks[2].SinkID != "capture" {
		t.Fatalf("unexpected plugin sink: %+v", loaded.Runtime.Audit.Sinks[2])
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"config", "audit", "remove", "plugin", "audit-demo", "--sink-id", "capture"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("config audit remove plugin returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	loaded, err = store.Load()
	if err != nil {
		t.Fatalf("failed to reload written config: %v", err)
	}
	if len(loaded.Runtime.Audit.Sinks) != 2 {
		t.Fatalf("expected plugin audit sink to be removed, got: %+v", loaded.Runtime.Audit.Sinks)
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"config", "audit", "remove", "webhook", "env:CLAWRISE_AUDIT_WEBHOOK_URL"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("config audit remove webhook returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"config", "audit", "remove", "stdout"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("config audit remove stdout returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	loaded, err = store.Load()
	if err != nil {
		t.Fatalf("failed to reload written config after removals: %v", err)
	}
	if len(loaded.Runtime.Audit.Sinks) != 0 {
		t.Fatalf("expected all audit sinks to be removed, got: %+v", loaded.Runtime.Audit.Sinks)
	}
}

func TestRunConfigAuditAndAuthSecretHelpFlags(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	for _, tc := range []struct {
		args     []string
		expected string
	}{
		{[]string{"config", "audit", "--help"}, "Usage: clawrise config audit mode <auto|manual|disabled>"},
		{[]string{"auth", "secret", "--help"}, "Usage: clawrise auth secret [set|put|delete] <account> <field>"},
	} {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := Run(tc.args, Dependencies{
				Version:       "test",
				Stdout:        &stdout,
				Stderr:        &stderr,
				PluginManager: newTestPluginManager(t),
			})
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			if !bytes.Contains(stdout.Bytes(), []byte(tc.expected)) {
				t.Fatalf("expected help output %q, got: %s", tc.expected, stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("expected empty stderr, got: %s", stderr.String())
			}
		})
	}
}

func TestRunAccountHelpIncludesEnsure(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"account", "--help"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Usage: clawrise account [list|inspect|use|current|add|ensure|remove]")) {
		t.Fatalf("expected account ensure in account help, got: %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}

func TestRunDoctorUsesLocatorResolvedPaths(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")
	t.Setenv("CLAWRISE_CONFIG", configPath)
	stateDir := filepath.Join(t.TempDir(), "env-state")
	t.Setenv("CLAWRISE_STATE_DIR", stateDir)

	if err := os.WriteFile(configPath, []byte("defaults: {}\n"), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"doctor"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}

	expectedRuntimeDir := filepath.Join(stateDir, "runtime")
	if !bytes.Contains(stdout.Bytes(), []byte(stateDir)) {
		t.Fatalf("expected doctor output to expose resolved state dir, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(expectedRuntimeDir)) {
		t.Fatalf("expected doctor output to expose resolved runtime dir, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"source": "env.CLAWRISE_STATE_DIR"`)) {
		t.Fatalf("expected doctor output to expose locator source, got: %s", stdout.String())
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
	return newTestPluginManagerWithOptions(t, feishuClient, nil, nil, nil, nil)
}

func newTestPluginManagerWithNotionClient(t *testing.T, factory func(sessionStore authcache.Store) (*notionadapter.Client, error)) *pluginruntime.Manager {
	t.Helper()

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu test client: %v", err)
	}
	return newTestPluginManagerWithOptions(t, feishuClient, factory, nil, nil, nil)
}

func newTestPluginManagerWithLaunchers(t *testing.T, launchers []pluginruntime.AuthLauncherRuntime) *pluginruntime.Manager {
	return newTestPluginManagerWithManagerOptions(t, launchers, nil)
}

func newTestPluginManagerWithManagerOptions(t *testing.T, launchers []pluginruntime.AuthLauncherRuntime, preferences map[string][]string) *pluginruntime.Manager {
	t.Helper()

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu test client: %v", err)
	}
	return newTestPluginManagerWithOptions(t, feishuClient, nil, launchers, nil, preferences)
}

func newTestPluginManagerWithOptions(t *testing.T, feishuClient *feishuadapter.Client, notionFactory func(sessionStore authcache.Store) (*notionadapter.Client, error), launchers []pluginruntime.AuthLauncherRuntime, extraRuntimes []pluginruntime.Runtime, preferences map[string][]string) *pluginruntime.Manager {
	t.Helper()

	notionClient, err := notionadapter.NewClient(notionadapter.Options{})
	if notionFactory != nil {
		notionClient, err = notionFactory(nil)
	}
	if err != nil {
		t.Fatalf("failed to construct notion test client: %v", err)
	}

	feishuRegistry := adapter.NewRegistry()
	feishuadapter.RegisterOperations(feishuRegistry, feishuClient)

	notionRegistry := adapter.NewRegistry()
	notionadapter.RegisterOperations(notionRegistry, notionClient)

	runtimes := []pluginruntime.Runtime{
		pluginruntime.NewRegistryRuntimeWithOptions("feishu", "test", []string{"feishu"}, feishuRegistry, pluginruntime.CatalogFromRegistry(feishuRegistry), pluginruntime.RegistryRuntimeOptions{
			AuthProvider: feishuadapter.NewAuthProvider(feishuClient),
		}),
		pluginruntime.NewRegistryRuntimeWithOptions("notion", "test", []string{"notion"}, notionRegistry, pluginruntime.CatalogFromRegistry(notionRegistry), pluginruntime.RegistryRuntimeOptions{
			AuthProvider: notionadapter.NewAuthProvider(notionClient),
		}),
	}
	runtimes = append(runtimes, extraRuntimes...)

	manager, err := pluginruntime.NewManagerWithOptions(context.Background(), runtimes, pluginruntime.ManagerOptions{
		AuthLaunchers:           launchers,
		AuthLauncherPreferences: preferences,
	})
	if err != nil {
		t.Fatalf("failed to construct test plugin manager: %v", err)
	}
	return manager
}

type testAuthLauncherRuntime struct {
	descriptor pluginruntime.AuthLauncherDescriptor
	launch     func(params pluginruntime.AuthLaunchParams) (pluginruntime.AuthLaunchResult, error)
}

func (r *testAuthLauncherRuntime) Name() string {
	return r.descriptor.ID
}

func (r *testAuthLauncherRuntime) Handshake(ctx context.Context) (pluginruntime.HandshakeResult, error) {
	return pluginruntime.HandshakeResult{
		ProtocolVersion: pluginruntime.ProtocolVersion,
		Name:            r.descriptor.ID,
		Version:         "test",
	}, nil
}

func (r *testAuthLauncherRuntime) DescribeAuthLauncher(ctx context.Context) (pluginruntime.AuthLauncherDescriptor, error) {
	return r.descriptor, nil
}

func (r *testAuthLauncherRuntime) LaunchAuth(ctx context.Context, params pluginruntime.AuthLaunchParams) (pluginruntime.AuthLaunchResult, error) {
	if r.launch == nil {
		return pluginruntime.AuthLaunchResult{Handled: false, Status: "skipped"}, nil
	}
	return r.launch(params)
}

type testAuthProvider struct {
	listMethods func(ctx context.Context) ([]pluginruntime.AuthMethodDescriptor, error)
	listPresets func(ctx context.Context) ([]pluginruntime.AuthPresetDescriptor, error)
	inspect     func(ctx context.Context, params pluginruntime.AuthInspectParams) (pluginruntime.AuthInspectResult, error)
	begin       func(ctx context.Context, params pluginruntime.AuthBeginParams) (pluginruntime.AuthBeginResult, error)
	complete    func(ctx context.Context, params pluginruntime.AuthCompleteParams) (pluginruntime.AuthCompleteResult, error)
	resolve     func(ctx context.Context, params pluginruntime.AuthResolveParams) (pluginruntime.AuthResolveResult, error)
}

func (p *testAuthProvider) ListMethods(ctx context.Context) ([]pluginruntime.AuthMethodDescriptor, error) {
	if p.listMethods == nil {
		return nil, nil
	}
	return p.listMethods(ctx)
}

func (p *testAuthProvider) ListPresets(ctx context.Context) ([]pluginruntime.AuthPresetDescriptor, error) {
	if p.listPresets == nil {
		return nil, nil
	}
	return p.listPresets(ctx)
}

func (p *testAuthProvider) Inspect(ctx context.Context, params pluginruntime.AuthInspectParams) (pluginruntime.AuthInspectResult, error) {
	if p.inspect == nil {
		return pluginruntime.AuthInspectResult{}, nil
	}
	return p.inspect(ctx, params)
}

func (p *testAuthProvider) Begin(ctx context.Context, params pluginruntime.AuthBeginParams) (pluginruntime.AuthBeginResult, error) {
	if p.begin == nil {
		return pluginruntime.AuthBeginResult{}, nil
	}
	return p.begin(ctx, params)
}

func (p *testAuthProvider) Complete(ctx context.Context, params pluginruntime.AuthCompleteParams) (pluginruntime.AuthCompleteResult, error) {
	if p.complete == nil {
		return pluginruntime.AuthCompleteResult{}, nil
	}
	return p.complete(ctx, params)
}

func (p *testAuthProvider) Resolve(ctx context.Context, params pluginruntime.AuthResolveParams) (pluginruntime.AuthResolveResult, error) {
	if p.resolve == nil {
		return pluginruntime.AuthResolveResult{}, nil
	}
	return p.resolve(ctx, params)
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
	t.Setenv("CLAWRISE_STATE_DIR", filepath.Join(t.TempDir(), "state"))
	t.Setenv("CLAWRISE_MASTER_KEY", "test-master-key")

	configBytes, err := os.ReadFile("../../examples/config.example.yaml")
	if err != nil {
		t.Fatalf("failed to read example config: %v", err)
	}
	if err := os.WriteFile(configPath, configBytes, 0o600); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// 单测统一改用临时文件型 secret store，避免误读开发机真实 keychain 等外部凭证。
	store := config.NewStore(configPath)
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("failed to load test config for isolation setup: %v", err)
	}
	cfg.Auth.SecretStore.Backend = "encrypted_file"
	cfg.Auth.SecretStore.FallbackBackend = "encrypted_file"
	if err := store.Save(cfg); err != nil {
		t.Fatalf("failed to persist isolated secret store config for tests: %v", err)
	}

	return configPath
}

func runSecretSet(t *testing.T, connectionName string, fieldName string, value string) {
	t.Helper()

	t.Setenv("CLAWRISE_MASTER_KEY", "test-master-key")
	configPath := os.Getenv("CLAWRISE_CONFIG")
	if strings.TrimSpace(configPath) != "" {
		store := config.NewStore(configPath)
		cfg, err := store.Load()
		if err != nil {
			t.Fatalf("failed to load test config before seeding secret: %v", err)
		}
		cfg.Auth.SecretStore.Backend = "encrypted_file"
		cfg.Auth.SecretStore.FallbackBackend = "encrypted_file"
		if err := store.Save(cfg); err != nil {
			t.Fatalf("failed to persist encrypted_file secret backend for tests: %v", err)
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Run([]string{"auth", "secret", "set", connectionName, fieldName, "--value", value, "--allow-insecure-cli-secret"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("failed to seed secret via CLI: %v, stdout=%s, stderr=%s", err, stdout.String(), stderr.String())
	}
}

func runAuthLoginForTest(t *testing.T, args []string) map[string]any {
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

func nestedStringFromBytes(t *testing.T, data []byte, keys ...string) string {
	t.Helper()

	result := map[string]any{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to decode json output: %v; output=%s", err, string(data))
	}
	return nestedString(result, keys...)
}

func writePolicyCapabilityFixture(t *testing.T, root string, pluginName string, policyID string) {
	t.Helper()

	pluginDir := filepath.Join(root, pluginName, "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create policy plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, pluginName+".sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to write policy plugin executable: %v", err)
	}
	manifest := `{
  "schema_version": 2,
  "name": "` + pluginName + `",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "policy",
      "id": "` + policyID + `"
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./` + pluginName + `.sh"]
  }
}`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("failed to write policy plugin manifest: %v", err)
	}
}

func writeAuditSinkCapabilityFixture(t *testing.T, root string, pluginName string, sinkID string) {
	t.Helper()

	pluginDir := filepath.Join(root, pluginName, "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create audit plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, pluginName+".sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("failed to write audit plugin executable: %v", err)
	}
	manifest := `{
  "schema_version": 2,
  "name": "` + pluginName + `",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "audit_sink",
      "id": "` + sinkID + `"
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./` + pluginName + `.sh"]
  }
}`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("failed to write audit plugin manifest: %v", err)
	}
}
