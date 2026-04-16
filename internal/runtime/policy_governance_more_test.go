package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/config"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

type testGovernanceAuditSink struct {
	name string
	err  error
	hits int
}

func (s *testGovernanceAuditSink) Name() string { return s.name }
func (s *testGovernanceAuditSink) Emit(ctx context.Context, record auditRecord) error {
	s.hits++
	return s.err
}

type testWriteAuditStore struct {
	err     error
	day     string
	records []auditRecord
}

func (s *testWriteAuditStore) EnsureIdempotencyDir() error { return nil }
func (s *testWriteAuditStore) LoadIdempotencyRecord(key string) (*persistedIdempotencyRecord, error) {
	return nil, nil
}
func (s *testWriteAuditStore) SaveIdempotencyRecord(record *persistedIdempotencyRecord) error {
	return nil
}
func (s *testWriteAuditStore) AppendAuditRecord(day string, record auditRecord) error {
	s.day = day
	s.records = append(s.records, record)
	return s.err
}

func TestLocalPolicyAndPolicyHelperFunctions(t *testing.T) {
	result, warnings, appErr := evaluateLocalPolicy(config.PolicyConfig{
		DenyOperations: []string{"notion.page.*"},
	}, "notion.page.update")
	if appErr == nil || appErr.Code != "POLICY_DENIED" || result.FinalDecision != policyDecisionDeny || len(result.Hits) != 1 || len(warnings) != 0 {
		t.Fatalf("unexpected deny policy result: result=%+v warnings=%+v err=%+v", result, warnings, appErr)
	}

	result, warnings, appErr = evaluateLocalPolicy(config.PolicyConfig{
		RequireApprovalOperations: []string{"notion.page.update"},
	}, "notion.page.update")
	if appErr == nil || appErr.Code != "POLICY_APPROVAL_REQUIRED" || result.FinalDecision != policyDecisionRequireApproval || len(result.Hits) != 1 || len(warnings) != 0 {
		t.Fatalf("unexpected approval policy result: result=%+v warnings=%+v err=%+v", result, warnings, appErr)
	}

	result, warnings, appErr = evaluateLocalPolicy(config.PolicyConfig{
		AnnotateOperations: map[string]string{
			" ":               "ignored",
			"notion.page.*":   "annotated by local rule",
			"notion.page.get": "",
		},
	}, "notion.page.get")
	if appErr != nil || result.FinalDecision != policyDecisionAllow || len(result.Hits) != 2 || len(warnings) != 2 {
		t.Fatalf("unexpected annotate policy result: result=%+v warnings=%+v err=%+v", result, warnings, appErr)
	}
	if warnings[0] != "annotated by local rule" || !strings.Contains(warnings[1], "matched rule notion.page.get") {
		t.Fatalf("unexpected annotate warnings: %+v", warnings)
	}

	if got := normalizePolicyDecision(" REQUIRE_APPROVAL "); got != policyDecisionRequireApproval {
		t.Fatalf("unexpected normalized policy decision: %q", got)
	}
	if got := normalizePolicyDecision(" "); got != policyDecisionAllow {
		t.Fatalf("expected blank policy decision to normalize to allow, got %q", got)
	}
	if got := messageWithColon(" hi "); got != ": hi" {
		t.Fatalf("unexpected messageWithColon result: %q", got)
	}
	if got := messageWithColon(" "); got != "" {
		t.Fatalf("expected blank messageWithColon result, got %q", got)
	}

	policyRuntime := &testPolicyRuntime{name: "policy-demo", id: "review", platforms: []string{"notion"}, priority: 12}
	if got := policyRuntimeKey(policyRuntime); got != "policy-demo|review" {
		t.Fatalf("unexpected policyRuntimeKey: %q", got)
	}
	if got := policyRuntimeKey(nil); got != "" {
		t.Fatalf("expected nil policyRuntimeKey to be empty, got %q", got)
	}
	decisionMessage := buildPolicyDecisionMessage(policyRuntime, pluginruntime.PolicyEvaluateResult{Message: "blocked by reviewer"}, "fallback")
	if decisionMessage != "policy plugin policy-demo/review: blocked by reviewer" {
		t.Fatalf("unexpected policy decision message: %q", decisionMessage)
	}
	annotationWarning := buildPolicyAnnotationWarning(policyRuntime, pluginruntime.PolicyEvaluateResult{})
	if annotationWarning != "policy plugin policy-demo/review: added an execution annotation" {
		t.Fatalf("unexpected annotation warning: %q", annotationWarning)
	}

	hit := buildPluginPolicyHit(policyRuntime, policyDecisionAnnotate, "annotated", map[string]any{"reviewer": "ops"})
	if hit.SourceType != policySourceTypePlugin || hit.SourceName != "policy-demo" || hit.Decision != policyDecisionAnnotate || hit.MatchedRule != "review" || hit.Annotations["reviewer"] != "ops" {
		t.Fatalf("unexpected plugin policy hit: %+v", hit)
	}

	merged := mergePolicyResults(newPolicyResult(), PolicyResult{FinalDecision: policyDecisionAnnotate, Hits: []PolicyHit{{Message: "note"}}})
	if merged.FinalDecision != policyDecisionAnnotate || len(merged.Hits) != 1 {
		t.Fatalf("unexpected merged policy result: %+v", merged)
	}
	merged = mergePolicyResults(PolicyResult{}, PolicyResult{})
	if merged.FinalDecision != policyDecisionAllow {
		t.Fatalf("expected zero-value merge to normalize to allow, got %+v", merged)
	}

	summary := summarizeLocalPolicy(config.PolicyConfig{
		DenyOperations:            []string{"deny.a"},
		RequireApprovalOperations: []string{"approve.a", "approve.b"},
		AnnotateOperations: map[string]string{
			" notion.page.create ": " annotate ",
			" ":                    "ignored",
		},
	})
	if summary.RuleCount != 5 || summary.AnnotateOperations["notion.page.create"] != "annotate" {
		t.Fatalf("unexpected local policy summary: %+v", summary)
	}

	selectorViews := buildPolicySelectorViews([]config.PolicyPluginBinding{{Plugin: "policy-demo", PolicyID: "review"}})
	if len(selectorViews) != 1 || selectorViews[0].Plugin != "policy-demo" || selectorViews[0].PolicyID != "review" {
		t.Fatalf("unexpected policy selector views: %+v", selectorViews)
	}
	if got := buildPolicySelectorViews(nil); got != nil {
		t.Fatalf("expected nil selector views for nil input, got %+v", got)
	}

	selections := summarizePolicySelections([]selectedPolicyRuntime{{Runtime: policyRuntime, Source: "configured"}, {Runtime: nil, Source: "ignored"}})
	if len(selections) != 1 || selections[0].Label != "policy-demo/review" || selections[0].Source != "configured" {
		t.Fatalf("unexpected policy selections summary: %+v", selections)
	}
	if got := summarizePolicySelections(nil); got != nil {
		t.Fatalf("expected nil summary for nil selections, got %+v", got)
	}

	closePolicyRuntimes([]pluginruntime.PolicyRuntime{policyRuntime})
	if !policyRuntime.closed {
		t.Fatal("expected closePolicyRuntimes to close runtime")
	}
}

func TestGovernanceReplayAuditAndAuditLabels(t *testing.T) {
	now := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	store := &testWriteAuditStore{err: errors.New("audit store down")}
	sinkA := &testGovernanceAuditSink{name: "sink-a"}
	sinkB := &testGovernanceAuditSink{name: "sink-b", err: errors.New("emit failed")}
	g := &runtimeGovernance{
		store:        store,
		sinks:        []auditSink{sinkA, sinkB},
		sinkWarnings: []string{"selector warning"},
		now:          func() time.Time { return now },
	}

	state := &IdempotencyState{Key: "idem-1", Status: "prepared"}
	replayed := g.buildReplayEnvelope(now.Add(-time.Second), "req-1", ExecutionProfile{Platform: "notion", Subject: "integration", Account: "notion_live", Name: "docs"}, state, &persistedIdempotencyRecord{
		Operation:  "notion.page.update",
		Status:     "executed",
		Data:       map[string]any{"id": "page_demo"},
		Meta:       Meta{Platform: "notion"},
		RetryCount: 2,
		UpdatedAt:  now.Format(time.RFC3339),
	})
	if !replayed.OK || replayed.Idempotency == nil || replayed.Idempotency.Status != "replayed" || replayed.Context == nil || replayed.Context.Account != "notion_live" {
		t.Fatalf("unexpected replay envelope: %+v", replayed)
	}

	conflict := g.validateIdempotencyConflict(&IdempotencyState{Key: "idem-1"}, &persistedIdempotencyRecord{Operation: "notion.page.create", InputHash: "other", UpdatedAt: now.Format(time.RFC3339)}, "notion.page.update", map[string]any{"id": "page_demo"})
	if conflict == nil || conflict.Code != "IDEMPOTENCY_KEY_CONFLICT" {
		t.Fatalf("expected idempotency conflict, got %+v", conflict)
	}
	if ok := g.validateIdempotencyConflict(&IdempotencyState{Key: "idem-2"}, &persistedIdempotencyRecord{Operation: "notion.page.update", InputHash: mustInputHash(t, "notion.page.update", map[string]any{"id": "page_demo"})}, "notion.page.update", map[string]any{"id": "page_demo"}); ok != nil {
		t.Fatalf("expected matching idempotency inputs not to conflict, got %+v", ok)
	}
	if hashErr := g.validateIdempotencyConflict(&IdempotencyState{Key: "idem-3"}, &persistedIdempotencyRecord{Operation: "notion.page.update", InputHash: "x"}, "notion.page.update", map[string]any{"bad": func() {}}); hashErr == nil || hashErr.Code != "IDEMPOTENCY_HASH_FAILED" {
		t.Fatalf("expected hash failure, got %+v", hashErr)
	}

	warnings := g.writeAudit(Envelope{RequestID: "req-1", Operation: "notion.page.update", OK: false, Error: &ErrorBody{Code: "UPSTREAM_FAILED", Message: "boom"}, Warnings: []string{"runtime warning"}}, map[string]any{"token": "secret", "title": "Demo"})
	if len(store.records) != 1 || store.day != "2026-04-10" {
		t.Fatalf("expected audit record to be written once, got day=%q records=%+v", store.day, store.records)
	}
	if len(warnings) != 3 || warnings[0] != "selector warning" || !strings.Contains(warnings[1], "failed to write governance audit record") || !strings.Contains(warnings[2], "failed to emit event to audit sink sink-b") {
		t.Fatalf("unexpected audit warnings: %+v", warnings)
	}
	if sinkA.hits != 1 || sinkB.hits != 1 {
		t.Fatalf("expected sinks to receive one audit event each, got sinkA=%d sinkB=%d", sinkA.hits, sinkB.hits)
	}

	runtimeA := &testAuditRuntime{name: "audit-demo", id: "sink-main"}
	runtimeB := &testAuditRuntime{name: "plugin-only"}
	if got := auditSinkRuntimeLabel(runtimeA); got != "audit-demo/sink-main" {
		t.Fatalf("unexpected auditSinkRuntimeLabel: %q", got)
	}
	if got := auditSinkRuntimeLabel(runtimeB); got != "plugin-only" {
		t.Fatalf("unexpected plugin-only auditSinkRuntimeLabel: %q", got)
	}
	if got := auditSinkRuntimeLabel(nil); got != "" {
		t.Fatalf("expected nil auditSinkRuntimeLabel to be empty, got %q", got)
	}
	if names := headerNames(map[string]string{" Authorization ": "a", "X-Test": "b", " ": "ignored"}); len(names) != 2 || names[0] != "Authorization" || names[1] != "X-Test" {
		t.Fatalf("unexpected headerNames result: %+v", names)
	}
	if got := headerNames(nil); got != nil {
		t.Fatalf("expected nil headerNames result, got %+v", got)
	}

	g2 := &runtimeGovernance{sinks: []auditSink{&pluginAuditSink{runtime: runtimeA}, &pluginAuditSink{runtime: runtimeB}}}
	g2.closeSinks()
	if !runtimeA.closed || !runtimeB.closed {
		t.Fatalf("expected closeSinks to close plugin audit runtimes, got runtimeA=%v runtimeB=%v", runtimeA.closed, runtimeB.closed)
	}
}

func mustInputHash(t *testing.T, operation string, input map[string]any) string {
	t.Helper()
	hash, err := calculateInputHash(operation, input)
	if err != nil {
		t.Fatalf("calculateInputHash returned error: %v", err)
	}
	return hash
}
