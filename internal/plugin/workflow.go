package plugin

import (
	"context"
	"sort"
	"strings"
)

// WorkflowRuntime describes one workflow planning runtime exposed by a plugin.
type WorkflowRuntime interface {
	Name() string
	ID() string
	Priority() int
	Handshake(ctx context.Context) (HandshakeResult, error)
	Plan(ctx context.Context, params WorkflowPlanParams) (WorkflowPlanResult, error)
	Close() error
}

// ProcessWorkflow executes JSON-RPC calls against one external workflow plugin.
type ProcessWorkflow struct {
	runtime    *ProcessRuntime
	capability CapabilityDescriptor
}

// NewProcessWorkflow creates one process-backed workflow plugin client.
func NewProcessWorkflow(manifest Manifest, capability CapabilityDescriptor) *ProcessWorkflow {
	return &ProcessWorkflow{
		runtime:    NewProcessRuntime(manifest),
		capability: capability,
	}
}

func (w *ProcessWorkflow) Name() string {
	if w == nil || w.runtime == nil {
		return ""
	}
	return w.runtime.Name()
}

func (w *ProcessWorkflow) ID() string {
	if w == nil {
		return ""
	}
	return strings.TrimSpace(w.capability.ID)
}

func (w *ProcessWorkflow) Priority() int {
	if w == nil {
		return 0
	}
	return w.capability.Priority
}

func (w *ProcessWorkflow) Handshake(ctx context.Context) (HandshakeResult, error) {
	return w.runtime.Handshake(ctx)
}

func (w *ProcessWorkflow) Plan(ctx context.Context, params WorkflowPlanParams) (WorkflowPlanResult, error) {
	var result WorkflowPlanResult
	if strings.TrimSpace(params.WorkflowID) == "" {
		params.WorkflowID = w.ID()
	}
	if err := w.runtime.call(ctx, "clawrise.workflow.plan", params, &result); err != nil {
		return WorkflowPlanResult{}, err
	}
	return result, nil
}

func (w *ProcessWorkflow) Close() error {
	if w == nil || w.runtime == nil {
		return nil
	}
	return w.runtime.Close()
}

// DiscoverWorkflowRuntimes discovers all enabled workflow capabilities.
func DiscoverWorkflowRuntimes(options DiscoveryOptions) ([]WorkflowRuntime, error) {
	roots, err := DefaultDiscoveryRoots()
	if err != nil {
		return nil, err
	}
	manifests, err := DiscoverManifests(roots)
	if err != nil {
		return nil, err
	}
	manifests = filterManifestsByEnabledRules(manifests, options.EnabledPlugins)

	items := make([]WorkflowRuntime, 0)
	for _, manifest := range manifests {
		for _, capability := range manifest.CapabilitiesByType(CapabilityTypeWorkflow) {
			items = append(items, NewProcessWorkflow(manifest, capability))
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
