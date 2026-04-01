package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// ServeWorkflowRuntime serves the workflow plugin protocol over stdio.
func ServeWorkflowRuntime(runtime WorkflowRuntime) error {
	reader := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	for reader.Scan() {
		var request RPCRequest
		if err := json.Unmarshal(reader.Bytes(), &request); err != nil {
			if writeErr := writeRPCResponse(writer, RPCResponse{
				JSONRPC: "2.0",
				ID:      "",
				Error: &RPCError{
					Code:    -32700,
					Message: "failed to decode request",
				},
			}); writeErr != nil {
				return writeErr
			}
			continue
		}

		response := handleWorkflowRPCRequest(runtime, request)
		if err := writeRPCResponse(writer, response); err != nil {
			return err
		}
	}

	if err := reader.Err(); err != nil {
		return fmt.Errorf("plugin stdio reader failed: %w", err)
	}
	return nil
}

func handleWorkflowRPCRequest(runtime WorkflowRuntime, request RPCRequest) RPCResponse {
	switch request.Method {
	case "clawrise.handshake":
		return callRPC(request, func(ctx context.Context) (any, error) {
			return runtime.Handshake(ctx)
		})
	case "clawrise.capabilities.list":
		return callRPC(request, func(ctx context.Context) (any, error) {
			return CapabilityListResult{
				Capabilities: []CapabilityDescriptor{{
					Type:     CapabilityTypeWorkflow,
					ID:       runtime.ID(),
					Priority: runtime.Priority(),
				}},
			}, nil
		})
	case "clawrise.workflow.plan":
		return callRPC(request, func(ctx context.Context) (any, error) {
			params, err := decodeRPCParams[WorkflowPlanParams](request.Params)
			if err != nil {
				return nil, err
			}
			return runtime.Plan(ctx, params)
		})
	default:
		return RPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: "method not found",
			},
		}
	}
}
