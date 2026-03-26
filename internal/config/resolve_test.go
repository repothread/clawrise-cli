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
