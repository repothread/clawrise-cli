package config

import "testing"

func TestBuildInitConfigForFeishuBot(t *testing.T) {
	result, err := BuildInitConfig(InitOptions{
		Platform:   "feishu",
		Subject:    "bot",
		Connection: "feishu_bot_ops",
	})
	if err != nil {
		t.Fatalf("BuildInitConfig returned error: %v", err)
	}

	if result.ConnectionName != "feishu_bot_ops" {
		t.Fatalf("unexpected connection name: %s", result.ConnectionName)
	}
	connection := result.Config.Connections["feishu_bot_ops"]
	if connection.Method != "feishu.app_credentials" {
		t.Fatalf("unexpected method: %s", connection.Method)
	}
	if connection.Params.AppID == "" {
		t.Fatalf("expected app_id placeholder to be generated")
	}
	if len(result.SecretFields) != 1 || result.SecretFields[0] != "app_secret" {
		t.Fatalf("unexpected secret fields: %+v", result.SecretFields)
	}
	if result.Config.Runtime.Retry.MaxAttempts != 1 {
		t.Fatalf("unexpected retry max attempts: %+v", result.Config.Runtime.Retry)
	}
	if result.Config.Runtime.Retry.BaseDelayMS != 200 || result.Config.Runtime.Retry.MaxDelayMS != 1000 {
		t.Fatalf("unexpected retry delay config: %+v", result.Config.Runtime.Retry)
	}
}

func TestBuildInitConfigForNotionIntegration(t *testing.T) {
	result, err := BuildInitConfig(InitOptions{
		Platform:   "notion",
		Subject:    "integration",
		Connection: "notion_team_docs",
	})
	if err != nil {
		t.Fatalf("BuildInitConfig returned error: %v", err)
	}

	connection := result.Config.Connections["notion_team_docs"]
	if connection.Method != "notion.internal_token" {
		t.Fatalf("unexpected method: %s", connection.Method)
	}
	if len(result.SecretFields) != 1 || result.SecretFields[0] != "token" {
		t.Fatalf("unexpected secret fields: %+v", result.SecretFields)
	}
	if connection.Params.NotionVersion != defaultNotionVersion {
		t.Fatalf("unexpected notion version: %s", connection.Params.NotionVersion)
	}
}
