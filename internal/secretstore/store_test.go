package secretstore

import (
	"path/filepath"
	"strings"
	"testing"
)

type testStore struct {
	backend string
	status  Status
}

func (s *testStore) Backend() string {
	return s.backend
}

func (s *testStore) Status() Status {
	return s.status
}

func (s *testStore) Get(connectionName string, field string) (string, error) {
	return "", ErrSecretNotFound
}

func (s *testStore) Set(connectionName string, field string, value string) error {
	return nil
}

func (s *testStore) Delete(connectionName string, field string) error {
	return nil
}

func TestRegisterStoreBackend(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	backendName := "test_backend_" + sanitizeTestName(t.Name())

	var capturedConfigPath string
	var capturedStateDir string
	RegisterStoreBackend(backendName, func(configPath string, stateDir string) (Store, error) {
		capturedConfigPath = configPath
		capturedStateDir = stateDir
		return &testStore{
			backend: backendName,
			status: Status{
				Backend:   backendName,
				Supported: true,
				Readable:  true,
				Writable:  true,
				Secure:    true,
			},
		}, nil
	})

	store, err := Open(Options{
		ConfigPath: configPath,
		Backend:    backendName,
	})
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if store.Backend() != backendName {
		t.Fatalf("unexpected backend name: %s", store.Backend())
	}
	if capturedConfigPath != configPath {
		t.Fatalf("unexpected config path passed to custom store: got=%s want=%s", capturedConfigPath, configPath)
	}
	expectedStateDir := filepath.Join(filepath.Dir(configPath), "state")
	if capturedStateDir != expectedStateDir {
		t.Fatalf("unexpected state dir passed to custom store: got=%s want=%s", capturedStateDir, expectedStateDir)
	}
}

func sanitizeTestName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	replacer := strings.NewReplacer("/", "_", " ", "_", "-", "_")
	return replacer.Replace(name)
}
