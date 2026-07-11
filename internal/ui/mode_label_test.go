package ui

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/storage"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

func TestDisplayModeLabel(t *testing.T) {
	tests := []struct {
		name   string
		mode   string
		length int
		want   string
	}{
		{name: "time", mode: "time", length: 30, want: "time 30"},
		{name: "words", mode: "words", length: 50, want: "words 50"},
		{name: "quote ignores length", mode: "quote", length: 42, want: "quote"},
		{name: "code ignores length", mode: "code", length: 142, want: "code"},
		{name: "unknown keeps identifier", mode: "custom", length: 10, want: "custom"},
		{name: "empty is unknown", mode: "", length: 10, want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := displayModeLabel(tt.mode, tt.length); got != tt.want {
				t.Fatalf("displayModeLabel(%q, %d) = %q, want %q", tt.mode, tt.length, got, tt.want)
			}
		})
	}
}

func TestHistoryView_CodeAndStrictRunsDoNotShowBestMarker(t *testing.T) {
	tests := []struct {
		name string
		rec  storage.Record
	}{
		{name: "code", rec: storage.Record{Mode: "code", Length: 142, WPM: 120, NetWPM: 120}},
		{name: "strict", rec: storage.Record{Mode: "time", Length: 30, WPM: 120, NetWPM: 120, Strict: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewHistory([]storage.Record{tt.rec}, theme.Default(), config.DefaultKeymap()).SetSize(80, 24).View()
			if strings.Contains(view, "★") {
				t.Fatalf("ineligible %s run rendered a best marker:\n%s", tt.name, view)
			}
		})
	}
}

func TestHistoryView_CodeLabelIsNotQuote(t *testing.T) {
	rec := storage.Record{Mode: "code", Length: 142, WPM: 94, NetWPM: 94}
	view := NewHistory([]storage.Record{rec}, theme.Default(), config.DefaultKeymap()).SetSize(80, 24).View()
	if !strings.Contains(view, "code") {
		t.Fatalf("history did not render code label:\n%s", view)
	}
	if strings.Contains(view, "quote") {
		t.Fatalf("history mislabeled code as quote:\n%s", view)
	}
}

func TestHistoryView_EligibleTiesShowBestMarkers(t *testing.T) {
	rows := []storage.Record{
		{Mode: "time", Length: 30, WPM: 94, NetWPM: 94},
		{Mode: "time", Length: 30, WPM: 94, NetWPM: 94},
	}
	view := NewHistory(rows, theme.Default(), config.DefaultKeymap()).SetSize(80, 24).View()
	if got := strings.Count(view, "★"); got != 2 {
		t.Fatalf("eligible tied runs rendered %d best markers, want 2:\n%s", got, view)
	}
}

func TestResultView_CodeMetaIsNotQuote(t *testing.T) {
	msg := ResultMsg{Result: makeTestMetricsResult(), Mode: config.ModeCode, CodeText: "func main() {}"}
	view := NewResult(msg, theme.Default(), config.DefaultKeymap()).SetSize(80, 24).View()
	if !strings.Contains(view, "· code · english") {
		t.Fatalf("result did not render code metadata:\n%s", view)
	}
	if strings.Contains(view, "· quote · english") {
		t.Fatalf("result mislabeled code as quote:\n%s", view)
	}
}
