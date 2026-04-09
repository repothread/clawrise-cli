package authflow

import (
	"slices"
	"testing"
)

func TestLookupMethodSpecRecognizesSupportedMethods(t *testing.T) {
	feishuSpec, ok := LookupMethodSpec("feishu.oauth_user")
	if !ok {
		t.Fatal("expected feishu.oauth_user to be recognized")
	}
	if !feishuSpec.Interactive || !feishuSpec.SupportsCodeFlow {
		t.Fatalf("unexpected feishu method spec: %+v", feishuSpec)
	}
	if feishuSpec.DefaultMode != "local_browser" {
		t.Fatalf("unexpected default mode: %s", feishuSpec.DefaultMode)
	}
	if !slices.Equal(feishuSpec.Modes, []string{"local_browser", "manual_url", "manual_code"}) {
		t.Fatalf("unexpected feishu modes: %+v", feishuSpec.Modes)
	}

	notionSpec, ok := LookupMethodSpec("notion.internal_token")
	if !ok {
		t.Fatal("expected notion.internal_token to be recognized")
	}
	if notionSpec.Interactive || notionSpec.SupportsCodeFlow {
		t.Fatalf("unexpected notion method spec: %+v", notionSpec)
	}
}

func TestLookupMethodSpecRejectsUnknownMethod(t *testing.T) {
	spec, ok := LookupMethodSpec("unknown.method")
	if ok {
		t.Fatalf("expected method lookup to fail, got: %+v", spec)
	}
}

func TestBuildActionsReturnsNilForCompletedFlow(t *testing.T) {
	actions := BuildActions(Flow{State: "completed"})
	if actions != nil {
		t.Fatalf("expected no actions for completed flow, got: %+v", actions)
	}
}

func TestBuildActionsIncludesDeviceCodeAndCallbackActions(t *testing.T) {
	actions := BuildActions(Flow{
		State:            "pending",
		Mode:             "device_code",
		VerificationURL:  "https://example.com/verify",
		AuthorizationURL: "https://example.com/authorize",
	})

	if len(actions) != 4 {
		t.Fatalf("expected 4 actions, got %d: %+v", len(actions), actions)
	}
	if actions[0].Type != "device_code" || actions[0].URL != "https://example.com/verify" {
		t.Fatalf("unexpected device code action: %+v", actions[0])
	}
	if actions[1].Type != "open_url" || actions[1].URL != "https://example.com/authorize" {
		t.Fatalf("unexpected open url action: %+v", actions[1])
	}
	if actions[2].Type != "submit_callback_url" || actions[3].Type != "submit_code" {
		t.Fatalf("unexpected terminal actions: %+v", actions)
	}
}

func TestBuildActionsIncludesLoopbackWaitHintForLocalBrowser(t *testing.T) {
	actions := BuildActions(Flow{
		State:            "pending",
		Mode:             "local_browser",
		AuthorizationURL: "https://example.com/authorize",
		CallbackHost:     "127.0.0.1",
		CallbackPort:     43821,
		CallbackPath:     "/callback",
	})

	var waitAction *Action
	for i := range actions {
		if actions[i].Type == "wait_callback" {
			waitAction = &actions[i]
			break
		}
	}
	if waitAction == nil {
		t.Fatalf("expected wait_callback action, got: %+v", actions)
	}
	if waitAction.Host != "127.0.0.1" || waitAction.Port != 43821 || waitAction.Path != "/callback" {
		t.Fatalf("unexpected callback action: %+v", *waitAction)
	}
	if waitAction.TimeoutSec != int(DefaultFlowTTL.Seconds()) {
		t.Fatalf("unexpected timeout: %+v", *waitAction)
	}
}
