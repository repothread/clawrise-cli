package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifestAndProcessRuntimeHandshake(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, ManifestFileName)
	pluginPath := filepath.Join(pluginDir, "demo-plugin.sh")
	if err := os.WriteFile(pluginPath, []byte("#!/bin/sh\nprintf '{\"jsonrpc\":\"2.0\",\"id\":\"1\",\"result\":{\"protocol_version\":1,\"name\":\"demo\",\"version\":\"0.1.0\",\"platforms\":[\"demo\"]}}\\n'\n"), 0o755); err != nil {
		t.Fatalf("failed to write demo plugin: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte(`{
  "schema_version": 1,
  "name": "demo",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./demo-plugin.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write plugin manifest: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}

	runtime := NewProcessRuntime(manifest)
	result, err := runtime.Handshake(context.Background())
	if err != nil {
		t.Fatalf("Handshake returned error: %v", err)
	}
	if result.Name != "demo" {
		t.Fatalf("unexpected handshake result: %+v", result)
	}
}

func TestProcessRuntimeListCatalogExecuteAndHealth(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	pluginPath := filepath.Join(pluginDir, "demo-plugin.sh")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.handshake"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"protocol_version":1,"name":"demo","version":"0.1.0","platforms":["demo"]}}'"\n"
      ;;
    *'"method":"clawrise.operations.list"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"operations":[{"operation":"demo.page.echo","platform":"demo","mutating":false,"default_timeout_ms":1000,"allowed_subjects":["integration"],"spec":{"summary":"Echo one demo page.","dry_run_supported":true,"input":{"required":["message"],"sample":{"message":"hello"}}}}]}}'"\n"
      ;;
    *'"method":"clawrise.catalog.get"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"entries":[{"operation":"demo.page.echo"}]}}'"\n"
      ;;
    *'"method":"clawrise.execute"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"ok":true,"data":{"message":"echoed"},"meta":{"retry_count":0}}}'"\n"
      ;;
    *'"method":"clawrise.health"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"ok":true,"details":{"plugin_name":"demo"}}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(pluginPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write demo plugin: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(`{
  "schema_version": 1,
  "name": "demo",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./demo-plugin.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write plugin manifest: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	runtime := NewProcessRuntime(manifest)

	definitions, err := runtime.ListOperations(context.Background())
	if err != nil {
		t.Fatalf("ListOperations returned error: %v", err)
	}
	if len(definitions) != 1 || definitions[0].Operation != "demo.page.echo" {
		t.Fatalf("unexpected definitions: %+v", definitions)
	}

	entries, err := runtime.GetCatalog(context.Background())
	if err != nil {
		t.Fatalf("GetCatalog returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].Operation != "demo.page.echo" {
		t.Fatalf("unexpected catalog entries: %+v", entries)
	}

	result, err := runtime.Execute(context.Background(), ExecuteRequest{
		Operation: "demo.page.echo",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Data["message"] != "echoed" {
		t.Fatalf("unexpected execute result: %+v", result)
	}

	health, err := runtime.Health(context.Background())
	if err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if !health.OK {
		t.Fatalf("unexpected health result: %+v", health)
	}
}

func TestProcessRuntimeDescribeAndLaunchAuthLauncher(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "launcher")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	pluginPath := filepath.Join(pluginDir, "launcher-plugin.sh")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.handshake"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"protocol_version":1,"name":"launcher","version":"0.1.0"}}'"\n"
      ;;
    *'"method":"clawrise.auth.launcher.describe"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"launcher":{"id":"launcher","display_name":"Launcher","action_types":["open_url"],"priority":10}}}'"\n"
      ;;
    *'"method":"clawrise.auth.launcher.run"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"handled":true,"status":"launched","launcher_id":"launcher"}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(pluginPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write launcher plugin: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(`{
  "schema_version": 1,
  "name": "launcher",
  "version": "0.1.0",
  "kind": "auth_launcher",
  "protocol_version": 1,
  "entry": {
    "type": "binary",
    "command": ["./launcher-plugin.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write plugin manifest: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	runtime := NewProcessRuntime(manifest)

	descriptor, err := runtime.DescribeAuthLauncher(context.Background())
	if err != nil {
		t.Fatalf("DescribeAuthLauncher returned error: %v", err)
	}
	if descriptor.ID != "launcher" {
		t.Fatalf("unexpected launcher descriptor: %+v", descriptor)
	}

	result, err := runtime.LaunchAuth(context.Background(), AuthLaunchParams{
		Context: AuthLaunchContext{
			AccountName: "demo",
			Platform:    "demo",
		},
		Action: AuthAction{
			Type: "open_url",
			URL:  "https://example.com/auth",
		},
	})
	if err != nil {
		t.Fatalf("LaunchAuth returned error: %v", err)
	}
	if !result.Handled || result.LauncherID != "launcher" {
		t.Fatalf("unexpected launch result: %+v", result)
	}
}
