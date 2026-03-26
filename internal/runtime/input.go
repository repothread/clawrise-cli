package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// ReadInput loads JSON input from inline text, a file, or stdin.
func ReadInput(jsonText, inputPath string, stdin io.Reader) (map[string]any, error) {
	switch {
	case strings.TrimSpace(jsonText) != "" && strings.TrimSpace(inputPath) != "":
		return nil, fmt.Errorf("cannot use --json and --input at the same time")
	case strings.TrimSpace(jsonText) != "":
		return decodeJSON(strings.NewReader(jsonText))
	case strings.TrimSpace(inputPath) != "":
		path := strings.TrimSpace(strings.TrimPrefix(inputPath, "@"))
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read input file: %w", err)
		}
		return decodeJSON(strings.NewReader(string(data)))
	case stdin != nil:
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read stdin: %w", err)
		}
		if strings.TrimSpace(string(data)) == "" {
			return map[string]any{}, nil
		}
		return decodeJSON(strings.NewReader(string(data)))
	default:
		return map[string]any{}, nil
	}
}

func decodeJSON(reader io.Reader) (map[string]any, error) {
	var input map[string]any
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	if err := decoder.Decode(&input); err != nil {
		return nil, fmt.Errorf("failed to decode JSON input: %w", err)
	}
	if input == nil {
		return map[string]any{}, nil
	}
	return input, nil
}
