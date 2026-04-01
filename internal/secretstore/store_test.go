package secretstore

import (
	"os"
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

func TestOpenUsesDiscoveredPluginSecretStore(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	pluginDir := filepath.Join(homeDir, ".clawrise", "plugins", "secret-demo", "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}

	pluginPath := filepath.Join(pluginDir, "secret-plugin.sh")
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
		t.Fatalf("failed to write plugin executable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{
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
    "command": ["./secret-plugin.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write plugin manifest: %v", err)
	}

	store, err := Open(Options{
		ConfigPath: filepath.Join(t.TempDir(), "config.yaml"),
		Backend:    "plugin.demo_secret",
	})
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	if store.Backend() != "plugin.demo_secret" {
		t.Fatalf("unexpected plugin store backend: %s", store.Backend())
	}
	status := store.Status()
	if !status.Supported || !status.Readable || !status.Writable {
		t.Fatalf("unexpected plugin store status: %+v", status)
	}

	value, err := store.Get("demo_account", "token")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if value != "demo-secret" {
		t.Fatalf("unexpected plugin store value: %s", value)
	}
	if err := store.Set("demo_account", "token", "demo-secret"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if err := store.Delete("demo_account", "token"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestOpenRejectsDisabledPluginSecretStore(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	pluginDir := filepath.Join(homeDir, ".clawrise", "plugins", "secret-demo", "0.1.0")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(`{
  "schema_version": 1,
  "name": "secret-demo",
  "version": "0.1.0",
  "kind": "storage_backend",
  "protocol_version": 1,
  "storage_backend": {
    "target": "secret_store",
    "backend": "plugin.demo_secret"
  },
  "entry": {
    "type": "binary",
    "command": ["./secret-plugin.sh"]
  }
}`), 0o644); err != nil {
		t.Fatalf("failed to write plugin manifest: %v", err)
	}

	_, err := Open(Options{
		ConfigPath: filepath.Join(t.TempDir(), "config.yaml"),
		Backend:    "plugin.demo_secret",
		Plugin:     "secret-demo",
		EnabledPlugins: map[string]string{
			"secret-demo": "disabled",
		},
	})
	if err == nil {
		t.Fatal("expected disabled plugin secret store to be rejected")
	}
	if err.Error() != "storage backend plugin secret-demo for target secret_store is disabled by plugins.enabled" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEncryptedFileStoreGeneratesLocalKeyAndPersistsSecrets(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	store, err := Open(Options{
		ConfigPath: configPath,
		Backend:    "encrypted_file",
	})
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}

	if err := store.Set("demo_account", "token", "demo-secret"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	value, err := store.Get("demo_account", "token")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if value != "demo-secret" {
		t.Fatalf("unexpected secret value: %s", value)
	}

	stateDir := filepath.Join(filepath.Dir(configPath), "state", "auth")
	if _, err := os.Stat(filepath.Join(stateDir, "secrets.v1.enc")); err != nil {
		t.Fatalf("expected encrypted secret file to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stateDir, "secrets.v1.key")); err != nil {
		t.Fatalf("expected local secret key file to exist: %v", err)
	}
}

func TestEncryptedFileStoreMigratesFromEnvMasterKeyToLocalKeyFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("CLAWRISE_MASTER_KEY", "test-master-key")

	store, err := Open(Options{
		ConfigPath: configPath,
		Backend:    "encrypted_file",
	})
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if err := store.Set("demo_account", "token", "demo-secret"); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	keyPath := filepath.Join(filepath.Dir(configPath), "state", "auth", "secrets.v1.key")
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("expected migrated local key file to exist: %v", err)
	}

	t.Setenv("CLAWRISE_MASTER_KEY", "")
	store, err = Open(Options{
		ConfigPath: configPath,
		Backend:    "encrypted_file",
	})
	if err != nil {
		t.Fatalf("Open returned error after env removal: %v", err)
	}
	value, err := store.Get("demo_account", "token")
	if err != nil {
		t.Fatalf("Get returned error after env removal: %v", err)
	}
	if value != "demo-secret" {
		t.Fatalf("unexpected secret value after env removal: %s", value)
	}
}

func sanitizeTestName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	replacer := strings.NewReplacer("/", "_", " ", "_", "-", "_")
	return replacer.Replace(name)
}
