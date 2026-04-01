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

func TestResolvePluginInstallAllowedSourcesNormalizesConfig(t *testing.T) {
	cfg := New()
	cfg.Plugins.Install.AllowedSources = []string{" https ", "npm", "https", "  ", "local"}

	allowed := ResolvePluginInstallAllowedSources(cfg)
	if len(allowed) != 3 {
		t.Fatalf("unexpected allowed sources: %+v", allowed)
	}
	if allowed[0] != "https" || allowed[1] != "npm" || allowed[2] != "local" {
		t.Fatalf("unexpected normalized allowed sources: %+v", allowed)
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

func TestResolvePolicyPluginsNormalizesConfig(t *testing.T) {
	cfg := New()
	cfg.Runtime.Policy.Mode = " manual "
	cfg.Runtime.Policy.Plugins = []PolicyPluginBinding{
		{Plugin: " policy-a ", PolicyID: " review "},
		{Plugin: "   "},
		{PolicyID: " audit "},
	}

	if mode := ResolvePolicyMode(cfg); mode != RuntimeSelectionModeManual {
		t.Fatalf("unexpected policy mode: %s", mode)
	}
	plugins := ResolvePolicyPlugins(cfg)
	if len(plugins) != 2 {
		t.Fatalf("unexpected normalized policy plugins: %+v", plugins)
	}
	if plugins[0].Plugin != "policy-a" || plugins[0].PolicyID != "review" {
		t.Fatalf("unexpected first policy binding: %+v", plugins[0])
	}
	if plugins[1].Plugin != "" || plugins[1].PolicyID != "audit" {
		t.Fatalf("unexpected second policy binding: %+v", plugins[1])
	}
}

func TestResolveAuditSinksNormalizesConfig(t *testing.T) {
	cfg := New()
	cfg.Runtime.Audit.Mode = "manual"
	cfg.Runtime.Audit.Sinks = []AuditSinkConfig{
		{
			Type:      " webhook ",
			URL:       " env:CLAWRISE_AUDIT_URL ",
			Headers:   map[string]string{" Authorization ": " env:CLAWRISE_AUDIT_TOKEN ", " ": "ignored"},
			TimeoutMS: 2500,
		},
		{
			Plugin: " audit-demo ",
			SinkID: " capture ",
		},
		{},
	}

	if mode := ResolveAuditMode(cfg); mode != RuntimeSelectionModeManual {
		t.Fatalf("unexpected audit mode: %s", mode)
	}
	sinks := ResolveAuditSinks(cfg)
	if len(sinks) != 2 {
		t.Fatalf("unexpected normalized audit sinks: %+v", sinks)
	}
	if sinks[0].Type != AuditSinkTypeWebhook || sinks[0].URL != "env:CLAWRISE_AUDIT_URL" {
		t.Fatalf("unexpected webhook sink: %+v", sinks[0])
	}
	if sinks[0].Headers["Authorization"] != "env:CLAWRISE_AUDIT_TOKEN" {
		t.Fatalf("unexpected webhook headers: %+v", sinks[0].Headers)
	}
	if sinks[1].Type != AuditSinkTypePlugin || sinks[1].Plugin != "audit-demo" || sinks[1].SinkID != "capture" {
		t.Fatalf("unexpected plugin sink: %+v", sinks[1])
	}
}

func TestSetPolicyModeValidatesInput(t *testing.T) {
	cfg := New()

	mode, err := SetPolicyMode(cfg, " manual ")
	if err != nil {
		t.Fatalf("SetPolicyMode returned error: %v", err)
	}
	if mode != RuntimeSelectionModeManual || cfg.Runtime.Policy.Mode != RuntimeSelectionModeManual {
		t.Fatalf("unexpected policy mode result: %s %+v", mode, cfg.Runtime.Policy)
	}

	if _, err := SetPolicyMode(cfg, "sometimes"); err == nil {
		t.Fatal("expected SetPolicyMode to reject an invalid mode")
	}
}

func TestAddAndRemovePolicyPluginBinding(t *testing.T) {
	cfg := New()

	items := AddPolicyPluginBinding(cfg, PolicyPluginBinding{Plugin: " sample-policy ", PolicyID: " review "})
	if len(items) != 1 || items[0].Plugin != "sample-policy" || items[0].PolicyID != "review" {
		t.Fatalf("unexpected policy bindings after add: %+v", items)
	}

	items = AddPolicyPluginBinding(cfg, PolicyPluginBinding{Plugin: "sample-policy", PolicyID: "review"})
	if len(items) != 1 {
		t.Fatalf("expected duplicate add to be ignored, got: %+v", items)
	}

	items = RemovePolicyPluginBinding(cfg, PolicyPluginBinding{Plugin: "sample-policy"})
	if len(items) != 0 {
		t.Fatalf("expected bindings to be removed, got: %+v", items)
	}
}

func TestSetAuditModeValidatesInput(t *testing.T) {
	cfg := New()

	mode, err := SetAuditMode(cfg, "disabled")
	if err != nil {
		t.Fatalf("SetAuditMode returned error: %v", err)
	}
	if mode != RuntimeSelectionModeDisabled || cfg.Runtime.Audit.Mode != RuntimeSelectionModeDisabled {
		t.Fatalf("unexpected audit mode result: %s %+v", mode, cfg.Runtime.Audit)
	}

	if _, err := SetAuditMode(cfg, "on"); err == nil {
		t.Fatal("expected SetAuditMode to reject an invalid mode")
	}
}

func TestAddAndRemoveAuditSinks(t *testing.T) {
	cfg := New()

	sinks, err := AddAuditSink(cfg, AuditSinkConfig{Type: "stdout"})
	if err != nil {
		t.Fatalf("AddAuditSink returned error for stdout: %v", err)
	}
	if len(sinks) != 1 || sinks[0].Type != AuditSinkTypeStdout {
		t.Fatalf("unexpected stdout sink state: %+v", sinks)
	}

	sinks, err = AddAuditSink(cfg, AuditSinkConfig{
		Type:      "webhook",
		URL:       " env:CLAWRISE_AUDIT_URL ",
		Headers:   map[string]string{" Authorization ": " env:TOKEN "},
		TimeoutMS: 3000,
	})
	if err != nil {
		t.Fatalf("AddAuditSink returned error for webhook: %v", err)
	}
	if len(sinks) != 2 || sinks[1].Type != AuditSinkTypeWebhook || sinks[1].URL != "env:CLAWRISE_AUDIT_URL" {
		t.Fatalf("unexpected webhook sink state: %+v", sinks)
	}

	sinks, err = AddAuditSink(cfg, AuditSinkConfig{Plugin: " audit-demo ", SinkID: " capture "})
	if err != nil {
		t.Fatalf("AddAuditSink returned error for plugin sink: %v", err)
	}
	if len(sinks) != 3 || sinks[2].Type != AuditSinkTypePlugin || sinks[2].Plugin != "audit-demo" || sinks[2].SinkID != "capture" {
		t.Fatalf("unexpected plugin sink state: %+v", sinks)
	}

	sinks = RemoveAuditSink(cfg, AuditSinkConfig{Type: AuditSinkTypeWebhook, URL: "env:CLAWRISE_AUDIT_URL"})
	if len(sinks) != 2 {
		t.Fatalf("expected webhook sink to be removed, got: %+v", sinks)
	}

	sinks = RemoveAuditSink(cfg, AuditSinkConfig{Type: AuditSinkTypePlugin, Plugin: "audit-demo"})
	if len(sinks) != 1 || sinks[0].Type != AuditSinkTypeStdout {
		t.Fatalf("expected plugin sink to be removed, got: %+v", sinks)
	}
}
