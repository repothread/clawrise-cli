package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/clawrise/clawrise-cli/internal/paths"
)

const (
	// SessionVersion leaves room for future session file migrations.
	SessionVersion = 1

	// DefaultRefreshSkew is the lead time before expiry when a token should be refreshed.
	DefaultRefreshSkew = 2 * time.Minute
)

var sessionAccountNameSanitizer = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// StoreFactory describes a session-store backend constructor.
type StoreFactory func(configPath string) Store

var sessionStoreFactories = map[string]StoreFactory{
	"file": func(configPath string) Store {
		return NewFileStore(configPath)
	},
}

// Session describes reusable runtime auth state for one account.
type Session struct {
	Version            int               `json:"version"`
	AccountName        string            `json:"account_name"`
	Platform           string            `json:"platform"`
	Subject            string            `json:"subject"`
	GrantType          string            `json:"grant_type"`
	AccessToken        string            `json:"access_token,omitempty"`
	RefreshToken       string            `json:"refresh_token,omitempty"`
	TokenType          string            `json:"token_type,omitempty"`
	ProfileFingerprint string            `json:"profile_fingerprint,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	ExpiresAt          *time.Time        `json:"expires_at,omitempty"`
	CreatedAt          *time.Time        `json:"created_at,omitempty"`
	UpdatedAt          *time.Time        `json:"updated_at,omitempty"`
}

// HasAccessToken reports whether the session already carries an access token.
func (s Session) HasAccessToken() bool {
	return strings.TrimSpace(s.AccessToken) != ""
}

// CanRefresh reports whether the session can be refreshed.
func (s Session) CanRefresh() bool {
	return strings.TrimSpace(s.RefreshToken) != ""
}

// NeedsRefreshAt reports whether the token should be refreshed at the given time.
func (s Session) NeedsRefreshAt(now time.Time, skew time.Duration) bool {
	if !s.HasAccessToken() {
		return true
	}
	if s.ExpiresAt == nil {
		return false
	}
	if skew < 0 {
		skew = 0
	}
	return !s.ExpiresAt.After(now.Add(skew))
}

// UsableAt reports whether the session can still be reused at the given time.
func (s Session) UsableAt(now time.Time, skew time.Duration) bool {
	return s.HasAccessToken() && !s.NeedsRefreshAt(now, skew)
}

// Store defines the minimal session/token cache persistence interface.
type Store interface {
	Load(accountName string) (*Session, error)
	Save(session Session) error
	Delete(accountName string) error
	Path(accountName string) string
}

// FileStore implements a lightweight file-based session cache.
type FileStore struct {
	rootDir string
	now     func() time.Time
}

// RegisterStoreBackend registers one session-store backend.
func RegisterStoreBackend(name string, factory StoreFactory) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" || factory == nil {
		return
	}
	sessionStoreFactories[name] = factory
}

// OpenStore creates a session store from the selected backend name.
func OpenStore(configPath string, backend string) (Store, error) {
	backend = strings.TrimSpace(strings.ToLower(backend))
	if backend == "" || backend == "auto" {
		backend = "file"
	}
	factory, ok := sessionStoreFactories[backend]
	if !ok {
		return nil, fmt.Errorf("unsupported session store backend: %s", backend)
	}
	return factory(configPath), nil
}

// NewFileStore derives the session cache directory from the main config path.
func NewFileStore(configPath string) *FileStore {
	return &FileStore{
		rootDir: ResolveSessionDir(configPath),
		now:     time.Now,
	}
}

// ResolveSessionDir returns the default session cache directory.
func ResolveSessionDir(configPath string) string {
	stateDir, err := paths.ResolveStateDir(configPath)
	if err != nil {
		return filepath.Join(filepath.Dir(configPath), "state", "auth", "sessions")
	}
	return filepath.Join(stateDir, "auth", "sessions")
}

// Path returns the session file path for one account.
func (s *FileStore) Path(accountName string) string {
	return filepath.Join(s.rootDir, sanitizeAccountName(accountName)+".json")
}

// Load reads the session for one account.
func (s *FileStore) Load(accountName string) (*Session, error) {
	data, err := os.ReadFile(s.Path(accountName))
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to decode session file: %w", err)
	}
	return &session, nil
}

// Save writes one session atomically to avoid partial refresh writes.
func (s *FileStore) Save(session Session) error {
	accountName := strings.TrimSpace(session.AccountName)
	if accountName == "" {
		return fmt.Errorf("session account_name is required")
	}

	now := s.now().UTC()
	if session.Version == 0 {
		session.Version = SessionVersion
	}
	if session.CreatedAt == nil {
		session.CreatedAt = &now
	}
	session.UpdatedAt = &now

	if err := os.MkdirAll(s.rootDir, 0o700); err != nil {
		return fmt.Errorf("failed to create session cache directory: %w", err)
	}

	encoded, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode session file: %w", err)
	}

	targetPath := s.Path(accountName)
	tempPath := targetPath + ".tmp"
	if err := os.WriteFile(tempPath, encoded, 0o600); err != nil {
		return fmt.Errorf("failed to write session temp file: %w", err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to move session file into place: %w", err)
	}
	return nil
}

// Delete removes the session cache for one account.
func (s *FileStore) Delete(accountName string) error {
	err := os.Remove(s.Path(accountName))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete session file: %w", err)
	}
	return nil
}

func sanitizeAccountName(accountName string) string {
	accountName = strings.TrimSpace(accountName)
	accountName = sessionAccountNameSanitizer.ReplaceAllString(accountName, "_")
	accountName = strings.Trim(accountName, "_")
	if accountName == "" {
		return "default"
	}
	return accountName
}
