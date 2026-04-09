package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clawrise/clawrise-cli/internal/config"
)

type testGovernanceStore struct{}

func (s *testGovernanceStore) EnsureIdempotencyDir() error {
	return nil
}

func (s *testGovernanceStore) LoadIdempotencyRecord(key string) (*persistedIdempotencyRecord, error) {
	return nil, nil
}

func (s *testGovernanceStore) SaveIdempotencyRecord(record *persistedIdempotencyRecord) error {
	return nil
}

func (s *testGovernanceStore) AppendAuditRecord(day string, record auditRecord) error {
	return nil
}

func TestRegisterGovernanceStoreBackend(t *testing.T) {
	backendName := "test_backend_" + sanitizeGovernanceTestName(t.Name())
	runtimeRootDir := filepath.Join(t.TempDir(), "runtime")

	var capturedRootDir string
	RegisterGovernanceStoreBackend(backendName, func(rootDir string) governanceStore {
		capturedRootDir = rootDir
		return &testGovernanceStore{}
	})

	store := openGovernanceStore(runtimePaths{
		rootDir:        runtimeRootDir,
		idempotencyDir: filepath.Join(runtimeRootDir, "idempotency"),
		auditDir:       filepath.Join(runtimeRootDir, "audit"),
	}, config.StoragePluginBinding{
		Backend: backendName,
	}, nil)
	if store == nil {
		t.Fatal("expected custom governance store to be opened")
	}
	if capturedRootDir != runtimeRootDir {
		t.Fatalf("unexpected runtime root dir passed to custom backend: got=%s want=%s", capturedRootDir, runtimeRootDir)
	}
}

func TestOpenGovernanceStoreRejectsDisabledPlugin(t *testing.T) {
	pluginRoot := t.TempDir()
	t.Setenv("CLAWRISE_PLUGIN_PATHS", pluginRoot)
	t.Setenv("HOME", t.TempDir())

	pluginDir := filepath.Join(pluginRoot, "governance-demo", "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{
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
    "command": ["./governance-demo.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write plugin manifest: %v", err)
	}

	store := openGovernanceStore(runtimePaths{
		rootDir:        filepath.Join(t.TempDir(), "runtime"),
		idempotencyDir: filepath.Join(t.TempDir(), "runtime", "idempotency"),
		auditDir:       filepath.Join(t.TempDir(), "runtime", "audit"),
	}, config.StoragePluginBinding{
		Backend: "plugin.demo_governance",
		Plugin:  "governance-demo",
	}, map[string]string{
		"governance-demo": "disabled",
	})
	if err := store.EnsureIdempotencyDir(); err == nil {
		t.Fatal("expected disabled governance plugin to return an error")
	} else if err.Error() != "storage backend plugin governance-demo for target governance is disabled by plugins.enabled" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func sanitizeGovernanceTestName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	replacer := strings.NewReplacer("/", "_", " ", "_", "-", "_")
	return replacer.Replace(name)
}
