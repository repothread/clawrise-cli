package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

type testRuntimeAuthProvider struct {
	resolve func(ctx context.Context, params pluginruntime.AuthResolveParams) (pluginruntime.AuthResolveResult, error)
}

func (p *testRuntimeAuthProvider) ListMethods(ctx context.Context) ([]pluginruntime.AuthMethodDescriptor, error) {
	return nil, nil
}

func (p *testRuntimeAuthProvider) ListPresets(ctx context.Context) ([]pluginruntime.AuthPresetDescriptor, error) {
	return nil, nil
}

func (p *testRuntimeAuthProvider) Inspect(ctx context.Context, params pluginruntime.AuthInspectParams) (pluginruntime.AuthInspectResult, error) {
	return pluginruntime.AuthInspectResult{}, nil
}

func (p *testRuntimeAuthProvider) Begin(ctx context.Context, params pluginruntime.AuthBeginParams) (pluginruntime.AuthBeginResult, error) {
	return pluginruntime.AuthBeginResult{}, nil
}

func (p *testRuntimeAuthProvider) Complete(ctx context.Context, params pluginruntime.AuthCompleteParams) (pluginruntime.AuthCompleteResult, error) {
	return pluginruntime.AuthCompleteResult{}, nil
}

func (p *testRuntimeAuthProvider) Resolve(ctx context.Context, params pluginruntime.AuthResolveParams) (pluginruntime.AuthResolveResult, error) {
	if p.resolve == nil {
		return pluginruntime.AuthResolveResult{}, nil
	}
	return p.resolve(ctx, params)
}

type testPolicyRuntime struct {
	name      string
	id        string
	priority  int
	platforms []string
	closed    bool
}

func (r *testPolicyRuntime) Name() string  { return r.name }
func (r *testPolicyRuntime) ID() string    { return r.id }
func (r *testPolicyRuntime) Priority() int { return r.priority }
func (r *testPolicyRuntime) Platforms() []string {
	return append([]string(nil), r.platforms...)
}
func (r *testPolicyRuntime) Handshake(ctx context.Context) (pluginruntime.HandshakeResult, error) {
	return pluginruntime.HandshakeResult{Name: r.name, ProtocolVersion: pluginruntime.ProtocolVersion}, nil
}
func (r *testPolicyRuntime) Evaluate(ctx context.Context, params pluginruntime.PolicyEvaluateParams) (pluginruntime.PolicyEvaluateResult, error) {
	return pluginruntime.PolicyEvaluateResult{}, nil
}
func (r *testPolicyRuntime) Close() error {
	r.closed = true
	return nil
}

type testAuditRuntime struct {
	name     string
	id       string
	priority int
	closed   bool
}

func (r *testAuditRuntime) Name() string  { return r.name }
func (r *testAuditRuntime) ID() string    { return r.id }
func (r *testAuditRuntime) Priority() int { return r.priority }
func (r *testAuditRuntime) Handshake(ctx context.Context) (pluginruntime.HandshakeResult, error) {
	return pluginruntime.HandshakeResult{Name: r.name, ProtocolVersion: pluginruntime.ProtocolVersion}, nil
}
func (r *testAuditRuntime) Emit(ctx context.Context, params pluginruntime.AuditEmitParams) error {
	return nil
}
func (r *testAuditRuntime) Close() error {
	r.closed = true
	return nil
}

type errReader struct{ err error }

func (r errReader) Read(p []byte) (int, error) { return 0, r.err }

func writeRuntimePluginManifest(t *testing.T, root string, pluginName string, capabilities string) {
	t.Helper()

	pluginDir := filepath.Join(root, pluginName, "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create runtime plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, pluginName+".sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to write runtime plugin executable: %v", err)
	}
	manifest := fmt.Sprintf(`{
  "schema_version": 2,
  "name": %q,
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": %s,
  "entry": {
    "type": "binary",
    "command": ["./%s.sh"]
  }
}`, pluginName, capabilities, pluginName)
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("failed to write runtime plugin manifest: %v", err)
	}
}

func newRuntimeTestManager(t *testing.T, platform string, authProvider pluginruntime.AuthProvider) *pluginruntime.Manager {
	t.Helper()

	manager, err := pluginruntime.NewManager(context.Background(), []pluginruntime.Runtime{
		pluginruntime.NewRegistryRuntimeWithOptions(
			platform,
			"test",
			[]string{platform},
			adapter.NewRegistry(),
			nil,
			pluginruntime.RegistryRuntimeOptions{AuthProvider: authProvider},
		),
	})
	if err != nil {
		t.Fatalf("failed to construct runtime test manager: %v", err)
	}
	return manager
}

func containsWarning(warnings []string, fragment string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, fragment) {
			return true
		}
	}
	return false
}

func writeGovernanceRuntimePluginManifest(t *testing.T, root string, pluginName string) pluginruntime.Manifest {
	return writeGovernanceRuntimePluginManifestWithLoadResult(t, root, pluginName, true)
}

func writeGovernanceRuntimePluginManifestWithLoadResult(t *testing.T, root string, pluginName string, found bool) pluginruntime.Manifest {
	t.Helper()

	pluginDir := filepath.Join(root, pluginName)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create governance wrapper plugin dir: %v", err)
	}

	pluginPath := filepath.Join(pluginDir, pluginName+".sh")
	loadResponse := `{"jsonrpc":"2.0","id":"1","result":{"found":true,"record":{"key":"idem_demo","operation":"demo.page.update","input_hash":"hash_demo","status":"executed","request_id":"req_demo","created_at":"2026-03-28T10:00:00Z","updated_at":"2026-03-28T10:00:01Z","retry_count":1,"meta":{"platform":"demo","duration_ms":12,"retry_count":1,"dry_run":false}}}}`
	if !found {
		loadResponse = `{"jsonrpc":"2.0","id":"1","result":{"found":false}}`
	}
	script := `#!/bin/sh
while IFS= read -r line; do
  case "$line" in
    *'"method":"clawrise.handshake"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"protocol_version":1,"name":"governance-demo","version":"0.1.0"}}'"\n"
      ;;
    *'"method":"clawrise.storage.governance.idempotency.load"'*)
      printf '` + loadResponse + `'"\n"
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
		t.Fatalf("failed to write governance wrapper plugin executable: %v", err)
	}

	manifestPath := filepath.Join(pluginDir, pluginruntime.ManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(fmt.Sprintf(`{
  "schema_version": 2,
  "name": %q,
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
    "command": ["./%s.sh"]
  }
}`, pluginName, pluginName)), 0o644); err != nil {
		t.Fatalf("failed to write governance wrapper manifest: %v", err)
	}

	manifest, err := pluginruntime.LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("failed to load governance wrapper manifest: %v", err)
	}
	return manifest
}
