package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/clawrise/clawrise-cli/internal/config"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

// auditSink describes one audit event fan-out target.
type auditSink interface {
	Name() string
	Emit(ctx context.Context, record auditRecord) error
}

// AuditSinkSelectorView 描述一个来自配置文件的 audit sink 项。
type AuditSinkSelectorView struct {
	Type        string   `json:"type,omitempty"`
	Plugin      string   `json:"plugin,omitempty"`
	SinkID      string   `json:"sink_id,omitempty"`
	URL         string   `json:"url,omitempty"`
	HeaderNames []string `json:"header_names,omitempty"`
	TimeoutMS   int      `json:"timeout_ms,omitempty"`
}

// AuditSinkSummary 描述一个实际生效的 audit sink。
type AuditSinkSummary struct {
	Type        string   `json:"type,omitempty"`
	Plugin      string   `json:"plugin,omitempty"`
	SinkID      string   `json:"sink_id,omitempty"`
	Label       string   `json:"label,omitempty"`
	URL         string   `json:"url,omitempty"`
	HeaderNames []string `json:"header_names,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	Source      string   `json:"source,omitempty"`
}

// AuditSinkInspection 描述 audit sink 链的配置与生效结果。
type AuditSinkInspection struct {
	Mode            string                  `json:"mode"`
	ConfiguredSinks []AuditSinkSelectorView `json:"configured_sinks,omitempty"`
	ActiveSinks     []AuditSinkSummary      `json:"active_sinks,omitempty"`
	Warnings        []string                `json:"warnings,omitempty"`
}

type pluginAuditSink struct {
	runtime pluginruntime.AuditSinkRuntime
}

type stdoutAuditSink struct {
	writer io.Writer
}

type webhookAuditSink struct {
	url       string
	headers   map[string]string
	timeout   time.Duration
	newClient func(timeout time.Duration) *http.Client
}

type selectedAuditSink struct {
	Sink    auditSink
	Summary AuditSinkSummary
}

const defaultAuditWebhookTimeout = 5 * time.Second

var builtinAuditStdoutWriter io.Writer = os.Stdout

var newAuditWebhookHTTPClient = func(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

func openAuditSinks(cfg *config.Config) ([]auditSink, []string) {
	if cfg == nil {
		cfg = config.New()
	}

	selections, warnings := resolveSelectedAuditSinks(cfg)
	return auditSelectionsToSinks(selections), warnings
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
			warnings = append(warnings, fmt.Sprintf("failed to emit event to audit sink %s: %s", sink.Name(), err.Error()))
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
		// Audit sinks are currently used per emission and then closed to avoid lingering processes.
		_ = s.runtime.Close()
	}()

	return s.runtime.Emit(ctx, pluginruntime.AuditEmitParams{
		SinkID: s.runtime.ID(),
		Record: convertAuditRecordToPlugin(record),
	})
}

func (s *stdoutAuditSink) Name() string {
	return "builtin/stdout"
}

func (s *stdoutAuditSink) Emit(ctx context.Context, record auditRecord) error {
	_ = ctx
	if s == nil || s.writer == nil {
		return nil
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if _, err := s.writer.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *webhookAuditSink) Name() string {
	return "builtin/webhook"
}

func (s *webhookAuditSink) Emit(ctx context.Context, record auditRecord) error {
	if s == nil {
		return nil
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	for key, value := range s.headers {
		request.Header.Set(key, value)
	}

	newClient := s.newClient
	if newClient == nil {
		newClient = newAuditWebhookHTTPClient
	}
	response, err := newClient(s.timeout).Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", response.StatusCode)
	}
	return nil
}

// InspectAuditSinks 返回当前配置下会参与扇出的审计 sink 摘要。
func InspectAuditSinks(cfg *config.Config) AuditSinkInspection {
	if cfg == nil {
		cfg = config.New()
	}

	selections, warnings := resolveSelectedAuditSinks(cfg)
	return AuditSinkInspection{
		Mode:            config.ResolveAuditMode(cfg),
		ConfiguredSinks: buildAuditSinkSelectorViews(config.ResolveAuditSinks(cfg)),
		ActiveSinks:     summarizeSelectedAuditSinks(selections),
		Warnings:        warnings,
	}
}

func resolveSelectedAuditSinks(cfg *config.Config) ([]selectedAuditSink, []string) {
	if cfg == nil {
		cfg = config.New()
	}

	mode := config.ResolveAuditMode(cfg)
	if mode == config.RuntimeSelectionModeDisabled {
		return nil, nil
	}

	configured := config.ResolveAuditSinks(cfg)
	runtimes, warnings := discoverAuditSinkRuntimesIfNeeded(cfg, configured, mode)

	if len(configured) == 0 {
		if mode == config.RuntimeSelectionModeManual {
			return nil, warnings
		}
		items := make([]selectedAuditSink, 0, len(runtimes))
		for _, runtime := range runtimes {
			items = append(items, selectedAuditSink{
				Sink: &pluginAuditSink{runtime: runtime},
				Summary: AuditSinkSummary{
					Type:     config.AuditSinkTypePlugin,
					Plugin:   strings.TrimSpace(runtime.Name()),
					SinkID:   strings.TrimSpace(runtime.ID()),
					Label:    auditSinkRuntimeLabel(runtime),
					Priority: runtime.Priority(),
					Source:   "auto",
				},
			})
		}
		return items, warnings
	}

	selected := make([]selectedAuditSink, 0, len(configured))
	seen := make(map[string]struct{})
	for _, item := range configured {
		switch item.Type {
		case config.AuditSinkTypeStdout:
			key := "builtin|stdout"
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			selected = append(selected, selectedAuditSink{
				Sink: &stdoutAuditSink{writer: builtinAuditStdoutWriter},
				Summary: AuditSinkSummary{
					Type:   config.AuditSinkTypeStdout,
					Label:  "builtin/stdout",
					Source: "configured",
				},
			})
		case config.AuditSinkTypeWebhook:
			sink, summary, err := buildWebhookAuditSink(item)
			if err != nil {
				warnings = append(warnings, "configured audit sink webhook is invalid: "+err.Error())
				continue
			}
			key := "builtin|webhook|" + summary.URL
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			selected = append(selected, selectedAuditSink{
				Sink:    sink,
				Summary: summary,
			})
		case config.AuditSinkTypePlugin:
			matches := matchAuditSinkRuntimes(runtimes, item)
			if len(matches) == 0 {
				warnings = append(warnings, fmt.Sprintf("configured audit sink selector %s did not match any discovered audit sink capability", auditSinkSelectorLabel(item)))
				continue
			}
			for _, index := range matches {
				runtime := runtimes[index]
				key := "plugin|" + strings.TrimSpace(runtime.Name()) + "|" + strings.TrimSpace(runtime.ID())
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				selected = append(selected, selectedAuditSink{
					Sink: &pluginAuditSink{runtime: runtime},
					Summary: AuditSinkSummary{
						Type:     config.AuditSinkTypePlugin,
						Plugin:   strings.TrimSpace(runtime.Name()),
						SinkID:   strings.TrimSpace(runtime.ID()),
						Label:    auditSinkRuntimeLabel(runtime),
						Priority: runtime.Priority(),
						Source:   "configured",
					},
				})
			}
		default:
			warnings = append(warnings, fmt.Sprintf("configured audit sink type %s is not supported", strings.TrimSpace(item.Type)))
		}
	}
	return selected, warnings
}

func discoverAuditSinkRuntimesIfNeeded(cfg *config.Config, configured []config.AuditSinkConfig, mode string) ([]pluginruntime.AuditSinkRuntime, []string) {
	needsPluginDiscovery := len(configured) == 0 && mode != config.RuntimeSelectionModeManual
	if !needsPluginDiscovery {
		for _, item := range configured {
			if item.Type == config.AuditSinkTypePlugin {
				needsPluginDiscovery = true
				break
			}
		}
	}
	if !needsPluginDiscovery {
		return nil, nil
	}

	runtimes, err := pluginruntime.DiscoverAuditSinkRuntimes(pluginruntime.DiscoveryOptions{
		EnabledPlugins: config.ResolveEnabledPlugins(cfg),
	})
	if err != nil {
		return nil, []string{"failed to discover audit sink plugins: " + err.Error()}
	}
	return runtimes, nil
}

func buildWebhookAuditSink(item config.AuditSinkConfig) (auditSink, AuditSinkSummary, error) {
	rawURL := strings.TrimSpace(item.URL)
	if rawURL == "" {
		return nil, AuditSinkSummary{}, fmt.Errorf("missing url")
	}
	resolvedURL, err := config.ResolveSecret(rawURL)
	if err != nil {
		return nil, AuditSinkSummary{}, fmt.Errorf("failed to resolve url: %w", err)
	}

	headers := make(map[string]string, len(item.Headers))
	for key, value := range item.Headers {
		resolvedValue, err := config.ResolveSecret(value)
		if err != nil {
			return nil, AuditSinkSummary{}, fmt.Errorf("failed to resolve header %s: %w", key, err)
		}
		headers[key] = resolvedValue
	}

	timeout := defaultAuditWebhookTimeout
	if item.TimeoutMS > 0 {
		timeout = time.Duration(item.TimeoutMS) * time.Millisecond
	}
	return &webhookAuditSink{
			url:       resolvedURL,
			headers:   headers,
			timeout:   timeout,
			newClient: newAuditWebhookHTTPClient,
		}, AuditSinkSummary{
			Type:        config.AuditSinkTypeWebhook,
			Label:       "builtin/webhook",
			URL:         rawURL,
			HeaderNames: headerNames(item.Headers),
			Source:      "configured",
		}, nil
}

func matchAuditSinkRuntimes(runtimes []pluginruntime.AuditSinkRuntime, selector config.AuditSinkConfig) []int {
	matches := make([]int, 0)
	for index, runtime := range runtimes {
		if !auditSinkRuntimeMatchesSelector(runtime, selector) {
			continue
		}
		matches = append(matches, index)
	}
	return matches
}

func auditSinkRuntimeMatchesSelector(runtime pluginruntime.AuditSinkRuntime, selector config.AuditSinkConfig) bool {
	if runtime == nil {
		return false
	}
	if selector.Plugin != "" && selector.Plugin != strings.TrimSpace(runtime.Name()) {
		return false
	}
	if selector.SinkID != "" && selector.SinkID != strings.TrimSpace(runtime.ID()) {
		return false
	}
	return selector.Plugin != "" || selector.SinkID != ""
}

func auditSelectionsToSinks(items []selectedAuditSink) []auditSink {
	if len(items) == 0 {
		return nil
	}
	sinks := make([]auditSink, 0, len(items))
	for _, item := range items {
		if item.Sink == nil {
			continue
		}
		sinks = append(sinks, item.Sink)
	}
	return sinks
}

func summarizeSelectedAuditSinks(items []selectedAuditSink) []AuditSinkSummary {
	if len(items) == 0 {
		return nil
	}
	summaries := make([]AuditSinkSummary, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, item.Summary)
	}
	return summaries
}

func buildAuditSinkSelectorViews(items []config.AuditSinkConfig) []AuditSinkSelectorView {
	if len(items) == 0 {
		return nil
	}
	views := make([]AuditSinkSelectorView, 0, len(items))
	for _, item := range items {
		views = append(views, AuditSinkSelectorView{
			Type:        item.Type,
			Plugin:      item.Plugin,
			SinkID:      item.SinkID,
			URL:         item.URL,
			HeaderNames: headerNames(item.Headers),
			TimeoutMS:   item.TimeoutMS,
		})
	}
	return views
}

func headerNames(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	names := make([]string, 0, len(values))
	for key := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		names = append(names, key)
	}
	sort.Strings(names)
	return names
}

func auditSinkRuntimeLabel(runtime pluginruntime.AuditSinkRuntime) string {
	if runtime == nil {
		return ""
	}
	pluginName := strings.TrimSpace(runtime.Name())
	sinkID := strings.TrimSpace(runtime.ID())
	switch {
	case pluginName != "" && sinkID != "":
		return pluginName + "/" + sinkID
	case pluginName != "":
		return pluginName
	default:
		return sinkID
	}
}

func auditSinkSelectorLabel(item config.AuditSinkConfig) string {
	switch item.Type {
	case config.AuditSinkTypePlugin:
		switch {
		case item.Plugin != "" && item.SinkID != "":
			return item.Plugin + "/" + item.SinkID
		case item.Plugin != "":
			return item.Plugin
		default:
			return item.SinkID
		}
	case config.AuditSinkTypeWebhook:
		return item.URL
	default:
		return item.Type
	}
}
