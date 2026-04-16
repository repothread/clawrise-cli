package config

import (
	authcache "github.com/clawrise/clawrise-cli/internal/auth"
	"path/filepath"
	"testing"
	"time"
)

func TestSortedAccountInspectionsAndOAuthStatusVariants(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("CLAWRISE_CONFIG", configPath)
	t.Setenv("CLAWRISE_TEST_CLIENT_ID", "client-id")
	t.Setenv("CLAWRISE_TEST_CLIENT_SECRET", "client-secret")
	t.Setenv("CLAWRISE_TEST_ACCESS_TOKEN", "configured-access-token")

	store := authcache.NewFileStore(configPath)
	future := time.Now().UTC().Add(30 * time.Minute)
	past := time.Now().UTC().Add(-30 * time.Minute)

	if err := store.Save(authcache.Session{
		AccountName: "alpha",
		Platform:    "feishu",
		Subject:     "user",
		GrantType:   "feishu.oauth_user",
		AccessToken: "session-token",
		ExpiresAt:   &future,
	}); err != nil {
		t.Fatalf("failed to save valid session: %v", err)
	}
	if err := store.Save(authcache.Session{
		AccountName:  "bravo",
		Platform:     "feishu",
		Subject:      "user",
		GrantType:    "oauth_user",
		AccessToken:  "expired-token",
		RefreshToken: "refresh-token",
		ExpiresAt:    &past,
	}); err != nil {
		t.Fatalf("failed to save refreshable session: %v", err)
	}

	cfg := New()
	cfg.Accounts = map[string]Account{
		"charlie": {
			Platform: "feishu",
			Subject:  "user",
			Auth: AccountAuth{
				Method: "feishu.oauth_user",
				Public: map[string]any{"client_id": "env:CLAWRISE_TEST_CLIENT_ID"},
				SecretRefs: map[string]string{
					"client_secret": "env:CLAWRISE_TEST_CLIENT_SECRET",
					"access_token":  "env:CLAWRISE_TEST_ACCESS_TOKEN",
				},
			},
		},
		"bravo": {
			Platform: "feishu",
			Subject:  "user",
			Auth: AccountAuth{
				Method: "feishu.oauth_user",
				Public: map[string]any{"client_id": "env:CLAWRISE_TEST_CLIENT_ID"},
				SecretRefs: map[string]string{
					"client_secret": "env:CLAWRISE_TEST_CLIENT_SECRET",
				},
			},
		},
		"alpha": {
			Platform: "feishu",
			Subject:  "user",
			Auth: AccountAuth{
				Method: "feishu.oauth_user",
				Public: map[string]any{"client_id": "env:CLAWRISE_TEST_CLIENT_ID"},
				SecretRefs: map[string]string{
					"client_secret": "env:CLAWRISE_TEST_CLIENT_SECRET",
				},
			},
		},
	}

	inspections := SortedAccountInspections(cfg)
	if len(inspections) != 3 {
		t.Fatalf("unexpected inspection count: %+v", inspections)
	}
	if inspections[0].Name != "alpha" || inspections[1].Name != "bravo" || inspections[2].Name != "charlie" {
		t.Fatalf("expected inspections to be sorted by account name, got %+v", inspections)
	}
	if inspections[0].AuthStatus != "session_valid" || !inspections[0].Ready {
		t.Fatalf("expected alpha to use valid session, got %+v", inspections[0])
	}
	if inspections[1].AuthStatus != "refreshable" || !inspections[1].Ready {
		t.Fatalf("expected bravo to be refreshable, got %+v", inspections[1])
	}
	if inspections[2].AuthStatus != "access_token_configured" || !inspections[2].Ready {
		t.Fatalf("expected charlie to use configured access token, got %+v", inspections[2])
	}
}

func TestResolveSecretAndParseSecretReferenceErrors(t *testing.T) {
	if value, err := ResolveSecret(" literal "); err != nil || value != "literal" {
		t.Fatalf("expected literal secret to pass through, got value=%q err=%v", value, err)
	}
	if value, err := ResolveSecret("   "); err != nil || value != "" {
		t.Fatalf("expected blank secret to resolve to empty, got value=%q err=%v", value, err)
	}
	if _, err := ResolveSecret("env:   "); err == nil {
		t.Fatal("expected invalid env reference to fail")
	}
	if _, err := ResolveSecret("env:CLAWRISE_MISSING_SECRET"); err == nil {
		t.Fatal("expected missing env reference to fail")
	}
	if _, err := ResolveSecret("secret:missing-separator"); err == nil {
		t.Fatal("expected invalid secret reference to fail")
	}

	account, field, err := parseSecretReference("secret: demo-account : token ")
	if err != nil || account != "demo-account" || field != "token" {
		t.Fatalf("unexpected parsed secret reference: account=%q field=%q err=%v", account, field, err)
	}
}
