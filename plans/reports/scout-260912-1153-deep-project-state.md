---
title: "Deep Scout — Typeburn current state"
date: 2026-09-12
type: scout
mode: ultra (best-of-5 verifier)
branch: main
head: 447e84c2
---

# Deep Scout — Typeburn @ v2.9.0

Method: 5 independent read-only candidate passes + controller verification.
Every claim below re-verified in the controller session; unverifiable candidate
claims dropped.

## 1. Identity

- Module `github.com/bavanchun/Typeburn/v2`, Go 1.25.0. Single binary, no backend.
- Public stable **v2.9.0** (tag on `447e84c2`, 2026-08-02). No commits since — ~6 weeks quiet.
- 303 Go files / 31,881 LOC; 146 are `_test.go` (48%).
- Direct deps: bubbletea v2.0.8, lipgloss v2.0.5, bubbles v2.1.1, fang v1.0.0,
  cobra v1.10.2, x/ansi, x/sys, x/term. All inside the CLAUDE.md allow-list.

## 2. Package map (src LOC excl. tests | test files)

Pure logic — **0 charm deps, verified by `go list -deps | grep charm`**:

| pkg | src | tests | role |
|---|---|---|---|
| `typing` | 408 | 9 | keystroke state machine (`New`, `NewStrict`, `Apply`, `Backspace`) |
| `metrics` | 542 | 11 | `Compute(log, mode, endMs)`, `TrimAFK`, key heatmap — post-hoc replay only |
| `words` | 304 | 4 | embedded wordlist + quote pack + generator |
| `codetext` | 153 | 2 | `Load(path)` = the only file/stdin I/O boundary; pure `Normalize` shares the core |
| `version` | 98 | 1 | ldflags → `debug.ReadBuildInfo` → `"dev"` |
| `mode` | 27 | 1 | `Mode` enum + `LengthsFor` |
| `runner` | 62 | 1 | `NewSession`/`NewCodeSession` shared by TUI and `--no-tui` |
| `anim` | 163 | 4 | clock/tween/easing/color math |
| `update` | 1421 | 16 | check/download/verify/lock/atomic-replace pipeline |

`storage` (676 src, 10 tests): UI-free in its own files, but
`go list -deps ./internal/storage | grep -c charm` = **9**, transitively via
`internal/config`. Exactly the known debt CLAUDE.md documents — reproduced, not stale.

Deliberate UI boundary: `config` (212 / 4, `keymap.go` binds `tea.Key`),
`theme` (428 / 3, lipgloss `Role`→style, 8 themes + mono/NO_COLOR).

UI/app: `ui` (5173 / 38, six screens + components), `app` (985 / 19, root Elm
model), `cli` (2485 / 23, cobra tree + `notui`/`output`/`updateui`).
`cmd/typeburn` is 17 LOC.

**Layering verdict: clean.** No pure-logic package imports a UI lib.
`internal/ui` calls **zero** storage I/O functions (grep = 0) — it only uses the
`Record` type and pure helpers. Persistence lives solely in `internal/app`.

## 3. Runtime flow

`cmd/typeburn/main.go` → `fang.Execute(cli.NewRoot())` → root `RunE` calls pure
`cli.Decide(args)` (stdlib `flag`, no I/O) for legacy `--version` / `--text`
aliases → `launchHome` → `app.New(...)`.

Six screens (`internal/app/screen.go:8-13`): Home, Typing, Result, Settings,
History, CodePaste.

Domain messages — **spread across 4 files, not all in `messages.go`** as CLAUDE.md implies:

| msg | defined at |
|---|---|
| `FrameTickMsg`, `ResultMsg`, `NavCodePasteMsg`, `CodePastedMsg`, `SettingsChangedMsg` | `internal/ui/messages.go` |
| `StartTestMsg` | `internal/ui/screen_home.go:14` |
| `AbortMsg` | `internal/ui/screen_typing.go:62` |
| `NavHistoryMsg` | `internal/ui/screen_result.go:143` |

## 4. CLI surface

`typeburn` bare → TUI. Subcommands: `version` (`--json --check-update`),
`run` (`--mode --duration --words --quote-len --theme --text --no-tui --json`),
`history` (`--limit/-n --json`), `config {list|get|set} --json`,
`replay <log.json>` (schema-versioned, bounded 8 MiB / 24h before allocation),
`update` (`--yes/-y --check`).

`update`: checks GitHub releases, SHA-256 over HTTPS (binaries unsigned by
design), refuses Homebrew / `go install` builds with the right upgrade hint,
process lock that self-heals after Ctrl-C, atomic self-replace. Renders through
`internal/cli/updateui` (independent inline Bubble Tea program) with a plain-text
fallback on non-TTY / narrow terminals.

## 5. Quality gates — run live in this session, all green

- `gofmt -l .` → empty
- `go vet ./...` → clean
- `go test ./... -race -count=1` → all packages pass
- `make lint` = gofmt + vet + `notui-noexit-check` (greps `internal/cli/notui` for `os.Exit`). It does **not** enforce the 200-LOC file rule.
- `make size-check` = 10 MiB cap (`SIZE_LIMIT=10485760`); v2.8 measured 9,002,530 B (85.9%). No v2.9 figure recorded.

Conventions holding: **every** file over 200 LOC is a `_test.go`; production code
is all under the cap. **Zero** TODO/FIXME/HACK/XXX markers anywhere in Go source.

## 6. CI / release

- `ci.yml`: branches+PRs only, no tag trigger. Jobs — `test` (ubuntu+macos matrix, full gate), `vuln` (govulncheck@v1.1.4, fails only on non-stdlib reachable findings), `installer` (shellcheck + offline install.sh harness + `goreleaser check`). **No Windows job** — deliberate documented gap.
- `release.yml`: tag-triggered, own least-privilege `test` job (since ci.yml skips tags) → `verify-main` (tag must be reachable from origin/main) → publish. Asserts release-notes heading matches tag (prereleases exempt, that was #87) and exactly 7 assets. Actions SHA-pinned.
- `.goreleaser.yaml`: v2 schema, GoReleaser pinned **v2.15.4** in lockstep with release.yml; 6 platforms, CGO off, `changelog.filters.exclude: ['.*']` + curated `.github/release-notes.md`; live `homebrew_casks:` block → `bavanchun/homebrew-tap-typeburn`.

## 7. Findings — three staleness issues, no code defects

**F1. `teatest` is documented everywhere and used nowhere.** CLAUDE.md plus
`docs/{codebase-summary,system-architecture,code-standards,project-overview-pdr,project-roadmap}.md`
all describe UI tests as teatest golden files. `grep -rn teatest --include='*.go'`
returns exactly one hit — a comment in `internal/app/smoke_test.go:4` explaining
teatest targets bubbletea **v1** and is therefore not used. `go.sum` contains
zero teatest entries. Real strategy: hand-rolled smoke tests driving
`Model.Update` with synthetic `tea.KeyPressMsg`/`WindowSizeMsg`, plus baseline
text fixtures in `internal/ui/testdata/` (`result_baseline_{color,nocolor}_{60,80,120,200}.txt`,
`big_digits_strip.txt`). Impact: anyone adding a screen test follows an
abandoned pattern.

**F2. `docs/project-roadmap.md` contradicts itself.** Header says public stable
v2.9.0; line 414 still reads *"Typeburn v2.5.1 is the current public stable
release."* Four releases stale.

**F3. Completed plan never closed out.**
`plans/260801-2216-audit-remediation-three-roots/plan.md:7` says
`status: in-progress`, while all nine phase files say `status: done` and the
roadmap + PR chain (#75–#87) confirm the cycle shipped as v2.9.0. Index metadata
only; work is real and done.

Also noted (not a defect): the roadmap self-reports v2.9.0's own known gap — it
shipped without the manual live-TUI pass the plan required; every measurement
came from calling `View()` directly, "sound for structure and silent on judgement."

Working tree: 6 unstaged deletions, all under `.claude/skills/` (package-lock
files + one SKILL.md). Zero uncommitted changes to Go source, docs, plans, or CI.

## 8. Open backlog (from roadmap)

Only two live items: (1) gather user feedback — strategic, undecided;
(2) shell completions + man pages in release archives — deferred.
Deferred by design: multiplayer/sync, plugin system (v3+), sound, daily
challenges, custom wordlists, Windows CI.

## Unresolved questions

- Was `plan.md`'s `in-progress` status left deliberately, or simply forgotten at release?
- Do the six teatest doc claims predate the bubbletea v1→v2 migration, or was teatest aspirational and never used? (git blame not pulled.)
- `install.sh` / `scripts/test-install-sh.sh` were not read; behavior inferred from the CI job only.
- No v2.9 binary-size measurement recorded against the 10 MiB cap.
