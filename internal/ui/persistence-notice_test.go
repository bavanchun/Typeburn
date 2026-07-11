package ui

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

func TestPersistenceNotice_ContainsMessageAndHint(t *testing.T) {
	th := theme.Load("default", false)
	got := PersistenceNotice("Couldn't save result to disk", th)
	if !strings.Contains(got, "Couldn't save result to disk") {
		t.Errorf("notice missing message; got %q", got)
	}
	if !strings.Contains(got, "dismiss") {
		t.Errorf("notice missing dismiss hint; got %q", got)
	}
}

func TestPersistenceNotice_EmptyMsgYieldsEmpty(t *testing.T) {
	th := theme.Load("default", false)
	if got := PersistenceNotice("", th); got != "" {
		t.Errorf("empty msg: want empty string, got %q", got)
	}
}

func TestPersistenceNotice_NoColorStillLegible(t *testing.T) {
	th := theme.Load("default", true) // no-color (attribute-only)
	got := PersistenceNotice("disk full", th)
	if !strings.Contains(got, "disk full") {
		t.Errorf("no-color notice must still carry the message text; got %q", got)
	}
}
