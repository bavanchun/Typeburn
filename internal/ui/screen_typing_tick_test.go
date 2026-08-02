package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
)

// startsTickLoop reports whether cmd starts a 100ms timer chain. It runs the
// command, because a chain is only real once its tea.Tick has been handed to
// the runtime — counting call sites instead would miss one hidden in a batch.
func startsTickLoop(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	switch msg := cmd().(type) {
	case tickMsg:
		return true
	case tea.BatchMsg:
		for _, c := range msg {
			if startsTickLoop(c) {
				return true
			}
		}
	}
	return false
}

// TestTyping_RestartsDoNotStackTickLoops asserts restarting leaves exactly one
// timer chain running.
//
// A tea.Tick chain re-arms itself forever and two chains never merge, so every
// unguarded start is a loop that ticks for the rest of the session — three tab
// presses used to leave four of them, all waking up ten times a second and all
// racing to end the same test.
func TestTyping_RestartsDoNotStackTickLoops(t *testing.T) {
	m := newTestTyping(config.ModeTime, 30)

	// The root bootstraps one chain the moment the screen opens.
	loops := 0
	if startsTickLoop(m.InitCmd()) {
		loops++
	}

	first := []rune(m.target)[0]
	var cmd tea.Cmd
	m, cmd = m.Update(pressText(string(first))) // starting the run must not add one
	if startsTickLoop(cmd) {
		loops++
	}

	for i := 0; i < 3; i++ {
		m, cmd = m.Update(press(tea.KeyTab, 0)) // restart same target
		if startsTickLoop(cmd) {
			loops++
		}
	}
	m, cmd = m.Update(press('r', tea.ModCtrl)) // fresh target
	if startsTickLoop(cmd) {
		loops++
	}

	if loops != 1 {
		t.Errorf("want exactly 1 live tick loop, got %d", loops)
	}

	// And the one chain is still going: a tick has to come back out, or the
	// timer and the live WPM header would both be dead after a restart.
	_, cmd = m.Update(tickMsg{t: time.Now()})
	if !startsTickLoop(cmd) {
		t.Error("the surviving chain stopped re-arming")
	}
}
