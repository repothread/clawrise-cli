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
		return decodeJSON(strings.NewReader(jsonText), "--json")
	case strings.TrimSpace(inputPath) != "":
		path := strings.TrimSpace(strings.TrimPrefix(inputPath, "@"))
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read input file: %w", err)
		}
		return decodeJSON(strings.NewReader(string(data)), "--input")
	case stdin != nil:
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read stdin: %w", err)
		}
		if strings.TrimSpace(string(data)) == "" {
			return map[string]any{}, nil
		}
		return decodeJSON(strings.NewReader(string(data)), "stdin")
	default:
		return map[string]any{}, nil
	}
}

func decodeJSON(reader io.Reader, source string) (map[string]any, error) {
	var input map[string]any
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()
	if err := decoder.Decode(&input); err != nil {
		message := fmt.Sprintf("failed to decode JSON input from %s: %v", source, err)
		if source == "--json" {
			message += "; check shell quoting/escaping or prefer --input <file> for automation"
		}
		return nil, fmt.Errorf("%s", message)
	}
	if input == nil {
		return map[string]any{}, nil
	}
	return input, nil
}
