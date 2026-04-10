package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreSaveLoadResolveAndPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("CLAWRISE_CONFIG", "")

	store, err := ResolveStore()
	if err != nil {
		t.Fatalf("ResolveStore returned error: %v", err)
	}
	wantPath := filepath.Join(homeDir, ".clawrise", "config.yaml")
	if store.Path() != wantPath {
		t.Fatalf("unexpected resolved store path: got=%s want=%s", store.Path(), wantPath)
	}

	cfg := &Config{}
	cfg.Defaults.Platform = "feishu"
	cfg.Defaults.Account = "ops-bot"
	cfg.Accounts = map[string]Account{
		"ops-bot": {
			Platform: "feishu",
			Subject:  "bot",
			Auth: AccountAuth{
				Method: "feishu.app_credentials",
			},
		},
	}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	info, err := os.Stat(store.Path())
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected config file mode 0600, got %#o", info.Mode().Perm())
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Defaults.Platform != "feishu" || loaded.Defaults.Account != "ops-bot" {
		t.Fatalf("unexpected loaded defaults: %+v", loaded.Defaults)
	}
	if loaded.Accounts["ops-bot"].Auth.Method != "feishu.app_credentials" {
		t.Fatalf("unexpected loaded account: %+v", loaded.Accounts["ops-bot"])
	}
}

func TestStoreLoadHandlesMissingEmptyAndInvalidFiles(t *testing.T) {
	missingStore := NewStore(filepath.Join(t.TempDir(), "missing.yaml"))
	cfg, err := missingStore.Load()
	if err != nil {
		t.Fatalf("Load returned error for missing file: %v", err)
	}
	if cfg == nil || cfg.Accounts == nil {
		t.Fatalf("expected missing-file load to return initialized config, got %+v", cfg)
	}

	emptyPath := filepath.Join(t.TempDir(), "empty.yaml")
	if err := os.WriteFile(emptyPath, nil, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	emptyCfg, err := NewStore(emptyPath).Load()
	if err != nil {
		t.Fatalf("Load returned error for empty file: %v", err)
	}
	if emptyCfg == nil || emptyCfg.Accounts == nil {
		t.Fatalf("expected empty-file load to return initialized config, got %+v", emptyCfg)
	}

	invalidPath := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(invalidPath, []byte("defaults: ["), 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	_, err = NewStore(invalidPath).Load()
	if err == nil || !strings.Contains(err.Error(), "failed to parse config file") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestConfigMarshalInitializesMaps(t *testing.T) {
	cfg := &Config{}
	data, err := cfg.Marshal()
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected Marshal to return yaml data")
	}
	if cfg.Accounts == nil || cfg.Defaults.PlatformAccounts == nil || cfg.Plugins.Enabled == nil || cfg.Plugins.Bindings.Providers == nil || cfg.Plugins.Bindings.AuthLaunchers == nil || cfg.Plugins.PluginConfig == nil {
		t.Fatalf("expected Marshal to ensure nested maps, got %+v", cfg)
	}
}
