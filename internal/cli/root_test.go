package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	feishuadapter "github.com/clawrise/clawrise-cli/internal/adapter/feishu"
	notionadapter "github.com/clawrise/clawrise-cli/internal/adapter/notion"
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
	if !bytes.Contains(stdout.Bytes(), []byte("clawrise auth [list|inspect|check]")) {
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
	t.Setenv("FEISHU_BOT_OPS_APP_ID", "app-id")
	t.Setenv("FEISHU_BOT_OPS_APP_SECRET", "app-secret")
	t.Setenv("CLAWRISE_CONFIG", "../../examples/config.example.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run([]string{"auth", "check", "feishu_bot_ops"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
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
