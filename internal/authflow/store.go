package authflow

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

var flowIDSanitizer = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// Store 定义授权 flow 的最小持久化接口。
type Store interface {
	Load(flowID string) (*Flow, error)
	Save(flow Flow) error
	Delete(flowID string) error
	Path(flowID string) string
}

// FileStore 使用本地文件保存授权 flow。
type FileStore struct {
	rootDir string
	now     func() time.Time
}

// NewFileStore 基于配置路径推导 flow 存储目录。
func NewFileStore(configPath string) *FileStore {
	stateDir, err := paths.ResolveStateDir(configPath)
	if err != nil {
		stateDir = filepath.Join(filepath.Dir(configPath), "state")
	}
	return &FileStore{
		rootDir: filepath.Join(stateDir, "auth", "flows"),
		now:     time.Now,
	}
}

// Path 返回指定 flow 的状态文件路径。
func (s *FileStore) Path(flowID string) string {
	return filepath.Join(s.rootDir, sanitizeFlowID(flowID)+".json")
}

// Load 读取指定 flow。
func (s *FileStore) Load(flowID string) (*Flow, error) {
	data, err := os.ReadFile(s.Path(flowID))
	if err != nil {
		return nil, err
	}

	var flow Flow
	if err := json.Unmarshal(data, &flow); err != nil {
		return nil, fmt.Errorf("failed to decode auth flow file: %w", err)
	}
	return &flow, nil
}

// Save 原子写入 flow 文件。
func (s *FileStore) Save(flow Flow) error {
	flow.ID = strings.TrimSpace(flow.ID)
	if flow.ID == "" {
		return fmt.Errorf("auth flow id is required")
	}

	now := s.now().UTC()
	if flow.CreatedAt.IsZero() {
		flow.CreatedAt = now
	}
	flow.UpdatedAt = now
	if flow.ExpiresAt.IsZero() {
		flow.ExpiresAt = now.Add(DefaultFlowTTL)
	}

	if err := os.MkdirAll(s.rootDir, 0o700); err != nil {
		return fmt.Errorf("failed to create auth flow directory: %w", err)
	}

	encoded, err := json.MarshalIndent(flow, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode auth flow file: %w", err)
	}

	targetPath := s.Path(flow.ID)
	tempPath := targetPath + ".tmp"
	if err := os.WriteFile(tempPath, encoded, 0o600); err != nil {
		return fmt.Errorf("failed to write auth flow temp file: %w", err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to move auth flow file into place: %w", err)
	}
	return nil
}

// Delete 删除 flow 文件。
func (s *FileStore) Delete(flowID string) error {
	err := os.Remove(s.Path(flowID))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete auth flow file: %w", err)
	}
	return nil
}

func sanitizeFlowID(flowID string) string {
	flowID = strings.TrimSpace(flowID)
	flowID = flowIDSanitizer.ReplaceAllString(flowID, "_")
	flowID = strings.Trim(flowID, "_")
	if flowID == "" {
		return "default"
	}
	return flowID
}
