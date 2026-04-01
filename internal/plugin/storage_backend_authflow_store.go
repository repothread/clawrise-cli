package plugin

import "context"

// AuthFlowStorePluginRuntime 描述一个外部 authflow store plugin 的最小协议面。
type AuthFlowStorePluginRuntime interface {
	Name() string
	Handshake(ctx context.Context) (HandshakeResult, error)
	Status(ctx context.Context) (StorageStatus, error)
	Load(ctx context.Context, params AuthFlowStoreLoadParams) (AuthFlowStoreLoadResult, error)
	Save(ctx context.Context, params AuthFlowStoreSaveParams) error
	Delete(ctx context.Context, params AuthFlowStoreDeleteParams) error
}

// ProcessAuthFlowStore 使用 stdio JSON-RPC 调用一个外部 authflow store plugin。
type ProcessAuthFlowStore struct {
	runtime *ProcessRuntime
	backend string
}

// NewProcessAuthFlowStore 创建一个进程化的 authflow store plugin 客户端。
func NewProcessAuthFlowStore(manifest Manifest) *ProcessAuthFlowStore {
	capability, _ := findStorageCapability(manifest, "authflow_store", "")
	return &ProcessAuthFlowStore{
		runtime: NewProcessRuntime(manifest),
		backend: capability.Backend,
	}
}

func (s *ProcessAuthFlowStore) Name() string {
	return s.runtime.Name()
}

func (s *ProcessAuthFlowStore) Handshake(ctx context.Context) (HandshakeResult, error) {
	return s.runtime.Handshake(ctx)
}

func (s *ProcessAuthFlowStore) Backend() string {
	return s.backend
}

func (s *ProcessAuthFlowStore) Status(ctx context.Context) (StorageStatus, error) {
	var result AuthFlowStoreStatusResult
	if err := s.runtime.call(ctx, "clawrise.storage.authflow.status", map[string]any{}, &result); err != nil {
		return StorageStatus{}, err
	}
	return result.Status, nil
}

func (s *ProcessAuthFlowStore) Load(ctx context.Context, params AuthFlowStoreLoadParams) (AuthFlowStoreLoadResult, error) {
	var result AuthFlowStoreLoadResult
	if err := s.runtime.call(ctx, "clawrise.storage.authflow.load", params, &result); err != nil {
		return AuthFlowStoreLoadResult{}, err
	}
	return result, nil
}

func (s *ProcessAuthFlowStore) Save(ctx context.Context, params AuthFlowStoreSaveParams) error {
	return s.runtime.call(ctx, "clawrise.storage.authflow.save", params, nil)
}

func (s *ProcessAuthFlowStore) Delete(ctx context.Context, params AuthFlowStoreDeleteParams) error {
	return s.runtime.call(ctx, "clawrise.storage.authflow.delete", params, nil)
}

// Close 关闭外部 authflow store plugin 进程。
func (s *ProcessAuthFlowStore) Close() error {
	if s == nil || s.runtime == nil {
		return nil
	}
	return s.runtime.Close()
}
