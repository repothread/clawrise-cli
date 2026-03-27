package main

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/clawrise/clawrise-cli/internal/adapter"
	pluginruntime "github.com/clawrise/clawrise-cli/internal/plugin"
	speccatalog "github.com/clawrise/clawrise-cli/internal/spec/catalog"
)

func main() {
	reader := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	for reader.Scan() {
		var request pluginruntime.RPCRequest
		if err := json.Unmarshal(reader.Bytes(), &request); err != nil {
			writeResponse(writer, pluginruntime.RPCResponse{
				JSONRPC: "2.0",
				ID:      "",
				Error: &pluginruntime.RPCError{
					Code:    -32700,
					Message: "failed to decode request",
				},
			})
			continue
		}

		switch request.Method {
		case "clawrise.handshake":
			writeResponse(writer, pluginruntime.RPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Result: pluginruntime.HandshakeResult{
					ProtocolVersion: pluginruntime.ProtocolVersion,
					Name:            "demo",
					Version:         "0.1.0",
					Platforms:       []string{"demo"},
				},
			})
		case "clawrise.operations.list":
			writeResponse(writer, pluginruntime.RPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Result: pluginruntime.OperationsListResult{
					Operations: []pluginruntime.OperationDescriptor{
						{
							Operation:        "demo.page.echo",
							Platform:         "demo",
							Mutating:         false,
							DefaultTimeoutMS: 1000,
							AllowedSubjects:  []string{"integration"},
							Spec: adapter.OperationSpec{
								Summary:         "Echo one demo page.",
								DryRunSupported: true,
								Input: adapter.InputSpec{
									Required: []string{"message"},
									Sample: map[string]any{
										"message": "hello",
									},
								},
							},
						},
					},
				},
			})
		case "clawrise.catalog.get":
			writeResponse(writer, pluginruntime.RPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Result: pluginruntime.CatalogResult{
					Entries: []speccatalog.Entry{
						{Operation: "demo.page.echo"},
					},
				},
			})
		case "clawrise.execute":
			writeResponse(writer, pluginruntime.RPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Result: pluginruntime.ExecuteRPCResult{
					OK: true,
					Data: map[string]any{
						"message": "echoed",
					},
					Meta: map[string]any{
						"retry_count": 0,
					},
				},
			})
		case "clawrise.health":
			writeResponse(writer, pluginruntime.RPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Result: pluginruntime.HealthResult{
					OK: true,
					Details: map[string]any{
						"plugin_name": "demo",
					},
				},
			})
		default:
			writeResponse(writer, pluginruntime.RPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Error: &pluginruntime.RPCError{
					Code:    -32601,
					Message: "method not found",
				},
			})
		}
	}
}

func writeResponse(writer *bufio.Writer, response pluginruntime.RPCResponse) {
	encoded, err := json.Marshal(response)
	if err != nil {
		return
	}
	_, _ = writer.Write(append(encoded, '\n'))
	_ = writer.Flush()
}
