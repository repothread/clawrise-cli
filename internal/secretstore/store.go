package secretstore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/paths"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

// ErrSecretNotFound 表示目标密钥不存在。
var ErrSecretNotFound = errors.New("secret not found")

// Status 描述当前 secret store 的可用状态。
type Status struct {
	Backend   string `json:"backend"`
	Supported bool   `json:"supported"`
	Readable  bool   `json:"readable"`
	Writable  bool   `json:"writable"`
	Secure    bool   `json:"secure"`
	Detail    string `json:"detail,omitempty"`
}

// Store 定义 secret store 的最小读写接口。
type Store interface {
	Backend() string
	Status() Status
	Get(connectionName string, field string) (string, error)
	Set(connectionName string, field string, value string) error
	Delete(connectionName string, field string) error
}

// StoreFactory 描述一个可注册的 secret store backend 构造器。
type StoreFactory func(configPath string, stateDir string) (Store, error)

// Options 描述创建 store 时的基础参数。
type Options struct {
	ConfigPath      string
	Backend         string
	FallbackBackend string
	Plugin          string
}

var storeFactories = map[string]StoreFactory{
	"keychain": func(configPath string, stateDir string) (Store, error) {
		return newMacOSKeychainStore(), nil
	},
	"secret_service": func(configPath string, stateDir string) (Store, error) {
		return newLinuxSecretServiceStore(), nil
	},
	"encrypted_file": func(configPath string, stateDir string) (Store, error) {
		return newEncryptedFileStore(stateDir, "encrypted_file"), nil
	},
	"windows_dpapi_file": func(configPath string, stateDir string) (Store, error) {
		return newEncryptedFileStore(stateDir, "windows_dpapi_file"), nil
	},
}

// RegisterStoreBackend 注册一个可扩展的 secret store backend。
func RegisterStoreBackend(name string, factory StoreFactory) {
	name = normalizeBackendName(name)
	if name == "" || name == "auto" || factory == nil {
		return
	}
	storeFactories[name] = factory
}

// Open 根据配置和当前操作系统创建 secret store。
func Open(options Options) (Store, error) {
	configPath := strings.TrimSpace(options.ConfigPath)
	if configPath == "" {
		var err error
		configPath, err = paths.ResolveConfigPath()
		if err != nil {
			return nil, err
		}
	}

	stateDir, err := paths.ResolveStateDir(configPath)
	if err != nil {
		return nil, err
	}

	backend := normalizeBackendName(options.Backend)
	if backend == "" || backend == "auto" {
		return openAutoStore(configPath, stateDir, options.FallbackBackend, options.Plugin)
	}
	return openExplicitStore(configPath, stateDir, backend, options.Plugin)
}

func openAutoStore(configPath string, stateDir string, fallback string, plugin string) (Store, error) {
	switch runtime.GOOS {
	case "darwin":
		store, err := openNamedStore(configPath, stateDir, "keychain", plugin)
		if err == nil {
			status := store.Status()
			if status.Supported && status.Readable && status.Writable {
				return store, nil
			}
		}
	case "linux":
		store, err := openNamedStore(configPath, stateDir, "secret_service", plugin)
		if err == nil {
			status := store.Status()
			if status.Supported && status.Readable && status.Writable {
				return store, nil
			}
		}
	case "windows":
		return openNamedStore(configPath, stateDir, "windows_dpapi_file", plugin)
	}

	backend := normalizeBackendName(fallback)
	if backend == "" || backend == "auto" {
		backend = "encrypted_file"
	}
	return openExplicitStore(configPath, stateDir, backend, plugin)
}

func openExplicitStore(configPath string, stateDir string, backend string, plugin string) (Store, error) {
	store, err := openNamedStore(configPath, stateDir, backend, plugin)
	if err != nil {
		return nil, err
	}
	if status := store.Status(); !status.Supported {
		if strings.TrimSpace(status.Detail) != "" {
			return nil, fmt.Errorf("secret store backend %s is not supported: %s", backend, status.Detail)
		}
		return nil, fmt.Errorf("secret store backend %s is not supported", backend)
	}
	return store, nil
}

func openNamedStore(configPath string, stateDir string, backend string, plugin string) (Store, error) {
	plugin = strings.TrimSpace(plugin)
	if plugin == "" || plugin == "builtin" {
		factory, ok := storeFactories[backend]
		if ok {
			return factory(configPath, stateDir)
		}
	} else {
		manifest, found, err := discoverSecretStorePlugin(backend, plugin)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("unsupported secret store backend: %s", backend)
		}
		return newPluginSecretStore(manifest), nil
	}

	manifest, found, err := discoverSecretStorePlugin(backend, plugin)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("unsupported secret store backend: %s", backend)
	}
	return newPluginSecretStore(manifest), nil
}

func normalizeBackendName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "auto":
		return value
	case "macos_keychain":
		return "keychain"
	case "linux_secret_service":
		return "secret_service"
	default:
		return value
	}
}

func discoverSecretStorePlugin(backend string, pluginName string) (pluginruntime.Manifest, bool, error) {
	return pluginruntime.FindStorageBackendManifest(pluginruntime.StorageBackendLookup{
		Target:  "secret_store",
		Backend: backend,
		Plugin:  pluginName,
	})
}

type pluginSecretStore struct {
	client *pluginruntime.ProcessSecretStore
}

func newPluginSecretStore(manifest pluginruntime.Manifest) Store {
	return &pluginSecretStore{
		client: pluginruntime.NewProcessSecretStore(manifest),
	}
}

func (s *pluginSecretStore) Backend() string {
	if s == nil || s.client == nil {
		return ""
	}
	if backend := strings.TrimSpace(s.client.Backend()); backend != "" {
		return backend
	}
	descriptor, err := s.client.DescribeStorageBackend(context.Background())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(descriptor.Backend)
}

func (s *pluginSecretStore) Status() Status {
	if s == nil || s.client == nil {
		return Status{
			Supported: false,
			Detail:    "storage backend plugin client is not initialized",
		}
	}
	status, err := s.client.Status(context.Background())
	if err != nil {
		return Status{
			Backend:   s.Backend(),
			Supported: false,
			Readable:  false,
			Writable:  false,
			Secure:    true,
			Detail:    err.Error(),
		}
	}
	return Status{
		Backend:   status.Backend,
		Supported: status.Supported,
		Readable:  status.Readable,
		Writable:  status.Writable,
		Secure:    status.Secure,
		Detail:    status.Detail,
	}
}

func (s *pluginSecretStore) Get(connectionName string, field string) (string, error) {
	result, err := s.client.Get(context.Background(), pluginruntime.SecretStoreGetParams{
		AccountName: connectionName,
		Field:       field,
	})
	if err != nil {
		return "", err
	}
	if !result.Found {
		return "", ErrSecretNotFound
	}
	return result.Value, nil
}

func (s *pluginSecretStore) Set(connectionName string, field string, value string) error {
	return s.client.Set(context.Background(), pluginruntime.SecretStoreSetParams{
		AccountName: connectionName,
		Field:       field,
		Value:       value,
	})
}

func (s *pluginSecretStore) Delete(connectionName string, field string) error {
	return s.client.Delete(context.Background(), pluginruntime.SecretStoreDeleteParams{
		AccountName: connectionName,
		Field:       field,
	})
}

type encryptedFileStore struct {
	rootDir string
	backend string
}

func newEncryptedFileStore(stateDir string, backend string) *encryptedFileStore {
	return &encryptedFileStore{
		rootDir: filepath.Join(stateDir, "auth"),
		backend: backend,
	}
}

func (s *encryptedFileStore) Backend() string {
	return s.backend
}

func (s *encryptedFileStore) Status() Status {
	if s.backend == "windows_dpapi_file" {
		return Status{
			Backend:   s.backend,
			Supported: true,
			Readable:  true,
			Writable:  true,
			Secure:    true,
		}
	}

	_, err := s.resolveEncryptionKey()
	switch {
	case err == nil:
		return Status{
			Backend:   s.backend,
			Supported: true,
			Readable:  true,
			Writable:  true,
			Secure:    true,
		}
	default:
		return Status{
			Backend:   s.backend,
			Supported: true,
			Readable:  false,
			Writable:  false,
			Secure:    true,
			Detail:    err.Error(),
		}
	}
}

func (s *encryptedFileStore) Get(connectionName string, field string) (string, error) {
	vault, err := s.loadVault()
	if err != nil {
		return "", err
	}

	value, ok := vault[secretEntryKey(connectionName, field)]
	if !ok || strings.TrimSpace(value) == "" {
		return "", ErrSecretNotFound
	}
	return value, nil
}

func (s *encryptedFileStore) Set(connectionName string, field string, value string) error {
	vault, err := s.loadVault()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if vault == nil {
		vault = map[string]string{}
	}
	vault[secretEntryKey(connectionName, field)] = value
	return s.saveVault(vault)
}

func (s *encryptedFileStore) Delete(connectionName string, field string) error {
	vault, err := s.loadVault()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	delete(vault, secretEntryKey(connectionName, field))
	return s.saveVault(vault)
}

func (s *encryptedFileStore) vaultPath() string {
	return filepath.Join(s.rootDir, "secrets.v1.enc")
}

func (s *encryptedFileStore) keyPath() string {
	return filepath.Join(s.rootDir, "secrets.v1.key")
}

func (s *encryptedFileStore) resolveEncryptionKey() ([]byte, error) {
	if s.backend == "windows_dpapi_file" {
		return nil, nil
	}

	if masterKey := strings.TrimSpace(os.Getenv("CLAWRISE_MASTER_KEY")); masterKey != "" {
		hash := sha256.Sum256([]byte(masterKey))
		key := hash[:]
		if _, err := os.Stat(s.keyPath()); errors.Is(err, os.ErrNotExist) {
			if err := s.writeKeyFile(key); err != nil {
				return nil, err
			}
		}
		return key, nil
	}

	return s.loadOrCreateLocalKey()
}

func (s *encryptedFileStore) loadOrCreateLocalKey() ([]byte, error) {
	data, err := os.ReadFile(s.keyPath())
	if err == nil {
		if len(data) != 32 {
			return nil, fmt.Errorf("invalid secret store key file: expected 32 bytes, got %d", len(data))
		}
		return data, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to read secret store key file: %w", err)
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate secret store key: %w", err)
	}
	if err := s.writeKeyFile(key); err != nil {
		return nil, err
	}
	return key, nil
}

func (s *encryptedFileStore) writeKeyFile(key []byte) error {
	if len(key) != 32 {
		return fmt.Errorf("invalid secret store key length: %d", len(key))
	}
	if err := os.MkdirAll(s.rootDir, 0o700); err != nil {
		return fmt.Errorf("failed to create secret store directory: %w", err)
	}

	targetPath := s.keyPath()
	tempPath := targetPath + ".tmp"
	if err := os.WriteFile(tempPath, key, 0o600); err != nil {
		return fmt.Errorf("failed to write secret store key temp file: %w", err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to move secret store key file into place: %w", err)
	}
	return nil
}

func (s *encryptedFileStore) loadVault() (map[string]string, error) {
	data, err := os.ReadFile(s.vaultPath())
	if err != nil {
		return nil, err
	}

	key, err := s.resolveEncryptionKey()
	if err != nil {
		return nil, err
	}

	plainData, err := decryptSecretPayload(data, key)
	if err != nil {
		return nil, err
	}

	vault := map[string]string{}
	if len(plainData) == 0 {
		return vault, nil
	}
	if err := json.Unmarshal(plainData, &vault); err != nil {
		return nil, fmt.Errorf("failed to decode encrypted secret file: %w", err)
	}
	return vault, nil
}

func (s *encryptedFileStore) saveVault(vault map[string]string) error {
	encoded, err := json.MarshalIndent(vault, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode encrypted secret file: %w", err)
	}

	key, err := s.resolveEncryptionKey()
	if err != nil {
		return err
	}

	encrypted, err := encryptSecretPayload(encoded, key)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.rootDir, 0o700); err != nil {
		return fmt.Errorf("failed to create secret store directory: %w", err)
	}

	targetPath := s.vaultPath()
	tempPath := targetPath + ".tmp"
	if err := os.WriteFile(tempPath, encrypted, 0o600); err != nil {
		return fmt.Errorf("failed to write secret temp file: %w", err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to move secret file into place: %w", err)
	}
	return nil
}

type commandSecretStore struct {
	backend string
	program string
	support func() Status
	get     func(connectionName string, field string) (string, error)
	set     func(connectionName string, field string, value string) error
	delete  func(connectionName string, field string) error
}

func (s *commandSecretStore) Backend() string {
	return s.backend
}

func (s *commandSecretStore) Status() Status {
	return s.support()
}

func (s *commandSecretStore) Get(connectionName string, field string) (string, error) {
	return s.get(connectionName, field)
}

func (s *commandSecretStore) Set(connectionName string, field string, value string) error {
	return s.set(connectionName, field, value)
}

func (s *commandSecretStore) Delete(connectionName string, field string) error {
	return s.delete(connectionName, field)
}

func newMacOSKeychainStore() Store {
	serviceName := "com.clawrise.cli"
	accountName := func(connectionName string, field string) string {
		return "connection/" + connectionName + "/field/" + field
	}

	return &commandSecretStore{
		backend: "keychain",
		program: "security",
		support: func() Status {
			_, err := exec.LookPath("security")
			if err != nil {
				return Status{
					Backend:   "keychain",
					Supported: false,
					Readable:  false,
					Writable:  false,
					Secure:    true,
					Detail:    "security command is not available",
				}
			}
			return Status{
				Backend:   "keychain",
				Supported: true,
				Readable:  true,
				Writable:  true,
				Secure:    true,
			}
		},
		get: func(connectionName string, field string) (string, error) {
			command := exec.Command("security", "find-generic-password", "-a", accountName(connectionName, field), "-s", serviceName, "-w")
			output, err := command.Output()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					stderr := strings.TrimSpace(string(exitErr.Stderr))
					if strings.Contains(stderr, "could not be found") {
						return "", ErrSecretNotFound
					}
				}
				return "", fmt.Errorf("failed to read macOS keychain secret: %w", err)
			}
			return strings.TrimSpace(string(output)), nil
		},
		set: func(connectionName string, field string, value string) error {
			command := exec.Command("security", "add-generic-password", "-a", accountName(connectionName, field), "-s", serviceName, "-U", "-w", value)
			if output, err := command.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to write macOS keychain secret: %s", strings.TrimSpace(string(output)))
			}
			return nil
		},
		delete: func(connectionName string, field string) error {
			command := exec.Command("security", "delete-generic-password", "-a", accountName(connectionName, field), "-s", serviceName)
			if output, err := command.CombinedOutput(); err != nil {
				text := strings.TrimSpace(string(output))
				if strings.Contains(text, "could not be found") {
					return nil
				}
				return fmt.Errorf("failed to delete macOS keychain secret: %s", text)
			}
			return nil
		},
	}
}

func newLinuxSecretServiceStore() Store {
	commonArgs := func(connectionName string, field string) []string {
		return []string{"application", "clawrise", "connection", connectionName, "field", field}
	}

	return &commandSecretStore{
		backend: "secret_service",
		program: "secret-tool",
		support: func() Status {
			_, err := exec.LookPath("secret-tool")
			if err != nil {
				return Status{
					Backend:   "secret_service",
					Supported: false,
					Readable:  false,
					Writable:  false,
					Secure:    true,
					Detail:    "secret-tool command is not available",
				}
			}
			return Status{
				Backend:   "secret_service",
				Supported: true,
				Readable:  true,
				Writable:  true,
				Secure:    true,
			}
		},
		get: func(connectionName string, field string) (string, error) {
			command := exec.Command("secret-tool", append([]string{"lookup"}, commonArgs(connectionName, field)...)...)
			output, err := command.Output()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) == 0 && len(output) == 0 {
					return "", ErrSecretNotFound
				}
				return "", fmt.Errorf("failed to read linux secret service secret: %w", err)
			}
			value := strings.TrimSpace(string(output))
			if value == "" {
				return "", ErrSecretNotFound
			}
			return value, nil
		},
		set: func(connectionName string, field string, value string) error {
			args := append([]string{"store", "--label=Clawrise " + connectionName + " " + field}, commonArgs(connectionName, field)...)
			command := exec.Command("secret-tool", args...)
			command.Stdin = strings.NewReader(value)
			if output, err := command.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to write linux secret service secret: %s", strings.TrimSpace(string(output)))
			}
			return nil
		},
		delete: func(connectionName string, field string) error {
			command := exec.Command("secret-tool", append([]string{"clear"}, commonArgs(connectionName, field)...)...)
			if output, err := command.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to delete linux secret service secret: %s", strings.TrimSpace(string(output)))
			}
			return nil
		},
	}
}

func secretEntryKey(connectionName string, field string) string {
	return strings.TrimSpace(connectionName) + "::" + strings.TrimSpace(field)
}
