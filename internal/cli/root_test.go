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
	if !bytes.Contains(stdout.Bytes(), []byte("clawrise auth [list|methods|presets|inspect|check|login|complete|logout|secret]")) {
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

	err := Run([]string{"account", "use", "notion_team_docs"}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"name": "notion_team_docs"`)) {
		t.Fatalf("expected account response payload, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"account"`)) {
		t.Fatalf("expected account payload shape, got: %s", stdout.String())
	}
}

func TestRunAccountUseSynchronizesPlatformForBareOperation(t *testing.T) {
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

	err = Run([]string{"account", "use", "notion_team_docs"}, Dependencies{
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
	if !bytes.Contains(stdout.Bytes(), []byte(`"account": "notion_team_docs"`)) {
		t.Fatalf("expected notion account in output, got: %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"subject": "integration"`)) {
		t.Fatalf("expected integration subject in output, got: %s", stdout.String())
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
		"--account", "notion_team_docs",
	}, Dependencies{
		Version:       "test",
		Stdout:        &stdout,
		Stderr:        &stderr,
		PluginManager: newTestPluginManager(t),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !bytes.Contains(stdout.Bytes(), []byte(`"account": "notion_team_docs"`)) {
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

func TestRunAuthBeginNotionPublic(t *testing.T) {
	copyExampleConfig(t)

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
	manager := newTestPluginManagerWithOptions(t, feishuClient, nil, []pluginruntime.AuthLauncherRuntime{launcher}, []pluginruntime.Runtime{deviceRuntime})

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
	err = Run([]string{"auth", "secret", "set", "notion_team_docs", "token", "--from-env", "TEST_SECRET_VALUE"}, Dependencies{
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
	value, err := secretStore.Get("notion_team_docs", "token")
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
	err := Run([]string{"auth", "secret", "set", "notion_team_docs", "token", "--value", "plain-secret"}, Dependencies{
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
	return newTestPluginManagerWithOptions(t, feishuClient, nil, nil, nil)
}

func newTestPluginManagerWithNotionClient(t *testing.T, factory func(sessionStore authcache.Store) (*notionadapter.Client, error)) *pluginruntime.Manager {
	t.Helper()

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu test client: %v", err)
	}
	return newTestPluginManagerWithOptions(t, feishuClient, factory, nil, nil)
}

func newTestPluginManagerWithLaunchers(t *testing.T, launchers []pluginruntime.AuthLauncherRuntime) *pluginruntime.Manager {
	t.Helper()

	feishuClient, err := feishuadapter.NewClient(feishuadapter.Options{})
	if err != nil {
		t.Fatalf("failed to construct feishu test client: %v", err)
	}
	return newTestPluginManagerWithOptions(t, feishuClient, nil, launchers, nil)
}

func newTestPluginManagerWithOptions(t *testing.T, feishuClient *feishuadapter.Client, notionFactory func(sessionStore authcache.Store) (*notionadapter.Client, error), launchers []pluginruntime.AuthLauncherRuntime, extraRuntimes []pluginruntime.Runtime) *pluginruntime.Manager {
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
		AuthLaunchers: launchers,
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
