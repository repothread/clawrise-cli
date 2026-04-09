package plugin

import "context"

// GovernanceStorePluginRuntime 描述一个外部 governance store plugin 的最小协议面。
type GovernanceStorePluginRuntime interface {
	Name() string
	Handshake(ctx context.Context) (HandshakeResult, error)
	Status(ctx context.Context) (StorageStatus, error)
	LoadIdempotency(ctx context.Context, params GovernanceIdempotencyLoadParams) (GovernanceIdempotencyLoadResult, error)
	SaveIdempotency(ctx context.Context, params GovernanceIdempotencySaveParams) error
	AppendAudit(ctx context.Context, params GovernanceAuditAppendParams) error
}

// ProcessGovernanceStore 使用 stdio JSON-RPC 调用一个外部 governance store plugin。
type ProcessGovernanceStore struct {
	runtime *ProcessRuntime
	backend string
}

// NewProcessGovernanceStore 创建一个进程化的 governance store plugin 客户端。
func NewProcessGovernanceStore(manifest Manifest) *ProcessGovernanceStore {
	capability, _ := findStorageCapability(manifest, "governance", "")
	return &ProcessGovernanceStore{
		runtime: NewProcessRuntime(manifest),
		backend: capability.Backend,
	}
}

func (s *ProcessGovernanceStore) Name() string {
	return s.runtime.Name()
}

func (s *ProcessGovernanceStore) Handshake(ctx context.Context) (HandshakeResult, error) {
	return s.runtime.Handshake(ctx)
}

func (s *ProcessGovernanceStore) Backend() string {
	return s.backend
}

func (s *ProcessGovernanceStore) Status(ctx context.Context) (StorageStatus, error) {
	var result GovernanceStoreStatusResult
	if err := s.runtime.call(ctx, "clawrise.storage.governance.status", map[string]any{}, &result); err != nil {
		return StorageStatus{}, err
	}
	return result.Status, nil
}

func (s *ProcessGovernanceStore) LoadIdempotency(ctx context.Context, params GovernanceIdempotencyLoadParams) (GovernanceIdempotencyLoadResult, error) {
	var result GovernanceIdempotencyLoadResult
	if err := s.runtime.call(ctx, "clawrise.storage.governance.idempotency.load", params, &result); err != nil {
		return GovernanceIdempotencyLoadResult{}, err
	}
	return result, nil
}

func (s *ProcessGovernanceStore) SaveIdempotency(ctx context.Context, params GovernanceIdempotencySaveParams) error {
	return s.runtime.call(ctx, "clawrise.storage.governance.idempotency.save", params, nil)
}

func (s *ProcessGovernanceStore) AppendAudit(ctx context.Context, params GovernanceAuditAppendParams) error {
	return s.runtime.call(ctx, "clawrise.storage.governance.audit.append", params, nil)
}

// Close 关闭外部 governance store plugin 进程。
func (s *ProcessGovernanceStore) Close() error {
	if s == nil || s.runtime == nil {
		return nil
	}
	return s.runtime.Close()
}
