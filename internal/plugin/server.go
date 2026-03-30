package plugin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/clawrise/clawrise-cli/internal/config"
)

// ServeRuntime serves the plugin protocol over stdio for one runtime.
func ServeRuntime(runtime Runtime) error {
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

		response := handleRPCRequest(runtime, request)
		if err := writeRPCResponse(writer, response); err != nil {
			return err
		}
	}

	if err := reader.Err(); err != nil {
		return fmt.Errorf("plugin stdio reader failed: %w", err)
	}
	return nil
}

func handleRPCRequest(runtime Runtime, request RPCRequest) RPCResponse {
	switch request.Method {
	case "clawrise.handshake":
		return callRPC(request, func(ctx context.Context) (any, error) {
			return runtime.Handshake(ctx)
		})
	case "clawrise.operations.list":
		return callRPC(request, func(ctx context.Context) (any, error) {
			definitions, err := runtime.ListOperations(ctx)
			if err != nil {
				return nil, err
			}
			operations := make([]OperationDescriptor, 0, len(definitions))
			for _, definition := range definitions {
				operations = append(operations, OperationDescriptor{
					Operation:        definition.Operation,
					Platform:         definition.Platform,
					Mutating:         definition.Mutating,
					DefaultTimeoutMS: definition.DefaultTimeout.Milliseconds(),
					AllowedSubjects:  append([]string(nil), definition.AllowedSubjects...),
					Spec:             definition.Spec,
				})
			}
			return OperationsListResult{Operations: operations}, nil
		})
	case "clawrise.catalog.get":
		return callRPC(request, func(ctx context.Context) (any, error) {
			entries, err := runtime.GetCatalog(ctx)
			if err != nil {
				return nil, err
			}
			return CatalogResult{Entries: entries}, nil
		})
	case "clawrise.auth.methods.list":
		return callRPC(request, func(ctx context.Context) (any, error) {
			methods, err := runtime.ListAuthMethods(ctx)
			if err != nil {
				return nil, err
			}
			return AuthMethodsListResult{Methods: methods}, nil
		})
	case "clawrise.auth.presets.list":
		return callRPC(request, func(ctx context.Context) (any, error) {
			presets, err := runtime.ListAuthPresets(ctx)
			if err != nil {
				return nil, err
			}
			return AuthPresetsListResult{Presets: presets}, nil
		})
	case "clawrise.auth.inspect":
		return callRPC(request, func(ctx context.Context) (any, error) {
			params, err := decodeRPCParams[AuthInspectParams](request.Params)
			if err != nil {
				return nil, err
			}
			return runtime.InspectAuth(ctx, params)
		})
	case "clawrise.auth.begin":
		return callRPC(request, func(ctx context.Context) (any, error) {
			params, err := decodeRPCParams[AuthBeginParams](request.Params)
			if err != nil {
				return nil, err
			}
			return runtime.BeginAuth(ctx, params)
		})
	case "clawrise.auth.complete":
		return callRPC(request, func(ctx context.Context) (any, error) {
			params, err := decodeRPCParams[AuthCompleteParams](request.Params)
			if err != nil {
				return nil, err
			}
			return runtime.CompleteAuth(ctx, params)
		})
	case "clawrise.auth.resolve":
		return callRPC(request, func(ctx context.Context) (any, error) {
			params, err := decodeRPCParams[AuthResolveParams](request.Params)
			if err != nil {
				return nil, err
			}
			return runtime.ResolveAuth(ctx, params)
		})
	case "clawrise.execute":
		return callRPC(request, func(ctx context.Context) (any, error) {
			params, err := decodeRPCParams[ExecuteParams](request.Params)
			if err != nil {
				return nil, err
			}
			timeout := time.Duration(params.Request.TimeoutMS) * time.Millisecond
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			result, err := runtime.Execute(ctx, ExecuteRequest{
				Operation:      params.Request.Operation,
				AccountName:    params.Identity.AccountName,
				Profile:        buildProfileFromIdentity(params.Identity),
				IdentityAuth:   params.Identity.Auth,
				Input:          params.Request.Input,
				IdempotencyKey: params.Request.IdempotencyKey,
			})
			if err != nil {
				return nil, err
			}

			return ExecuteRPCResult{
				OK:    result.Error == nil,
				Data:  result.Data,
				Error: result.Error,
				Meta: map[string]any{
					"retry_count": 0,
				},
			}, nil
		})
	case "clawrise.health":
		return callRPC(request, func(ctx context.Context) (any, error) {
			return runtime.Health(ctx)
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

func callRPC(request RPCRequest, fn func(ctx context.Context) (any, error)) RPCResponse {
	result, err := fn(context.Background())
	if err != nil {
		return RPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &RPCError{
				Code:    -32000,
				Message: err.Error(),
			},
		}
	}
	return RPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  result,
	}
}

func decodeRPCParams[T any](raw any) (T, error) {
	var target T
	if raw == nil {
		return target, nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return target, fmt.Errorf("failed to encode rpc params: %w", err)
	}
	if err := json.Unmarshal(data, &target); err != nil {
		return target, fmt.Errorf("failed to decode rpc params: %w", err)
	}
	return target, nil
}

func writeRPCResponse(writer io.Writer, response RPCResponse) error {
	encoded, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to encode rpc response: %w", err)
	}
	if _, err := writer.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("failed to write rpc response: %w", err)
	}
	if flusher, ok := writer.(*bufio.Writer); ok {
		return flusher.Flush()
	}
	return nil
}

func buildProfileFromIdentity(identity ExecuteIdentity) config.Profile {
	authType := asStringOrEmpty(identity.Auth["type"])
	if authType == "resolved_access_token" {
		return config.Profile{
			Platform: identity.Platform,
			Subject:  identity.Subject,
			Grant: config.Grant{
				Type:        authType,
				AccessToken: asStringOrEmpty(identity.Auth["access_token"]),
				NotionVer:   asStringOrEmpty(identity.Auth["notion_version"]),
			},
		}
	}

	profile := config.Profile{
		Platform: identity.Platform,
		Subject:  identity.Subject,
		Grant: config.Grant{
			Type: authType,
		},
	}
	profile.Grant.AppID = asStringOrEmpty(identity.Auth["app_id"])
	profile.Grant.AppSecret = asStringOrEmpty(identity.Auth["app_secret"])
	profile.Grant.Token = asStringOrEmpty(identity.Auth["token"])
	profile.Grant.ClientID = asStringOrEmpty(identity.Auth["client_id"])
	profile.Grant.ClientSecret = asStringOrEmpty(identity.Auth["client_secret"])
	profile.Grant.AccessToken = asStringOrEmpty(identity.Auth["access_token"])
	profile.Grant.RefreshToken = asStringOrEmpty(identity.Auth["refresh_token"])
	profile.Grant.NotionVer = asStringOrEmpty(identity.Auth["notion_version"])
	return profile
}

func asStringOrEmpty(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}
