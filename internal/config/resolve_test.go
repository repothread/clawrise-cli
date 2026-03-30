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

func TestValidateGrantAllowsInteractiveOAuthWithoutRefreshTokenBeforeAuthorization(t *testing.T) {
	t.Setenv("CLAWRISE_TEST_FEISHU_CLIENT_ID", "client-id")
	t.Setenv("CLAWRISE_TEST_FEISHU_CLIENT_SECRET", "client-secret")

	err := ValidateGrant(Profile{
		Platform: "feishu",
		Subject:  "user",
		Grant: Grant{
			Type:         "oauth_user",
			ClientID:     "env:CLAWRISE_TEST_FEISHU_CLIENT_ID",
			ClientSecret: "env:CLAWRISE_TEST_FEISHU_CLIENT_SECRET",
		},
	})
	if err != nil {
		t.Fatalf("ValidateGrant returned error: %v", err)
	}
}

func TestInspectAccountRedactsResolvedSecrets(t *testing.T) {
	t.Setenv("CLAWRISE_TEST_APP_ID", "app-id")
	t.Setenv("CLAWRISE_TEST_APP_SECRET", "app-secret")

	inspection := InspectAccount("feishu_bot_ops", Account{
		Platform: "feishu",
		Subject:  "bot",
		Auth: AccountAuth{
			Method: "feishu.app_credentials",
			Public: map[string]any{
				"app_id": "env:CLAWRISE_TEST_APP_ID",
			},
			SecretRefs: map[string]string{
				"app_secret": "env:CLAWRISE_TEST_APP_SECRET",
			},
		},
	})

	if !inspection.ShapeValid || !inspection.ResolvedValid {
		t.Fatalf("expected inspection to be valid: %+v", inspection)
	}
	if len(inspection.Fields) != 2 {
		t.Fatalf("unexpected inspection fields: %+v", inspection.Fields)
	}
	if inspection.Fields[0].ResolvedValue == "app-id" || inspection.Fields[1].ResolvedValue == "app-secret" {
		t.Fatalf("expected resolved values to be redacted: %+v", inspection.Fields)
	}
}

func TestInspectAccountMarksInteractiveOAuthAsAuthorizationRequiredBeforeAuthorization(t *testing.T) {
	t.Setenv("CLAWRISE_CONFIG", t.TempDir()+"/config.yaml")
	t.Setenv("CLAWRISE_TEST_FEISHU_CLIENT_ID", "client-id")
	t.Setenv("CLAWRISE_TEST_FEISHU_CLIENT_SECRET", "client-secret")

	inspection := InspectAccount("feishu_user_alice", Account{
		Platform: "feishu",
		Subject:  "user",
		Auth: AccountAuth{
			Method: "feishu.oauth_user",
			Public: map[string]any{
				"client_id": "env:CLAWRISE_TEST_FEISHU_CLIENT_ID",
			},
			SecretRefs: map[string]string{
				"client_secret": "env:CLAWRISE_TEST_FEISHU_CLIENT_SECRET",
			},
		},
	})

	if !inspection.ShapeValid || !inspection.ResolvedValid {
		t.Fatalf("expected inspection config to be valid: %+v", inspection)
	}
	if inspection.Ready {
		t.Fatalf("expected inspection to require authorization: %+v", inspection)
	}
	if inspection.AuthStatus != "authorization_required" {
		t.Fatalf("unexpected auth status: %+v", inspection)
	}
}
