package plugin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectDiscoveryReportsRuntimeCapabilityMismatch(t *testing.T) {
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)

	pluginDir := filepath.Join(pluginRoot, "demo", "0.1.0")
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
    *'"method":"clawrise.capabilities.list"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"capabilities":[{"type":"storage_backend","target":"session_store","backend":"plugin.demo_session"}]}}'"\n"
      ;;
    *'"method":"clawrise.operations.list"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"operations":[]}}'"\n"
      ;;
    *'"method":"clawrise.catalog.get"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"entries":[]}}'"\n"
      ;;
    *'"method":"clawrise.health"'*)
      printf '{"jsonrpc":"2.0","id":"1","result":{"ok":true}}'"\n"
      ;;
  esac
done
`
	if err := os.WriteFile(pluginPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write plugin executable: %v", err)
	}

	if err := os.WriteFile(filepath.Join(pluginDir, ManifestFileName), []byte(`{
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

	report, err := InspectDiscoveryWithOptions(context.Background(), DiscoveryOptions{})
	if err != nil {
		t.Fatalf("InspectDiscoveryWithOptions returned error: %v", err)
	}
	if len(report.Plugins) != 1 {
		t.Fatalf("unexpected plugins: %+v", report.Plugins)
	}

	item := report.Plugins[0]
	if len(item.RuntimeCapabilities) != 1 || item.RuntimeCapabilities[0].Type != CapabilityTypeStorageBackend {
		t.Fatalf("unexpected runtime capabilities: %+v", item.RuntimeCapabilities)
	}
	if len(item.InspectionWarnings) == 0 {
		t.Fatalf("expected inspection warning, got: %+v", item)
	}
	if !strings.Contains(item.InspectionWarnings[0], "runtime capabilities") {
		t.Fatalf("unexpected inspection warnings: %+v", item.InspectionWarnings)
	}
}
