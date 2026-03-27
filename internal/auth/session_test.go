package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionUsableAt(t *testing.T) {
	now := time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC)
	expiresAt := now.Add(10 * time.Minute)

	session := Session{
		AccessToken: "access-token",
		ExpiresAt:   &expiresAt,
	}
	if !session.UsableAt(now, DefaultRefreshSkew) {
		t.Fatal("expected session to be usable before refresh window")
	}
	if session.NeedsRefreshAt(now.Add(9*time.Minute), DefaultRefreshSkew) == false {
		t.Fatal("expected session to require refresh inside refresh window")
	}
}

func TestFileStoreSaveAndLoad(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	store := NewFileStore(configPath)

	now := time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC)
	store.now = func() time.Time {
		return now
	}

	expiresAt := now.Add(30 * time.Minute)
	session := Session{
		ProfileName:  "feishu_user_alice",
		Platform:     "feishu",
		Subject:      "user",
		GrantType:    "oauth_user",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    &expiresAt,
		Metadata: map[string]string{
			"scope": "docx:read",
		},
	}

	if err := store.Save(session); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := store.Load("feishu_user_alice")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.ProfileName != session.ProfileName {
		t.Fatalf("unexpected profile name: %s", loaded.ProfileName)
	}
	if loaded.AccessToken != session.AccessToken {
		t.Fatalf("unexpected access token: %s", loaded.AccessToken)
	}
	if loaded.UpdatedAt == nil || !loaded.UpdatedAt.Equal(now) {
		t.Fatalf("expected updated_at to be written with current time, got: %#v", loaded.UpdatedAt)
	}
	if loaded.CreatedAt == nil || !loaded.CreatedAt.Equal(now) {
		t.Fatalf("expected created_at to be written with current time, got: %#v", loaded.CreatedAt)
	}
}

func TestFileStorePathSanitizesProfileName(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "config.yaml"))
	path := store.Path("team/docs bot")

	if filepath.Base(path) != "team_docs_bot.json" {
		t.Fatalf("unexpected sanitized file name: %s", filepath.Base(path))
	}
}

func TestFileStoreDeleteIgnoresMissingFile(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "config.yaml"))
	if err := store.Delete("missing-profile"); err != nil {
		t.Fatalf("Delete returned error for missing file: %v", err)
	}
}

func TestFileStoreSaveUsesPrivatePermissions(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	store := NewFileStore(configPath)

	session := Session{
		ProfileName: "notion_team_docs",
		Platform:    "notion",
		Subject:     "integration",
		GrantType:   "oauth_refreshable",
		AccessToken: "access-token",
	}
	if err := store.Save(session); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	info, err := os.Stat(store.Path("notion_team_docs"))
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected session file mode 0600, got %#o", info.Mode().Perm())
	}
}
