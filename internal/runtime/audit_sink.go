package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/config"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

// auditSink 描述一个审计事件扇出目标。
type auditSink interface {
	Name() string
	Emit(ctx context.Context, record auditRecord) error
}

type pluginAuditSink struct {
	runtime pluginruntime.AuditSinkRuntime
}

func openAuditSinks(cfg *config.Config) []auditSink {
	if cfg == nil {
		cfg = config.New()
	}

	runtimes, err := pluginruntime.DiscoverAuditSinkRuntimes(pluginruntime.DiscoveryOptions{
		EnabledPlugins: config.ResolveEnabledPlugins(cfg),
	})
	if err != nil {
		return nil
	}

	items := make([]auditSink, 0, len(runtimes))
	for _, runtime := range runtimes {
		items = append(items, &pluginAuditSink{runtime: runtime})
	}
	return items
}

func (g *runtimeGovernance) emitAuditSinks(record auditRecord) []string {
	if g == nil || len(g.sinks) == 0 {
		return nil
	}

	warnings := make([]string, 0)
	for _, sink := range g.sinks {
		if sink == nil {
			continue
		}
		if err := sink.Emit(context.Background(), record); err != nil {
			warnings = append(warnings, fmt.Sprintf("audit sink %s 投递失败: %s", sink.Name(), err.Error()))
		}
	}
	return warnings
}

func (s *pluginAuditSink) Name() string {
	if s == nil || s.runtime == nil {
		return ""
	}
	id := strings.TrimSpace(s.runtime.ID())
	name := strings.TrimSpace(s.runtime.Name())
	switch {
	case id != "" && name != "" && id != name:
		return name + "/" + id
	case id != "":
		return id
	default:
		return name
	}
}

func (s *pluginAuditSink) Emit(ctx context.Context, record auditRecord) error {
	if s == nil || s.runtime == nil {
		return nil
	}
	defer func() {
		// 审计 sink 当前按次使用，发完即释放进程资源，避免长期悬挂。
		_ = s.runtime.Close()
	}()

	return s.runtime.Emit(ctx, pluginruntime.AuditEmitParams{
		SinkID: s.runtime.ID(),
		Record: convertAuditRecordToPlugin(record),
	})
}
