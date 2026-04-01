package plugin

import (
	"context"
	"sort"
	"strings"
)

// AuditSinkRuntime 描述一个可接收审计事件的运行时。
type AuditSinkRuntime interface {
	Name() string
	ID() string
	Priority() int
	Handshake(ctx context.Context) (HandshakeResult, error)
	Emit(ctx context.Context, params AuditEmitParams) error
	Close() error
}

// ProcessAuditSink 使用 stdio JSON-RPC 调用一个外部 audit sink plugin。
type ProcessAuditSink struct {
	runtime    *ProcessRuntime
	capability CapabilityDescriptor
}

// NewProcessAuditSink 创建一个进程化的 audit sink plugin 客户端。
func NewProcessAuditSink(manifest Manifest, capability CapabilityDescriptor) *ProcessAuditSink {
	return &ProcessAuditSink{
		runtime:    NewProcessRuntime(manifest),
		capability: capability,
	}
}

func (s *ProcessAuditSink) Name() string {
	if s == nil || s.runtime == nil {
		return ""
	}
	return s.runtime.Name()
}

func (s *ProcessAuditSink) ID() string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(s.capability.ID)
}

func (s *ProcessAuditSink) Priority() int {
	if s == nil {
		return 0
	}
	return s.capability.Priority
}

func (s *ProcessAuditSink) Handshake(ctx context.Context) (HandshakeResult, error) {
	return s.runtime.Handshake(ctx)
}

func (s *ProcessAuditSink) Emit(ctx context.Context, params AuditEmitParams) error {
	if strings.TrimSpace(params.SinkID) == "" {
		params.SinkID = s.ID()
	}
	return s.runtime.call(ctx, "clawrise.audit.emit", params, nil)
}

func (s *ProcessAuditSink) Close() error {
	if s == nil || s.runtime == nil {
		return nil
	}
	return s.runtime.Close()
}

// DiscoverAuditSinkRuntimes 发现所有启用中的 audit sink capability。
func DiscoverAuditSinkRuntimes(options DiscoveryOptions) ([]AuditSinkRuntime, error) {
	roots, err := DefaultDiscoveryRoots()
	if err != nil {
		return nil, err
	}
	manifests, err := DiscoverManifests(roots)
	if err != nil {
		return nil, err
	}
	manifests = filterManifestsByEnabledRules(manifests, options.EnabledPlugins)

	items := make([]AuditSinkRuntime, 0)
	for _, manifest := range manifests {
		for _, capability := range manifest.CapabilitiesByType(CapabilityTypeAuditSink) {
			items = append(items, NewProcessAuditSink(manifest, capability))
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Priority() == items[j].Priority() {
			if items[i].Name() == items[j].Name() {
				return items[i].ID() < items[j].ID()
			}
			return items[i].Name() < items[j].Name()
		}
		return items[i].Priority() > items[j].Priority()
	})
	return items, nil
}
