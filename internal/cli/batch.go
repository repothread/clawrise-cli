package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"github.com/clawrise/clawrise-cli/internal/output"
	"github.com/clawrise/clawrise-cli/internal/runtime"
)

type batchRequestEnvelope struct {
	Requests []batchRequest `json:"requests"`
}

type batchRequest struct {
	Operation            string         `json:"operation"`
	Account              string         `json:"account,omitempty"`
	Subject              string         `json:"subject,omitempty"`
	Input                map[string]any `json:"input,omitempty"`
	Timeout              string         `json:"timeout,omitempty"`
	DryRun               bool           `json:"dry_run,omitempty"`
	DebugProviderPayload bool           `json:"debug_provider_payload,omitempty"`
	VerifyAfterWrite     bool           `json:"verify_after_write,omitempty"`
	IdempotencyKey       string         `json:"idempotency_key,omitempty"`
	Quiet                bool           `json:"quiet,omitempty"`
}

func runBatch(args []string, stdout io.Writer, stderr io.Writer, executor *runtime.Executor) error {
	flags := pflag.NewFlagSet("clawrise batch", pflag.ContinueOnError)
	flags.SetOutput(stderr)

	var inputJSON string
	var inputFile string
	flags.StringVar(&inputJSON, "json", "", "pass inline JSON batch input")
	flags.StringVar(&inputFile, "input", "", "read JSON batch input from a file")

	if err := flags.Parse(args); err != nil {
		if err == pflag.ErrHelp {
			return nil
		}
		return err
	}
	if len(flags.Args()) != 0 {
		return fmt.Errorf("usage: clawrise batch [--json <payload> | --input <path>]")
	}

	payload, err := loadBatchPayload(strings.TrimSpace(inputJSON), strings.TrimSpace(inputFile), readPipedInput())
	if err != nil {
		return err
	}
	if len(payload.Requests) == 0 {
		return fmt.Errorf("batch requests must not be empty")
	}

	results := make([]map[string]any, 0, len(payload.Requests))
	successCount := 0
	for index, request := range payload.Requests {
		timeout, err := parseBatchTimeout(request.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout for batch request %d: %w", index, err)
		}

		inputJSON, err := marshalBatchInput(request.Input)
		if err != nil {
			return fmt.Errorf("failed to encode batch request %d input: %w", index, err)
		}

		envelope, err := executor.ExecuteContext(context.Background(), runtime.ExecuteOptions{
			OperationInput:       request.Operation,
			AccountName:          strings.TrimSpace(request.Account),
			SubjectName:          strings.TrimSpace(request.Subject),
			InputJSON:            inputJSON,
			Timeout:              timeout,
			DryRun:               request.DryRun,
			DebugProviderPayload: request.DebugProviderPayload,
			VerifyAfterWrite:     request.VerifyAfterWrite,
			IdempotencyKey:       strings.TrimSpace(request.IdempotencyKey),
			Output:               "json",
			Quiet:                request.Quiet,
		})
		if err != nil {
			return err
		}
		if envelope.OK {
			successCount++
		}
		results = append(results, map[string]any{
			"index":     index,
			"operation": request.Operation,
			"ok":        envelope.OK,
			"result":    envelope,
		})
	}

	ok := successCount == len(payload.Requests)
	if err := output.WriteJSON(stdout, map[string]any{
		"ok": ok,
		"data": map[string]any{
			"count":         len(payload.Requests),
			"success_count": successCount,
			"failure_count": len(payload.Requests) - successCount,
			"results":       results,
		},
	}); err != nil {
		return err
	}
	if !ok {
		return ExitError{Code: 1}
	}
	return nil
}

func loadBatchPayload(inputJSON string, inputFile string, stdin io.Reader) (batchRequestEnvelope, error) {
	switch {
	case inputJSON != "":
		return decodeBatchPayload([]byte(inputJSON))
	case inputFile != "":
		data, err := os.ReadFile(inputFile)
		if err != nil {
			return batchRequestEnvelope{}, fmt.Errorf("failed to read batch input file: %w", err)
		}
		return decodeBatchPayload(data)
	case stdin != nil:
		data, err := io.ReadAll(stdin)
		if err != nil {
			return batchRequestEnvelope{}, fmt.Errorf("failed to read batch stdin: %w", err)
		}
		return decodeBatchPayload(data)
	default:
		return batchRequestEnvelope{}, fmt.Errorf("batch input is required via --json, --input, or stdin")
	}
}

func decodeBatchPayload(data []byte) (batchRequestEnvelope, error) {
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		return batchRequestEnvelope{}, fmt.Errorf("batch input must not be empty")
	}

	var requests []batchRequest
	if err := json.Unmarshal(data, &requests); err == nil {
		return batchRequestEnvelope{Requests: requests}, nil
	}

	var envelope batchRequestEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return batchRequestEnvelope{}, fmt.Errorf("failed to decode batch input: %w", err)
	}
	return envelope, nil
}

func parseBatchTimeout(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	return time.ParseDuration(raw)
}

func marshalBatchInput(input map[string]any) (string, error) {
	if input == nil {
		return "", nil
	}
	data, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
