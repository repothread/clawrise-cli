package config

import "testing"

func TestResolveStorageBindingPrefersPluginsConfig(t *testing.T) {
	cfg := New()
	cfg.Auth.SecretStore.Backend = "encrypted_file"
	cfg.Auth.SecretStore.Plugin = "builtin"
	cfg.Plugins.Bindings.Storage.SecretStore = StoragePluginBinding{
		Backend: "plugin.demo_secret",
		Plugin:  "secret-demo",
	}

	binding := ResolveStorageBinding(cfg, "secret_store")
	if binding.Backend != "plugin.demo_secret" {
		t.Fatalf("unexpected backend: %+v", binding)
	}
	if binding.Plugin != "secret-demo" {
		t.Fatalf("unexpected plugin: %+v", binding)
	}
}

func TestResolveStorageBindingFallsBackToLegacyFields(t *testing.T) {
	cfg := New()
	cfg.Auth.SessionStore.Backend = "file"
	cfg.Auth.SessionStore.Plugin = "builtin"

	binding := ResolveStorageBinding(cfg, "session_store")
	if binding.Backend != "file" {
		t.Fatalf("unexpected backend: %+v", binding)
	}
	if binding.Plugin != "builtin" {
		t.Fatalf("unexpected plugin: %+v", binding)
	}
}

func TestResolveEnabledPluginsNormalizesConfig(t *testing.T) {
	cfg := New()
	cfg.Ensure()
	cfg.Plugins.Enabled[" demo-provider "] = " ^0.4.0 "
	cfg.Plugins.Enabled["disabled-provider"] = " disabled "
	cfg.Plugins.Enabled["   "] = "ignored"

	rules := ResolveEnabledPlugins(cfg)
	if len(rules) != 2 {
		t.Fatalf("unexpected enabled rules: %+v", rules)
	}
	if rules["demo-provider"] != "^0.4.0" {
		t.Fatalf("unexpected normalized rule: %+v", rules)
	}
	if rules["disabled-provider"] != "disabled" {
		t.Fatalf("unexpected disabled rule: %+v", rules)
	}
}

func TestSetAuthLauncherPreferenceMovesLauncherToFront(t *testing.T) {
	cfg := New()
	cfg.Ensure()
	cfg.Plugins.Bindings.AuthLaunchers["open_url"] = []string{"browser", "device"}

	preferences := SetAuthLauncherPreference(cfg, "open_url", "device")
	if len(preferences) != 2 || preferences[0] != "device" || preferences[1] != "browser" {
		t.Fatalf("unexpected preferences: %+v", preferences)
	}
}

func TestUnsetAuthLauncherPreferenceRemovesLauncherAndActionGroup(t *testing.T) {
	cfg := New()
	cfg.Ensure()
	cfg.Plugins.Bindings.AuthLaunchers["open_url"] = []string{"browser", "device"}

	preferences := UnsetAuthLauncherPreference(cfg, "open_url", "browser")
	if len(preferences) != 1 || preferences[0] != "device" {
		t.Fatalf("unexpected preferences after removing one launcher: %+v", preferences)
	}

	preferences = UnsetAuthLauncherPreference(cfg, "open_url", "device")
	if preferences != nil {
		t.Fatalf("expected empty preferences after removing all launchers, got: %+v", preferences)
	}
	if _, exists := cfg.Plugins.Bindings.AuthLaunchers["open_url"]; exists {
		t.Fatalf("expected action group to be removed, got: %+v", cfg.Plugins.Bindings.AuthLaunchers)
	}
}

func TestResolveAllAuthLauncherPreferencesNormalizesConfig(t *testing.T) {
	cfg := New()
	cfg.Ensure()
	cfg.Plugins.Bindings.AuthLaunchers[" open_url "] = []string{" browser ", "browser", "device"}
	cfg.Plugins.Bindings.AuthLaunchers["   "] = []string{"ignored"}

	preferences := ResolveAllAuthLauncherPreferences(cfg)
	if len(preferences) != 1 {
		t.Fatalf("unexpected preferences: %+v", preferences)
	}
	if got := preferences["open_url"]; len(got) != 2 || got[0] != "browser" || got[1] != "device" {
		t.Fatalf("unexpected normalized preferences: %+v", preferences)
	}
}
