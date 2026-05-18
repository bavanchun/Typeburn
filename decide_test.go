package main

import "testing"

func TestDecide_VersionFlag(t *testing.T) {
	if !decide([]string{"--version"}) {
		t.Fatal("--version must request the version banner")
	}
}

func TestDecide_UnknownFlag_FallsThrough(t *testing.T) {
	if decide([]string{"--bogus"}) {
		t.Fatal("unknown flag must fall through to the TUI, not print version")
	}
}

func TestDecide_DashH_FallsThrough(t *testing.T) {
	if decide([]string{"-h"}) {
		t.Fatal("-h must fall through to the TUI (no usage dump, no exit 2)")
	}
}

func TestDecide_NoArgs_FallsThrough(t *testing.T) {
	if decide(nil) {
		t.Fatal("no args must launch the TUI")
	}
}

func TestDecide_NoShortV(t *testing.T) {
	if decide([]string{"-v"}) {
		t.Fatal("-v must NOT be bound to version (reserved for future --verbose)")
	}
}
