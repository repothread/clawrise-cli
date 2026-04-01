package plugin

import "context"

// SessionStorePluginRuntime 描述一个外部 session store plugin 的最小协议面。
type SessionStorePluginRuntime interface {
	Name() string
	Handshake(ctx context.Context) (HandshakeResult, error)
	Status(ctx context.Context) (StorageStatus, error)
	Load(ctx context.Context, params SessionStoreLoadParams) (SessionStoreLoadResult, error)
	Save(ctx context.Context, params SessionStoreSaveParams) error
	Delete(ctx context.Context, params SessionStoreDeleteParams) error
}

// ProcessSessionStore 使用 stdio JSON-RPC 调用一个外部 session store plugin。
type ProcessSessionStore struct {
	runtime *ProcessRuntime
	backend string
}

// NewProcessSessionStore 创建一个进程化的 session store plugin 客户端。
func NewProcessSessionStore(manifest Manifest) *ProcessSessionStore {
	capability, _ := findStorageCapability(manifest, "session_store", "")
	return &ProcessSessionStore{
		runtime: NewProcessRuntime(manifest),
		backend: capability.Backend,
	}
}

func (s *ProcessSessionStore) Name() string {
	return s.runtime.Name()
}

func (s *ProcessSessionStore) Handshake(ctx context.Context) (HandshakeResult, error) {
	return s.runtime.Handshake(ctx)
}

func (s *ProcessSessionStore) Backend() string {
	return s.backend
}

func (s *ProcessSessionStore) Status(ctx context.Context) (StorageStatus, error) {
	var result SessionStoreStatusResult
	if err := s.runtime.call(ctx, "clawrise.storage.session.status", map[string]any{}, &result); err != nil {
		return StorageStatus{}, err
	}
	return result.Status, nil
}

func (s *ProcessSessionStore) Load(ctx context.Context, params SessionStoreLoadParams) (SessionStoreLoadResult, error) {
	var result SessionStoreLoadResult
	if err := s.runtime.call(ctx, "clawrise.storage.session.load", params, &result); err != nil {
		return SessionStoreLoadResult{}, err
	}
	return result, nil
}

func (s *ProcessSessionStore) Save(ctx context.Context, params SessionStoreSaveParams) error {
	return s.runtime.call(ctx, "clawrise.storage.session.save", params, nil)
}

func (s *ProcessSessionStore) Delete(ctx context.Context, params SessionStoreDeleteParams) error {
	return s.runtime.call(ctx, "clawrise.storage.session.delete", params, nil)
}

// Close 关闭外部 session store plugin 进程。
func (s *ProcessSessionStore) Close() error {
	if s == nil || s.runtime == nil {
		return nil
	}
	return s.runtime.Close()
}
