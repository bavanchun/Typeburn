package app

import (
	"testing"
	"time"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/ui"
)

// ui.FrameTickCmd's command, when run, must produce a FrameTickMsg carrying the
// fire time — this is what re-arms the loop.
func TestFrameTickCmd_ProducesFrameTickMsg(t *testing.T) {
	cmd := ui.FrameTickCmd()
	if cmd == nil {
		t.Fatal("ui.FrameTickCmd returned nil")
	}
	msg := cmd()
	if _, ok := msg.(ui.FrameTickMsg); !ok {
		t.Fatalf("FrameTickCmd produced %T, want ui.FrameTickMsg", msg)
	}
}

// With every screen reporting idle, a frame tick must NOT re-arm: the batch
// collapses to nil so the loop self-stops.
func TestHandleFrameTick_StopsWhenIdle(t *testing.T) {
	m := newTestModel()
	now := time.UnixMilli(5000)

	model, cmd := m.handleFrameTick(ui.FrameTickMsg{T: now})
	got := model.(Model)

	if got.animNowMs != 5000 {
		t.Errorf("animNowMs=%d want 5000", got.animNowMs)
	}
	if cmd != nil {
		t.Errorf("idle frame tick re-armed (cmd != nil); loop should self-stop")
	}
}

// maybeFrameCmd is the self-stop seam: nil when nothing is live.
func TestMaybeFrameCmd_NilWhenIdle(t *testing.T) {
	m := newTestModel()
	m.screen = ScreenTyping
	if cmd := m.maybeFrameCmd(); cmd != nil {
		t.Errorf("maybeFrameCmd non-nil while typing idle")
	}
	m.screen = ScreenResult
	if cmd := m.maybeFrameCmd(); cmd != nil {
		t.Errorf("maybeFrameCmd non-nil while result idle")
	}
}

// A frame tick must never advance WPM/completion: routing it through the typing
// screen leaves the screen idle (no spurious test start, loop stops).
func TestFrameTick_DoesNotStartTest(t *testing.T) {
	m := newTestModel()
	m.screen = ScreenTyping
	m.typing = ui.NewTyping(config.ModeTime, 30, 0, m.theme, m.keys, false, false).SetSize(80, 24)

	model, _ := m.handleFrameTick(ui.FrameTickMsg{T: time.UnixMilli(1234)})
	got := model.(Model)
	if got.animActive(got.animNowMs) {
		t.Errorf("frame tick made an idle typing screen report active")
	}
}

func TestResultMsg_BootstrapsFrameLoop(t *testing.T) {
	m := newTestModel()
	staleNow := time.Now().UTC().UnixMilli() - 10_000
	m.animNowMs = staleNow
	msg := ui.ResultMsg{
		Result: metrics.Result{NetWPM: 80, Accuracy: 100, DurationMs: 30000},
		Mode:   config.ModeTime,
		Length: 30,
	}

	model, cmd := m.Update(msg)
	got := model.(Model)
	if got.screen != ScreenResult {
		t.Fatalf("screen=%v want ScreenResult", got.screen)
	}
	if cmd == nil {
		t.Fatal("ResultMsg should bootstrap a frame tick")
	}
	if !got.result.HasActiveAnim(time.Now().UTC().UnixMilli()) {
		t.Fatal("result reveal should start fresh even when the shared clock is stale")
	}

	got.animNowMs = time.Now().UTC().UnixMilli() - 10_000
	model, cmd = got.Update(msg)
	next := model.(Model)
	if cmd == nil {
		t.Fatal("second ResultMsg should bootstrap a frame tick")
	}
	if !next.result.HasActiveAnim(time.Now().UTC().UnixMilli()) {
		t.Fatal("second result reveal should restart from the current frame time")
	}
}
