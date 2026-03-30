package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// ServeAuthLauncherRuntime 通过 stdio 暴露 auth launcher plugin 协议。
func ServeAuthLauncherRuntime(runtime AuthLauncherRuntime) error {
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

		response := handleAuthLauncherRPCRequest(runtime, request)
		if err := writeRPCResponse(writer, response); err != nil {
			return err
		}
	}

	if err := reader.Err(); err != nil {
		return fmt.Errorf("plugin stdio reader failed: %w", err)
	}
	return nil
}

func handleAuthLauncherRPCRequest(runtime AuthLauncherRuntime, request RPCRequest) RPCResponse {
	switch request.Method {
	case "clawrise.handshake":
		return callRPC(request, func(ctx context.Context) (any, error) {
			return runtime.Handshake(ctx)
		})
	case "clawrise.auth.launcher.describe":
		return callRPC(request, func(ctx context.Context) (any, error) {
			descriptor, err := runtime.DescribeAuthLauncher(ctx)
			if err != nil {
				return nil, err
			}
			return AuthLauncherDescribeResult{Launcher: descriptor}, nil
		})
	case "clawrise.auth.launcher.run":
		return callRPC(request, func(ctx context.Context) (any, error) {
			params, err := decodeRPCParams[AuthLaunchParams](request.Params)
			if err != nil {
				return nil, err
			}
			return runtime.LaunchAuth(ctx, params)
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
