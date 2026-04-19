package runtime

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadInputCoversInlineFileAndStdinPaths(t *testing.T) {
	t.Run("rejects json and input together", func(t *testing.T) {
		_, err := ReadInput(`{"a":1}`, "input.json", nil)
		if err == nil || !strings.Contains(err.Error(), "cannot use --json and --input") {
			t.Fatalf("expected mutual exclusion error, got: %v", err)
		}
	})

	t.Run("reads inline json", func(t *testing.T) {
		input, err := ReadInput(`{"answer":42}`, "", nil)
		if err != nil {
			t.Fatalf("ReadInput returned error: %v", err)
		}
		if input["answer"].(json.Number).String() != "42" {
			t.Fatalf("expected json number to round-trip, got: %+v", input)
		}
	})

	t.Run("reads file with at-prefix", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "input.json")
		if err := os.WriteFile(path, []byte(`{"source":"file"}`), 0o600); err != nil {
			t.Fatalf("failed to write input file: %v", err)
		}
		input, err := ReadInput("", "@"+path, nil)
		if err != nil {
			t.Fatalf("ReadInput returned error: %v", err)
		}
		if input["source"] != "file" {
			t.Fatalf("unexpected file input payload: %+v", input)
		}
	})

	t.Run("returns empty map for empty stdin", func(t *testing.T) {
		input, err := ReadInput("", "", strings.NewReader(" \n\t "))
		if err != nil {
			t.Fatalf("ReadInput returned error: %v", err)
		}
		if len(input) != 0 {
			t.Fatalf("expected empty stdin to produce empty map, got: %+v", input)
		}
	})

	t.Run("reads stdin json", func(t *testing.T) {
		input, err := ReadInput("", "", strings.NewReader(`{"source":"stdin"}`))
		if err != nil {
			t.Fatalf("ReadInput returned error: %v", err)
		}
		if input["source"] != "stdin" {
			t.Fatalf("unexpected stdin payload: %+v", input)
		}
	})

	t.Run("surfaces read stdin failure", func(t *testing.T) {
		_, err := ReadInput("", "", io.NopCloser(errReader{err: errors.New("stdin broken")}))
		if err == nil || !strings.Contains(err.Error(), "failed to read stdin") {
			t.Fatalf("expected stdin read failure, got: %v", err)
		}
	})

	t.Run("surfaces decode failure", func(t *testing.T) {
		_, err := ReadInput("{", "", nil)
		if err == nil || !strings.Contains(err.Error(), "failed to decode JSON input") {
			t.Fatalf("expected json decode failure, got: %v", err)
		}
		if !strings.Contains(err.Error(), "prefer --input <file> for automation") {
			t.Fatalf("expected inline json guidance in decode failure, got: %v", err)
		}
	})

	t.Run("keeps file decode failures source-specific without shell quoting hint", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "broken.json")
		if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
			t.Fatalf("failed to write broken input file: %v", err)
		}
		_, err := ReadInput("", path, nil)
		if err == nil || !strings.Contains(err.Error(), "failed to decode JSON input from --input") {
			t.Fatalf("expected file decode failure, got: %v", err)
		}
		if strings.Contains(err.Error(), "prefer --input <file> for automation") {
			t.Fatalf("did not expect shell quoting hint for file input, got: %v", err)
		}
	})

	t.Run("returns empty map for null json", func(t *testing.T) {
		input, err := ReadInput("null", "", nil)
		if err != nil {
			t.Fatalf("ReadInput returned error: %v", err)
		}
		if len(input) != 0 {
			t.Fatalf("expected null json to produce empty map, got: %+v", input)
		}
	})

	t.Run("returns empty map when no source is provided", func(t *testing.T) {
		input, err := ReadInput("", "", nil)
		if err != nil {
			t.Fatalf("ReadInput returned error: %v", err)
		}
		if len(input) != 0 {
			t.Fatalf("expected empty default input map, got: %+v", input)
		}
	})
}
