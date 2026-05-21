package cli

import "testing"

func TestDecide_VersionFlag(t *testing.T) {
	pv, _ := Decide([]string{"--version"})
	if !pv {
		t.Fatal("--version must request the version banner")
	}
}

func TestDecide_UnknownFlag_FallsThrough(t *testing.T) {
	pv, _ := Decide([]string{"--bogus"})
	if pv {
		t.Fatal("unknown flag must fall through to the TUI, not print version")
	}
}

func TestDecide_DashH_FallsThrough(t *testing.T) {
	pv, _ := Decide([]string{"-h"})
	if pv {
		t.Fatal("-h must not print version")
	}
}

func TestDecide_NoArgs_FallsThrough(t *testing.T) {
	pv, _ := Decide(nil)
	if pv {
		t.Fatal("no args must launch the TUI")
	}
}

func TestDecide_NoShortV(t *testing.T) {
	pv, _ := Decide([]string{"-v"})
	if pv {
		t.Fatal("-v must NOT be bound to version")
	}
}

func TestDecide_TextFlag_SetsPath(t *testing.T) {
	pv, path := Decide([]string{"--text", "myfile.go"})
	if pv {
		t.Fatal("--text must not trigger the version banner")
	}
	if path != "myfile.go" {
		t.Fatalf("want textPath=%q, got %q", "myfile.go", path)
	}
}

func TestDecide_TextFlag_Stdin(t *testing.T) {
	pv, path := Decide([]string{"--text", "-"})
	if pv {
		t.Fatal("--text - must not trigger the version banner")
	}
	if path != "-" {
		t.Fatalf("want textPath=%q, got %q", "-", path)
	}
}

func TestDecide_NoTextFlag_EmptyPath(t *testing.T) {
	_, path := Decide([]string{"--version"})
	if path != "" {
		t.Fatalf("want empty textPath when --text absent, got %q", path)
	}
}
