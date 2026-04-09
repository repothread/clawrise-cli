package authflow

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStoreSaveLoadDelete(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	store := NewFileStore(configPath)

	fixedNow := time.Date(2026, 4, 8, 12, 45, 0, 0, time.UTC)
	store.now = func() time.Time { return fixedNow }

	flow := Flow{
		ID:          " flow / with spaces ",
		AccountName: "notion_bot",
		Platform:    "notion",
		Method:      "notion.oauth_public",
		State:       "pending",
	}
	if err := store.Save(flow); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	expectedPath := filepath.Join(filepath.Dir(configPath), "state", "auth", "flows", "flow_with_spaces.json")
	if store.Path(flow.ID) != expectedPath {
		t.Fatalf("unexpected flow path: %s", store.Path(flow.ID))
	}

	loaded, err := store.Load(flow.ID)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.ID != "flow / with spaces" {
		t.Fatalf("unexpected stored flow id: %q", loaded.ID)
	}
	if loaded.CreatedAt != fixedNow || loaded.UpdatedAt != fixedNow {
		t.Fatalf("unexpected timestamps: created=%s updated=%s", loaded.CreatedAt, loaded.UpdatedAt)
	}
	if loaded.ExpiresAt != fixedNow.Add(DefaultFlowTTL) {
		t.Fatalf("unexpected expiry: %s", loaded.ExpiresAt)
	}

	if err := store.Delete(flow.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := store.Load(flow.ID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist after delete, got: %v", err)
	}
}

func TestOpenStoreWithOptionsDefaultsToFileBackend(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	store, err := OpenStoreWithOptions(StoreOptions{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("OpenStoreWithOptions returned error: %v", err)
	}

	fileStore, ok := store.(*FileStore)
	if !ok {
		t.Fatalf("expected file store, got %T", store)
	}
	if fileStore.rootDir == "" {
		t.Fatal("expected file store rootDir to be initialized")
	}
}

func TestOpenStoreWithOptionsRejectsUnknownBackend(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	_, err := OpenStoreWithOptions(StoreOptions{
		ConfigPath: configPath,
		Backend:    "unknown-backend",
	})
	if err == nil {
		t.Fatal("expected unsupported backend error")
	}
}
