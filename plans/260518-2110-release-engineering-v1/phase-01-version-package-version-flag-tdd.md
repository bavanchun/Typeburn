---
phase: 1
title: "Version package & --version flag (TDD)"
status: pending
priority: P1
effort: "2h"
dependencies: []
---

# Phase 1: Version package & --version flag (TDD)

## Overview

Hybrid version mechanism: `internal/version` package whose `Version/Commit/Date` vars are
overridable via `-ldflags -X`; when empty, fall back to `debug.ReadBuildInfo()`. Add a
`--version` flag to `main.go` that prints the resolved version block and exits, **without
regressing the current "bare launch ignores stray args" behavior**.

Red-team applied: F12 (graceful unknown-arg/`-h` handling, no `os.Exit(2)` regression),
F13 (drop `-v` short alias — only `--version`; reserve `-v` for future verbose).

## Requirements

- Functional:
  - `internal/version.Resolve()` returns `{Version, Commit, Date}` with precedence:
    ldflags-set wins; if `Version==""` → `BuildInfo.Main.Version` (skip `(devel)`/empty);
    fill `Commit`/`Date` from `vcs.revision`/`vcs.time` when empty; final fallback
    `Version="dev"`. Never panic.
  - `version.String()` → one line, e.g.
    `typeburn v1.0.0 (61a4afd, 2026-05-18T21:10:00Z, go1.26.2 darwin/arm64)`.
  - `main.go`: `--version` → print `version.String()` to stdout, exit 0, before `tea.NewProgram`.
  - **No `-v` short flag** (frozen public CLI surface; `-v`/`--verbose` reserved for future).
  - **Unknown args / `-h` must NOT crash or `os.Exit(2)`-regress**: current behavior is
    `typeburn <anything>` launches the TUI. Use a dedicated `flag.NewFlagSet("typeburn",
    flag.ContinueOnError)` with `SetOutput(io.Discard)`; on parse error (unknown flag, `-h`)
    do NOT exit — fall through into the TUI exactly as today. Only an explicit, successfully
    parsed `--version` short-circuits.
- Non-functional: file <200 lines; no new deps (stdlib `runtime/debug`, `runtime`, `flag`, `io`).

## Architecture

```
main.go
  fs := flag.NewFlagSet("typeburn", flag.ContinueOnError); fs.SetOutput(io.Discard)
  showVersion := fs.Bool("version", false, "print version and exit")
  if fs.Parse(os.Args[1:]) == nil && *showVersion { print version.String(); return }
  // any parse error OR no --version → unchanged: tea.NewProgram(app.NewFromDisk())
internal/version/version.go
  var Version,Commit,Date string  (ldflags -X targets)
  Resolve(): ldflags ? use : debug.ReadBuildInfo() fallback
```

ldflags target path (Makefile + GoReleaser, Phase 2):
`-X github.com/bavanchun/Typeburn/internal/version.Version=...` (+ `.Commit`, `.Date`).
Capital-`T` case matches `go.mod:1` `module github.com/bavanchun/Typeburn` — verified, not assumed.
`go install …@v1.0.0` applies NO ldflags → `BuildInfo.Main.Version` == `v1.0.0` (proxy stamps it;
proxy *ingestion* lag is a Phase 5 concern, not a build concern).

## Related Code Files

- Create: `internal/version/version.go`
- Create: `internal/version/version_test.go`
- Modify: `main.go` (add FlagSet parse before `tea.NewProgram`; graceful fallthrough)

## Implementation Steps (TDD — tests first)

1. **Write `version_test.go` first (RED):**
   - `TestResolve_LdflagsWin`: set vars to `v9.9.9/abc/2026`; assert `Resolve()` returns them;
     restore via `t.Cleanup`.
   - `TestResolve_FallbackNoPanic`: vars empty → non-empty `Version` (≥ `"dev"`), no panic,
     no literal `(devel)` leaked.
   - `TestString_Format`: known `Info` → `String()` contains version, short commit,
     `runtime.Version()`, `GOOS/GOARCH`.
2. **Write flag-behavior tests (RED)** — extract the arg→action decision into a testable
   pure func `decide(args []string) (printVersion bool)` (no `os.Exit` inside):
   - `TestDecide_VersionFlag`: `["--version"]` → true.
   - `TestDecide_UnknownFlag_FallsThrough`: `["--bogus"]` → false (no panic, TUI path).
   - `TestDecide_DashH_FallsThrough`: `["-h"]` → false (does NOT exit/usage-dump).
   - `TestDecide_NoArgs_FallsThrough`: `[]` → false.
   - `TestDecide_NoShortV`: `["-v"]` → false (not bound to version; TUI path).
3. Run `go test ./internal/version/` → FAIL (symbols absent).
4. **Implement `version.go` (GREEN):** vars + `Info` + `Resolve()` + `String()` + short-commit helper.
5. **Implement `decide()` + wire `main.go` (GREEN):** `ContinueOnError` FlagSet, discard output,
   parse-error/`-h`/unknown → fallthrough to TUI; `--version` → print + return.
6. `go build ./... && go vet ./... && gofmt -l .` clean.
7. Manual: `go run . --version` prints line & exits; `go run . -v` / `go run . -h` /
   `go run . --bogus` all still launch the TUI (no exit 2, no usage dump).
8. Full regression: `go test ./... -race -count=1` GREEN.

## Success Criteria

- [ ] version + flag-decide tests written before impl; initial RED, final GREEN
- [ ] `go run . --version` prints line, exit 0, no TUI
- [ ] `-v`, `-h`, unknown args → TUI launches (NO `os.Exit(2)`, no usage dump) — covered by tests
- [ ] no `-v` short alias bound to version (reserved)
- [ ] ldflags-set path returns injected values (deterministic test)
- [ ] fallback path: non-empty, no panic, no literal `(devel)`
- [ ] `go build`/`vet`/`gofmt -l`/`go test ./... -race` clean
- [ ] `internal/version/version.go` < 200 lines, no new deps

## Risk Assessment

- Build-info fallback environment-dependent → test only deterministic ldflags path strictly;
  assert fallback invariants only (documented honest limitation).
- `flag` default behavior is `ExitOnError` + stderr usage → explicitly use `ContinueOnError`
  + `SetOutput(io.Discard)` so the distraction-free TUI contract is preserved. Covered by
  `TestDecide_*` regression tests, not left to "revisit later".
- `-v` permanently reserved (not bound) so a future `--verbose` keeps the short form — public
  v1.0.0 CLI surface decided now, not deferred.

## Next Steps

Phase 2 wires Makefile + GoReleaser to set these ldflags targets.
