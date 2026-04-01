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
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.handshake"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"protocol_version":1,"name":"demo","version":"0.1.0","platforms":["demo"]}}'"\n"
      exit 0
      ;;
  esac
done
`
	if err := os.WriteFile(pluginPath, []byte(script), 0o755); err != nil {
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

func TestProcessRuntimeListCapabilities(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	pluginPath := filepath.Join(pluginDir, "demo-plugin.sh")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.capabilities.list"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"capabilities":[{"type":"provider","platforms":["demo"]},{"type":"storage_backend","target":"session_store","backend":"plugin.demo_session"}]}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(pluginPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write demo plugin: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(`{
  "schema_version": 2,
  "name": "demo",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "provider",
      "platforms": ["demo"]
    }
  ],
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

	capabilities, err := runtime.ListCapabilities(context.Background())
	if err != nil {
		t.Fatalf("ListCapabilities returned error: %v", err)
	}
	if len(capabilities) != 2 {
		t.Fatalf("unexpected capabilities: %+v", capabilities)
	}
	if capabilities[0].Type != CapabilityTypeProvider || capabilities[1].Target != "session_store" {
		t.Fatalf("unexpected capability payload: %+v", capabilities)
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

func TestProcessPolicyEvaluate(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "policy")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	pluginPath := filepath.Join(pluginDir, "policy-plugin.sh")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.policy.evaluate"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"decision":"annotate","message":"matched policy annotation"}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(pluginPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write policy plugin: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(`{
  "schema_version": 2,
  "name": "policy-demo",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "policy",
      "id": "review",
      "priority": 80
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./policy-plugin.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write plugin manifest: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	runtime := NewProcessPolicy(manifest, manifest.CapabilitiesByType(CapabilityTypePolicy)[0])

	result, err := runtime.Evaluate(context.Background(), PolicyEvaluateParams{
		Request: PolicyEvaluationRequest{
			RequestID: "req_demo",
			Operation: "demo.page.update",
			Mutating:  true,
		},
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if result.Decision != "annotate" || result.Message != "matched policy annotation" {
		t.Fatalf("unexpected policy result: %+v", result)
	}
}

func TestProcessAuditSinkEmit(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "audit-sink")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	pluginPath := filepath.Join(pluginDir, "audit-sink-plugin.sh")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.audit.emit"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(pluginPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write audit sink plugin: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(`{
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
    "command": ["./audit-sink-plugin.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write plugin manifest: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	runtime := NewProcessAuditSink(manifest, manifest.CapabilitiesByType(CapabilityTypeAuditSink)[0])

	if err := runtime.Emit(context.Background(), AuditEmitParams{
		Record: GovernanceAuditRecord{
			RequestID: "req_demo",
			Operation: "demo.page.update",
			OK:        true,
		},
	}); err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}
}

func TestProcessSecretStoreDescribeStatusAndCRUD(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "secret-store")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	pluginPath := filepath.Join(pluginDir, "secret-store-plugin.sh")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.handshake"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"protocol_version":1,"name":"secret-demo","version":"0.1.0"}}'"\n"
      ;;
    *'"method":"clawrise.storage.backend.describe"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"backend":{"target":"secret_store","backend":"plugin.demo_secret","display_name":"Demo Secret Store"}}}'"\n"
      ;;
    *'"method":"clawrise.storage.secret.status"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"status":{"backend":"plugin.demo_secret","supported":true,"readable":true,"writable":true,"secure":true}}}'"\n"
      ;;
    *'"method":"clawrise.storage.secret.get"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"found":true,"value":"demo-secret"}}'"\n"
      ;;
    *'"method":"clawrise.storage.secret.set"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{}}'"\n"
      ;;
    *'"method":"clawrise.storage.secret.delete"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(pluginPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write storage backend plugin: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(`{
  "schema_version": 1,
  "name": "secret-demo",
  "version": "0.1.0",
  "kind": "storage_backend",
  "protocol_version": 1,
  "storage_backend": {
    "target": "secret_store",
    "backend": "plugin.demo_secret",
    "display_name": "Demo Secret Store"
  },
  "entry": {
    "type": "binary",
    "command": ["./secret-store-plugin.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write storage backend manifest: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	store := NewProcessSecretStore(manifest)
	defer func() {
		_ = store.Close()
	}()

	descriptor, err := store.DescribeStorageBackend(context.Background())
	if err != nil {
		t.Fatalf("DescribeStorageBackend returned error: %v", err)
	}
	if descriptor.Target != "secret_store" || descriptor.Backend != "plugin.demo_secret" {
		t.Fatalf("unexpected storage backend descriptor: %+v", descriptor)
	}

	status, err := store.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if !status.Supported || status.Backend != "plugin.demo_secret" {
		t.Fatalf("unexpected storage status: %+v", status)
	}

	getResult, err := store.Get(context.Background(), SecretStoreGetParams{
		AccountName: "demo_account",
		Field:       "token",
	})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !getResult.Found || getResult.Value != "demo-secret" {
		t.Fatalf("unexpected get result: %+v", getResult)
	}

	if err := store.Set(context.Background(), SecretStoreSetParams{
		AccountName: "demo_account",
		Field:       "token",
		Value:       "demo-secret",
	}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	if err := store.Delete(context.Background(), SecretStoreDeleteParams{
		AccountName: "demo_account",
		Field:       "token",
	}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestProcessSessionStoreStatusLoadSaveAndDelete(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "session-store")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	pluginPath := filepath.Join(pluginDir, "session-store-plugin.sh")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.handshake"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"protocol_version":1,"name":"session-demo","version":"0.1.0"}}'"\n"
      ;;
    *'"method":"clawrise.storage.session.status"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"status":{"backend":"plugin.demo_session","supported":true,"readable":true,"writable":true,"secure":true}}}'"\n"
      ;;
    *'"method":"clawrise.storage.session.load"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"found":true,"session":{"account_name":"demo_account","platform":"demo","subject":"user","grant_type":"oauth_user","access_token":"demo-access-token"}}}'"\n"
      ;;
    *'"method":"clawrise.storage.session.save"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{}}'"\n"
      ;;
    *'"method":"clawrise.storage.session.delete"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(pluginPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write storage backend plugin: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(`{
  "schema_version": 2,
  "name": "session-demo",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "storage_backend",
      "target": "session_store",
      "backend": "plugin.demo_session"
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./session-store-plugin.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write storage backend manifest: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	store := NewProcessSessionStore(manifest)
	defer func() {
		_ = store.Close()
	}()

	status, err := store.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if !status.Supported || status.Backend != "plugin.demo_session" {
		t.Fatalf("unexpected session store status: %+v", status)
	}

	loadResult, err := store.Load(context.Background(), SessionStoreLoadParams{
		AccountName: "demo_account",
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !loadResult.Found || loadResult.Session == nil || loadResult.Session.AccessToken != "demo-access-token" {
		t.Fatalf("unexpected session load result: %+v", loadResult)
	}

	if err := store.Save(context.Background(), SessionStoreSaveParams{
		Session: *loadResult.Session,
	}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if err := store.Delete(context.Background(), SessionStoreDeleteParams{
		AccountName: "demo_account",
	}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestProcessAuthFlowStoreStatusLoadSaveAndDelete(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "authflow-store")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	pluginPath := filepath.Join(pluginDir, "authflow-store-plugin.sh")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.handshake"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"protocol_version":1,"name":"authflow-demo","version":"0.1.0"}}'"\n"
      ;;
    *'"method":"clawrise.storage.authflow.status"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"status":{"backend":"plugin.demo_authflow","supported":true,"readable":true,"writable":true,"secure":true}}}'"\n"
      ;;
    *'"method":"clawrise.storage.authflow.load"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"found":true,"flow":{"id":"flow_demo","account_name":"demo_account","platform":"demo","method":"oauth_user","mode":"local_browser","state":"pending","created_at":"2026-03-28T10:00:00Z","updated_at":"2026-03-28T10:00:00Z","expires_at":"2026-03-28T10:10:00Z"}}}'"\n"
      ;;
    *'"method":"clawrise.storage.authflow.save"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{}}'"\n"
      ;;
    *'"method":"clawrise.storage.authflow.delete"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(pluginPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write storage backend plugin: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(`{
  "schema_version": 2,
  "name": "authflow-demo",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "storage_backend",
      "target": "authflow_store",
      "backend": "plugin.demo_authflow"
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./authflow-store-plugin.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write storage backend manifest: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	store := NewProcessAuthFlowStore(manifest)
	defer func() {
		_ = store.Close()
	}()

	status, err := store.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if !status.Supported || status.Backend != "plugin.demo_authflow" {
		t.Fatalf("unexpected authflow store status: %+v", status)
	}

	loadResult, err := store.Load(context.Background(), AuthFlowStoreLoadParams{
		FlowID: "flow_demo",
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !loadResult.Found || loadResult.Flow == nil || loadResult.Flow.ID != "flow_demo" {
		t.Fatalf("unexpected authflow load result: %+v", loadResult)
	}

	if err := store.Save(context.Background(), AuthFlowStoreSaveParams{
		Flow: *loadResult.Flow,
	}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if err := store.Delete(context.Background(), AuthFlowStoreDeleteParams{
		FlowID: "flow_demo",
	}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestProcessGovernanceStoreStatusLoadSaveAndAppend(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "governance-store")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	pluginPath := filepath.Join(pluginDir, "governance-store-plugin.sh")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.handshake"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"protocol_version":1,"name":"governance-demo","version":"0.1.0"}}'"\n"
      ;;
    *'"method":"clawrise.storage.governance.status"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"status":{"backend":"plugin.demo_governance","supported":true,"readable":true,"writable":true,"secure":true}}}'"\n"
      ;;
    *'"method":"clawrise.storage.governance.idempotency.load"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"found":true,"record":{"key":"idem_demo","operation":"demo.page.update","input_hash":"hash_demo","status":"executed","request_id":"req_demo","created_at":"2026-03-28T10:00:00Z","updated_at":"2026-03-28T10:00:01Z","retry_count":1,"meta":{"platform":"demo","duration_ms":12,"retry_count":1,"dry_run":false}}}}'"\n"
      ;;
    *'"method":"clawrise.storage.governance.idempotency.save"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{}}'"\n"
      ;;
    *'"method":"clawrise.storage.governance.audit.append"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(pluginPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write storage backend plugin: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(`{
  "schema_version": 2,
  "name": "governance-demo",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "storage_backend",
      "target": "governance",
      "backend": "plugin.demo_governance"
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./governance-store-plugin.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write storage backend manifest: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	store := NewProcessGovernanceStore(manifest)
	defer func() {
		_ = store.Close()
	}()

	status, err := store.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if !status.Supported || status.Backend != "plugin.demo_governance" {
		t.Fatalf("unexpected governance store status: %+v", status)
	}

	loadResult, err := store.LoadIdempotency(context.Background(), GovernanceIdempotencyLoadParams{
		Key: "idem_demo",
	})
	if err != nil {
		t.Fatalf("LoadIdempotency returned error: %v", err)
	}
	if !loadResult.Found || loadResult.Record == nil || loadResult.Record.Key != "idem_demo" {
		t.Fatalf("unexpected governance load result: %+v", loadResult)
	}

	if err := store.SaveIdempotency(context.Background(), GovernanceIdempotencySaveParams{
		Record: *loadResult.Record,
	}); err != nil {
		t.Fatalf("SaveIdempotency returned error: %v", err)
	}

	if err := store.AppendAudit(context.Background(), GovernanceAuditAppendParams{
		Day: "2026-03-28",
		Record: GovernanceAuditRecord{
			Time:      "2026-03-28T10:00:02Z",
			RequestID: "req_demo",
			Operation: "demo.page.update",
			OK:        true,
			Meta: GovernanceMeta{
				Platform:   "demo",
				DurationMS: 12,
				RetryCount: 1,
				DryRun:     false,
			},
		},
	}); err != nil {
		t.Fatalf("AppendAudit returned error: %v", err)
	}
}

func TestNewDiscoveredManagerWithOptionsFiltersDisabledProviders(t *testing.T) {
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())

	writeProviderDiscoveryTestPlugin(t, pluginRoot, "demo-a", "demo.a.echo")
	writeProviderDiscoveryTestPlugin(t, pluginRoot, "demo-b", "demo.b.echo")

	manager, err := NewDiscoveredManagerWithOptions(context.Background(), DiscoveryOptions{
		EnabledPlugins: map[string]string{
			"demo-a": "disabled",
		},
	})
	if err != nil {
		t.Fatalf("NewDiscoveredManagerWithOptions returned error: %v", err)
	}

	if _, ok := manager.Registry().Resolve("demo.b.echo"); !ok {
		t.Fatalf("expected enabled provider operation to be registered")
	}
	if _, ok := manager.Registry().Resolve("demo.a.echo"); ok {
		t.Fatalf("did not expect disabled provider operation to be registered")
	}
}

func TestNewDiscoveredManagerWithOptionsRejectsBindingToDisabledProvider(t *testing.T) {
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())

	writeProviderDiscoveryTestPlugin(t, pluginRoot, "demo-a", "demo.a.echo")

	_, err := NewDiscoveredManagerWithOptions(context.Background(), DiscoveryOptions{
		EnabledPlugins: map[string]string{
			"demo-a": "disabled",
		},
		ProviderBindings: map[string]string{
			"demo": "demo-a",
		},
	})
	if err == nil {
		t.Fatal("expected disabled provider binding to be rejected")
	}
	if err.Error() != "provider binding for platform demo points to demo-a, but the plugin is disabled by plugins.enabled" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeProviderDiscoveryTestPlugin(t *testing.T, root string, pluginName string, operation string) {
	t.Helper()

	pluginDir := filepath.Join(root, pluginName, "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	pluginPath := filepath.Join(pluginDir, pluginName+".sh")
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.handshake"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"protocol_version":1,"name":"` + pluginName + `","version":"0.1.0","platforms":["demo"]}}'"\n"
      ;;
    *'"method":"clawrise.operations.list"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"operations":[{"operation":"` + operation + `","platform":"demo","mutating":false,"default_timeout_ms":1000,"allowed_subjects":["integration"],"spec":{"summary":"Echo one demo payload.","dry_run_supported":true,"input":{"sample":{"message":"hello"}}}}]}}'"\n"
      ;;
    *'"method":"clawrise.catalog.get"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"entries":[{"operation":"` + operation + `"}]}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(pluginPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write provider plugin: %v", err)
	}

	manifest := `{
  "schema_version": 1,
  "name": "` + pluginName + `",
  "version": "0.1.0",
  "kind": "provider",
  "protocol_version": 1,
  "platforms": ["demo"],
  "entry": {
    "type": "binary",
    "command": ["./` + pluginName + `.sh"]
  }
}`
	if err := os.WriteFile(filepath.Join(pluginDir, ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatalf("failed to write provider manifest: %v", err)
	}
}
