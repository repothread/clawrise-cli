package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

const (
	sampleAuditPluginName    = "sample-audit"
	sampleAuditPluginVersion = "0.1.0"
	sampleAuditSinkID        = "file_capture"
	sampleAuditLogPathEnv    = "CLAWRISE_SAMPLE_AUDIT_LOG"
)

// sampleAuditRuntime appends audit events to a newline-delimited JSON file.
// The file path comes from an environment variable so the sample remains easy
// to run in tests and local development without depending on the main config.
type sampleAuditRuntime struct{}

func (r *sampleAuditRuntime) Name() string {
	return sampleAuditPluginName
}

func (r *sampleAuditRuntime) ID() string {
	return sampleAuditSinkID
}

func (r *sampleAuditRuntime) Priority() int {
	return 50
}

func (r *sampleAuditRuntime) Handshake(ctx context.Context) (pluginruntime.HandshakeResult, error) {
	_ = ctx

	return pluginruntime.HandshakeResult{
		ProtocolVersion: pluginruntime.ProtocolVersion,
		Name:            sampleAuditPluginName,
		Version:         sampleAuditPluginVersion,
	}, nil
}

func (r *sampleAuditRuntime) Emit(ctx context.Context, params pluginruntime.AuditEmitParams) error {
	_ = ctx

	logPath := strings.TrimSpace(os.Getenv(sampleAuditLogPathEnv))
	if logPath == "" {
		logPath = filepath.Join(os.TempDir(), "clawrise-sample-audit.ndjson")
	}

	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("failed to create audit log directory: %w", err)
	}

	entry := map[string]any{
		"plugin":  sampleAuditPluginName,
		"sink_id": firstNonEmpty(strings.TrimSpace(params.SinkID), sampleAuditSinkID),
		"record":  params.Record,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to encode audit entry: %w", err)
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open audit log: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to append audit entry: %w", err)
	}
	return nil
}

func (r *sampleAuditRuntime) Close() error {
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func main() {
	if err := pluginruntime.ServeAuditSinkRuntime(&sampleAuditRuntime{}); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
