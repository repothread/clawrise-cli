package secretstore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeBackendNameAndSecretEntryKey(t *testing.T) {
	cases := map[string]string{
		"":                     "",
		" auto ":               "auto",
		"macos_keychain":       "keychain",
		"linux_secret_service": "secret_service",
		"encrypted_file":       "encrypted_file",
	}
	for input, want := range cases {
		if got := normalizeBackendName(input); got != want {
			t.Fatalf("normalizeBackendName(%q)=%q want %q", input, got, want)
		}
	}

	if got := secretEntryKey(" demo ", " token "); got != "demo::token" {
		t.Fatalf("unexpected secret entry key: %q", got)
	}
}

func TestPluginSecretStoreNilClientStatus(t *testing.T) {
	var store *pluginSecretStore
	if got := store.Backend(); got != "" {
		t.Fatalf("expected nil plugin store backend to be empty, got %q", got)
	}
	status := store.Status()
	if status.Supported || status.Detail == "" {
		t.Fatalf("expected nil plugin store status to report unsupported client, got %+v", status)
	}
}

func TestEncryptedFileStoreKeyValidationAndCommandDelegation(t *testing.T) {
	store := newEncryptedFileStore(t.TempDir(), "encrypted_file")
	if err := store.writeKeyFile([]byte("short")); err == nil {
		t.Fatal("expected invalid key length to be rejected")
	}

	badStore := newEncryptedFileStore(t.TempDir(), "encrypted_file")
	if err := badStore.writeKeyFile(bytesOfLen(31)); err == nil {
		t.Fatal("expected 31-byte key to be rejected")
	}

	invalidKeyStore := newEncryptedFileStore(t.TempDir(), "encrypted_file")
	if err := os.MkdirAll(filepath.Dir(invalidKeyStore.keyPath()), 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(invalidKeyStore.keyPath(), bytesOfLen(8), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if _, err := invalidKeyStore.loadOrCreateLocalKey(); err == nil {
		t.Fatal("expected invalid key file length to be rejected")
	}
	status := invalidKeyStore.Status()
	if status.Readable || status.Writable || status.Detail == "" {
		t.Fatalf("expected invalid key file to make status unreadable, got %+v", status)
	}

	called := 0
	commandStore := &commandSecretStore{
		backend: "command-demo",
		support: func() Status {
			return Status{Backend: "command-demo", Supported: true, Readable: true, Writable: true, Secure: true}
		},
		get: func(connectionName string, field string) (string, error) {
			called++
			return connectionName + ":" + field, nil
		},
		set: func(connectionName string, field string, value string) error {
			called++
			if value != "value" {
				return errors.New("bad value")
			}
			return nil
		},
		delete: func(connectionName string, field string) error {
			called++
			return nil
		},
	}
	if commandStore.Backend() != "command-demo" {
		t.Fatalf("unexpected command store backend: %q", commandStore.Backend())
	}
	if !commandStore.Status().Supported {
		t.Fatal("expected command store status to delegate to support func")
	}
	value, err := commandStore.Get("demo", "token")
	if err != nil || value != "demo:token" {
		t.Fatalf("unexpected command store get result: value=%q err=%v", value, err)
	}
	if err := commandStore.Set("demo", "token", "value"); err != nil {
		t.Fatalf("unexpected command store set error: %v", err)
	}
	if err := commandStore.Delete("demo", "token"); err != nil {
		t.Fatalf("unexpected command store delete error: %v", err)
	}
	if called != 3 {
		t.Fatalf("expected command store delegates to be invoked 3 times, got %d", called)
	}
}

func bytesOfLen(n int) []byte {
	return make([]byte, n)
}
