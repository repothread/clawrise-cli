package runtime

import (
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
	})
	if store == nil {
		t.Fatal("expected custom governance store to be opened")
	}
	if capturedRootDir != runtimeRootDir {
		t.Fatalf("unexpected runtime root dir passed to custom backend: got=%s want=%s", capturedRootDir, runtimeRootDir)
	}
}

func sanitizeGovernanceTestName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	replacer := strings.NewReplacer("/", "_", " ", "_", "-", "_")
	return replacer.Replace(name)
}
