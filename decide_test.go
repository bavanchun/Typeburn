package main

import "testing"

func TestDecide_VersionFlag(t *testing.T) {
	pv, _ := decide([]string{"--version"})
	if !pv {
		t.Fatal("--version must request the version banner")
	}
}

func TestDecide_UnknownFlag_FallsThrough(t *testing.T) {
	pv, _ := decide([]string{"--bogus"})
	if pv {
		t.Fatal("unknown flag must fall through to the TUI, not print version")
	}
}

func TestDecide_DashH_FallsThrough(t *testing.T) {
	pv, _ := decide([]string{"-h"})
	if pv {
		t.Fatal("-h must fall through to the TUI (no usage dump, no exit 2)")
	}
}

func TestDecide_NoArgs_FallsThrough(t *testing.T) {
	pv, _ := decide(nil)
	if pv {
		t.Fatal("no args must launch the TUI")
	}
}

func TestDecide_NoShortV(t *testing.T) {
	pv, _ := decide([]string{"-v"})
	if pv {
		t.Fatal("-v must NOT be bound to version (reserved for future --verbose)")
	}
}

// TestDecide_TextFlag_SetsPath verifies --text <path> populates textPath and
// leaves printVersion false.
func TestDecide_TextFlag_SetsPath(t *testing.T) {
	pv, path := decide([]string{"--text", "myfile.go"})
	if pv {
		t.Fatal("--text must not trigger the version banner")
	}
	if path != "myfile.go" {
		t.Fatalf("want textPath=%q, got %q", "myfile.go", path)
	}
}

// TestDecide_TextFlag_Stdin verifies --text - is recognized (stdin sentinel).
func TestDecide_TextFlag_Stdin(t *testing.T) {
	pv, path := decide([]string{"--text", "-"})
	if pv {
		t.Fatal("--text - must not trigger the version banner")
	}
	if path != "-" {
		t.Fatalf("want textPath=%q, got %q", "-", path)
	}
}

// TestDecide_NoTextFlag_EmptyPath verifies textPath is "" when --text is absent.
func TestDecide_NoTextFlag_EmptyPath(t *testing.T) {
	_, path := decide([]string{"--version"})
	if path != "" {
		t.Fatalf("want empty textPath when --text absent, got %q", path)
	}
}
