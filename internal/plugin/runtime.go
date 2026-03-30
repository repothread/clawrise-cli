package plugin

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

// Runtime describes one provider runtime backend.
type Runtime interface {
	Name() string
	Handshake(ctx context.Context) (HandshakeResult, error)
	ListOperations(ctx context.Context) ([]adapter.Definition, error)
	GetCatalog(ctx context.Context) ([]speccatalog.Entry, error)
	ListAuthMethods(ctx context.Context) ([]AuthMethodDescriptor, error)
	ListAuthPresets(ctx context.Context) ([]AuthPresetDescriptor, error)
	InspectAuth(ctx context.Context, params AuthInspectParams) (AuthInspectResult, error)
	BeginAuth(ctx context.Context, params AuthBeginParams) (AuthBeginResult, error)
	CompleteAuth(ctx context.Context, params AuthCompleteParams) (AuthCompleteResult, error)
	ResolveAuth(ctx context.Context, params AuthResolveParams) (AuthResolveResult, error)
	Execute(ctx context.Context, req ExecuteRequest) (ExecuteResult, error)
	Health(ctx context.Context) (HealthResult, error)
}

// AuthProvider describes the auth capability surface exposed by a provider plugin.
type AuthProvider interface {
	ListMethods(ctx context.Context) ([]AuthMethodDescriptor, error)
	ListPresets(ctx context.Context) ([]AuthPresetDescriptor, error)
	Inspect(ctx context.Context, params AuthInspectParams) (AuthInspectResult, error)
	Begin(ctx context.Context, params AuthBeginParams) (AuthBeginResult, error)
	Complete(ctx context.Context, params AuthCompleteParams) (AuthCompleteResult, error)
	Resolve(ctx context.Context, params AuthResolveParams) (AuthResolveResult, error)
}

// AuthLauncherRuntime describes one auth-action launcher runtime.
type AuthLauncherRuntime interface {
	Name() string
	Handshake(ctx context.Context) (HandshakeResult, error)
	DescribeAuthLauncher(ctx context.Context) (AuthLauncherDescriptor, error)
	LaunchAuth(ctx context.Context, params AuthLaunchParams) (AuthLaunchResult, error)
}

// HandshakeResult describes one provider runtime handshake result.
type HandshakeResult struct {
	ProtocolVersion int      `json:"protocol_version"`
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Platforms       []string `json:"platforms"`
}

// ExecuteRequest describes one normalized provider execution request.
type ExecuteRequest struct {
	Operation      string
	Identity       ExecuteIdentity
	Input          map[string]any
	IdempotencyKey string
}

// ExecuteResult describes one provider execution result.
type ExecuteResult struct {
	Data  map[string]any   `json:"data"`
	Error *apperr.AppError `json:"error,omitempty"`
}

// HealthResult describes one provider runtime health result.
type HealthResult struct {
	OK      bool           `json:"ok"`
	Details map[string]any `json:"details,omitempty"`
}

// ManagerOptions describes optional manager capabilities.
type ManagerOptions struct {
	AuthLaunchers []AuthLauncherRuntime
}

type authLauncherRegistration struct {
	runtime    AuthLauncherRuntime
	descriptor AuthLauncherDescriptor
	order      int
}

// Manager aggregates provider runtimes into one execution and discovery view.
type Manager struct {
	registry          *adapter.Registry
	catalogEntries    []speccatalog.Entry
	operationRuntimes map[string]Runtime
	platformRuntimes  map[string]Runtime
	authLaunchers     []authLauncherRegistration
}

// NewManager creates one aggregated provider runtime manager.
func NewManager(ctx context.Context, runtimes []Runtime) (*Manager, error) {
	return NewManagerWithOptions(ctx, runtimes, ManagerOptions{})
}

// NewManagerWithOptions creates one aggregated provider runtime manager with optional launchers.
func NewManagerWithOptions(ctx context.Context, runtimes []Runtime, options ManagerOptions) (*Manager, error) {
	registry := adapter.NewRegistry()
	operationRuntimes := make(map[string]Runtime)
	platformRuntimes := make(map[string]Runtime)
	catalogEntries := make([]speccatalog.Entry, 0)

	for _, runtime := range runtimes {
		handshake, err := runtime.Handshake(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to handshake provider runtime %s: %w", runtime.Name(), err)
		}
		if handshake.Name == "" {
			return nil, fmt.Errorf("provider runtime %s returned an empty handshake name", runtime.Name())
		}
		for _, platform := range handshake.Platforms {
			platform = strings.TrimSpace(platform)
			if platform == "" {
				continue
			}
			if _, exists := platformRuntimes[platform]; exists {
				return nil, fmt.Errorf("duplicate provider runtime registered for platform: %s", platform)
			}
			platformRuntimes[platform] = runtime
		}

		definitions, err := runtime.ListOperations(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list operations for provider runtime %s: %w", runtime.Name(), err)
		}
		for _, definition := range definitions {
			if _, exists := operationRuntimes[definition.Operation]; exists {
				return nil, fmt.Errorf("duplicate operation registered by provider runtimes: %s", definition.Operation)
			}

			runtimeRef := runtime
			operation := definition.Operation
			definition.Handler = func(ctx context.Context, call adapter.Call) (map[string]any, *apperr.AppError) {
				result, err := runtimeRef.Execute(ctx, ExecuteRequest{
					Operation:      operation,
					Identity:       buildExecuteIdentityFromCall(call),
					Input:          call.Input,
					IdempotencyKey: call.IdempotencyKey,
				})
				if err != nil {
					return nil, apperr.New("PROVIDER_RUNTIME_FAILED", err.Error())
				}
				return result.Data, result.Error
			}

			registry.Register(definition)
			operationRuntimes[definition.Operation] = runtime
		}

		entries, err := runtime.GetCatalog(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load catalog for provider runtime %s: %w", runtime.Name(), err)
		}
		catalogEntries = append(catalogEntries, entries...)
	}

	authLaunchers := make([]authLauncherRegistration, 0, len(options.AuthLaunchers))
	for index, runtime := range options.AuthLaunchers {
		handshake, err := runtime.Handshake(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to handshake auth launcher %s: %w", runtime.Name(), err)
		}
		if handshake.Name == "" {
			return nil, fmt.Errorf("auth launcher %s returned an empty handshake name", runtime.Name())
		}
		descriptor, err := runtime.DescribeAuthLauncher(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe auth launcher %s: %w", runtime.Name(), err)
		}
		if strings.TrimSpace(descriptor.ID) == "" {
			descriptor.ID = handshake.Name
		}
		if strings.TrimSpace(descriptor.DisplayName) == "" {
			descriptor.DisplayName = descriptor.ID
		}
		authLaunchers = append(authLaunchers, authLauncherRegistration{
			runtime:    runtime,
			descriptor: descriptor,
			order:      index,
		})
	}

	return &Manager{
		registry:          registry,
		catalogEntries:    catalogEntries,
		operationRuntimes: operationRuntimes,
		platformRuntimes:  platformRuntimes,
		authLaunchers:     authLaunchers,
	}, nil
}

// Registry returns the aggregated operation registry view.
func (m *Manager) Registry() *adapter.Registry {
	return m.registry
}

// CatalogEntries returns the aggregated structured catalog view.
func (m *Manager) CatalogEntries() []speccatalog.Entry {
	return append([]speccatalog.Entry(nil), m.catalogEntries...)
}

// RuntimeForPlatform returns the provider runtime registered for one platform.
func (m *Manager) RuntimeForPlatform(platform string) (Runtime, bool) {
	if m == nil {
		return nil, false
	}
	runtime, ok := m.platformRuntimes[strings.TrimSpace(platform)]
	return runtime, ok
}

func buildExecuteIdentityFromCall(call adapter.Call) ExecuteIdentity {
	identity := call.Identity
	return ExecuteIdentity{
		Platform:    strings.TrimSpace(identity.Platform),
		Subject:     strings.TrimSpace(identity.Subject),
		AccountName: strings.TrimSpace(identity.AccountName),
		Auth: ExecuteAuth{
			Method:        strings.TrimSpace(identity.AuthMethod),
			ExecutionAuth: cloneMap(identity.ExecutionAuth),
		},
	}
}

// AuthLaunchers returns all registered auth launcher descriptors.
func (m *Manager) AuthLaunchers() []AuthLauncherDescriptor {
	if m == nil || len(m.authLaunchers) == 0 {
		return nil
	}
	items := make([]AuthLauncherDescriptor, 0, len(m.authLaunchers))
	for _, launcher := range m.authLaunchers {
		items = append(items, launcher.descriptor)
	}
	return items
}

// ListAuthMethods returns the auth method descriptors exposed by provider runtimes.
func (m *Manager) ListAuthMethods(ctx context.Context, platform string) ([]AuthMethodDescriptor, error) {
	if strings.TrimSpace(platform) != "" {
		runtime, ok := m.RuntimeForPlatform(platform)
		if !ok {
			return nil, fmt.Errorf("no provider runtime is registered for platform %s", platform)
		}
		return runtime.ListAuthMethods(ctx)
	}

	items := make([]AuthMethodDescriptor, 0)
	for _, runtime := range m.platformRuntimes {
		methods, err := runtime.ListAuthMethods(ctx)
		if err != nil {
			return nil, err
		}
		items = append(items, methods...)
	}
	return items, nil
}

// ListAuthPresets returns the account preset descriptors exposed by provider runtimes.
func (m *Manager) ListAuthPresets(ctx context.Context, platform string) ([]AuthPresetDescriptor, error) {
	if strings.TrimSpace(platform) != "" {
		runtime, ok := m.RuntimeForPlatform(platform)
		if !ok {
			return nil, fmt.Errorf("no provider runtime is registered for platform %s", platform)
		}
		return runtime.ListAuthPresets(ctx)
	}

	items := make([]AuthPresetDescriptor, 0)
	for _, runtime := range m.platformRuntimes {
		presets, err := runtime.ListAuthPresets(ctx)
		if err != nil {
			return nil, err
		}
		items = append(items, presets...)
	}
	return items, nil
}

// InspectAuth runs provider auth inspection for one platform account.
func (m *Manager) InspectAuth(ctx context.Context, platform string, params AuthInspectParams) (AuthInspectResult, error) {
	runtime, ok := m.RuntimeForPlatform(platform)
	if !ok {
		return AuthInspectResult{}, fmt.Errorf("no provider runtime is registered for platform %s", platform)
	}
	return runtime.InspectAuth(ctx, params)
}

// BeginAuth starts an auth flow for one platform account.
func (m *Manager) BeginAuth(ctx context.Context, platform string, params AuthBeginParams) (AuthBeginResult, error) {
	runtime, ok := m.RuntimeForPlatform(platform)
	if !ok {
		return AuthBeginResult{}, fmt.Errorf("no provider runtime is registered for platform %s", platform)
	}
	return runtime.BeginAuth(ctx, params)
}

// CompleteAuth completes an auth flow for one platform account.
func (m *Manager) CompleteAuth(ctx context.Context, platform string, params AuthCompleteParams) (AuthCompleteResult, error) {
	runtime, ok := m.RuntimeForPlatform(platform)
	if !ok {
		return AuthCompleteResult{}, fmt.Errorf("no provider runtime is registered for platform %s", platform)
	}
	return runtime.CompleteAuth(ctx, params)
}

// ResolveAuth resolves provider auth before one execution.
func (m *Manager) ResolveAuth(ctx context.Context, platform string, params AuthResolveParams) (AuthResolveResult, error) {
	runtime, ok := m.RuntimeForPlatform(platform)
	if !ok {
		return AuthResolveResult{}, fmt.Errorf("no provider runtime is registered for platform %s", platform)
	}
	return runtime.ResolveAuth(ctx, params)
}

// LaunchAuth delegates one launchable auth action to the most suitable launcher runtime.
func (m *Manager) LaunchAuth(ctx context.Context, params AuthLaunchParams) (AuthLaunchResult, error) {
	if m == nil || len(m.authLaunchers) == 0 {
		return AuthLaunchResult{
			Handled: false,
			Status:  "no_launcher_available",
			Message: "no auth launcher is registered",
		}, nil
	}

	candidates := make([]authLauncherRegistration, 0)
	for _, launcher := range m.authLaunchers {
		if !launcherSupportsAction(launcher.descriptor, params.Context.Platform, params.Action.Type) {
			continue
		}
		candidates = append(candidates, launcher)
	}
	if len(candidates) == 0 {
		return AuthLaunchResult{
			Handled: false,
			Status:  "no_matching_launcher",
			Message: fmt.Sprintf("no auth launcher supports action %s", strings.TrimSpace(params.Action.Type)),
		}, nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].descriptor.Priority == candidates[j].descriptor.Priority {
			return candidates[i].order < candidates[j].order
		}
		return candidates[i].descriptor.Priority > candidates[j].descriptor.Priority
	})

	var launchErr error
	for _, launcher := range candidates {
		result, err := launcher.runtime.LaunchAuth(ctx, params)
		if err != nil {
			launchErr = err
			continue
		}
		if strings.TrimSpace(result.LauncherID) == "" {
			result.LauncherID = launcher.descriptor.ID
		}
		if strings.TrimSpace(result.Status) == "" {
			if result.Handled {
				result.Status = "launched"
			} else {
				result.Status = "skipped"
			}
		}
		return result, nil
	}

	if launchErr == nil {
		return AuthLaunchResult{
			Handled: false,
			Status:  "launcher_skipped",
		}, nil
	}
	return AuthLaunchResult{
		Handled: false,
		Status:  "launcher_failed",
		Message: launchErr.Error(),
	}, launchErr
}

// NewDiscoveredManager creates a manager backed by discovered provider plugins
// and available auth launchers.
func NewDiscoveredManager(ctx context.Context) (*Manager, error) {
	roots, err := DefaultDiscoveryRoots()
	if err != nil {
		return nil, err
	}
	manifests, err := DiscoverManifests(roots)
	if err != nil {
		return nil, err
	}
	providerManifests, launcherManifests := SplitManifestsByKind(manifests)
	launchers := append([]AuthLauncherRuntime{
		NewSystemAuthLauncherRuntime(),
	}, NewProcessAuthLaunchers(launcherManifests)...)
	return NewManagerWithOptions(ctx, NewProcessRuntimes(providerManifests), ManagerOptions{
		AuthLaunchers: launchers,
	})
}

func launcherSupportsAction(descriptor AuthLauncherDescriptor, platform string, actionType string) bool {
	actionType = strings.TrimSpace(actionType)
	if actionType == "" {
		return false
	}

	if len(descriptor.ActionTypes) > 0 && !stringSliceContainsTrimmed(descriptor.ActionTypes, actionType) {
		return false
	}
	if len(descriptor.Platforms) > 0 && !stringSliceContainsTrimmed(descriptor.Platforms, platform) {
		return false
	}
	return true
}

func stringSliceContainsTrimmed(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}
