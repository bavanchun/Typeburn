package app

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/ui"
)

// sandboxXDG points history/settings writes at temp dirs so completing a test in
// these cases never touches the real user data dir.
func sandboxXDG(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

// startedTypingModel returns a root model on the typing screen at a usable size.
func startedTypingModel(t *testing.T, w, h int) Model {
	t.Helper()
	sandboxXDG(t)
	m := newTestModel()
	m.w, m.h = w, h
	out, _ := m.Update(ui.StartTestMsg{Mode: config.ModeWords, Length: 10})
	return out.(Model)
}

func sampleResultMsg() ui.ResultMsg {
	return ui.ResultMsg{
		Result: metrics.Result{NetWPM: 80, RawWPM: 95, Accuracy: 98, Consistency: 90, DurationMs: 30000},
		Mode:   config.ModeWords,
		Length: 10,
	}
}

// A resize during a transition cancels it (snapshot geometry is stale).
func TestEdge_ResizeCancelsTransition(t *testing.T) {
	m := startedTypingModel(t, 80, 24)
	out, _ := m.Update(sampleResultMsg())
	m = out.(Model)
	if m.transition == nil {
		t.Fatal("expected a transition after completing on the typing screen")
	}
	out, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	if out.(Model).transition != nil {
		t.Error("resize should cancel the in-flight transition")
	}
}

// Each result arms its own fresh reveal — a second consecutive result is active
// at its own start, never inheriting an elapsed window from the first.
func TestEdge_ConsecutiveResultsAnimateFresh(t *testing.T) {
	m := startedTypingModel(t, 80, 24)

	out, _ := m.Update(sampleResultMsg())
	first := out.(Model)
	if !first.result.HasActiveAnim(time.Now().UnixMilli()) {
		t.Error("first result reveal should be active right after completion")
	}

	out, _ = first.Update(ui.StartTestMsg{Mode: config.ModeWords, Length: 10})
	out, _ = out.(Model).Update(sampleResultMsg())
	second := out.(Model)
	if !second.result.HasActiveAnim(time.Now().UnixMilli()) {
		t.Error("second result reveal should start fresh (active right after)")
	}
}

// Below the degraded size, completing a test must NOT start a transition.
func TestEdge_DegradedSkipsTransition(t *testing.T) {
	m := startedTypingModel(t, 50, 18) // below 60×20
	out, _ := m.Update(sampleResultMsg())
	if out.(Model).transition != nil {
		t.Error("degraded terminal should skip the transition")
	}
}

// Once everything settles, a frame tick far past every window must not re-arm —
// the loop self-stops.
func TestEdge_LoopSelfStopsWhenSettled(t *testing.T) {
	m := startedTypingModel(t, 80, 24)
	out, _ := m.Update(sampleResultMsg())
	m = out.(Model)

	future := time.UnixMilli(time.Now().UnixMilli() + 10_000)
	out, cmd := m.Update(ui.FrameTickMsg{T: future})
	got := out.(Model)
	if got.animActive(got.animNowMs) {
		t.Error("settled result should report no active animation")
	}
	if cmd != nil {
		t.Error("frame loop should self-stop (nil cmd) once settled")
	}
}

// Abort from the typing screen clears any in-flight transition and returns Home.
func TestEdge_AbortClearsTransition(t *testing.T) {
	m := startedTypingModel(t, 80, 24)
	out, _ := m.Update(sampleResultMsg())
	m = out.(Model)
	if m.transition == nil {
		t.Fatal("expected a transition")
	}
	out, _ = m.Update(ui.AbortMsg{})
	got := out.(Model)
	if got.transition != nil {
		t.Error("abort should clear the transition")
	}
	if got.screen != ScreenHome {
		t.Error("abort should return to Home")
	}
}

// The composed result frame stays a sensible multi-line block with animation
// plumbing in place (regression guard on frame composition).
func TestEdge_FrameCompositionStable(t *testing.T) {
	m := startedTypingModel(t, 80, 24)
	out, _ := m.Update(sampleResultMsg())
	content := out.(Model).View().Content
	if strings.Count(content, "\n")+1 < 10 {
		t.Errorf("result frame unexpectedly short: %d lines", strings.Count(content, "\n")+1)
	}
}
