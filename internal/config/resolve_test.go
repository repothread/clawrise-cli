package config

import "testing"

func TestResolveSecretWithEnvReference(t *testing.T) {
	t.Setenv("CLAWRISE_TEST_SECRET", "secret-value")

	value, err := ResolveSecret("env:CLAWRISE_TEST_SECRET")
	if err != nil {
		t.Fatalf("ResolveSecret returned error: %v", err)
	}
	if value != "secret-value" {
		t.Fatalf("unexpected secret value: %s", value)
	}
}

func TestValidateGrantClientCredentials(t *testing.T) {
	t.Setenv("CLAWRISE_TEST_APP_ID", "app-id")
	t.Setenv("CLAWRISE_TEST_APP_SECRET", "app-secret")

	err := ValidateGrant(Profile{
		Platform: "feishu",
		Subject:  "bot",
		Grant: Grant{
			Type:      "client_credentials",
			AppID:     "env:CLAWRISE_TEST_APP_ID",
			AppSecret: "env:CLAWRISE_TEST_APP_SECRET",
		},
	})
	if err != nil {
		t.Fatalf("ValidateGrant returned error: %v", err)
	}
}

func TestValidateGrantNotionStaticToken(t *testing.T) {
	t.Setenv("CLAWRISE_TEST_NOTION_TOKEN", "notion-token")

	err := ValidateGrant(Profile{
		Platform: "notion",
		Subject:  "integration",
		Grant: Grant{
			Type:  "static_token",
			Token: "env:CLAWRISE_TEST_NOTION_TOKEN",
		},
	})
	if err != nil {
		t.Fatalf("ValidateGrant returned error: %v", err)
	}
}

func TestValidateGrantRejectsNotionSubjectMismatch(t *testing.T) {
	t.Setenv("CLAWRISE_TEST_NOTION_TOKEN", "notion-token")

	err := ValidateGrant(Profile{
		Platform: "notion",
		Subject:  "user",
		Grant: Grant{
			Type:  "static_token",
			Token: "env:CLAWRISE_TEST_NOTION_TOKEN",
		},
	})
	if err == nil {
		t.Fatal("expected ValidateGrant to reject notion subject mismatch")
	}
}

func TestValidateGrantNotionOAuthRefreshable(t *testing.T) {
	t.Setenv("CLAWRISE_TEST_NOTION_CLIENT_ID", "client-id")
	t.Setenv("CLAWRISE_TEST_NOTION_CLIENT_SECRET", "client-secret")
	t.Setenv("CLAWRISE_TEST_NOTION_REFRESH_TOKEN", "refresh-token")

	err := ValidateGrant(Profile{
		Platform: "notion",
		Subject:  "integration",
		Grant: Grant{
			Type:         "oauth_refreshable",
			ClientID:     "env:CLAWRISE_TEST_NOTION_CLIENT_ID",
			ClientSecret: "env:CLAWRISE_TEST_NOTION_CLIENT_SECRET",
			RefreshToken: "env:CLAWRISE_TEST_NOTION_REFRESH_TOKEN",
		},
	})
	if err != nil {
		t.Fatalf("ValidateGrant returned error: %v", err)
	}
}
