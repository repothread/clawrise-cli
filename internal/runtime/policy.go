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

// evaluatePolicies 依次执行本地策略规则与外部 policy plugin。
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
		return nil, apperr.New("POLICY_DENIED", fmt.Sprintf("本地策略拒绝执行 %s（命中规则 %s）", operation, matched))
	}
	if matched, ok := firstMatchingOperationPattern(policy.RequireApprovalOperations, operation); ok {
		return nil, apperr.New("POLICY_APPROVAL_REQUIRED", fmt.Sprintf("本地策略要求人工审批后才能执行 %s（命中规则 %s）", operation, matched))
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
			message = fmt.Sprintf("本地策略对 %s 添加了标注（命中规则 %s）", operation, pattern)
		}
		warnings = append(warnings, message)
	}
	return warnings, nil
}

func evaluatePluginPolicies(ctx context.Context, cfg *config.Config, definition adapter.Definition, requestID string, operation string, input map[string]any, profile ExecutionProfile, dryRun bool) ([]string, *apperr.AppError) {
	runtimes, err := pluginruntime.DiscoverPolicyRuntimes(pluginruntime.DiscoveryOptions{
		EnabledPlugins: config.ResolveEnabledPlugins(cfg),
	})
	if err != nil {
		return nil, apperr.New("POLICY_EVALUATION_FAILED", err.Error())
	}
	defer closePolicyRuntimes(runtimes)

	warnings := make([]string, 0)
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
			return warnings, apperr.New("POLICY_EVALUATION_FAILED", fmt.Sprintf("策略插件 %s 执行失败: %s", policyRuntimeLabel(runtime), err.Error()))
		}

		switch normalizePolicyDecision(result.Decision) {
		case policyDecisionAllow:
			continue
		case policyDecisionAnnotate:
			warnings = append(warnings, buildPolicyAnnotationWarning(runtime, result))
		case policyDecisionDeny:
			return warnings, apperr.New("POLICY_DENIED", buildPolicyDecisionMessage(runtime, result, "拒绝了当前请求"))
		case policyDecisionRequireApproval:
			return warnings, apperr.New("POLICY_APPROVAL_REQUIRED", buildPolicyDecisionMessage(runtime, result, "要求人工审批后再继续执行"))
		default:
			return warnings, apperr.New("POLICY_EVALUATION_FAILED", fmt.Sprintf("策略插件 %s 返回了不支持的 decision: %s", policyRuntimeLabel(runtime), strings.TrimSpace(result.Decision)))
		}
	}
	return warnings, nil
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
	return fmt.Sprintf("策略插件 %s%s", policyRuntimeLabel(runtime), messageWithChineseColon(message))
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
		message = "添加了执行标注"
	}
	return fmt.Sprintf("策略插件 %s%s", policyRuntimeLabel(runtime), messageWithChineseColon(message))
}

func messageWithChineseColon(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	return "：" + message
}
