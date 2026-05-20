---
phase: 2
title: "cobra+fang skeleton with bare-TUI fall-through and version subcommand"
status: completed
priority: P1
effort: "6h"
dependencies: [1]
---

# Phase 2: cobra+fang skeleton with bare-TUI fall-through and `version` subcommand

## Overview

Add the new CLI surface foundation: cobra root cmd, fang styling, `version`
subcommand, `--version`/`--text` back-compat aliases, `--help`/`-h` working,
and bare-`typeburn` still launching TUI. Preserves `decide()`-style purity.

Per researcher-01: root cmd uses `DisableFlagParsing: true` so the pure
`decide()` runs FIRST on raw args, then cobra/fang takes over for recognized
subcommands. Fang doesn't `os.Exit`; we own exit codes.

## Requirements

- Functional: `typeburn`, `typeburn --version`, `typeburn --text foo.txt`, `typeburn -h`, `typeburn version`, `typeburn version --json` all work
- Functional: `typeburn anything-unknown` still launches TUI (fall-through preserved)
- Functional: `typeburn version --bogus` exits 1 (subcommands are strict)
- Non-functional: parse path remains pure-testable; no `os.Exit` from parse

## Architecture

```
main.go                  # ~30 LOC: ctx, cli.NewRoot(), fang.Execute, os.Exit(code)
internal/cli/
  root.go                # cobra root cmd, DisableFlagParsing, RunE → TUI or alias-dispatch
  cmd_version.go         # `version` subcommand + --json
  exitcodes.go           # 0/1/2/3/4 constants
  decide.go              # pure decide(args) → (printVersion, textPath); migrated from main.go
  root_test.go           # tests decide() + cmd routing without process spawn
```

Key patterns (from researcher-01):

```go
var root = &cobra.Command{
    Use:                "typeburn",
    Short:              "Distraction-free terminal typing test",
    DisableFlagParsing: true,
    DisableSuggestions: true,   // prevent "did you mean 'version'?" spam on fall-through
    SilenceErrors:      true,
    SilenceUsage:       true,
    Args:               cobra.ArbitraryArgs,
    RunE: func(cmd *cobra.Command, args []string) error {
        // Help shim: under DisableFlagParsing, cobra does NOT auto-handle -h.
        // Without this shim, `typeburn -h` would fall through to TUI (acceptance #1 fails).
        if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
            return cmd.Help()
        }
        printVer, textPath := decide(args)
        if printVer { return runVersion(cmd, false) }
        // Any unrecognized root-level flag (e.g. `--bogus`) lands here too:
        // decide() returned zeros, so we launch TUI. Documented fall-through.
        return runTUI(textPath)
    },
}
root.AddCommand(versionCmd)
```

Root-level fall-through rule (explicit per F2): any args that `decide()` does not recognize
fall through to TUI. Strict subcommand parsing applies ONLY after a recognized subcommand
keyword (e.g. `run`, `version`, `history`). `typeburn --bogus` → TUI; `typeburn version --bogus` → exit 1.

`fang.Execute(ctx, root)` returns an error; main maps to exit code via `exitcodes.go`.

## Related Code Files

- Modify: `main.go` (thin wrapper)
- Create: `internal/cli/root.go`, `internal/cli/cmd_version.go`, `internal/cli/exitcodes.go`, `internal/cli/decide.go`, `internal/cli/root_test.go`, `internal/cli/decide_test.go`
- Modify: `go.mod` / `go.sum` (add `github.com/spf13/cobra`, `github.com/charmbracelet/fang`)

## Implementation Steps

1. Add deps with pinned tags (NOT `@latest` — fang is pre-1.0):
   - `go get github.com/spf13/cobra@v1.8.1` (or latest tagged stable at implementation time, recorded here)
   - `go get github.com/charmbracelet/fang@<concrete-tag>` — implementer MUST resolve the tag before running, record it in this step + in CONTRIBUTING.md's "Dep upgrade gate" section (added in Phase 7). Reproducibility requires explicit pin.
2. Move `main.go:decide()` + `decide_test.go` → `internal/cli/decide.go` + `decide_test.go`. Same signature, same tests.
3. Write `internal/cli/exitcodes.go` with `OK=0 Usage=1 IO=2 Abort=3 Internal=4`.
4. Implement `internal/cli/root.go`:
   - `func NewRoot() *cobra.Command` returns configured root
   - Root has `DisableFlagParsing: true`, `Args: cobra.ArbitraryArgs`, RunE calls `decide(args)` then dispatches
   - Attach `versionCmd`
5. Implement `internal/cli/cmd_version.go` with `--json` flag → emit `{version, commit, date, go, os, arch}` JSON or plain banner.
6. Rewrite `main.go` to ~15 LOC: build root, `fang.Execute(ctx, root)`, map error→exit-code, `os.Exit(code)`.
7. Tests in `root_test.go` — must cover BOTH root fall-through and subcommand strictness:
   - `decide()` purity preserved (existing test passes from new location)
   - `[]string{"anything-unknown"}` → RunE returns nil (TUI launcher injectable for tests)
   - `[]string{"--bogus-root-flag"}` → RunE returns nil (root fall-through; explicit per F2)
   - `[]string{"--text", "foo.txt"}` → RunE invokes TUI with textPath="foo.txt" (back-compat)
   - `[]string{"-h"}` → returns help (via shim), exit 0 (acceptance #1 protected per F3)
   - `[]string{"--help"}` → returns help
   - `[]string{"version", "--json"}` → emits JSON to buffer
   - `[]string{"version", "--bogus"}` → returns error (subcommand strict)
   - `[]string{"version"}` → emits banner
   - Verify NO `os.Exit` is called from any of these paths (mock os.Exit via injected exiter or wrap fang.Execute)
8. Run `make test`, `make lint`, `make build`.

## Success Criteria

- [ ] `go.mod` has cobra + fang; `go.sum` updated
- [ ] `typeburn` builds, runs, shows TUI Home (no regression)
- [ ] `typeburn --version` prints v1.5-format banner (back-compat)
- [ ] `typeburn version` prints same banner; `typeburn version --json` prints valid JSON
- [ ] `typeburn -h` and `typeburn --help` print fang-styled help; exit 0
- [ ] `typeburn anything-unknown` launches TUI; `typeburn --bogus-root-flag` also launches TUI; `typeburn version --bogus` exits 1
- [ ] `decide()` test moved + still passing
- [ ] All existing tests pass
- [ ] Binary size growth ≤ 1.2 MB at this phase
- [ ] `go.mod` records pinned tags for both cobra AND fang (no `@latest`); pin tags recorded in CONTRIBUTING.md "Dep upgrade gate"
- [ ] `-h` shim path covered by test (acceptance #1 hardened against DisableFlagParsing interaction)

## Risk Assessment

- **Risk:** cobra's `DisableFlagParsing` interacts oddly with subcommands.
  **Mitigation:** Researcher-01 confirmed pattern; integration test in `root_test.go` covers both branches.
- **Risk:** fang `Execute` signature changes between versions.
  **Mitigation:** Pin exact version via `go get …@<sha>` and document in CONTRIBUTING.
- **Risk:** Help output drift breaks teatest goldens.
  **Mitigation:** Help is non-TUI; teatest snapshots only TUI screens.
- **Risk:** Existing `gofmt -l .` empty assertion fails if cobra/fang generated code intrudes.
  **Mitigation:** Generated code lives in module cache, not formatted by `gofmt -l .` on `./...`.
