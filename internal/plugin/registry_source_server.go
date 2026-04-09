package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// ServeRegistrySourceRuntime serves the registry source plugin protocol over stdio.
func ServeRegistrySourceRuntime(runtime RegistrySourceRuntime) error {
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

		response := handleRegistrySourceRPCRequest(runtime, request)
		if err := writeRPCResponse(writer, response); err != nil {
			return err
		}
	}

	if err := reader.Err(); err != nil {
		return fmt.Errorf("plugin stdio reader failed: %w", err)
	}
	return nil
}

func handleRegistrySourceRPCRequest(runtime RegistrySourceRuntime, request RPCRequest) RPCResponse {
	switch request.Method {
	case "clawrise.handshake":
		return callRPC(request, func(ctx context.Context) (any, error) {
			return runtime.Handshake(ctx)
		})
	case "clawrise.capabilities.list":
		return callRPC(request, func(ctx context.Context) (any, error) {
			return CapabilityListResult{
				Capabilities: []CapabilityDescriptor{{
					Type:     CapabilityTypeRegistrySource,
					ID:       runtime.ID(),
					Priority: runtime.Priority(),
				}},
			}, nil
		})
	case "clawrise.registry_source.list":
		return callRPC(request, func(ctx context.Context) (any, error) {
			params, err := decodeRPCParams[RegistrySourceListParams](request.Params)
			if err != nil {
				return nil, err
			}
			return runtime.List(ctx, params)
		})
	case "clawrise.registry_source.resolve":
		return callRPC(request, func(ctx context.Context) (any, error) {
			params, err := decodeRPCParams[RegistrySourceResolveParams](request.Params)
			if err != nil {
				return nil, err
			}
			return runtime.Resolve(ctx, params)
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
