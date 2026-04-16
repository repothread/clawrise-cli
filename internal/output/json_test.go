package output

import (
	"bytes"
	"strings"
	"testing"
)

type samplePayload struct {
	HTML string `json:"html"`
	Name string `json:"name"`
}

func TestWriteJSONProducesIndentedUnescapedOutput(t *testing.T) {
	var buffer bytes.Buffer
	err := WriteJSON(&buffer, samplePayload{
		HTML: "<b>hi</b>",
		Name: "clawrise",
	})
	if err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	output := buffer.String()
	if !strings.Contains(output, "\n  \"html\": \"<b>hi</b>\",") {
		t.Fatalf("expected indented json with unescaped html, got %q", output)
	}
	if !strings.HasSuffix(output, "\n") {
		t.Fatalf("expected json output to end with newline, got %q", output)
	}
}
