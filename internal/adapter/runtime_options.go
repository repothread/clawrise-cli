package adapter

import (
	"context"
	"sync"
)

type runtimeOptionsContextKey string

const runtimeOptionsKey runtimeOptionsContextKey = "runtime_options"

type providerDebugContextKey string

const providerDebugKey providerDebugContextKey = "provider_debug_capture"

// RuntimeOptions 描述 runtime 透传给 adapter 的执行期增强选项。
type RuntimeOptions struct {
	DebugProviderPayload bool
	VerifyAfterWrite     bool
}

// ProviderDebugCapture 保存一次 operation 内部记录的 provider 请求轨迹。
type ProviderDebugCapture struct {
	mu      sync.Mutex
	entries []map[string]any
}

// WithRuntimeOptions 将 runtime 选项附着到当前执行上下文。
func WithRuntimeOptions(ctx context.Context, options RuntimeOptions) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, runtimeOptionsKey, options)
}

// RuntimeOptionsFromContext 从上下文中读取 runtime 选项。
func RuntimeOptionsFromContext(ctx context.Context) RuntimeOptions {
	if ctx == nil {
		return RuntimeOptions{}
	}
	options, _ := ctx.Value(runtimeOptionsKey).(RuntimeOptions)
	return options
}

// WithProviderDebugCapture 在上下文中创建或复用 provider 调试采集器。
func WithProviderDebugCapture(ctx context.Context) (context.Context, *ProviderDebugCapture) {
	if ctx == nil {
		ctx = context.Background()
	}
	if capture, ok := ctx.Value(providerDebugKey).(*ProviderDebugCapture); ok && capture != nil {
		return ctx, capture
	}
	capture := &ProviderDebugCapture{}
	return context.WithValue(ctx, providerDebugKey, capture), capture
}

// AddProviderDebugEvent 向当前上下文里的调试采集器追加一条 provider 请求记录。
func AddProviderDebugEvent(ctx context.Context, entry map[string]any) {
	if ctx == nil {
		return
	}
	capture, _ := ctx.Value(providerDebugKey).(*ProviderDebugCapture)
	if capture == nil || len(entry) == 0 {
		return
	}

	cloned := make(map[string]any, len(entry))
	for key, value := range entry {
		cloned[key] = value
	}

	capture.mu.Lock()
	defer capture.mu.Unlock()
	capture.entries = append(capture.entries, cloned)
}

// ProviderDebugFromContext 导出当前上下文内采集到的 provider 请求轨迹。
func ProviderDebugFromContext(ctx context.Context) map[string]any {
	if ctx == nil {
		return nil
	}
	capture, _ := ctx.Value(providerDebugKey).(*ProviderDebugCapture)
	if capture == nil {
		return nil
	}

	capture.mu.Lock()
	defer capture.mu.Unlock()
	if len(capture.entries) == 0 {
		return nil
	}

	items := make([]map[string]any, 0, len(capture.entries))
	for _, entry := range capture.entries {
		cloned := make(map[string]any, len(entry))
		for key, value := range entry {
			cloned[key] = value
		}
		items = append(items, cloned)
	}
	return map[string]any{
		"provider_requests": items,
	}
}
