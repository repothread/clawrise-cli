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
	// SessionVersion 用于后续平滑升级 session 文件结构。
	SessionVersion = 1

	// DefaultRefreshSkew 表示 access token 提前多久视为需要刷新。
	DefaultRefreshSkew = 2 * time.Minute
)

var sessionProfileNameSanitizer = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// Session 描述某个 profile 在运行时可直接复用的认证态。
type Session struct {
	Version            int               `json:"version"`
	ProfileName        string            `json:"profile_name"`
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

// HasAccessToken 判断当前 session 是否已经持有可用的 access token。
func (s Session) HasAccessToken() bool {
	return strings.TrimSpace(s.AccessToken) != ""
}

// CanRefresh 判断当前 session 是否具备刷新能力。
func (s Session) CanRefresh() bool {
	return strings.TrimSpace(s.RefreshToken) != ""
}

// NeedsRefreshAt 判断在给定时间点是否应该提前刷新 token。
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

// UsableAt 判断当前 session 在给定时间点是否仍可直接复用。
func (s Session) UsableAt(now time.Time, skew time.Duration) bool {
	return s.HasAccessToken() && !s.NeedsRefreshAt(now, skew)
}

// Store 定义 session/token cache 的最小存储接口。
type Store interface {
	Load(profileName string) (*Session, error)
	Save(session Session) error
	Delete(profileName string) error
	Path(profileName string) string
}

// FileStore 通过本地文件实现轻量 session cache。
type FileStore struct {
	rootDir string
	now     func() time.Time
}

// NewFileStore 基于主配置路径推导 session cache 目录。
func NewFileStore(configPath string) *FileStore {
	return &FileStore{
		rootDir: ResolveSessionDir(configPath),
		now:     time.Now,
	}
}

// ResolveSessionDir 返回 session cache 的默认目录。
func ResolveSessionDir(configPath string) string {
	stateDir, err := paths.ResolveStateDir(configPath)
	if err != nil {
		return filepath.Join(filepath.Dir(configPath), "state", "auth", "sessions")
	}
	return filepath.Join(stateDir, "auth", "sessions")
}

// Path 返回指定 profile 的 session 文件路径。
func (s *FileStore) Path(profileName string) string {
	return filepath.Join(s.rootDir, sanitizeProfileName(profileName)+".json")
}

// Load 读取指定 profile 的 session。
func (s *FileStore) Load(profileName string) (*Session, error) {
	data, err := os.ReadFile(s.Path(profileName))
	if err != nil {
		return nil, err
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to decode session file: %w", err)
	}
	return &session, nil
}

// Save 原子写入 session 文件，避免刷新流程写出半文件。
func (s *FileStore) Save(session Session) error {
	profileName := strings.TrimSpace(session.ProfileName)
	if profileName == "" {
		return fmt.Errorf("session profile_name is required")
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

	targetPath := s.Path(profileName)
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

// Delete 删除指定 profile 的 session cache。
func (s *FileStore) Delete(profileName string) error {
	err := os.Remove(s.Path(profileName))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete session file: %w", err)
	}
	return nil
}

func sanitizeProfileName(profileName string) string {
	profileName = strings.TrimSpace(profileName)
	profileName = sessionProfileNameSanitizer.ReplaceAllString(profileName, "_")
	profileName = strings.Trim(profileName, "_")
	if profileName == "" {
		return "default"
	}
	return profileName
}
