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

// StoreFactory describes an auth-flow store backend constructor.
type StoreFactory func(configPath string) Store

// StoreOptions 描述打开 authflow store 时的可选参数。
type StoreOptions struct {
	ConfigPath     string
	Backend        string
	Plugin         string
	EnabledPlugins map[string]string
}

// ExternalStoreResolver 描述一个外部 authflow store 解析器。
type ExternalStoreResolver func(options StoreOptions) (Store, bool, error)

var storeFactories = map[string]StoreFactory{
	"file": func(configPath string) Store {
		return NewFileStore(configPath)
	},
}

var externalStoreResolvers []ExternalStoreResolver

// Store defines the minimal persistence contract for auth flows.
type Store interface {
	Load(flowID string) (*Flow, error)
	Save(flow Flow) error
	Delete(flowID string) error
	Path(flowID string) string
}

// FileStore persists auth flows as local files.
type FileStore struct {
	rootDir string
	now     func() time.Time
}

// RegisterStoreBackend registers one auth-flow store backend.
func RegisterStoreBackend(name string, factory StoreFactory) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" || factory == nil {
		return
	}
	storeFactories[name] = factory
}

// RegisterExternalStoreResolver 注册一个外部 authflow store 解析器。
func RegisterExternalStoreResolver(resolver ExternalStoreResolver) {
	if resolver == nil {
		return
	}
	externalStoreResolvers = append(externalStoreResolvers, resolver)
}

// OpenStore creates an auth-flow store from the selected backend name.
func OpenStore(configPath string, backend string) (Store, error) {
	return OpenStoreWithOptions(StoreOptions{
		ConfigPath: configPath,
		Backend:    backend,
	})
}

// OpenStoreWithOptions 根据配置与 plugin 绑定打开 authflow store。
func OpenStoreWithOptions(options StoreOptions) (Store, error) {
	backend := strings.TrimSpace(strings.ToLower(options.Backend))
	if backend == "" || backend == "auto" {
		backend = "file"
	}

	pluginName := strings.TrimSpace(options.Plugin)
	if pluginName != "" && pluginName != "builtin" {
		store, handled, err := openExternalStore(StoreOptions{
			ConfigPath:     options.ConfigPath,
			Backend:        backend,
			Plugin:         pluginName,
			EnabledPlugins: options.EnabledPlugins,
		})
		if err != nil {
			return nil, err
		}
		if handled {
			return store, nil
		}
		return nil, fmt.Errorf("unsupported auth flow store backend: %s", backend)
	}

	if factory, ok := storeFactories[backend]; ok {
		return factory(options.ConfigPath), nil
	}

	store, handled, err := openExternalStore(StoreOptions{
		ConfigPath:     options.ConfigPath,
		Backend:        backend,
		Plugin:         pluginName,
		EnabledPlugins: options.EnabledPlugins,
	})
	if err != nil {
		return nil, err
	}
	if handled {
		return store, nil
	}
	return nil, fmt.Errorf("unsupported auth flow store backend: %s", backend)
}

func openExternalStore(options StoreOptions) (Store, bool, error) {
	for _, resolver := range externalStoreResolvers {
		store, handled, err := resolver(options)
		if err != nil {
			return nil, false, err
		}
		if handled {
			return store, true, nil
		}
	}
	return nil, false, nil
}

// NewFileStore derives the auth-flow storage directory from the config path.
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

// Path returns the state-file path for one auth flow.
func (s *FileStore) Path(flowID string) string {
	return filepath.Join(s.rootDir, sanitizeFlowID(flowID)+".json")
}

// Load reads one auth flow.
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

// Save writes one auth flow atomically.
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

// Delete removes one auth-flow file.
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
