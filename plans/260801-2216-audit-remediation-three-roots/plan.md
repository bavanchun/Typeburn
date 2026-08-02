---
title: "Audit Remediation: Three Root Causes"
description: >-
  Close the three structural root causes behind the audit's findings, plus the
  Result A2 redesign, the CLI/update hardening, and the supply-chain gaps.
  Restructured after red-team.
status: in-progress
priority: P1
effort: "8-10d"
tags: [correctness, ui, metrics, storage, cli, ci]
blockedBy: []
blocks: []
created: 2026-08-01
---

# Audit Remediation: Three Root Causes

## Overview

A five-scope audit (`plans/reports/code-review-260801-2002-codebase-audit.md`)
produced ~77 findings including 5 blockers. Most reduce to three structural
causes:

| Root | What it is |
|---|---|
| **R1** | No test asserts rendered/computed output against *reality* — only that a component agrees with itself |
| **R2** | No bounds at input boundaries |
| **R3** | The typing viewport is wired only for the Code branch |

**R1/R2/R3 do not cover everything, and this plan does not pretend they do.**
Red-team established that the reduction omitted an entire audit scope. Phase 7
now owns `internal/cli`/`internal/update` (B1–B4), and the deferral list below is
explicit. "77 findings → 3 roots" was achieved partly by not counting; that is
corrected here.

## Corrections applied after red-team

Recorded because each one falsified a claim this plan previously made:

1. **`golang.org/x/text` is NOT on the render path.** `go mod why -m` →
   `cmd/typeburn → fang → golang.org/x/text/cases` (help-text title casing);
   `go list -deps charm.land/lipgloss/v2 | grep -c x/text` = **0**. The
   govulncheck trace through `lipgloss.Style.Render` is a sound
   over-approximation through interface dispatch, not a real call path. The
   vulnerability is real and still worth fixing; the urgency framing was wrong.
2. **A2's zone widths were derived from a 2-digit sample — the same
   methodological error that produced the glyph bug.** Measured (ANSI stripped):
   `BigDigits(87)=17`, `(96)=19`, `(100)=22`, `(200)=26`. The specified
   `17 + 6 + 19 + 6 + 40 = 88` breaks at 100 wpm / 100% accuracy:
   `22 + 6 + 24 + 6 + 40 = 98 > 88`, and `TestLayoutFor_MatchesRenderedWidth`
   asserts exact panel width — a hard failure, not a cosmetic one. Zone widths
   must be **derived at render time**, never hardcoded. Phase 6 opens with a
   falsification gate.
3. **A2's 24-row budget ignored two rows that already exist.**
   `screen_result_view.go:36-43` reserves an `updateLine` row when an update is
   available, and Phase 3 makes a persistence/AFK notice load-bearing. The
   budget must be proven with both present.
4. **The golden fixture cannot see the glyph defect it baked in.**
   `result_layout_test.go:38` uses `NetWPM: 74` — no `0`, no ragged `3/6/9`. The
   glyph fix would produce an *empty* golden diff, and Phase 1's criterion
   "golden diff contains digit changes only" would pass vacuously.
5. **`knownOverflow` as originally specified could never reach empty**, so the
   phase that asserts it is empty was unreachable. Fixed by scoping the matrix to
   supported sizes and assigning owners to every measured overflow.
6. **Phase 3 was three phases.** Split into 3, 4, 5.
7. **Import-purity test is unimplementable as a doc fix.** `internal/storage` →
   `internal/config` → bubbletea is real (`go list -deps ./internal/storage |
   grep -c charm` = 9). A test cannot make a false sentence true; decoupling is
   unscoped. Phase 9 corrects the sentences instead.

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | A test proves every screen fits its terminal at every *supported* size | P1 |
| 2 | No metric can report an impossible value, and none is persisted | P1 |
| 3 | Every input boundary is bounded before allocation | P1 |
| 4 | Typing windows its viewport in every mode, never splitting words, correct for wide runes | P1 |
| 5 | History survives concurrent writers and corrupt files | P1 |
| 6 | Result fills its width with information and fits 80×24 | P2 |
| 7 | Ctrl-C during an update does not brick the update feature | P2 |
| 8 | The known vulnerability is gone and CI gates what it claims | P2 |

## Explicitly deferred (not silently dropped)

| Finding | Why deferred |
|---|---|
| C4 `release.yml` parity with `ci.yml` | Release infra; CLAUDE.md requires `ci.yml` byte-stability when touching it. Separate deliberate change. |
| C5 `notui-noexit-check` is a defeatable grep | Needs an AST/import-graph assertion — different kind of work. |
| C6 Windows CI / Windows targets | Open question 1: add CI or drop the targets. Product call. |
| Tier 4 UX items (Home modifier row, History filter/de-chrome, first-run states, Code Paste footer, vertical-slack split) | Real, but a separate UX plan. Phase 6 covers only the Result work. |
| `.claude/package-lock.json` untracking | Hygiene, no functional effect. |
| A4 strict-mode `Errors` semantics | Open question 2 — product call. |
| LOW batch (`ExitCode` `errors.As`, `archive.go` decompress cap, `classifyInstall`, `config set` second mutation, `for_mode.go` default) | Phase 7 takes the two in its files; the rest is opportunistic. |

## Key decisions (user-confirmed)

**AFK-trimmed runs are ineligible for best and are not persisted** — the
Monkeytype behaviour, via the existing `storage.EligibleForBest` seam. The
withholding must be **visible**; silently discarding a result is worse than the
bug being fixed.

**The A2 Result redesign is in scope** — but its geometry is re-derived, not
inherited (correction 2).

**Supply-chain fixes are in scope.**

## Key design decisions

**R1 lands as a ratchet, corrected.** `main` is protected and CI must stay green,
so failing tests cannot land. `knownOverflow` enumerates currently-failing cases.
Three corrections from red-team:
- **Matrix scoped to supported sizes** (`w>=60, h>=20`, the app's own degraded
  gate at `model_view.go:34`). Below that the app deliberately renders
  `DegradedNotice`, so asserting fit there is asserting against a configuration
  the product refuses.
- **Key on the failing dimension and its measured value**, not a boolean. A
  width-only failure that becomes a height failure at the same key would
  otherwise stay "correctly listed" and hide a regression.
- **No phase numbers in the map values** — `.claude/rules/review-audit-self-decision.md`
  forbids plan IDs in code artifacts. Values record the measured overflow; the
  plan maps entries to phases.

**Every measured overflow has an owner.** The orphan files red-team found
(`footer.go`, `settings_rows.go`, `history_table.go` rules, `typing_log_helpers.go`,
`mode_header.go`, `notui/runner.go`, `runner/session.go`, `screen_typing.go`,
`model_key_handler.go`, `celebration.go`, `screen_result.go`, `stat_card.go`) are
assigned in the ownership table below.

## Phases

| # | Phase | Status | Depends on | Group |
|---|-------|--------|------------|-------|
| 1 | [Correctness Harness And Glyph Fix](./phase-01-start.md) | Done | — | A |
| 8 | [Supply Chain And CI Gates](./phase-08-p8.md) | Done | — | A |
| 2 | [Typing Viewport, Wrapping, Display Width](./phase-02-p2.md) | Pending | 1 | B |
| 3 | [Metrics And Typing Correctness](./phase-03-p3.md) | Pending | 1 | B |
| 4 | [Input Bounds And Layout Overflows](./phase-04-p4.md) | Pending | 1 | C |
| 5 | [Storage Integrity](./phase-05-p5.md) | Pending | 1 | C |
| 7 | [CLI And Update Hardening](./phase-07-p7.md) | Pending | — | C |
| 6 | [Result A2 Redesign](./phase-06-p6.md) | Pending | 1, 3 | D |
| 9 | [Verify, Docs, Release](./phase-09-p9.md) | Pending | all | E |

Groups run in order; phases within a group run concurrently.
**A = {1, 8} → B = {2, 3} → C = {4, 5, 7} → D = {6} → E = {9}.**

Group B is only two phases because 2 and 3 both reach into `internal/ui` and
`internal/typing`; adding a third concurrent editor there is how the collisions
red-team found would recur.

### File ownership (parallel safety)

No two phases in the same group may touch the same file. Verified against the
repo, not asserted.

| Phase | Owns |
|---|---|
| 1 | `internal/ui/render_harness_test.go`, `internal/ui/frame_fits_test.go` + `frame_fits_known_overflow_test.go`, `internal/app/frame_fits_test.go` + `frame_fits_cases_test.go` + `frame_fits_known_overflow_test.go`, `internal/metrics/plausibility_test.go`, `internal/ui/ascii_big_digits.go` + its test, `internal/ui/testdata/big_digits_strip.txt`, `internal/ui/testdata/result_baseline_*.txt` (glyph regeneration), `internal/ui/result_layout_test.go` (fixture WPM only) |
| 2 | `internal/ui/word_stream_renderer.go`, `word_stream_anim.go`, `screen_typing_view.go`, `code_stream_renderer.go`, `screen_typing.go` (tick-guard), their tests |
| 3 | `internal/metrics/*` (except `plausibility_test.go`), `internal/typing/engine.go`, `internal/typing/completion.go`, `internal/ui/typing_log_helpers.go`, `internal/ui/mode_header.go`, `internal/cli/notui/runner.go`, `internal/runner/session.go`, `internal/storage/new_best.go`, `internal/app/model_history.go`, `internal/app/model_view.go` |
| 4 | `internal/codetext/codetext.go`, `internal/cli/cmd_replay.go`, `internal/words/generator.go`, `internal/words/for_mode.go`, `internal/ui/screen_history_view.go`, `internal/ui/history_table.go`, `internal/ui/footer.go`, `internal/ui/screen_settings_view.go`, `internal/ui/settings_rows.go` |
| 5 | `internal/storage/history_store.go`, `internal/storage/atomic_write.go` |
| 6 | `internal/ui/screen_result_hero.go`, `screen_result_view.go`, `screen_result.go`, `screen_result_reveal.go`, `result_layout.go`, `result_comparison_rail.go` (new), `result_context.go` (new), `result_graph.go`, `result_graph_axes.go`, `stat_card.go`, `celebration.go`, `internal/ui/testdata/result_baseline_*.txt` (layout regeneration), Result test files |
| 7 | `internal/cli/cmd_update_run.go`, `cmd_update.go`, `exitcodes.go`, `cmd/typeburn/main.go`, `internal/update/lock.go`, `download.go`, `check.go`, `replace_unix.go`, `replace_windows.go`, `archive.go` |
| 8 | `go.mod`, `go.sum`, `.github/workflows/ci.yml`, `.github/dependabot.yml` (new), `.gitignore`, `Makefile`, branch-protection settings |
| 9 | docs, `CHANGELOG.md`, `.github/release-notes.md`, `internal/ui/screen_home.go` + `screen_home_actions.go` (new), scaffolding deletion |

**Sequenced couplings, deliberate:**
- `internal/app/model_history.go` — Phase 3 owns it; Phase 6 depends on 3.
- `internal/ui/frame_fits_known_overflow_test.go` — Phase 1 creates it; **phases
  2, 4, 6 each delete their own entries**. They are in different groups, so no
  concurrent edit. Phase 3 does not touch it (its work is not layout). The map
  lives in its own file, separate from the assertions that police it, so the
  debt list shrinks without either file crossing the 200-LOC ceiling. The same
  split exists in `internal/app/` for the root-frame map.
- `internal/ui/testdata/result_baseline_*.txt` — Phase 1 regenerates for glyphs,
  Phase 6 for layout. Different groups. The plan's earlier "once, not twice"
  claim was false; twice is correct and safe because they are sequenced.
- Phase 8 touches `go.mod`; it merges in group A so later phases rebase once.

## Constraints

- Protected `main`: branch → PR → green `ci.yml` → squash-merge. One PR per phase.
- **200-LOC ceiling, and two files are already at the edge:** `internal/typing/engine.go`
  is 197 and `internal/app/model.go` is 199. Phase 3 must split `engine.go` as part
  of its work, not as an afterthought. `make lint` does NOT enforce this — it is
  gofmt + vet + the notui grep only.
- `NO_COLOR`/mono layout-identical; reveal invariants hold; `theme.Role` only, no hex.
- No information carried by colour alone.
- Allowed deps unchanged.
- **Rollback:** every phase records how to revert. Phase 8's branch-protection
  change is external state, not `git revert`-able — its pre-state is recorded in
  phase-08 verbatim before the write.

## Success Criteria

- [ ] `go test ./... -race -count=1`, `go vet ./...`, empty `gofmt -l .`
- [ ] `make lint && make size-check && make build`
- [ ] `knownOverflow` empty and deleted; every entry had an owning phase
- [ ] Glyph table rectangular, no two digits equal, and the golden fixture exercises `0` and a ragged digit
- [ ] No impossible WPM persisted; AFK runs withheld with a visible notice
- [ ] Cross-package assertion covers `TrimAFK → Compute → IsNewBest → AppendHistory`
- [ ] Typing windows correctly, splits no word, and is correct for CJK/emoji
- [ ] Two subprocesses × 60 appends → 120 records
- [ ] Result fits 80×24 **with the update hint and a notice present**
- [ ] Ctrl-C during a plain update leaves no lock; a stale lock self-heals
- [ ] `govulncheck` in CI with a defined stdlib policy; all `ci.yml` jobs required
- [ ] Manual pass at 80×24, 120×32, full-screen × {colour, NO_COLOR} — **before** release prep
- [ ] Every Go file under 200 LOC

## Relationship to prior work

Supersedes `plans/260801-1747-result-responsive-layout/` (unfinished; its Option A
shipped as v2.8.0 and did not solve the emptiness). Close it when Phase 6 merges.

## Decisions resolved (were blocking)

- **Typing viewport scales with terminal height** — `clamp(3, (h-4)/2, 7)`. 80×24
  → 3 lines, 120×50 → 7. Accepted trade-off: reading rhythm differs between a
  laptop and an external monitor. Phase 2.
- **`govulncheck` fails on non-stdlib findings only**, with stdlib findings
  surfaced as a visible job-summary warning. A stdlib advisory is fixed by a
  toolchain bump, not by the author of an unrelated PR. Phase 8.

## Open questions — resolved

All four blocking questions are answered. Recorded here because each one binds a
later phase's implementation.

1. **Windows: leave as-is.** No `windows-latest` job, no change to the published
   targets. The mismatch — `install.sh` refuses Windows while two Windows zips
   ship — stays a known, deliberate gap rather than being closed by guesswork in
   a supply-chain phase. Phase 8 shipped without touching it.
2. **Strict-mode `Errors`: keystroke-level.** Every wrong key counts, including
   ones the user corrected. Consistent with `KeystrokeAccuracy`, which strict
   runs already use, and it is what "strict" means. Note the user-visible
   consequence: a strict run now reports a *higher* `Errors` than the same run
   would today, because final-state counting is near-zero under strict — the
   cursor is blocked on a wrong key, so few errors survive to the final state.
   Phase 3.
3. **`--words` upper bound: 10 000.** Matches `codetext`'s existing 10 000-rune
   cap, so both input boundaries carry the same number. Phase 4.
4. **Rank in the A2 rail: bucket-scoped**, comparing within the same
   mode+length. `#2 of 6` on a fresh profile is accepted — a rank across
   incomparable buckets (time-15 against a quote run) would be worse than a
   small honest one. Phase 6.
5. **If A2's budget cannot absorb the update-hint row**, does the hint move, or
   does Result get a dedicated height change? Phase 6's Gate 0 decides during
   execution; no answer needed up front.
