# Brainstorm Summary — Update-UX Polish

**Date:** 2026-05-30
**Topic:** Improve the `typeburn update` self-update UX (post-v2.3.0)
**Outcome:** Approved scope = #1 + #2 + #3. Next step = `/ck:plan --tdd`.

## Problem Statement

`typeburn update` shipped in v2.3.0 but its UX has gaps. The most damaging: the
app's own in-app nudge points users to a check-only command, not the new
self-updater — so users never discover the feature exists. Two further polish
gaps (silent download, no "what changed") reduce trust during the swap.

## Audit Findings (ranked)

### 🔴 #1 — In-app hint points to the wrong command (DEFECT)
- `internal/ui/screen_result_view.go:75`:
  `↑ %s available — run "typeburn version --check-update"`
- `version --check-update` is **detect-only** — a dead end that re-reports
  availability without upgrading. Predates the `update` command.
- **Fix:** point the hint at `typeburn update`.
- **Blast radius:** the muted footer line on the Result screen; NO_COLOR/mono
  layouts must stay identical; a teatest golden file almost certainly captures
  this line → golden must be regenerated.

### 🟠 #2 — No progress feedback during the swap
- `runUpdate` (`cmd_update.go:123-124`) prints `updating X → Y ...` then blocks
  silently inside `update.Apply` (download ~9 MB → SHA-256 verify → extract →
  atomic swap). Terminal looks frozen; risk of mid-download `ctrl-c`.
- **Fix options (decide in plan):**
  - (a) Add a progress-reporter callback / `io.Writer` status param to `Apply`
    (or a thin wrapper) so the CLI prints `downloading… / verifying… /
    installing…` step lines. Stdlib-only, non-TUI, KISS-friendly.
  - (b) Spinner via `bubbles/spinner` — heavier for a non-TUI command; not
    preferred.
- **Constraint:** `Apply` already takes 6 params; prefer a single reporter
  rather than widening further, or a small wrapper in `internal/cli`.

### 🟡 #3 — "What changed?" never surfaced
- Update messages give a version but no changelog. Append the release URL
  (`…/releases/tag/v<latest>`) to the available/updated messages so users can
  read notes.
- **Check:** does `update.Result` already carry an HTML/release URL, or must the
  CLI compose it from the tag + repo prefix? (researcher/scout to confirm.)

### 🟢 #4 — Deferred (YAGNI)
- `--check` always exit-0 — intentional (mirrors `version --check-update`); skip.
- In-TUI "press U to update" — scope creep for a typing app; skip.

## Approved Scope

Act on #1 + #2 + #3 together as one "update-UX polish" change. Skip #4.

## Constraints (from CLAUDE.md)

- Stdlib-only in `internal/update`; charm allowed in UI/CLI seams.
- Every Go file <200 LOC.
- NO_COLOR + mono themes: layout identical, attributes only.
- Update trust model unchanged (checksum-only, unsigned) — do not touch.
- Protected main: branch → PR → squash-merge.

## Touchpoints

- `internal/ui/screen_result_view.go` (#1) + its teatest golden.
- `internal/cli/cmd_update.go` (#2, #3) + `cmd_update_test.go`.
- `internal/update/apply.go` and/or a new reporter seam (#2) + tests.
- `internal/update/result.go` / `check.go` (#3 — release URL source).
- Docs: `CHANGELOG.md`, README/usage if update output changes.

## Success Criteria

- Result-screen hint reads `run "typeburn update"`; golden updated; NO_COLOR/mono
  unchanged.
- `typeburn update` emits visible step feedback during download/verify/install.
- Update available/updated messages include the release URL.
- `go test ./... -race`, `go vet`, `gofmt -l` all clean; size-check passes.

## Open Questions

1. #2 feedback: reporter-callback into `Apply` vs CLI-side wrapper — plan to pick
   the lower-LOC option that keeps `internal/update` stdlib-only.
2. #3: confirm whether `update.Result` exposes a release URL or it must be
   composed from tag + repo prefix.
