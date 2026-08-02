package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

func TestPersistenceNotice_ContainsMessageAndHint(t *testing.T) {
	th := theme.Load("default", false)
	got := PersistenceNotice("Couldn't save result to disk", 120, th)
	if !strings.Contains(got, "Couldn't save result to disk") {
		t.Errorf("notice missing message; got %q", got)
	}
	if !strings.Contains(got, "dismiss") {
		t.Errorf("notice missing dismiss hint; got %q", got)
	}
}

func TestPersistenceNotice_EmptyMsgYieldsEmpty(t *testing.T) {
	th := theme.Load("default", false)
	if got := PersistenceNotice("", 80, th); got != "" {
		t.Errorf("empty msg: want empty string, got %q", got)
	}
}

func TestPersistenceNotice_NoColorStillLegible(t *testing.T) {
	th := theme.Load("default", true) // no-color (attribute-only)
	got := PersistenceNotice("disk full", 80, th)
	if !strings.Contains(got, "disk full") {
		t.Errorf("no-color notice must still carry the message text; got %q", got)
	}
}

// TestPersistenceNotice_FitsTheTerminal is the reason the width argument
// exists. The notice is written into a real frame, so a line wider than the
// terminal does not merely spill — every row around it is padded to match and
// the whole screen moves off centre.
func TestPersistenceNotice_FitsTheTerminal(t *testing.T) {
	th := theme.Load("default", false)
	msgs := []string{
		"could not write history.json: permission denied",
		"run withheld from history: away from keyboard for most of it",
		"短", // a single wide rune, to catch a byte-based trim
		strings.Repeat("very long reason ", 20),
	}

	for _, w := range []int{60, 61, 72, 80, 88, 120, 200} {
		for _, msg := range msgs {
			got := lipgloss.Width(PersistenceNotice(msg, w, th))
			if got > w {
				t.Errorf("width %d: notice is %d cells for %q", w, got, msg)
			}
		}
	}
}

// TestPersistenceNotice_KeepsTheReasonWhenItDropsTheHint: the hint is the
// disposable half. Losing the reason and keeping "press any key" would leave
// the user knowing only that something happened.
func TestPersistenceNotice_KeepsTheReasonWhenItDropsTheHint(t *testing.T) {
	th := theme.Load("default", false)
	const msg = "could not write history.json: permission denied"

	got := PersistenceNotice(msg, 60, th)

	if !strings.Contains(got, msg) {
		t.Errorf("the reason was trimmed before the hint was: %q", got)
	}
	if strings.Contains(got, "dismiss") {
		t.Errorf("the hint survived a width it does not fit: %q", got)
	}
}

// TestPersistenceNotice_MarksATruncatedReason: a reason cut short must look
// cut short, or the user reads a fragment as the whole thing.
func TestPersistenceNotice_MarksATruncatedReason(t *testing.T) {
	th := theme.Load("default", false)

	got := PersistenceNotice(strings.Repeat("x", 200), 60, th)

	if !strings.Contains(got, "…") {
		t.Errorf("a clipped reason carries no ellipsis: %q", got)
	}
}

// TestPersistenceNotice_UnknownWidthIsNotTrimmed: 0 means the terminal size has
// not arrived yet, which is not the same as "no room".
func TestPersistenceNotice_UnknownWidthIsNotTrimmed(t *testing.T) {
	th := theme.Load("default", false)
	const msg = "could not write history.json: permission denied"

	got := PersistenceNotice(msg, 0, th)

	if !strings.Contains(got, msg) || !strings.Contains(got, "dismiss") {
		t.Errorf("unsized notice lost content: %q", got)
	}
}
