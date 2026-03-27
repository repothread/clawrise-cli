package config

import "testing"

func TestBuildInitConfigForFeishuBot(t *testing.T) {
	result, err := BuildInitConfig(InitOptions{
		Platform: "feishu",
		Subject:  "bot",
		Profile:  "feishu_bot_ops",
	})
	if err != nil {
		t.Fatalf("BuildInitConfig returned error: %v", err)
	}

	if result.ProfileName != "feishu_bot_ops" {
		t.Fatalf("unexpected profile name: %s", result.ProfileName)
	}
	profile := result.Config.Profiles["feishu_bot_ops"]
	if profile.Grant.Type != "client_credentials" {
		t.Fatalf("unexpected grant type: %s", profile.Grant.Type)
	}
	if profile.Grant.AppID != "env:FEISHU_BOT_OPS_APP_ID" {
		t.Fatalf("unexpected app_id template: %s", profile.Grant.AppID)
	}
	if profile.Grant.AppSecret != "env:FEISHU_BOT_OPS_APP_SECRET" {
		t.Fatalf("unexpected app_secret template: %s", profile.Grant.AppSecret)
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
		Platform: "notion",
		Subject:  "integration",
		Profile:  "notion_team_docs",
	})
	if err != nil {
		t.Fatalf("BuildInitConfig returned error: %v", err)
	}

	profile := result.Config.Profiles["notion_team_docs"]
	if profile.Grant.Type != "static_token" {
		t.Fatalf("unexpected grant type: %s", profile.Grant.Type)
	}
	if profile.Grant.Token != "env:NOTION_TEAM_DOCS_TOKEN" {
		t.Fatalf("unexpected token template: %s", profile.Grant.Token)
	}
	if profile.Grant.NotionVer != defaultNotionVersion {
		t.Fatalf("unexpected notion version: %s", profile.Grant.NotionVer)
	}
}
