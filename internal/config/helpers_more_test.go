package config

import "testing"

func TestBuildAccountAuthBridgeFromAccountAndNormalizeHelpers(t *testing.T) {
	bridge := buildAccountAuthBridgeFromAccount("notion_team_docs", Account{
		Title:    "Docs",
		Platform: "notion",
		Subject:  "integration",
		Auth: AccountAuth{
			Method: " notion.oauth_public ",
			Public: map[string]any{
				"client_id":      " client-id ",
				"notion_version": " 2025-01-01 ",
				"redirect_mode":  " popup ",
				"scopes":         []any{" pages:read ", "", 1, "users:read"},
			},
			SecretRefs: map[string]string{
				"client_secret": " env:NOTION_CLIENT_SECRET ",
				"access_token":  " env:NOTION_ACCESS_TOKEN ",
			},
		},
	})
	if bridge.Method != "notion.oauth_public" {
		t.Fatalf("unexpected method: %+v", bridge)
	}
	if bridge.Params.ClientID != "client-id" || bridge.Params.NotionVersion != "2025-01-01" || bridge.Params.RedirectMode != "popup" {
		t.Fatalf("unexpected params: %+v", bridge.Params)
	}
	if len(bridge.Params.Scopes) != 2 || bridge.Params.Scopes[0] != "pages:read" || bridge.Params.Scopes[1] != "users:read" {
		t.Fatalf("unexpected scopes: %+v", bridge.Params.Scopes)
	}
	if bridge.LegacyAuth.Type != "oauth_refreshable" || bridge.LegacyAuth.ClientSecret != "env:NOTION_CLIENT_SECRET" || bridge.LegacyAuth.AccessToken != "env:NOTION_ACCESS_TOKEN" || bridge.LegacyAuth.RefreshToken != SecretRef("notion_team_docs", "refresh_token") {
		t.Fatalf("unexpected legacy auth bridge: %+v", bridge.LegacyAuth)
	}

	normalized := normalizeAccountAuthBridge("legacy-notion", accountAuthBridge{
		Platform: "notion",
		Subject:  "integration",
		LegacyAuth: legacyAuthConfig{
			Type:      "static_token",
			Token:     "literal-token",
			NotionVer: "2022-06-28",
		},
	})
	if normalized.Method != "notion.internal_token" {
		t.Fatalf("expected method to be inferred, got %+v", normalized)
	}
	if normalized.Params.NotionVersion != "2022-06-28" {
		t.Fatalf("expected notion version default to be copied, got %+v", normalized.Params)
	}
	if normalized.LegacyAuth.Token != "literal-token" {
		t.Fatalf("expected persisted literal token to be preserved, got %+v", normalized.LegacyAuth)
	}

	builtLegacy := buildLegacyAuthConfig("feishu_user_alice", accountAuthBridge{
		Method: "feishu.oauth_user",
		Params: legacyAuthParams{ClientID: "client-id"},
		LegacyAuth: legacyAuthConfig{
			AccessToken: "existing-access-token",
		},
	})
	if builtLegacy.Type != "oauth_user" || builtLegacy.ClientID != "client-id" || builtLegacy.ClientSecret != SecretRef("feishu_user_alice", "client_secret") || builtLegacy.RefreshToken != SecretRef("feishu_user_alice", "refresh_token") || builtLegacy.AccessToken != "existing-access-token" {
		t.Fatalf("unexpected built legacy auth config: %+v", builtLegacy)
	}

	if got := inferMethodFromLegacyAuthType("feishu", "client_credentials"); got != "feishu.app_credentials" {
		t.Fatalf("unexpected inferred method: %q", got)
	}
	if got := inferMethodFromLegacyAuthType("unknown", "type"); got != "" {
		t.Fatalf("expected unknown legacy auth type to return empty method, got %q", got)
	}
	if got := legacyAuthTypeForMethod("notion.oauth_public"); got != "oauth_refreshable" {
		t.Fatalf("unexpected legacy auth type: %q", got)
	}
	if got := legacyAuthTypeForMethod("unknown.method"); got != "" {
		t.Fatalf("expected unknown method to return empty type, got %q", got)
	}
	if got := firstNonEmpty(" ", " second ", "third"); got != "second" {
		t.Fatalf("unexpected firstNonEmpty result: %q", got)
	}
	if !hasPersistedLegacyAuthConfig(legacyAuthConfig{Type: "static_token", Token: "literal-token"}) {
		t.Fatal("expected literal legacy auth config to be treated as persisted")
	}
	if hasPersistedLegacyAuthConfig(legacyAuthConfig{Type: "static_token", Token: SecretRef("demo", "token")}) {
		t.Fatal("expected pure secret refs not to count as persisted legacy auth config")
	}
}

func TestResolveProviderBindingAndAccountValidationHelpers(t *testing.T) {
	cfg := New()
	cfg.Ensure()
	cfg.Plugins.Bindings.Providers[" feishu "] = ProviderPluginBinding{Plugin: " ignored "}
	cfg.Plugins.Bindings.Providers["feishu"] = ProviderPluginBinding{Plugin: " provider-demo "}
	if got := ResolveProviderBinding(cfg, " feishu "); got != "provider-demo" {
		t.Fatalf("unexpected provider binding: %q", got)
	}
	if got := ResolveProviderBinding(cfg, "notion"); got != "" {
		t.Fatalf("expected missing provider binding to be empty, got %q", got)
	}
	if got := ResolveStorageBinding(cfg, "unknown_target"); got != (StoragePluginBinding{}) {
		t.Fatalf("expected unknown storage target to return zero binding, got %+v", got)
	}

	valid := Account{
		Platform: "feishu",
		Subject:  "bot",
		Auth: AccountAuth{
			Method: "feishu.app_credentials",
			SecretRefs: map[string]string{
				"app_secret": "secret:demo:app_secret",
			},
		},
	}
	if err := ValidateAccountShape("demo", valid); err != nil {
		t.Fatalf("ValidateAccountShape returned error for valid account: %v", err)
	}
	if err := ValidateAccount("demo", valid); err != nil {
		t.Fatalf("ValidateAccount returned error for valid account: %v", err)
	}

	for _, tc := range []struct {
		name    string
		account Account
		wantErr string
	}{
		{name: "missing platform", account: Account{Subject: "bot", Auth: AccountAuth{Method: "m"}}, wantErr: "missing platform"},
		{name: "missing subject", account: Account{Platform: "feishu", Auth: AccountAuth{Method: "m"}}, wantErr: "missing subject"},
		{name: "missing method", account: Account{Platform: "feishu", Subject: "bot"}, wantErr: "missing auth method"},
		{name: "empty secret ref field", account: Account{Platform: "feishu", Subject: "bot", Auth: AccountAuth{Method: "m", SecretRefs: map[string]string{" ": "secret:x:y"}}}, wantErr: "secret_refs field must not be empty"},
		{name: "empty secret ref value", account: Account{Platform: "feishu", Subject: "bot", Auth: AccountAuth{Method: "m", SecretRefs: map[string]string{"token": " "}}}, wantErr: "secret_refs.token must not be empty"},
	} {
		if err := ValidateAccountShape("demo", tc.account); err == nil || err.Error() != tc.wantErr {
			t.Fatalf("%s: expected %q, got %v", tc.name, tc.wantErr, err)
		}
	}
}
