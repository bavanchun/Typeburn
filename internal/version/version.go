// Package version exposes the build-time version of typeburn.
//
// Version, Commit and Date are injected by the linker via
//
//	-ldflags "-X github.com/bavanchun/Typeburn/internal/version.Version=..."
//
// (wired in the Makefile and .goreleaser.yaml). When they are not set — for
// example after `go install github.com/bavanchun/Typeburn@v1.0.0`, which
// applies no ldflags — Resolve falls back to the module build information the
// Go toolchain embeds in every binary.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// ldflags-injection targets. Keep these as plain package-level string vars:
// the linker's -X only rewrites simple string variables.
var (
	Version string
	Commit  string
	Date    string
)

// Info is a fully resolved version triple ready for display.
type Info struct {
	Version string
	Commit  string
	Date    string
}

// Resolve returns the effective version information.
//
// Precedence: ldflags-injected values win. Any field left empty is filled from
// debug.ReadBuildInfo() — Main.Version for the version (ignoring the synthetic
// "(devel)"), and the vcs.revision / vcs.time build settings for commit/date.
// If the version is still unknown the final fallback is "dev". It never panics.
func Resolve() Info {
	v, c, d := Version, Commit, Date

	if v == "" || c == "" || d == "" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			if v == "" {
				if mv := bi.Main.Version; mv != "" && mv != "(devel)" {
					v = mv
				}
			}
			for _, s := range bi.Settings {
				switch s.Key {
				case "vcs.revision":
					if c == "" {
						c = s.Value
					}
				case "vcs.time":
					if d == "" {
						d = s.Value
					}
				}
			}
		}
	}

	if v == "" {
		v = "dev"
	}
	return Info{Version: v, Commit: c, Date: d}
}

// shortCommit trims a git SHA to 7 characters for compact display.
func shortCommit(c string) string {
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

// String renders a single-line banner, e.g.
//
//	typeburn v1.0.0 (61a4afd, 2026-05-18T21:10:00Z, go1.26.2 darwin/arm64)
func (i Info) String() string {
	commit := i.Commit
	if commit == "" {
		commit = "none"
	} else {
		commit = shortCommit(commit)
	}
	date := i.Date
	if date == "" {
		date = "unknown"
	}
	return fmt.Sprintf("typeburn %s (%s, %s, %s %s/%s)",
		i.Version, commit, date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// String resolves the build info and renders the one-line banner.
func String() string { return Resolve().String() }
