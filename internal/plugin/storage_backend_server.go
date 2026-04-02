package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// ServeSecretStoreRuntime 通过 stdio 暴露外部 secret store plugin 协议。
func ServeSecretStoreRuntime(runtime SecretStorePluginRuntime) error {
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

		response := handleSecretStoreRPCRequest(runtime, request)
		if err := writeRPCResponse(writer, response); err != nil {
			return err
		}
	}

	if err := reader.Err(); err != nil {
		return fmt.Errorf("plugin stdio reader failed: %w", err)
	}
	return nil
}

func handleSecretStoreRPCRequest(runtime SecretStorePluginRuntime, request RPCRequest) RPCResponse {
	switch request.Method {
	case "clawrise.handshake":
		return callRPC(request, func(ctx context.Context) (any, error) {
			return runtime.Handshake(ctx)
		})
	case "clawrise.capabilities.list":
		return callRPC(request, func(ctx context.Context) (any, error) {
			descriptor, err := runtime.DescribeStorageBackend(ctx)
			if err != nil {
				return nil, err
			}
			return CapabilityListResult{
				Capabilities: []CapabilityDescriptor{{
					Type:        CapabilityTypeStorageBackend,
					Target:      descriptor.Target,
					Backend:     descriptor.Backend,
					DisplayName: descriptor.DisplayName,
					Description: descriptor.Description,
				}},
			}, nil
		})
	case "clawrise.storage.backend.describe":
		return callRPC(request, func(ctx context.Context) (any, error) {
			descriptor, err := runtime.DescribeStorageBackend(ctx)
			if err != nil {
				return nil, err
			}
			return StorageBackendDescribeResult{Backend: descriptor}, nil
		})
	case "clawrise.storage.secret.status":
		return callRPC(request, func(ctx context.Context) (any, error) {
			status, err := runtime.Status(ctx)
			if err != nil {
				return nil, err
			}
			return SecretStoreStatusResult{Status: status}, nil
		})
	case "clawrise.storage.secret.get":
		return callRPC(request, func(ctx context.Context) (any, error) {
			params, err := decodeRPCParams[SecretStoreGetParams](request.Params)
			if err != nil {
				return nil, err
			}
			return runtime.Get(ctx, params)
		})
	case "clawrise.storage.secret.set":
		return callRPC(request, func(ctx context.Context) (any, error) {
			params, err := decodeRPCParams[SecretStoreSetParams](request.Params)
			if err != nil {
				return nil, err
			}
			return map[string]any{}, runtime.Set(ctx, params)
		})
	case "clawrise.storage.secret.delete":
		return callRPC(request, func(ctx context.Context) (any, error) {
			params, err := decodeRPCParams[SecretStoreDeleteParams](request.Params)
			if err != nil {
				return nil, err
			}
			return map[string]any{}, runtime.Delete(ctx, params)
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
