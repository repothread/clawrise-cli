package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	"github.com/clawrise/clawrise-cli/internal/apperr"
	"github.com/clawrise/clawrise-cli/internal/config"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

const (
	policyDecisionAllow           = "allow"
	policyDecisionDeny            = "deny"
	policyDecisionRequireApproval = "require_approval"
	policyDecisionAnnotate        = "annotate"
)

// PolicySelectorView 描述一个来自配置文件的 policy selector。
type PolicySelectorView struct {
	Plugin   string `json:"plugin,omitempty"`
	PolicyID string `json:"policy_id,omitempty"`
}

// PolicyLocalSummary 描述本地 policy 规则摘要。
type PolicyLocalSummary struct {
	DenyOperations            []string          `json:"deny_operations,omitempty"`
	RequireApprovalOperations []string          `json:"require_approval_operations,omitempty"`
	AnnotateOperations        map[string]string `json:"annotate_operations,omitempty"`
	RuleCount                 int               `json:"rule_count"`
}

// PolicyRuntimeSummary 描述一个实际参与链路的 policy runtime。
type PolicyRuntimeSummary struct {
	Plugin    string   `json:"plugin,omitempty"`
	PolicyID  string   `json:"policy_id,omitempty"`
	Label     string   `json:"label,omitempty"`
	Platforms []string `json:"platforms,omitempty"`
	Priority  int      `json:"priority,omitempty"`
	Source    string   `json:"source,omitempty"`
}

// PolicyChainInspection 描述 policy 链的配置与实际生效结果。
type PolicyChainInspection struct {
	Mode              string                 `json:"mode"`
	ConfiguredPlugins []PolicySelectorView   `json:"configured_plugins,omitempty"`
	Local             PolicyLocalSummary     `json:"local"`
	ActiveChain       []PolicyRuntimeSummary `json:"active_chain,omitempty"`
	Warnings          []string               `json:"warnings,omitempty"`
}

type selectedPolicyRuntime struct {
	Runtime pluginruntime.PolicyRuntime
	Source  string
}

// evaluatePolicies runs local policy rules first and then external policy plugins.
func (e *Executor) evaluatePolicies(ctx context.Context, cfg *config.Config, definition adapter.Definition, requestID string, operation string, input map[string]any, profile ExecutionProfile, dryRun bool) ([]string, *apperr.AppError) {
	warnings, appErr := evaluateLocalPolicy(cfg.Runtime.Policy, operation)
	if appErr != nil {
		return warnings, appErr
	}

	pluginWarnings, appErr := evaluatePluginPolicies(ctx, cfg, definition, requestID, operation, input, profile, dryRun)
	warnings = append(warnings, pluginWarnings...)
	if appErr != nil {
		return warnings, appErr
	}
	return warnings, nil
}

func evaluateLocalPolicy(policy config.PolicyConfig, operation string) ([]string, *apperr.AppError) {
	if matched, ok := firstMatchingOperationPattern(policy.DenyOperations, operation); ok {
		return nil, apperr.New("POLICY_DENIED", fmt.Sprintf("local policy denied %s (matched rule %s)", operation, matched))
	}
	if matched, ok := firstMatchingOperationPattern(policy.RequireApprovalOperations, operation); ok {
		return nil, apperr.New("POLICY_APPROVAL_REQUIRED", fmt.Sprintf("local policy requires manual approval before executing %s (matched rule %s)", operation, matched))
	}

	if len(policy.AnnotateOperations) == 0 {
		return nil, nil
	}

	patterns := make([]string, 0, len(policy.AnnotateOperations))
	for pattern := range policy.AnnotateOperations {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		patterns = append(patterns, pattern)
	}
	sort.Strings(patterns)

	warnings := make([]string, 0)
	for _, pattern := range patterns {
		if !matchesOperationPattern(pattern, operation) {
			continue
		}
		message := strings.TrimSpace(policy.AnnotateOperations[pattern])
		if message == "" {
			message = fmt.Sprintf("local policy annotated %s (matched rule %s)", operation, pattern)
		}
		warnings = append(warnings, message)
	}
	return warnings, nil
}

func evaluatePluginPolicies(ctx context.Context, cfg *config.Config, definition adapter.Definition, requestID string, operation string, input map[string]any, profile ExecutionProfile, dryRun bool) ([]string, *apperr.AppError) {
	selections, selectionWarnings, err := resolveSelectedPolicyRuntimes(cfg)
	if err != nil {
		return nil, apperr.New("POLICY_EVALUATION_FAILED", err.Error())
	}
	runtimes := policySelectionsToRuntimes(selections)
	defer closePolicyRuntimes(runtimes)

	warnings := append([]string(nil), selectionWarnings...)
	for _, runtime := range runtimes {
		if !policyRuntimeSupportsPlatform(runtime, profile.Platform) {
			continue
		}

		result, err := runtime.Evaluate(ctx, pluginruntime.PolicyEvaluateParams{
			PolicyID: runtime.ID(),
			Request: pluginruntime.PolicyEvaluationRequest{
				RequestID: requestID,
				Operation: operation,
				DryRun:    dryRun,
				Mutating:  definition.Mutating,
				Input:     cloneAnyMap(input),
				Context: pluginruntime.PolicyEvaluationContext{
					AccountName: profile.Account,
					Platform:    profile.Platform,
					Subject:     profile.Subject,
					AuthMethod:  profile.AuthMethod,
				},
			},
		})
		if err != nil {
			return warnings, apperr.New("POLICY_EVALUATION_FAILED", fmt.Sprintf("policy plugin %s failed: %s", policyRuntimeLabel(runtime), err.Error()))
		}

		switch normalizePolicyDecision(result.Decision) {
		case policyDecisionAllow:
			continue
		case policyDecisionAnnotate:
			warnings = append(warnings, buildPolicyAnnotationWarning(runtime, result))
		case policyDecisionDeny:
			return warnings, apperr.New("POLICY_DENIED", buildPolicyDecisionMessage(runtime, result, "denied the current request"))
		case policyDecisionRequireApproval:
			return warnings, apperr.New("POLICY_APPROVAL_REQUIRED", buildPolicyDecisionMessage(runtime, result, "requires manual approval before continuing"))
		default:
			return warnings, apperr.New("POLICY_EVALUATION_FAILED", fmt.Sprintf("policy plugin %s returned an unsupported decision: %s", policyRuntimeLabel(runtime), strings.TrimSpace(result.Decision)))
		}
	}
	return warnings, nil
}

// InspectPolicyChain 返回当前配置下会参与执行的 policy 链摘要。
func InspectPolicyChain(cfg *config.Config) PolicyChainInspection {
	if cfg == nil {
		cfg = config.New()
	}

	inspection := PolicyChainInspection{
		Mode:              config.ResolvePolicyMode(cfg),
		ConfiguredPlugins: buildPolicySelectorViews(config.ResolvePolicyPlugins(cfg)),
		Local:             summarizeLocalPolicy(cfg.Runtime.Policy),
	}

	selections, warnings, err := resolveSelectedPolicyRuntimes(cfg)
	if err != nil {
		inspection.Warnings = append(inspection.Warnings, "failed to discover policy plugins: "+err.Error())
		return inspection
	}

	inspection.ActiveChain = summarizePolicySelections(selections)
	inspection.Warnings = append(inspection.Warnings, warnings...)
	return inspection
}

func firstMatchingOperationPattern(patterns []string, operation string) (string, bool) {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if matchesOperationPattern(pattern, operation) {
			return pattern, true
		}
	}
	return "", false
}

func matchesOperationPattern(pattern string, operation string) bool {
	pattern = strings.TrimSpace(pattern)
	operation = strings.TrimSpace(operation)
	if pattern == "" || operation == "" {
		return false
	}
	if pattern == operation {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return prefix != "" && strings.HasPrefix(operation, prefix+".")
	}
	return false
}

func policyRuntimeSupportsPlatform(runtime pluginruntime.PolicyRuntime, platform string) bool {
	if runtime == nil {
		return false
	}
	platform = strings.TrimSpace(platform)
	platforms := runtime.Platforms()
	if len(platforms) == 0 || platform == "" {
		return true
	}
	for _, item := range platforms {
		if strings.TrimSpace(item) == platform {
			return true
		}
	}
	return false
}

func closePolicyRuntimes(runtimes []pluginruntime.PolicyRuntime) {
	for _, runtime := range runtimes {
		if runtime == nil {
			continue
		}
		_ = runtime.Close()
	}
}

func resolveSelectedPolicyRuntimes(cfg *config.Config) ([]selectedPolicyRuntime, []string, error) {
	if cfg == nil {
		cfg = config.New()
	}

	mode := config.ResolvePolicyMode(cfg)
	if mode == config.RuntimeSelectionModeDisabled {
		return nil, nil, nil
	}

	runtimes, err := pluginruntime.DiscoverPolicyRuntimes(pluginruntime.DiscoveryOptions{
		EnabledPlugins: config.ResolveEnabledPlugins(cfg),
	})
	if err != nil {
		return nil, nil, err
	}

	selectors := config.ResolvePolicyPlugins(cfg)
	if len(selectors) == 0 {
		if mode == config.RuntimeSelectionModeManual {
			return nil, nil, nil
		}
		items := make([]selectedPolicyRuntime, 0, len(runtimes))
		for _, runtime := range runtimes {
			items = append(items, selectedPolicyRuntime{
				Runtime: runtime,
				Source:  "auto",
			})
		}
		return items, nil, nil
	}

	selected := make([]selectedPolicyRuntime, 0, len(selectors))
	warnings := make([]string, 0)
	seen := make(map[string]struct{}, len(runtimes))
	for _, selector := range selectors {
		matches := matchPolicyRuntimes(runtimes, selector)
		if len(matches) == 0 {
			warnings = append(warnings, fmt.Sprintf("configured policy selector %s did not match any discovered policy capability", policySelectorLabel(selector)))
			continue
		}
		for _, index := range matches {
			runtime := runtimes[index]
			key := policyRuntimeKey(runtime)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			selected = append(selected, selectedPolicyRuntime{
				Runtime: runtime,
				Source:  "configured",
			})
		}
	}
	return selected, warnings, nil
}

func matchPolicyRuntimes(runtimes []pluginruntime.PolicyRuntime, selector config.PolicyPluginBinding) []int {
	matches := make([]int, 0)
	for index, runtime := range runtimes {
		if !policyRuntimeMatchesSelector(runtime, selector) {
			continue
		}
		matches = append(matches, index)
	}
	return matches
}

func policyRuntimeMatchesSelector(runtime pluginruntime.PolicyRuntime, selector config.PolicyPluginBinding) bool {
	if runtime == nil {
		return false
	}

	pluginName := strings.TrimSpace(runtime.Name())
	policyID := strings.TrimSpace(runtime.ID())
	if selector.Plugin != "" && selector.Plugin != pluginName {
		return false
	}
	if selector.PolicyID != "" && selector.PolicyID != policyID {
		return false
	}
	return selector.Plugin != "" || selector.PolicyID != ""
}

func policySelectionsToRuntimes(items []selectedPolicyRuntime) []pluginruntime.PolicyRuntime {
	if len(items) == 0 {
		return nil
	}
	runtimes := make([]pluginruntime.PolicyRuntime, 0, len(items))
	for _, item := range items {
		if item.Runtime == nil {
			continue
		}
		runtimes = append(runtimes, item.Runtime)
	}
	return runtimes
}

func summarizeLocalPolicy(policy config.PolicyConfig) PolicyLocalSummary {
	ruleCount := 0
	if len(policy.DenyOperations) > 0 {
		ruleCount += len(policy.DenyOperations)
	}
	if len(policy.RequireApprovalOperations) > 0 {
		ruleCount += len(policy.RequireApprovalOperations)
	}
	if len(policy.AnnotateOperations) > 0 {
		ruleCount += len(policy.AnnotateOperations)
	}

	annotateOperations := make(map[string]string, len(policy.AnnotateOperations))
	for pattern, message := range policy.AnnotateOperations {
		pattern = strings.TrimSpace(pattern)
		message = strings.TrimSpace(message)
		if pattern == "" {
			continue
		}
		annotateOperations[pattern] = message
	}
	if len(annotateOperations) == 0 {
		annotateOperations = nil
	}

	return PolicyLocalSummary{
		DenyOperations:            append([]string(nil), policy.DenyOperations...),
		RequireApprovalOperations: append([]string(nil), policy.RequireApprovalOperations...),
		AnnotateOperations:        annotateOperations,
		RuleCount:                 ruleCount,
	}
}

func buildPolicySelectorViews(selectors []config.PolicyPluginBinding) []PolicySelectorView {
	if len(selectors) == 0 {
		return nil
	}
	items := make([]PolicySelectorView, 0, len(selectors))
	for _, selector := range selectors {
		items = append(items, PolicySelectorView{
			Plugin:   selector.Plugin,
			PolicyID: selector.PolicyID,
		})
	}
	return items
}

func summarizePolicySelections(selections []selectedPolicyRuntime) []PolicyRuntimeSummary {
	if len(selections) == 0 {
		return nil
	}
	items := make([]PolicyRuntimeSummary, 0, len(selections))
	for _, selection := range selections {
		if selection.Runtime == nil {
			continue
		}
		items = append(items, PolicyRuntimeSummary{
			Plugin:    strings.TrimSpace(selection.Runtime.Name()),
			PolicyID:  strings.TrimSpace(selection.Runtime.ID()),
			Label:     policyRuntimeLabel(selection.Runtime),
			Platforms: append([]string(nil), selection.Runtime.Platforms()...),
			Priority:  selection.Runtime.Priority(),
			Source:    selection.Source,
		})
	}
	return items
}

func policySelectorLabel(selector config.PolicyPluginBinding) string {
	switch {
	case selector.Plugin != "" && selector.PolicyID != "":
		return selector.Plugin + "/" + selector.PolicyID
	case selector.Plugin != "":
		return selector.Plugin
	default:
		return selector.PolicyID
	}
}

func policyRuntimeKey(runtime pluginruntime.PolicyRuntime) string {
	if runtime == nil {
		return ""
	}
	return strings.TrimSpace(runtime.Name()) + "|" + strings.TrimSpace(runtime.ID())
}

func normalizePolicyDecision(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return policyDecisionAllow
	}
	return value
}

func policyRuntimeLabel(runtime pluginruntime.PolicyRuntime) string {
	if runtime == nil {
		return ""
	}
	id := strings.TrimSpace(runtime.ID())
	name := strings.TrimSpace(runtime.Name())
	switch {
	case id != "" && name != "" && id != name:
		return name + "/" + id
	case id != "":
		return id
	default:
		return name
	}
}

func buildPolicyDecisionMessage(runtime pluginruntime.PolicyRuntime, result pluginruntime.PolicyEvaluateResult, fallback string) string {
	message := strings.TrimSpace(result.Message)
	if message == "" {
		message = fallback
	}
	return fmt.Sprintf("policy plugin %s%s", policyRuntimeLabel(runtime), messageWithColon(message))
}

func buildPolicyAnnotationWarning(runtime pluginruntime.PolicyRuntime, result pluginruntime.PolicyEvaluateResult) string {
	message := strings.TrimSpace(result.Message)
	if message == "" && len(result.Annotations) > 0 {
		encoded, err := json.Marshal(result.Annotations)
		if err == nil {
			message = string(encoded)
		}
	}
	if message == "" {
		message = "added an execution annotation"
	}
	return fmt.Sprintf("policy plugin %s%s", policyRuntimeLabel(runtime), messageWithColon(message))
}

func messageWithColon(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	return ": " + message
}
