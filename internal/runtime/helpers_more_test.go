package runtime

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
)

func TestExecutorHelperFunctionsAndAccountSelection(t *testing.T) {
	if got := appendUniqueStrings([]string{" alpha ", "alpha", " ", "beta"}, "beta", " gamma ", ""); len(got) != 3 || got[0] != "alpha" || got[1] != "beta" || got[2] != "gamma" {
		t.Fatalf("unexpected appendUniqueStrings result: %+v", got)
	}
	if got := appendUniqueStrings(nil); got != nil {
		t.Fatalf("expected appendUniqueStrings(nil) to stay nil, got %+v", got)
	}

	cfg := config.New()
	cfg.Ensure()
	cfg.Defaults.Subject = "user"
	cfg.Accounts["feishu_user_alice"] = config.Account{Platform: "feishu", Subject: "user"}
	cfg.Accounts["feishu_user_bob"] = config.Account{Platform: "feishu", Subject: "user"}
	cfg.Accounts["feishu_bot_ops"] = config.Account{Platform: "feishu", Subject: "bot"}
	cfg.Accounts["notion_team_docs"] = config.Account{Platform: "notion", Subject: "integration"}

	if _, _, appErr := resolveAccountSelection(cfg, "feishu", "missing", ""); appErr == nil || appErr.Code != "ACCOUNT_NOT_FOUND" {
		t.Fatalf("expected ACCOUNT_NOT_FOUND, got %+v", appErr)
	}
	if _, _, appErr := resolveAccountSelection(cfg, "feishu", "notion_team_docs", ""); appErr == nil || appErr.Code != "ACCOUNT_PLATFORM_MISMATCH" {
		t.Fatalf("expected ACCOUNT_PLATFORM_MISMATCH, got %+v", appErr)
	}
	if _, _, appErr := resolveAccountSelection(cfg, "feishu", "feishu_bot_ops", "user"); appErr == nil || appErr.Code != "ACCOUNT_SUBJECT_MISMATCH" {
		t.Fatalf("expected ACCOUNT_SUBJECT_MISMATCH, got %+v", appErr)
	}

	cfg.Defaults.PlatformAccounts["feishu"] = "missing-default"
	if _, _, appErr := resolveAccountSelection(cfg, "feishu", "", ""); appErr == nil || appErr.Code != "DEFAULT_ACCOUNT_NOT_FOUND" {
		t.Fatalf("expected DEFAULT_ACCOUNT_NOT_FOUND, got %+v", appErr)
	}
	cfg.Defaults.PlatformAccounts["feishu"] = "notion_team_docs"
	if _, _, appErr := resolveAccountSelection(cfg, "feishu", "", ""); appErr == nil || appErr.Code != "DEFAULT_ACCOUNT_PLATFORM_MISMATCH" {
		t.Fatalf("expected DEFAULT_ACCOUNT_PLATFORM_MISMATCH, got %+v", appErr)
	}

	cfg.Defaults.PlatformAccounts["feishu"] = "feishu_bot_ops"
	cfg.Defaults.Account = "feishu_user_alice"
	name, account, appErr := resolveAccountSelection(cfg, "feishu", "", "bot")
	if appErr != nil || name != "feishu_bot_ops" || account.Subject != "bot" {
		t.Fatalf("expected subject-matching platform default account, got name=%q account=%+v err=%+v", name, account, appErr)
	}

	cfg.Defaults.PlatformAccounts["feishu"] = ""
	cfg.Defaults.Account = "feishu_user_alice"
	name, account, appErr = resolveAccountSelection(cfg, "feishu", "", "user")
	if appErr != nil || name != "feishu_user_alice" || account.Subject != "user" {
		t.Fatalf("expected global default account fallback, got name=%q account=%+v err=%+v", name, account, appErr)
	}

	cfg.Defaults.Account = ""
	if _, _, appErr := resolveAccountSelection(cfg, "google", "", "integration"); appErr == nil || appErr.Code != "ACCOUNT_REQUIRED" || !strings.Contains(appErr.Message, "integration") {
		t.Fatalf("expected subject-specific ACCOUNT_REQUIRED, got %+v", appErr)
	}
	if _, _, appErr := resolveAccountSelection(cfg, "feishu", "", "user"); appErr == nil || appErr.Code != "ACCOUNT_AMBIGUOUS" {
		t.Fatalf("expected ACCOUNT_AMBIGUOUS for multiple user accounts, got %+v", appErr)
	}

	delete(cfg.Accounts, "feishu_user_bob")
	name, account, appErr = resolveAccountSelection(cfg, "feishu", "", "user")
	if appErr != nil || name != "feishu_user_alice" || account.Subject != "user" {
		t.Fatalf("expected single candidate to be selected, got name=%q account=%+v err=%+v", name, account, appErr)
	}

	if !supportsProviderPayloadDebug(" notion.page.create ") || supportsProviderPayloadDebug("feishu.doc.create") {
		t.Fatal("unexpected supportsProviderPayloadDebug behavior")
	}
	if !supportsWriteVerification("notion.block.update") || supportsWriteVerification("notion.database.query") {
		t.Fatal("unexpected supportsWriteVerification behavior")
	}
	warnings := writeEnhancementWarnings("feishu.doc.create", ExecuteOptions{DebugProviderPayload: true, VerifyAfterWrite: true})
	if len(warnings) != 2 {
		t.Fatalf("expected unsupported enhancement warnings, got %+v", warnings)
	}
}

func TestGovernanceRetryAndAtomicWriteHelpers(t *testing.T) {
	g := &runtimeGovernance{retry: retryPolicy{maxAttempts: 2, baseDelay: time.Millisecond, maxDelay: 2 * time.Millisecond}}
	if g.shouldRetry(adapter.Definition{Mutating: false}, nil, 0) {
		t.Fatal("expected nil error not to retry")
	}
	if g.shouldRetry(adapter.Definition{Mutating: false}, apperr.New("NOPE", "boom"), 0) {
		t.Fatal("expected non-retryable error not to retry")
	}
	if !g.shouldRetry(adapter.Definition{Mutating: false}, apperr.New("RETRY", "boom").WithRetryable(true), 0) {
		t.Fatal("expected retryable read error to retry")
	}
	if g.shouldRetry(adapter.Definition{Mutating: false}, apperr.New("RETRY", "boom").WithRetryable(true), 2) {
		t.Fatal("expected retry count limit to stop retries")
	}
	if g.shouldRetry(adapter.Definition{Mutating: true}, apperr.New("RETRY", "boom").WithRetryable(true), 0) {
		t.Fatal("expected mutating operation without idempotency not to retry")
	}
	if !g.shouldRetry(adapter.Definition{Mutating: true, Spec: adapter.OperationSpec{Idempotency: adapter.IdempotencySpec{Required: true}}}, apperr.New("RETRY", "boom").WithRetryable(true), 0) {
		t.Fatal("expected mutating operation with idempotency to retry")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := g.waitBeforeRetry(ctx, 2); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled waitBeforeRetry, got %v", err)
	}
	if err := g.waitBeforeRetry(context.Background(), 1); err != nil {
		t.Fatalf("expected successful waitBeforeRetry, got %v", err)
	}

	path := filepath.Join(t.TempDir(), "state.json")
	if err := atomicWriteJSONFile(path, map[string]any{"ok": true}, 0o600); err != nil {
		t.Fatalf("atomicWriteJSONFile returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read atomic write output: %v", err)
	}
	if !bytes.Contains(data, []byte(`"ok": true`)) {
		t.Fatalf("unexpected atomic write contents: %s", string(data))
	}
	if err := atomicWriteJSONFile(filepath.Join(t.TempDir(), "bad.json"), map[string]any{"fn": func() {}}, 0o600); err == nil {
		t.Fatal("expected atomicWriteJSONFile to fail for unsupported json value")
	}
}

func TestAuditSinkAndPolicyHelpers(t *testing.T) {
	if got := (&pluginAuditSink{}).Name(); got != "" {
		t.Fatalf("expected nil plugin audit sink name to be empty, got %q", got)
	}
	if got := (&pluginAuditSink{runtime: &testAuditRuntime{name: "audit-demo", id: "sink-main"}}).Name(); got != "audit-demo/sink-main" {
		t.Fatalf("unexpected plugin audit sink name: %q", got)
	}
	if got := (&pluginAuditSink{runtime: &testAuditRuntime{name: "same", id: "same"}}).Name(); got != "same" {
		t.Fatalf("unexpected same-id plugin audit sink name: %q", got)
	}

	var stdout bytes.Buffer
	if err := (&stdoutAuditSink{writer: &stdout}).Emit(context.Background(), auditRecord{Operation: "demo.page.get", OK: true}); err != nil {
		t.Fatalf("stdout Emit returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"operation":"demo.page.get"`) {
		t.Fatalf("unexpected stdout audit output: %s", stdout.String())
	}
	if err := (&stdoutAuditSink{}).Emit(context.Background(), auditRecord{}); err != nil {
		t.Fatalf("expected nil stdout writer to no-op, got %v", err)
	}
	var nilWebhook *webhookAuditSink
	if err := nilWebhook.Emit(context.Background(), auditRecord{}); err != nil {
		t.Fatalf("expected nil webhook sink to no-op, got %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test") != "demo" {
			t.Fatalf("expected custom audit header, got %q", r.Header.Get("X-Test"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	if err := (&webhookAuditSink{url: server.URL, headers: map[string]string{"X-Test": "demo"}, timeout: time.Second}).Emit(context.Background(), auditRecord{Operation: "demo.page.update", OK: true}); err != nil {
		t.Fatalf("webhook Emit returned error: %v", err)
	}

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer failServer.Close()
	if err := (&webhookAuditSink{url: failServer.URL, timeout: time.Second}).Emit(context.Background(), auditRecord{}); err == nil || !strings.Contains(err.Error(), "webhook returned status 502") {
		t.Fatalf("expected webhook non-2xx error, got %v", err)
	}

	if !matchesOperationPattern("notion.page.*", "notion.page.update") {
		t.Fatal("expected wildcard operation pattern to match")
	}
	if matchesOperationPattern("notion.page.*", "notion.database.query") {
		t.Fatal("expected wildcard operation pattern not to match other resource path")
	}
	if matchesOperationPattern("", "notion.page.update") {
		t.Fatal("expected empty operation pattern not to match")
	}
	if matched, ok := firstMatchingOperationPattern([]string{" ", "notion.page.*", "notion.*"}, "notion.page.update"); !ok || matched != "notion.page.*" {
		t.Fatalf("unexpected firstMatchingOperationPattern result: matched=%q ok=%v", matched, ok)
	}

	if policyRuntimeSupportsPlatform(nil, "notion") {
		t.Fatal("expected nil policy runtime not to support platform")
	}
	if !policyRuntimeSupportsPlatform(&testPolicyRuntime{platforms: nil}, "notion") {
		t.Fatal("expected empty runtime platform list to allow all platforms")
	}
	if !policyRuntimeSupportsPlatform(&testPolicyRuntime{platforms: []string{"notion"}}, " notion ") {
		t.Fatal("expected exact platform match to succeed")
	}
	if policyRuntimeSupportsPlatform(&testPolicyRuntime{platforms: []string{"feishu"}}, "notion") {
		t.Fatal("expected mismatched platform not to match")
	}
}
