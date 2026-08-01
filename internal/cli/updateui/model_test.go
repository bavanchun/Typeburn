package updateui

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/update"
)

// driveable builds a model whose snapshot is controlled by the test, so the
// event loop can be stepped without a running program or a real download.
func driveable(p *update.Progress, results <-chan Result) Model {
	th := theme.Load("default", true)
	return New("v2.5.1", "v2.6.0", th, func() update.Progress { return *p }, results)
}

// step feeds one poll tick and returns the updated model.
func step(t *testing.T, m Model) Model {
	t.Helper()
	next, _ := m.Update(tickMsg(time.Now()))
	return next.(Model)
}

// The frame must track the snapshot: as the update advances, rows settle.
func TestModel_FrameFollowsSnapshot(t *testing.T) {
	p := update.Progress{Stage: update.StageChecksums}
	m := driveable(&p, nil)

	m = step(t, m)
	if got := ansi.Strip(m.Frame()); !strings.Contains(got, "·  downloading") {
		t.Errorf("downloading should still be pending:\n%s", got)
	}

	p = update.Progress{Stage: update.StageVerifying, Total: 4_513_792}
	m = step(t, m)
	got := ansi.Strip(m.Frame())
	if !strings.Contains(got, "✓  checksums") || !strings.Contains(got, "✓  downloading") {
		t.Errorf("earlier stages should be settled:\n%s", got)
	}
	if !strings.Contains(got, "·  installing") {
		t.Errorf("installing should still be pending:\n%s", got)
	}
}

// Percent must rise with the reported bytes — this is what makes the bar honest
// rather than decorative.
func TestModel_PercentTracksBytes(t *testing.T) {
	p := update.Progress{Stage: update.StageDownloading, Done: 0, Total: 1000}
	m := driveable(&p, nil)

	m = step(t, m)
	first := m.percent()

	p.Done = 750
	m = step(t, m)
	if second := m.percent(); second <= first {
		t.Errorf("percent did not advance: %v → %v", first, second)
	} else if second != 0.75 {
		t.Errorf("percent = %v, want 0.75", second)
	}
}

// A finished run quits and carries its outcome out for the caller to print.
func TestModel_ResultQuitsAndIsReported(t *testing.T) {
	p := update.Progress{Stage: update.StageInstalling}
	m := driveable(&p, nil)

	want := update.Outcome{From: "v2.5.1", To: "v2.6.0"}
	next, _ := m.Update(resultMsg(Result{Outcome: want}))
	m = next.(Model)

	res, settled := m.Result()
	if res.Outcome != want {
		t.Errorf("Result() = %+v, want %+v", res.Outcome, want)
	}
	if !settled {
		t.Error("a delivered result must mark the model settled")
	}
	if !m.done {
		t.Error("model should be marked done")
	}
	// Every row settles once the run has finished.
	if got := ansi.Strip(m.Frame()); strings.Contains(got, "·") {
		t.Errorf("finished frame still shows a pending row:\n%s", got)
	}
}

// Cancelling before the install stage is honoured.
func TestModel_CancelBeforeInstall(t *testing.T) {
	p := update.Progress{Stage: update.StageDownloading, Done: 10, Total: 100}
	m := driveable(&p, nil)
	m = step(t, m)

	next, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m = next.(Model)

	res, settled := m.Result()
	if res.Err != ErrCancelled {
		t.Errorf("Result().Err = %v, want ErrCancelled", res.Err)
	}
	if !settled {
		t.Error("cancelling must mark the model settled")
	}
	if cmd == nil {
		t.Error("cancelling should issue a quit command")
	}
}

// Once the install stage begins, the remaining work is the atomic swap. This is
// the only path in the feature that could leave a user without a working
// binary, so the interrupt must be refused.
func TestModel_CancelRefusedDuringInstall(t *testing.T) {
	p := update.Progress{Stage: update.StageInstalling}
	m := driveable(&p, nil)
	m = step(t, m)

	next, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m = next.(Model)

	res, settled := m.Result()
	if res.Err != nil {
		t.Errorf("Result().Err = %v, want nil (cancel refused)", res.Err)
	}
	if settled {
		t.Error("a refused cancel must not settle the model")
	}
	if cmd != nil {
		t.Error("cancelling during install must not issue a quit command")
	}
}

// Bubble Tea returns directly on QuitMsg (SIGTERM) and InterruptMsg without
// routing through the model. An unsettled model must therefore never read as a
// successful update, or a killed run would print a false success line.
func TestModel_UnsettledUntilTerminalState(t *testing.T) {
	p := update.Progress{Stage: update.StageDownloading, Done: 10, Total: 100}
	m := driveable(&p, nil)

	if _, settled := m.Result(); settled {
		t.Error("a fresh model must not be settled")
	}
	m = step(t, m)
	if _, settled := m.Result(); settled {
		t.Error("progress alone must not settle the model")
	}
}

// The guard must read the live stage, not the polled copy. m.cur only refreshes
// every pollInterval, and Apply reports StageInstalling immediately before it
// extracts and renames — so a snapshot that has advanced to installing must be
// honoured even when the model has not ticked since.
func TestModel_CancelRefusedOnLiveStageBeforeNextPoll(t *testing.T) {
	p := update.Progress{Stage: update.StageDownloading, Done: 99, Total: 100}
	m := driveable(&p, nil)
	m = step(t, m) // m.cur is now "downloading"

	// The update advances, but no tick has happened yet.
	p = update.Progress{Stage: update.StageInstalling}

	next, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m = next.(Model)

	if _, settled := m.Result(); settled {
		t.Error("cancel accepted against a stale stage: the swap window is unguarded")
	}
	if cmd != nil {
		t.Error("cancel must not quit once the update has reached installing")
	}
}

// A cancelled run did not finish, so its final frame must not paint every row
// as complete.
func TestModel_CancelledFrameDoesNotClaimCompletion(t *testing.T) {
	p := update.Progress{Stage: update.StageDownloading, Done: 10, Total: 100}
	m := driveable(&p, nil)
	m = step(t, m)

	next, _ := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m = next.(Model)

	got := ansi.Strip(m.Frame())
	if !strings.Contains(got, "·  verifying") || !strings.Contains(got, "·  installing") {
		t.Errorf("cancelled frame marked unreached stages as done:\n%s", got)
	}
}
