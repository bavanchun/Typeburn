package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/ui"
)

// blockedXDG points an XDG_* path at a location under a regular file so
// os.MkdirAll fails (ENOTDIR) — a real write failure, no mocking.
func blockedXDG(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(blocker, "xdg") // a dir under a file → unwritable
}

// TestPersistNotice_HistoryFailureShownThenDismissed exercises the full path:
// a failed AppendHistory sets persistErr, the View shows it, and any key
// clears it while still routing normally.
func TestPersistNotice_HistoryFailureShownThenDismissed(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", blockedXDG(t))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m := tea.Model(New(theme.Default(), config.Defaults(), "", ""))
	m = sm_sendSize(m, 80, 24)

	m, _ = m.Update(ui.ResultMsg{
		Result: sampleResult(),
		Mode:   config.ModeWords,
		Length: 10,
	})
	rm := m.(Model)
	if rm.persistErr != "Couldn't save result to disk" {
		t.Fatalf("want persistErr set after failed AppendHistory, got %q", rm.persistErr)
	}
	if v := rm.View().Content; !strings.Contains(v, "Couldn't save result to disk") {
		t.Fatalf("View should show the persistence notice; got:\n%s", v)
	}

	// Any key dismisses the toast; the keystroke still routes (here: nav Home).
	m2, _ := sm_sendKey(m, '1', 0)
	cleared := m2.(Model)
	if cleared.persistErr != "" {
		t.Errorf("persistErr should clear on keypress, got %q", cleared.persistErr)
	}
	if v := cleared.View().Content; strings.Contains(v, "Couldn't save result to disk") {
		t.Errorf("notice should be gone after dismiss; got:\n%s", v)
	}
}

// TestPersistNotice_SettingsFailureSetsError verifies the settings persist
// path surfaces the notice on a real write failure.
func TestPersistNotice_SettingsFailureSetsError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", blockedXDG(t))
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	m := New(theme.Default(), config.Defaults(), "", "")
	s := m.settings
	s.BlinkCursor = !s.BlinkCursor
	m = m.applySettings(s) // value receiver → use the returned model

	if m.persistErr != "Couldn't save settings to disk" {
		t.Fatalf("want persistErr set after failed SaveSettings, got %q", m.persistErr)
	}
}

// TestPersistNotice_NoFailureNoNotice guards the no-error path: a writable
// sandbox must not show any notice (layout/golden-neutral).
func TestPersistNotice_NoFailureNoNotice(t *testing.T) {
	m := tea.Model(sandboxedModel(t))
	m = sm_sendSize(m, 80, 24)
	m, _ = m.Update(ui.ResultMsg{
		Result: sampleResult(),
		Mode:   config.ModeWords,
		Length: 10,
	})
	rm := m.(Model)
	if rm.persistErr != "" {
		t.Fatalf("writable sandbox: persistErr must stay empty, got %q", rm.persistErr)
	}
	if v := rm.View().Content; strings.Contains(v, "press any key to dismiss") {
		t.Errorf("no notice expected on success path; got:\n%s", v)
	}
}
