package auth

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type stubSessionStore struct {
	backend string
}

func (s *stubSessionStore) Load(accountName string) (*Session, error) { return nil, nil }
func (s *stubSessionStore) Save(session Session) error                { return nil }
func (s *stubSessionStore) Delete(accountName string) error           { return nil }
func (s *stubSessionStore) Path(accountName string) string            { return s.backend + ":" + accountName }

func TestSessionHelperBranches(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	expiresLater := now.Add(time.Minute)

	empty := Session{}
	if empty.HasAccessToken() {
		t.Fatal("expected empty session to have no access token")
	}
	if empty.CanRefresh() {
		t.Fatal("expected empty session to have no refresh token")
	}
	if !empty.NeedsRefreshAt(now, DefaultRefreshSkew) {
		t.Fatal("expected empty session to need refresh")
	}
	if empty.UsableAt(now, DefaultRefreshSkew) {
		t.Fatal("expected empty session to be unusable")
	}

	session := Session{
		AccessToken:  " access-token ",
		RefreshToken: " refresh-token ",
		ExpiresAt:    &expiresLater,
	}
	if !session.HasAccessToken() {
		t.Fatal("expected trimmed access token to count as present")
	}
	if !session.CanRefresh() {
		t.Fatal("expected trimmed refresh token to count as present")
	}
	if session.NeedsRefreshAt(now, -time.Minute) {
		t.Fatal("expected negative skew to clamp to zero")
	}

	session.ExpiresAt = nil
	if session.NeedsRefreshAt(now, DefaultRefreshSkew) {
		t.Fatal("expected session without expiry to skip refresh")
	}
}

func TestOpenStoreWithOptionsSupportsBuiltinAndExternalBackends(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	backendName := "custom_builtin_" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))
	RegisterStoreBackend(backendName, func(configPath string) Store {
		return &stubSessionStore{backend: backendName}
	})

	builtinStore, err := OpenStoreWithOptions(StoreOptions{
		ConfigPath: configPath,
		Backend:    strings.ToUpper(backendName),
	})
	if err != nil {
		t.Fatalf("OpenStoreWithOptions returned error for builtin backend: %v", err)
	}
	if builtinStore.Path("demo") != backendName+":demo" {
		t.Fatalf("unexpected builtin store implementation: %T", builtinStore)
	}

	externalBackend := "external_backend_" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))
	pluginName := "external-plugin"
	var captured StoreOptions
	RegisterExternalStoreResolver(func(options StoreOptions) (Store, bool, error) {
		if options.Backend != externalBackend || options.Plugin != pluginName {
			return nil, false, nil
		}
		captured = options
		return &stubSessionStore{backend: "external"}, true, nil
	})

	externalStore, err := OpenStoreWithOptions(StoreOptions{
		ConfigPath:     configPath,
		Backend:        externalBackend,
		Plugin:         pluginName,
		EnabledPlugins: map[string]string{"demo": "enabled"},
	})
	if err != nil {
		t.Fatalf("OpenStoreWithOptions returned error for external backend: %v", err)
	}
	if externalStore.Path("demo") != "external:demo" {
		t.Fatalf("unexpected external store implementation: %T", externalStore)
	}
	if captured.ConfigPath != configPath || captured.Backend != externalBackend || captured.Plugin != pluginName {
		t.Fatalf("unexpected resolver options: %+v", captured)
	}
}

func TestOpenStoreWithOptionsPropagatesExternalResolverErrors(t *testing.T) {
	externalBackend := "failing_backend_" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "_"))
	resolverErr := errors.New("resolver boom")
	RegisterExternalStoreResolver(func(options StoreOptions) (Store, bool, error) {
		if options.Backend != externalBackend {
			return nil, false, nil
		}
		return nil, false, resolverErr
	})

	_, err := OpenStoreWithOptions(StoreOptions{Backend: externalBackend, Plugin: "custom-plugin"})
	if !errors.Is(err, resolverErr) {
		t.Fatalf("expected resolver error, got %v", err)
	}
}

func TestFileStoreLoadReturnsDecodeErrorAndSanitizeFallback(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "config.yaml"))
	if err := os.MkdirAll(filepath.Dir(store.Path("bad account")), 0o700); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(store.Path("bad account"), []byte("not-json"), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := store.Load("bad account")
	if err == nil || !strings.Contains(err.Error(), "failed to decode session file") {
		t.Fatalf("expected decode error, got %v", err)
	}

	if got := sanitizeAccountName("   "); got != "default" {
		t.Fatalf("expected blank account name to sanitize to default, got %q", got)
	}
	if got := sanitizeAccountName(" team/docs bot "); got != "team_docs_bot" {
		t.Fatalf("unexpected sanitized account name: %q", got)
	}
}
