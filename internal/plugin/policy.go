package plugin

import (
	"context"
	"sort"
	"strings"
)

// PolicyRuntime 描述一个可参与执行前判断的策略运行时。
type PolicyRuntime interface {
	Name() string
	ID() string
	Priority() int
	Platforms() []string
	Handshake(ctx context.Context) (HandshakeResult, error)
	Evaluate(ctx context.Context, params PolicyEvaluateParams) (PolicyEvaluateResult, error)
	Close() error
}

// ProcessPolicy 使用 stdio JSON-RPC 调用一个外部 policy plugin。
type ProcessPolicy struct {
	runtime    *ProcessRuntime
	capability CapabilityDescriptor
}

// NewProcessPolicy 创建一个进程化的 policy plugin 客户端。
func NewProcessPolicy(manifest Manifest, capability CapabilityDescriptor) *ProcessPolicy {
	return &ProcessPolicy{
		runtime:    NewProcessRuntime(manifest),
		capability: capability,
	}
}

func (p *ProcessPolicy) Name() string {
	if p == nil || p.runtime == nil {
		return ""
	}
	return p.runtime.Name()
}

func (p *ProcessPolicy) ID() string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(p.capability.ID)
}

func (p *ProcessPolicy) Priority() int {
	if p == nil {
		return 0
	}
	return p.capability.Priority
}

func (p *ProcessPolicy) Platforms() []string {
	if p == nil {
		return nil
	}
	return append([]string(nil), p.capability.Platforms...)
}

func (p *ProcessPolicy) Handshake(ctx context.Context) (HandshakeResult, error) {
	return p.runtime.Handshake(ctx)
}

func (p *ProcessPolicy) Evaluate(ctx context.Context, params PolicyEvaluateParams) (PolicyEvaluateResult, error) {
	var result PolicyEvaluateResult
	if strings.TrimSpace(params.PolicyID) == "" {
		params.PolicyID = p.ID()
	}
	if err := p.runtime.call(ctx, "clawrise.policy.evaluate", params, &result); err != nil {
		return PolicyEvaluateResult{}, err
	}
	return result, nil
}

func (p *ProcessPolicy) Close() error {
	if p == nil || p.runtime == nil {
		return nil
	}
	return p.runtime.Close()
}

// DiscoverPolicyRuntimes 发现所有启用中的 policy capability。
func DiscoverPolicyRuntimes(options DiscoveryOptions) ([]PolicyRuntime, error) {
	roots, err := DefaultDiscoveryRoots()
	if err != nil {
		return nil, err
	}
	manifests, err := DiscoverManifests(roots)
	if err != nil {
		return nil, err
	}
	manifests = filterManifestsByEnabledRules(manifests, options.EnabledPlugins)

	items := make([]PolicyRuntime, 0)
	for _, manifest := range manifests {
		for _, capability := range manifest.CapabilitiesByType(CapabilityTypePolicy) {
			items = append(items, NewProcessPolicy(manifest, capability))
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
