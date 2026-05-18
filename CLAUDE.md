# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Typeburn is a Monkeytype-style terminal typing test: Go 1.26 + Bubble Tea v2 / Lip Gloss v2, single binary, no backend, local XDG persistence.

## Commands

```sh
make build       # ldflags-stamped binary → ./bin/typeburn
make run         # go run . (launches TUI)
make test        # go test ./...
make test-race   # go test ./... -race -count=1   ← the CI gate; must be GREEN
make lint        # gofmt -l check (must be empty) + go vet ./...
make version     # build then print the resolved --version banner

# Single test / package
go test ./internal/metrics/ -run TestCompute -count=1
go test ./internal/version/ -run TestResolve_LdflagsWin -v
```

`go test ./... -race -count=1`, `go vet ./...`, and an empty `gofmt -l .` are exactly what CI enforces — run all three before considering work done. UI tests use `teatest` golden files; pure packages are table-driven with real data (no mocks).

## Architecture

**Strict dependency layering — do not violate.** The *pure-logic* packages are UI-free and must stay that way (no `bubbletea`/`lipgloss` imports):

- Pure logic (no UI deps): `internal/typing` (keystroke state machine), `internal/metrics` (WPM/accuracy/consistency formulas), `internal/words` (embedded wordlist + quote pack), `internal/storage` (atomic JSON persistence), `internal/version` (build stamp).
- Styling/input boundary (intentionally not reusable-core): `internal/config` binds Bubble Tea key types for its keymap, and `internal/theme` returns Lip Gloss styles/colors by design. These two depend on `charm.land` libs deliberately — they are the seam between pure logic and the TUI, not general-purpose libraries. Do not "fix" this by removing the imports; do not add new UI deps to the pure-logic packages above.
- `internal/ui` depends on the packages above and implements the five screen sub-models + reusable components.
- `internal/app` is the root Bubble Tea Elm model that wires everything together.

**Elm message flow.** `app.Model` owns a `Screen` enum and five sub-models. Screen sub-models in `internal/ui` emit *domain* messages — `StartTestMsg`, `ResultMsg`, `AbortMsg`, `NavHistoryMsg` (defined in `internal/ui/messages.go`). The root model's `Update` routes them, owns screen transitions, and is the *only* place that persists results (`AppendHistory`) and computes new-best. Sub-models never touch storage directly. To add a screen interaction, emit a message from the sub-model and handle routing/side-effects in `internal/app`.

**Metrics derive entirely from the keystroke log.** `typing.Engine` only records keystrokes (`Apply`/`Backspace`); nothing computes WPM live. `metrics.Compute(log, startMs, durationMs)` replays the log post-hoc. Never add live metric mutation — extend the log/replay path instead. `AFKTrim` (Time mode, >7s trailing idle) runs before compute.

**Theme is a semantic `Role` enum, never hex.** UI code asks for `theme.Style(RoleX)` / `theme.Color(RoleX)`. `NO_COLOR` and the `mono` (attribute-only) theme are first-class and must keep layouts identical (only attributes change). Adding a color means adding a `Role` and mapping it in every theme, not a literal in UI code.

**Storage is defensive.** Atomic temp-file + rename; any corrupt/missing file returns safe defaults and never panics; history is capped at the 200 newest records. Settings load once at startup (`app.NewFromDisk()`); history loads on demand and after each test.

**Versioning is hybrid.** `internal/version` reads ldflags-injected `Version/Commit/Date` (set by Makefile + GoReleaser); when empty (bare `go install`) it falls back to `debug.ReadBuildInfo()`, final fallback `"dev"`. `main.go`'s `decide()` is a pure, tested function: a single `--version` short-circuits; a `ContinueOnError` FlagSet with discarded output ensures unknown flags / `-h` / `-v` / positional args fall through to the TUI (no `os.Exit(2)`, no usage dump). `-v` is intentionally unbound (reserved for a future `--verbose`).

## Git Workflow (protected main — enforced)

`main` is protected. **Never commit or push directly to `main`.** This is hard-enforced two ways: GitHub branch protection (PR required, direct push denied, `ci.yml` must pass, linear history) and a local PreToolUse hook that blocks `git commit`/`git push` to main in this repo.

Every change — code, docs, config, release prep — follows:

1. Branch off `main`: `feat/…`, `fix/…`, or `chore/…`.
2. Commit on the branch (conventional commits, no AI references).
3. Push the branch; open a PR to `main`.
4. `ci.yml` must be green; **squash-merge** the PR (squash is the only enabled merge mode; branch auto-deletes).
5. Tags are cut on `main` **only after** the PR is merged. Tag pushes are allowed by the hook/protection (only branch refs are protected).

**Release runbook is now PR-based:** branch → commit phases → push branch → PR → squash-merge → `git tag` the merged commit on main → push tag → `release.yml`. The disposable-dry-run / fix-forward / never-re-tag invariants are unchanged (see `CONTRIBUTING.md`).

## Conventions & Constraints

- **File size:** keep every Go file < 200 LOC. Split by concern (`screen_x.go` / `screen_x_view.go` / `screen_x_actions.go` / `screen_x_test.go`). Core logic uses `snake_case` filenames; small utility/output modules use `kebab-case`.
- **Module path is case-sensitive:** `github.com/bavanchun/Typeburn` (capital `T`). ldflags `-X` targets and `go install` both depend on this exact casing.
- **No new dependencies** for core behavior without strong justification — the app deliberately uses only Bubble Tea / Lip Gloss + stdlib.

## Release Engineering (read before touching release files)

- **GoReleaser is pinned to exactly `v2.15.4`** in three places that must stay in lockstep: `.goreleaser.yaml` validation, `release.yml` `version:`, and `CONTRIBUTING.md`. Bump all three together, deliberately.
- **`ci.yml` does NOT trigger on tag pushes** (branches/PR only). The tagged commit therefore has zero CI unless `release.yml` provides it — that is why `release.yml` runs its own least-privilege `test` job that gates the `contents: write` publish job. Keep `ci.yml` byte-identical when working on release infra.
- **Non-obvious GoReleaser trap:** `changelog.disable: true` *also* makes GoReleaser ignore `--release-notes` and publish an empty release body. This repo deliberately uses a `changelog.filters.exclude: ['.*']` filter instead, with curated notes supplied via `.github/release-notes.md`. Do not "simplify" this back to `disable: true`.
- **Tags are immutable / fix-forward only.** The Go module proxy + sumdb are append-only: never delete-and-re-tag a version that was pushed (it becomes permanently uninstallable). A defect in a release is fixed by cutting the next patch (`v1.0.1`). The only safe delete-and-retry is the unadvertised disposable `v0.0.0-rc.test` dry-run tag.
- **Release procedure:** push `main`, run the disposable pre-release tag dry-run through `release.yml`, verify assets/notes/checksums, delete the dry-run (`gh release delete --cleanup-tag`), then annotate `v1.0.0` on the exact dry-run-proven SHA and push the tag in a *separate* push (never `git push --follow-tags`). Expected published assets = 7 (6 archives + `checksums.txt`); `release.yml` asserts this.
- Release binaries are **unsigned** by design (v1 scope); integrity = HTTPS + `checksums.txt` only (see `SECURITY.md`). Homebrew is a documented prose TODO in `CONTRIBUTING.md`, intentionally not a dead YAML block.

## Documentation

`docs/` is the source of truth: `codebase-summary.md` (per-package detail), `system-architecture.md`, `project-roadmap.md` (M2 new-best-precision is the tracked fast-follow), `code-standards.md`, `project-overview-pdr.md`. Update them when behavior or structure changes. Plans live in `plans/`; session journals in `docs/journals/`.
