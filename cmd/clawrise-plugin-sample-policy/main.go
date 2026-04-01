package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
)

const (
	samplePolicyPluginName    = "sample-policy"
	samplePolicyPluginVersion = "0.1.0"
	samplePolicyID            = "require_reason_for_mutations"
)

// samplePolicyRuntime demonstrates the smallest useful policy plugin surface.
// It keeps the decision logic intentionally simple so plugin authors can focus
// on the protocol shape first.
type samplePolicyRuntime struct{}

func (r *samplePolicyRuntime) Name() string {
	return samplePolicyPluginName
}

func (r *samplePolicyRuntime) ID() string {
	return samplePolicyID
}

func (r *samplePolicyRuntime) Priority() int {
	return 80
}

func (r *samplePolicyRuntime) Platforms() []string {
	// An empty platform list keeps the sample portable across providers.
	return nil
}

func (r *samplePolicyRuntime) Handshake(ctx context.Context) (pluginruntime.HandshakeResult, error) {
	_ = ctx

	return pluginruntime.HandshakeResult{
		ProtocolVersion: pluginruntime.ProtocolVersion,
		Name:            samplePolicyPluginName,
		Version:         samplePolicyPluginVersion,
	}, nil
}

func (r *samplePolicyRuntime) Evaluate(ctx context.Context, params pluginruntime.PolicyEvaluateParams) (pluginruntime.PolicyEvaluateResult, error) {
	_ = ctx

	request := params.Request
	if !request.Mutating || request.DryRun {
		return pluginruntime.PolicyEvaluateResult{
			Decision: "allow",
		}, nil
	}

	changeReason := stringInput(request.Input, "change_reason")
	if changeReason == "" {
		return pluginruntime.PolicyEvaluateResult{
			Decision: "require_approval",
			Message:  "mutating requests should include input.change_reason before they can run without manual review",
		}, nil
	}

	return pluginruntime.PolicyEvaluateResult{
		Decision: "annotate",
		Message:  "recorded change_reason for the mutating request",
		Annotations: map[string]any{
			"change_reason": changeReason,
			"policy_id":     samplePolicyID,
			"review_state":  "reason_recorded",
		},
	}, nil
}

func (r *samplePolicyRuntime) Close() error {
	return nil
}

func stringInput(input map[string]any, key string) string {
	if input == nil {
		return ""
	}

	value, ok := input[key]
	if !ok {
		return ""
	}

	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func main() {
	if err := pluginruntime.ServePolicyRuntime(&samplePolicyRuntime{}); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
