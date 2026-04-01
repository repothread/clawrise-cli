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
