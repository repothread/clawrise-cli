package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/config"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

// ProcessRuntime executes plugin RPC calls against one external plugin process.
type ProcessRuntime struct {
	manifest Manifest

	command *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader

	mu        sync.Mutex
	started   bool
	requestID uint64
}

// NewProcessRuntime creates one process-backed provider runtime.
func NewProcessRuntime(manifest Manifest) *ProcessRuntime {
	return &ProcessRuntime{
		manifest: manifest,
	}
}

func (r *ProcessRuntime) Name() string {
	return r.manifest.Name
}

func (r *ProcessRuntime) Handshake(ctx context.Context) (HandshakeResult, error) {
	var result HandshakeResult
	if err := r.call(ctx, "clawrise.handshake", HandshakeParams{
		ProtocolVersion: ProtocolVersion,
		Core: CoreVersion{
			Name:    "clawrise",
			Version: "dev",
		},
	}, &result); err != nil {
		return HandshakeResult{}, err
	}
	return result, nil
}

func (r *ProcessRuntime) ListOperations(ctx context.Context) ([]adapter.Definition, error) {
	var result OperationsListResult
	if err := r.call(ctx, "clawrise.operations.list", map[string]any{}, &result); err != nil {
		return nil, err
	}

	definitions := make([]adapter.Definition, 0, len(result.Operations))
	for _, operation := range result.Operations {
		definitions = append(definitions, adapter.Definition{
			Operation:       operation.Operation,
			Platform:        operation.Platform,
			Mutating:        operation.Mutating,
			DefaultTimeout:  time.Duration(operation.DefaultTimeoutMS) * time.Millisecond,
			AllowedSubjects: append([]string(nil), operation.AllowedSubjects...),
			Spec:            operation.Spec,
		})
	}
	return definitions, nil
}

func (r *ProcessRuntime) GetCatalog(ctx context.Context) ([]speccatalog.Entry, error) {
	var result CatalogResult
	if err := r.call(ctx, "clawrise.catalog.get", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return append([]speccatalog.Entry(nil), result.Entries...), nil
}

func (r *ProcessRuntime) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	var result ExecuteRPCResult
	if err := r.call(ctx, "clawrise.execute", ExecuteParams{
		Request: ExecuteEnvelope{
			RequestID:      "",
			Operation:      req.Operation,
			Input:          req.Input,
			IdempotencyKey: req.IdempotencyKey,
			DryRun:         false,
		},
		Identity: buildExecuteIdentity(req.Profile),
	}, &result); err != nil {
		return ExecuteResult{}, err
	}
	if result.OK {
		return ExecuteResult{Data: result.Data}, nil
	}
	return ExecuteResult{Data: result.Data, Error: result.Error}, nil
}

func (r *ProcessRuntime) Health(ctx context.Context) (HealthResult, error) {
	var result HealthResult
	if err := r.call(ctx, "clawrise.health", map[string]any{}, &result); err != nil {
		return HealthResult{}, err
	}
	return result, nil
}

func (r *ProcessRuntime) call(ctx context.Context, method string, params any, result any) error {
	if err := r.ensureStarted(ctx); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	requestID := fmt.Sprintf("%d", atomic.AddUint64(&r.requestID, 1))
	request := RPCRequest{
		JSONRPC: "2.0",
		ID:      requestID,
		Method:  method,
		Params:  params,
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to encode plugin RPC request: %w", err)
	}
	if _, err := r.stdin.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("failed to write plugin RPC request: %w", err)
	}

	line, err := r.stdout.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("failed to read plugin RPC response: %w", err)
	}

	var response RPCResponse
	if err := json.Unmarshal(line, &response); err != nil {
		return fmt.Errorf("failed to decode plugin RPC response: %w", err)
	}
	if response.Error != nil {
		return fmt.Errorf("plugin RPC error %d: %s", response.Error.Code, response.Error.Message)
	}
	if result == nil || response.Result == nil {
		return nil
	}

	data, err := json.Marshal(response.Result)
	if err != nil {
		return fmt.Errorf("failed to re-encode plugin RPC result: %w", err)
	}
	if err := json.Unmarshal(data, result); err != nil {
		return fmt.Errorf("failed to decode plugin RPC result payload: %w", err)
	}
	return nil
}

func (r *ProcessRuntime) ensureStarted(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return nil
	}

	command := r.manifest.ResolveCommand()
	if len(command) == 0 {
		return fmt.Errorf("plugin %s command is empty", r.manifest.Name)
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create plugin stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create plugin stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start plugin process: %w", err)
	}

	r.command = cmd
	r.stdin = stdin
	r.stdout = bufio.NewReader(stdout)
	r.started = true
	return nil
}

// NewProcessRuntimes creates process runtimes from discovered manifests.
func NewProcessRuntimes(manifests []Manifest) []Runtime {
	runtimes := make([]Runtime, 0, len(manifests))
	for _, manifest := range manifests {
		runtimes = append(runtimes, NewProcessRuntime(manifest))
	}
	return runtimes
}

func buildExecuteIdentity(profile config.Profile) ExecuteIdentity {
	return ExecuteIdentity{
		Platform: profile.Platform,
		Subject:  profile.Subject,
		Auth:     buildResolvedAuthPayload(profile),
	}
}

func buildResolvedAuthPayload(profile config.Profile) map[string]any {
	auth := map[string]any{
		"type": profile.Grant.Type,
	}

	resolveAndSet := func(key, raw string) {
		if raw == "" {
			return
		}
		if value, err := config.ResolveSecret(raw); err == nil && value != "" {
			auth[key] = value
		}
	}

	resolveAndSet("app_id", profile.Grant.AppID)
	resolveAndSet("app_secret", profile.Grant.AppSecret)
	resolveAndSet("token", profile.Grant.Token)
	resolveAndSet("client_id", profile.Grant.ClientID)
	resolveAndSet("client_secret", profile.Grant.ClientSecret)
	resolveAndSet("access_token", profile.Grant.AccessToken)
	resolveAndSet("refresh_token", profile.Grant.RefreshToken)
	if profile.Grant.NotionVer != "" {
		auth["notion_version"] = profile.Grant.NotionVer
	}
	return auth
}
