package config

import "testing"

// knownModes is every mode the app supports. If a new Mode constant is added
// it MUST be added here AND handled by LengthsFor + Normalize — this test is
// the guard against the multi-switch seam drifting (same discipline as the
// theme Available()/Normalize sync test).
var knownModes = []Mode{ModeTime, ModeWords, ModeQuote, ModeCode}

// TestModeSeam_LengthsForHandlesEveryKnownMode asserts LengthsFor never
// panics for a known mode and that the no-length modes (quote, code) return
// nil while the numeric modes return non-empty options.
func TestModeSeam_LengthsForHandlesEveryKnownMode(t *testing.T) {
	for _, m := range knownModes {
		lens := LengthsFor(m)
		switch m {
		case ModeQuote, ModeCode:
			if lens != nil {
				t.Errorf("LengthsFor(%q): want nil (no length selector), got %v", m, lens)
			}
		default:
			if len(lens) == 0 {
				t.Errorf("LengthsFor(%q): want non-empty options, got %v", m, lens)
			}
		}
	}
}

// TestModeSeam_NormalizePreservesKnownAndRepairsUnknown asserts every known
// mode survives Normalize and an unknown mode is repaired to ModeTime.
func TestModeSeam_NormalizePreservesKnownAndRepairsUnknown(t *testing.T) {
	for _, m := range knownModes {
		s := Defaults()
		s.DefaultMode = m
		s.Normalize()
		if s.DefaultMode != m {
			t.Errorf("Normalize reset known mode %q to %q", m, s.DefaultMode)
		}
	}

	s := Defaults()
	s.DefaultMode = "definitely-not-a-mode"
	s.Normalize()
	if s.DefaultMode != ModeTime {
		t.Errorf("unknown mode: want fallback ModeTime, got %q", s.DefaultMode)
	}
}
