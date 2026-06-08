package app

import (
	"math"
	"testing"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/ui"
)

func TestBuildRecord_QuoteModeLength(t *testing.T) {
	msg := ui.ResultMsg{
		Mode:   config.ModeQuote,
		Length: 42, // should be overridden to 0
		Result: metrics.Result{NetWPM: 80, Accuracy: 95, DurationMs: 30000},
	}
	rec := buildRecord(msg)
	if rec.Length != 0 {
		t.Errorf("quote mode length = %d, want 0", rec.Length)
	}
}

func TestBuildRecord_CodeModeRuneCount(t *testing.T) {
	msg := ui.ResultMsg{
		Mode:     config.ModeCode,
		CodeText: "hello 世界",
		Result:   metrics.Result{NetWPM: 50, Accuracy: 90, DurationMs: 15000},
	}
	rec := buildRecord(msg)
	wantLen := len([]rune("hello 世界")) // 8
	if rec.Length != wantLen {
		t.Errorf("code mode length = %d, want %d", rec.Length, wantLen)
	}
}

func TestBuildRecord_TimeModePreservesLength(t *testing.T) {
	msg := ui.ResultMsg{
		Mode:   config.ModeTime,
		Length: 30,
		Result: metrics.Result{NetWPM: 80, Accuracy: 95, DurationMs: 30000},
	}
	rec := buildRecord(msg)
	if rec.Length != 30 {
		t.Errorf("time mode length = %d, want 30", rec.Length)
	}
	if rec.Mode != "time" {
		t.Errorf("mode = %q, want %q", rec.Mode, "time")
	}
}

func TestBuildRecord_WPMRounded(t *testing.T) {
	msg := ui.ResultMsg{
		Mode:   config.ModeTime,
		Length: 15,
		Result: metrics.Result{NetWPM: 72.6, Accuracy: 98, DurationMs: 15000},
	}
	rec := buildRecord(msg)
	want := int(math.Round(72.6)) // 73
	if rec.WPM != want {
		t.Errorf("WPM = %d, want %d", rec.WPM, want)
	}
}

func TestBuildRecord_TimestampNonZero(t *testing.T) {
	msg := ui.ResultMsg{
		Mode:   config.ModeTime,
		Length: 30,
		Result: metrics.Result{NetWPM: 60, Accuracy: 100, DurationMs: 30000},
	}
	rec := buildRecord(msg)
	if rec.Time.IsZero() {
		t.Error("timestamp should be non-zero")
	}
}

func TestBuildRecord_MetricsPassedThrough(t *testing.T) {
	msg := ui.ResultMsg{
		Mode:   config.ModeWords,
		Length: 25,
		Result: metrics.Result{
			NetWPM:      65.5,
			RawWPM:      70.2,
			Accuracy:    93.4,
			Consistency: 88.1,
			DurationMs:  20000,
		},
	}
	rec := buildRecord(msg)
	if rec.NetWPM != 65.5 {
		t.Errorf("NetWPM = %f, want 65.5", rec.NetWPM)
	}
	if rec.RawWPM != 70.2 {
		t.Errorf("RawWPM = %f, want 70.2", rec.RawWPM)
	}
	if rec.Accuracy != 93.4 {
		t.Errorf("Accuracy = %f, want 93.4", rec.Accuracy)
	}
	if rec.Consistency != 88.1 {
		t.Errorf("Consistency = %f, want 88.1", rec.Consistency)
	}
}
