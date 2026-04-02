package plugin

import "context"

// SecretStorePluginRuntime 描述一个外部 secret store plugin 的最小协议面。
type SecretStorePluginRuntime interface {
	Name() string
	Handshake(ctx context.Context) (HandshakeResult, error)
	DescribeStorageBackend(ctx context.Context) (StorageBackendDescriptor, error)
	Status(ctx context.Context) (StorageStatus, error)
	Get(ctx context.Context, params SecretStoreGetParams) (SecretStoreGetResult, error)
	Set(ctx context.Context, params SecretStoreSetParams) error
	Delete(ctx context.Context, params SecretStoreDeleteParams) error
}

// ProcessSecretStore 使用 stdio JSON-RPC 调用一个外部 secret store plugin。
type ProcessSecretStore struct {
	runtime *ProcessRuntime
	backend string
}

// NewProcessSecretStore 创建一个进程化的外部 secret store plugin 客户端。
func NewProcessSecretStore(manifest Manifest) *ProcessSecretStore {
	capability, _ := findStorageCapability(manifest, "secret_store", "")
	return &ProcessSecretStore{
		runtime: NewProcessRuntime(manifest),
		backend: capability.Backend,
	}
}

func (s *ProcessSecretStore) Name() string {
	return s.runtime.Name()
}

func (s *ProcessSecretStore) Handshake(ctx context.Context) (HandshakeResult, error) {
	return s.runtime.Handshake(ctx)
}

func (s *ProcessSecretStore) Backend() string {
	return s.backend
}

func (s *ProcessSecretStore) DescribeStorageBackend(ctx context.Context) (StorageBackendDescriptor, error) {
	var result StorageBackendDescribeResult
	if err := s.runtime.call(ctx, "clawrise.storage.backend.describe", map[string]any{}, &result); err != nil {
		return StorageBackendDescriptor{}, err
	}
	return result.Backend, nil
}

func (s *ProcessSecretStore) Status(ctx context.Context) (StorageStatus, error) {
	var result SecretStoreStatusResult
	if err := s.runtime.call(ctx, "clawrise.storage.secret.status", map[string]any{}, &result); err != nil {
		return StorageStatus{}, err
	}
	return result.Status, nil
}

func (s *ProcessSecretStore) Get(ctx context.Context, params SecretStoreGetParams) (SecretStoreGetResult, error) {
	var result SecretStoreGetResult
	if err := s.runtime.call(ctx, "clawrise.storage.secret.get", params, &result); err != nil {
		return SecretStoreGetResult{}, err
	}
	return result, nil
}

func (s *ProcessSecretStore) Set(ctx context.Context, params SecretStoreSetParams) error {
	return s.runtime.call(ctx, "clawrise.storage.secret.set", params, nil)
}

func (s *ProcessSecretStore) Delete(ctx context.Context, params SecretStoreDeleteParams) error {
	return s.runtime.call(ctx, "clawrise.storage.secret.delete", params, nil)
}

// Close 关闭外部 secret store plugin 进程。
func (s *ProcessSecretStore) Close() error {
	if s == nil || s.runtime == nil {
		return nil
	}
	return s.runtime.Close()
}
