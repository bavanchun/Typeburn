package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderTable(t *testing.T) {
	var b bytes.Buffer
	err := RenderTable(&b, []string{"when", "wpm"}, [][]string{{"now", "100"}})
	if err != nil {
		t.Fatalf("RenderTable: %v", err)
	}
	got := b.String()
	if !strings.Contains(got, "when") || !strings.Contains(got, "100") {
		t.Fatalf("unexpected table:\n%s", got)
	}
}
