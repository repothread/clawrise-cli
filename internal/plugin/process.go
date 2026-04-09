package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/clawrise/clawrise-cli/internal/adapter"
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

// Manifest returns the manifest associated with the current process runtime.
func (r *ProcessRuntime) Manifest() Manifest {
	return r.manifest
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

// ListCapabilities returns the capability list declared by the plugin process.
func (r *ProcessRuntime) ListCapabilities(ctx context.Context) ([]CapabilityDescriptor, error) {
	var result CapabilityListResult
	if err := r.call(ctx, "clawrise.capabilities.list", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return cloneCapabilityList(result.Capabilities), nil
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

func (r *ProcessRuntime) ListAuthMethods(ctx context.Context) ([]AuthMethodDescriptor, error) {
	var result AuthMethodsListResult
	if err := r.call(ctx, "clawrise.auth.methods.list", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return append([]AuthMethodDescriptor(nil), result.Methods...), nil
}

func (r *ProcessRuntime) ListAuthPresets(ctx context.Context) ([]AuthPresetDescriptor, error) {
	var result AuthPresetsListResult
	if err := r.call(ctx, "clawrise.auth.presets.list", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return append([]AuthPresetDescriptor(nil), result.Presets...), nil
}

func (r *ProcessRuntime) InspectAuth(ctx context.Context, params AuthInspectParams) (AuthInspectResult, error) {
	var result AuthInspectResult
	if err := r.call(ctx, "clawrise.auth.inspect", params, &result); err != nil {
		return AuthInspectResult{}, err
	}
	return result, nil
}

func (r *ProcessRuntime) BeginAuth(ctx context.Context, params AuthBeginParams) (AuthBeginResult, error) {
	var result AuthBeginResult
	if err := r.call(ctx, "clawrise.auth.begin", params, &result); err != nil {
		return AuthBeginResult{}, err
	}
	return result, nil
}

func (r *ProcessRuntime) CompleteAuth(ctx context.Context, params AuthCompleteParams) (AuthCompleteResult, error) {
	var result AuthCompleteResult
	if err := r.call(ctx, "clawrise.auth.complete", params, &result); err != nil {
		return AuthCompleteResult{}, err
	}
	return result, nil
}

func (r *ProcessRuntime) ResolveAuth(ctx context.Context, params AuthResolveParams) (AuthResolveResult, error) {
	var result AuthResolveResult
	if err := r.call(ctx, "clawrise.auth.resolve", params, &result); err != nil {
		return AuthResolveResult{}, err
	}
	return result, nil
}

// DescribeAuthLauncher returns launcher capability metadata from one external plugin.
func (r *ProcessRuntime) DescribeAuthLauncher(ctx context.Context) (AuthLauncherDescriptor, error) {
	var result AuthLauncherDescribeResult
	if err := r.call(ctx, "clawrise.auth.launcher.describe", map[string]any{}, &result); err != nil {
		return AuthLauncherDescriptor{}, err
	}
	return result.Launcher, nil
}

// LaunchAuth delegates one auth action to the external launcher plugin.
func (r *ProcessRuntime) LaunchAuth(ctx context.Context, params AuthLaunchParams) (AuthLaunchResult, error) {
	var result AuthLaunchResult
	if err := r.call(ctx, "clawrise.auth.launcher.run", params, &result); err != nil {
		return AuthLaunchResult{}, err
	}
	return result, nil
}

func (r *ProcessRuntime) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error) {
	var result ExecuteRPCResult
	if err := r.call(ctx, "clawrise.execute", ExecuteParams{
		Request: ExecuteEnvelope{
			RequestID:            req.RequestID,
			Operation:            req.Operation,
			Input:                req.Input,
			TimeoutMS:            req.TimeoutMS,
			IdempotencyKey:       req.IdempotencyKey,
			DryRun:               false,
			DebugProviderPayload: req.DebugProviderPayload,
			VerifyAfterWrite:     req.VerifyAfterWrite,
		},
		Identity: req.Identity,
	}, &result); err != nil {
		return ExecuteResult{}, err
	}
	if result.OK {
		return ExecuteResult{Data: result.Data, Debug: result.Debug}, nil
	}
	return ExecuteResult{Data: result.Data, Debug: result.Debug, Error: result.Error}, nil
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

// NewProcessAuthLaunchers creates process-backed auth launchers from manifests.
func NewProcessAuthLaunchers(manifests []Manifest) []AuthLauncherRuntime {
	launchers := make([]AuthLauncherRuntime, 0, len(manifests))
	for _, manifest := range manifests {
		launchers = append(launchers, NewProcessRuntime(manifest))
	}
	return launchers
}

// Close terminates the plugin child process and releases related resources.
func (r *ProcessRuntime) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return nil
	}

	if r.stdin != nil {
		_ = r.stdin.Close()
	}

	var waitErr error
	if r.command != nil && r.command.Process != nil {
		if killErr := r.command.Process.Kill(); killErr != nil && killErr != os.ErrProcessDone {
			waitErr = killErr
		}
		if err := r.command.Wait(); err != nil && waitErr == nil {
			waitErr = err
		}
	}

	r.command = nil
	r.stdin = nil
	r.stdout = nil
	r.started = false
	return waitErr
}

// NewProcessRuntimes creates process runtimes from discovered manifests.
func NewProcessRuntimes(manifests []Manifest) []Runtime {
	runtimes := make([]Runtime, 0, len(manifests))
	for _, manifest := range manifests {
		runtimes = append(runtimes, NewProcessRuntime(manifest))
	}
	return runtimes
}

func cloneMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
