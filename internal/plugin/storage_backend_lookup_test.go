package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindStorageBackendManifestRejectsDisabledPlugin(t *testing.T) {
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())

	pluginDir := filepath.Join(pluginRoot, "session-demo", "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, ManifestFileName), []byte(`{
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
    "command": ["./session-demo.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write plugin manifest: %v", err)
	}

	_, _, err := FindStorageBackendManifest(StorageBackendLookup{
		Target:  "session_store",
		Backend: "plugin.demo_session",
		Plugin:  "session-demo",
		EnabledPlugins: map[string]string{
			"session-demo": "disabled",
		},
	})
	if err == nil {
		t.Fatal("expected disabled plugin lookup error")
	}
	if err.Error() != "storage backend plugin session-demo for target session_store is disabled by plugins.enabled" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFindStorageBackendManifestsFiltersDisabledPlugins(t *testing.T) {
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())

	writeStorageLookupTestManifest(t, pluginRoot, "session-a", "plugin.session_a")
	writeStorageLookupTestManifest(t, pluginRoot, "session-b", "plugin.session_b")

	manifests, err := FindStorageBackendManifests(StorageBackendLookup{
		Target: "session_store",
		EnabledPlugins: map[string]string{
			"session-a": "disabled",
		},
	})
	if err != nil {
		t.Fatalf("FindStorageBackendManifests returned error: %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("unexpected filtered manifests: %+v", manifests)
	}
	if manifests[0].Name != "session-b" {
		t.Fatalf("unexpected manifest after filtering: %+v", manifests[0])
	}
}

func writeStorageLookupTestManifest(t *testing.T, root string, pluginName string, backend string) {
	t.Helper()

	pluginDir := filepath.Join(root, pluginName, "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}
	manifest := `{
  "schema_version": 2,
  "name": "` + pluginName + `",
  "version": "0.1.0",
  "protocol_version": 1,
  "capabilities": [
    {
      "type": "storage_backend",
      "target": "session_store",
      "backend": "` + backend + `"
    }
  ],
  "entry": {
    "type": "binary",
    "command": ["./` + pluginName + `.sh"]
  }
}`
	if err := os.WriteFile(filepath.Join(pluginDir, ManifestFileName), []byte(manifest), 0o644); err != nil {
		t.Fatalf("failed to write plugin manifest: %v", err)
	}
}
