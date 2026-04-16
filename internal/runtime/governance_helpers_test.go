package runtime

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/clawrise/clawrise-cli/internal/config"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

func TestGovernanceConversionHelpersRoundTrip(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	record := persistedIdempotencyRecord{
		Key:        "idem-demo",
		Operation:  "notion.page.create",
		InputHash:  "hash-demo",
		Status:     "executed",
		RequestID:  "req-demo",
		CreatedAt:  now,
		UpdatedAt:  now,
		RetryCount: 2,
		Data:       map[string]any{"page_id": "page_demo"},
		Error:      &ErrorBody{Code: "UPSTREAM_FAILED", Message: "boom", Retryable: true, UpstreamCode: "500", HTTPStatus: 500},
		Meta:       Meta{Platform: "notion", DurationMS: 99, RetryCount: 2, DryRun: false},
	}

	pluginRecord := convertGovernanceRecordToPlugin(record)
	if pluginRecord.Key != "idem-demo" || pluginRecord.Meta.Platform != "notion" {
		t.Fatalf("unexpected plugin governance record: %+v", pluginRecord)
	}
	roundTrip := convertGovernanceRecordFromPlugin(pluginRecord)
	if roundTrip.Key != record.Key || roundTrip.Error == nil || roundTrip.Error.Code != record.Error.Code || roundTrip.Meta.Platform != record.Meta.Platform {
		t.Fatalf("unexpected converted runtime governance record: %+v", roundTrip)
	}

	contextPayload := convertGovernanceContextToPlugin(&Context{Platform: "notion", Subject: "integration", Account: "notion_live"})
	if contextPayload == nil || contextPayload.Account != "notion_live" {
		t.Fatalf("unexpected governance context conversion: %+v", contextPayload)
	}

	idempotencyPayload := convertGovernanceIdempotencyStateToPlugin(&IdempotencyState{Key: "idem-demo", Status: "executed", Persisted: true, UpdatedAt: now})
	if idempotencyPayload == nil || idempotencyPayload.Key != "idem-demo" {
		t.Fatalf("unexpected governance idempotency conversion: %+v", idempotencyPayload)
	}

	retryErr := buildRetryAbortError(errors.New("stop retrying"))
	if retryErr == nil || retryErr.Code != "RETRY_ABORTED" || retryErr.Retryable {
		t.Fatalf("unexpected retry abort error: %+v", retryErr)
	}
}

func TestFileGovernanceStoreAndErrorStoreHelpers(t *testing.T) {
	paths := runtimePaths{rootDir: filepath.Join(t.TempDir(), "runtime"), idempotencyDir: filepath.Join(t.TempDir(), "runtime", "idempotency"), auditDir: filepath.Join(t.TempDir(), "runtime", "audit")}
	store := &fileGovernanceStore{paths: paths}

	if err := store.EnsureIdempotencyDir(); err != nil {
		t.Fatalf("EnsureIdempotencyDir returned error: %v", err)
	}

	record := &persistedIdempotencyRecord{Key: "idem:file", Operation: "notion.page.update", Status: "executed", RequestID: "req-file"}
	if err := store.SaveIdempotencyRecord(record); err != nil {
		t.Fatalf("SaveIdempotencyRecord returned error: %v", err)
	}
	loaded, err := store.LoadIdempotencyRecord("idem:file")
	if err != nil {
		t.Fatalf("LoadIdempotencyRecord returned error: %v", err)
	}
	if loaded == nil || loaded.Key != "idem:file" {
		t.Fatalf("unexpected loaded idempotency record: %+v", loaded)
	}
	missing, err := store.LoadIdempotencyRecord("missing")
	if err != nil {
		t.Fatalf("LoadIdempotencyRecord returned error for missing key: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil missing record, got: %+v", missing)
	}
	if err := os.WriteFile(filepath.Join(paths.idempotencyDir, safeFilename("broken")+".json"), []byte("{"), 0o600); err != nil {
		t.Fatalf("failed to seed broken idempotency record: %v", err)
	}
	if _, err := store.LoadIdempotencyRecord("broken"); err == nil || !strings.Contains(err.Error(), "failed to decode idempotency record") {
		t.Fatalf("expected broken idempotency decode error, got: %v", err)
	}

	audit := auditRecord{Time: "2026-04-10T00:00:00Z", RequestID: "req-file", Operation: "notion.page.update", OK: true, Meta: Meta{Platform: "notion"}}
	if err := store.AppendAuditRecord("2026-04-10", audit); err != nil {
		t.Fatalf("AppendAuditRecord returned error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(paths.auditDir, "2026-04-10.jsonl"))
	if err != nil {
		t.Fatalf("failed to read audit file: %v", err)
	}
	if !bytes.Contains(data, []byte(`"operation":"notion.page.update"`)) {
		t.Fatalf("unexpected audit log file contents: %s", string(data))
	}

	failErr := errors.New("governance unavailable")
	errStore := &errorGovernanceStore{err: failErr}
	if err := errStore.EnsureIdempotencyDir(); !errors.Is(err, failErr) {
		t.Fatalf("expected EnsureIdempotencyDir to forward error, got: %v", err)
	}
	if _, err := errStore.LoadIdempotencyRecord("idem"); !errors.Is(err, failErr) {
		t.Fatalf("expected LoadIdempotencyRecord to forward error, got: %v", err)
	}
	if err := errStore.SaveIdempotencyRecord(record); !errors.Is(err, failErr) {
		t.Fatalf("expected SaveIdempotencyRecord to forward error, got: %v", err)
	}
	if err := errStore.AppendAuditRecord("2026-04-10", audit); !errors.Is(err, failErr) {
		t.Fatalf("expected AppendAuditRecord to forward error, got: %v", err)
	}
}

func TestPluginGovernanceStoreWrapperMethods(t *testing.T) {
	root := t.TempDir()
	manifest := writeGovernanceRuntimePluginManifest(t, root, "governance-wrapper")
	store := &pluginGovernanceStore{client: pluginruntime.NewProcessGovernanceStore(manifest)}
	defer func() { _ = store.client.Close() }()

	if err := store.EnsureIdempotencyDir(); err != nil {
		t.Fatalf("EnsureIdempotencyDir returned error: %v", err)
	}

	record, err := store.LoadIdempotencyRecord("idem_demo")
	if err != nil {
		t.Fatalf("LoadIdempotencyRecord returned error: %v", err)
	}
	if record == nil || record.Key != "idem_demo" || record.Meta.Platform != "demo" {
		t.Fatalf("unexpected governance wrapper load result: %+v", record)
	}

	if err := store.SaveIdempotencyRecord(nil); err != nil {
		t.Fatalf("SaveIdempotencyRecord(nil) returned error: %v", err)
	}
	if err := store.SaveIdempotencyRecord(&persistedIdempotencyRecord{Key: "idem_demo", Operation: "demo.page.update", Status: "executed", RequestID: "req_demo", Meta: Meta{Platform: "demo"}}); err != nil {
		t.Fatalf("SaveIdempotencyRecord returned error: %v", err)
	}

	if err := store.AppendAuditRecord("2026-04-10", auditRecord{Time: "2026-04-10T00:00:00Z", RequestID: "req_demo", Operation: "demo.page.update", OK: true, Meta: Meta{Platform: "demo"}}); err != nil {
		t.Fatalf("AppendAuditRecord returned error: %v", err)
	}
}

func TestPluginGovernanceStoreLoadIdempotencyNotFound(t *testing.T) {
	root := t.TempDir()
	manifest := writeGovernanceRuntimePluginManifestWithLoadResult(t, root, "governance-not-found", false)
	store := &pluginGovernanceStore{client: pluginruntime.NewProcessGovernanceStore(manifest)}
	defer func() { _ = store.client.Close() }()

	record, err := store.LoadIdempotencyRecord("missing")
	if err != nil {
		t.Fatalf("LoadIdempotencyRecord returned error: %v", err)
	}
	if record != nil {
		t.Fatalf("expected nil record for not-found result, got: %+v", record)
	}
}

func TestOpenPluginGovernanceStoreReturnsNotFoundWhenCapabilityMissing(t *testing.T) {
	t.Setenv("CLAWRISE_PLUGIN_PATHS", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	store, found, err := openPluginGovernanceStore("plugin.missing", "", nil)
	if err != nil {
		t.Fatalf("openPluginGovernanceStore returned error: %v", err)
	}
	if found || store != nil {
		t.Fatalf("expected no governance plugin match, got store=%T found=%v", store, found)
	}
}

func TestOpenGovernanceStoreHandlesFallbacksAndUnsupportedPluginBinding(t *testing.T) {
	paths := runtimePaths{rootDir: filepath.Join(t.TempDir(), "runtime"), idempotencyDir: filepath.Join(t.TempDir(), "runtime", "idempotency"), auditDir: filepath.Join(t.TempDir(), "runtime", "audit")}

	fileStore := openGovernanceStore(paths, config.StoragePluginBinding{Backend: "auto"}, nil)
	if _, ok := fileStore.(*fileGovernanceStore); !ok {
		t.Fatalf("expected auto backend to resolve to file store, got: %T", fileStore)
	}

	fallbackStore := openGovernanceStore(paths, config.StoragePluginBinding{Backend: "unknown"}, nil)
	if _, ok := fallbackStore.(*fileGovernanceStore); !ok {
		t.Fatalf("expected unknown backend without plugin to fall back to file store, got: %T", fallbackStore)
	}

	unsupportedPluginStore := openGovernanceStore(paths, config.StoragePluginBinding{Backend: "unknown", Plugin: "custom-plugin"}, nil)
	errStore, ok := unsupportedPluginStore.(*errorGovernanceStore)
	if !ok {
		t.Fatalf("expected explicit plugin binding to return error store, got: %T", unsupportedPluginStore)
	}
	if err := errStore.EnsureIdempotencyDir(); err == nil || !strings.Contains(err.Error(), "unsupported governance backend") {
		t.Fatalf("expected unsupported governance backend error, got: %v", err)
	}
}
