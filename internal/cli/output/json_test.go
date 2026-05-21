package output

import (
	"bytes"
	"testing"
)

func TestRenderJSON(t *testing.T) {
	var b bytes.Buffer
	err := RenderJSON(&b, map[string]int{"wpm": 100})
	if err != nil {
		t.Fatalf("RenderJSON: %v", err)
	}
	if got := b.String(); got != "{\n  \"wpm\": 100\n}\n" {
		t.Fatalf("unexpected JSON: %q", got)
	}
}
