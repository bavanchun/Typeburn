package version

import (
	"runtime"
	"strings"
	"testing"
)

// setVars overrides the ldflags-target package vars for the duration of a test
// and restores the originals via t.Cleanup so test order does not matter.
func setVars(t *testing.T, v, c, d string) {
	t.Helper()
	ov, oc, od := Version, Commit, Date
	Version, Commit, Date = v, c, d
	t.Cleanup(func() { Version, Commit, Date = ov, oc, od })
}

func TestResolve_LdflagsWin(t *testing.T) {
	setVars(t, "v9.9.9", "abc1234def", "2026-01-02T03:04:05Z")
	got := Resolve()
	if got.Version != "v9.9.9" || got.Commit != "abc1234def" || got.Date != "2026-01-02T03:04:05Z" {
		t.Fatalf("ldflags values must win, got %+v", got)
	}
}

func TestResolve_FallbackNoPanic(t *testing.T) {
	setVars(t, "", "", "")
	got := Resolve() // must not panic
	if got.Version == "" {
		t.Fatalf("fallback Version must be non-empty (at least %q)", "dev")
	}
	if got.Version == "(devel)" {
		t.Fatalf("literal %q must never leak", "(devel)")
	}
}

func TestString_Format(t *testing.T) {
	info := Info{Version: "v1.0.0", Commit: "61a4afdcafe", Date: "2026-05-18T21:10:00Z"}
	s := info.String()
	for _, want := range []string{
		"typeburn",
		"v1.0.0",
		"61a4afd", // short commit (7 chars)
		"2026-05-18T21:10:00Z",
		runtime.Version(),
		runtime.GOOS + "/" + runtime.GOARCH,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("String() %q missing %q", s, want)
		}
	}
	if strings.Contains(s, "61a4afdcafe") {
		t.Errorf("commit must be shortened, got %q", s)
	}
}
