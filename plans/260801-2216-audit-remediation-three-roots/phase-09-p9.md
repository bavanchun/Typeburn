---
phase: 9
title: "Verify, Docs, Release"
status: pending
priority: P2
effort: "1.5d"
dependencies: [2, 3, 4, 5, 6, 7, 8]
---

# Phase 9: Verify, Docs, Release

## Overview

Prove the roots are closed on an integration branch, delete the scaffolding that
tracked the debt, correct the documentation the audit found stale, then ship.

**Verification is a gate, not a step.** Red-team's objection stands: if the manual
pass runs after all eight phases are on `main`, a failure arrives when tags are
immutable and fix-forward is the only option. So this phase has two halves with a
hard boundary between them.

## Requirements

- Functional
  - `knownOverflow` is empty and deleted; the plausibility allowlist too.
  - Docs describe the code that exists.
- Non-functional
  - CI green; binary under cap; every Go file under 200 LOC.

## Architecture

### Half A — verify on an integration branch (gate)

Cut `integration/v2.9.0` from `main` after phases 2–8 merge. Everything below runs
there, **before** any release-prep PR exists.

**The manual pass is the point.** No agent in this plan has seen the live TUI —
this environment has no pty, so every measurement came from calling `View()`
directly. That is sound for structure and useless for judgement. v2.6.0 shipped a
Result redesign whose wide-terminal degradation was only caught from a
screenshot; v2.8.0 shipped a double-centring regression that a green suite *and* a
code review both missed because the guarding test asserted the wrong layer.

Cover: **80×24, 120×32, full-screen × {colour, `NO_COLOR`}**, plus
- a Time run (viewport scrolls, footer visible, caret always on screen)
- a 100-word run (downsampling path)
- a run scoring 100 wpm at 100% accuracy (the A2 geometry case)
- a run with the update hint present (the budget case)
- an AFK run (the notice is visible and the result is withheld)
- a Code-mode paste containing CJK

If any of these fails, fix on the integration branch and re-verify. Only then
proceed to Half B.

### Delete the scaffolding

`knownOverflow` was a ratchet, not a feature. When empty, delete the map, the
both-direction branch, and its doc comment, leaving a plain assertion that every
screen fits every supported size. Same for `assertPlausible`'s allowlist.

If any entry remains, **the plan is not done** — do not ship with a non-empty map
and do not relax the test. Every entry has an owning phase recorded in `plan.md`;
a leftover means that phase did not finish.

### Docs — the two "source of truth" files contradict each other and the code

| Doc | Correction |
|---|---|
| `CLAUDE.md:37` | `metrics.Compute(log, startMs, durationMs)` → actual `Compute(log, mode.Mode, endMs)`; `AFKTrim` → actual `TrimAFK`. **This is the always-loaded file and it is the wrong one** — `docs/codebase-summary.md:278,282` has both right. |
| `CLAUDE.md` | The Homebrew "prose TODO, intentionally not a dead YAML block" claim: `.goreleaser.yaml:95-116` is a **live** `homebrew_casks:` block with an existing tap and a token injected at `release.yml:88-97`. **An agent reading this would delete working release infrastructure.** |
| `CLAUDE.md:31`, `docs/system-architecture.md:131-132`, `docs/codebase-summary.md:9` | The layering invariant is stated as fact and is false: `internal/storage` → `internal/config` → bubbletea (`go list -deps ./internal/storage \| grep -c charm` = **9**). **Correct the sentences.** The earlier plan said "prefer an import-purity test" — that is unimplementable: a test cannot make a false statement true, and decoupling `storage` from `config` is unscoped. (`internal/theme` importing lipgloss is a sanctioned seam per CLAUDE.md, so only the `storage` half is a defect.) |
| `docs/project-overview-pdr.md` | Three releases stale: claims v2.5.1 is current stable, Bubble Tea v2.0.6, describes the pre-v2.6.0 Result screen, omits `update` from the CLI surface. |
| 5 files | A quote bucket **"Epic"** that does not exist — actual enum is `{QuoteShort, QuoteMedium, QuoteLong}` (`quotes.go:17-21`). `codebase-summary.md:294` invents an API member. |
| `docs/codebase-summary.md:451`, `:229` | "All files <200 LOC" false; the `internal/ui` file list names 45 files against an actual 68, three of which do not exist, and contradicts its own kebab-case naming 200 lines later. |
| `README.md:237`, `:219-224` | Understates `ci.yml` by five steps; data-paths table omits `$XDG_STATE_HOME`. |
| `internal/ui/screen_result.go:14` | Still describes "a WPM-over-time sparkline" for a screen that has drawn a braille dual-axis chart since v2.6.0. |

#### Doc claims re-verified against source before writing

Checked on current `main`, so step 8 corrects from evidence rather than prose:

```
internal/metrics/compute.go:43   func Compute(log []typing.Keystroke, mode mode.Mode, endMs int64) Result
internal/metrics/afk_trim.go:21  func TrimAFK(log []typing.Keystroke, m mode.Mode, endMs int64) ([]typing.Keystroke, int64)
go list -deps ./internal/storage | grep -c charm   -> 9
go list -f '{{join .Imports " "}}' ./internal/storage
    -> encoding/json fmt .../internal/config os path/filepath sort strconv time
.goreleaser.yaml:95              homebrew_casks:      (LIVE block, not a TODO)
internal/words/quotes.go:16-20   QuoteShort | QuoteMedium | QuoteLong   (no "Epic")
```

All four corrections in the table above are confirmed. The `internal/config`
import is the single edge carrying bubbletea into `internal/storage` — the
layering sentence must be rewritten to say so, not merely softened.

The Homebrew row is the highest-risk one: `homebrew_casks:` is live release
infrastructure with an existing tap and an injected token. Correct that sentence
early, independently of the rest of the docs sweep, so no future agent deletes it.

Plus the user-visible behaviour changes this plan introduces:
- **Consistency scores change for everyone** (partial-final-second correction) —
  document as an intentional correction with a worked before/after; stored history
  untouched.
- **AFK-trimmed runs are no longer saved** — document the rule and the notice.
- The Result screen looks different.
- `typeburn update` now handles Ctrl-C.

### Also here

- Split `internal/ui/screen_home.go` (208 LOC): move `startCmd` to
  `screen_home_actions.go`, mirroring `screen_typing_actions.go`.
- Sweep every Go file for the 200-LOC ceiling — `make lint` does **not** enforce
  it (gofmt + vet + notui grep only), so this is a manual check or it does not
  happen.
- Close `plans/260801-1747-result-responsive-layout/` as completed, recording that
  its Option A was superseded by Phase 6 and why.

### Half B — release

Only after Half A passes. Standard runbook: release-prep PR (CHANGELOG,
`.github/release-notes.md`, roadmap) → CI green → squash-merge → annotate and push
the tag on the merged commit in a **separate** push. Never `git push --follow-tags`.

Tags are immutable and the module proxy is append-only — a defect is fixed forward
with v2.9.1, never by re-tagging.

Version: **v2.9.0** (behaviour changes and a redesign, no breaking API).

## Related Code Files

- Modify: `internal/ui/frame_fits_test.go`, `internal/app/frame_fits_test.go`, `internal/metrics/plausibility_test.go` (delete scaffolding)
- Create: `internal/ui/screen_home_actions.go`; modify `screen_home.go`
- Modify: `CLAUDE.md`, `docs/system-architecture.md`, `docs/codebase-summary.md`, `docs/project-overview-pdr.md`, `docs/project-roadmap.md`, `README.md`, `CHANGELOG.md`, `.github/release-notes.md`, `internal/ui/screen_result.go` (comment)
- Modify: `plans/260801-1747-result-responsive-layout/plan.md` (close)

## Implementation Steps

**Half A — gate**
1. Cut `integration/v2.9.0` from `main`.
2. `go test ./... -race -count=1`, `go vet ./...`, `gofmt -l .` (must be empty).
3. `make lint && make size-check && make build`.
4. Confirm both allowlists are empty; delete them and simplify the tests.
5. Split `screen_home.go`; sweep every file for the 200-LOC ceiling.
6. **Manual pass** over the full matrix above. Capture output for the PR body.
7. Any failure: fix on the integration branch, return to step 2.

**Half B — release**
8. Correct every doc row above, verifying each claim against source as you write
   it — writing from memory is how the current drift happened.
9. CHANGELOG with the four user-visible behaviour changes called out.
10. Release-prep PR → CI green → squash-merge.
11. Tag `v2.9.0` annotated on the merged commit; push the tag separately.
12. Verify the published release: 7 assets, notes non-empty and matching the tag,
    one archive's checksum verified against `checksums.txt`.
13. Run `typeburn update` from an installed v2.8.1 to exercise Phase 7's work
    against a real download.
14. Close the superseded plan; write the journal entry.

## Success Criteria

- [ ] `go test ./... -race -count=1` green; `go vet` clean; `gofmt -l .` empty
- [ ] `make lint`, `make size-check`, `make build` pass
- [ ] Both allowlists **deleted**, not merely emptied
- [ ] Every Go file under 200 LOC (manual sweep — `make lint` does not check)
- [ ] Manual pass complete over the full matrix, output in the PR body — **before** release prep
- [ ] Every doc claim in the table corrected and traced to source
- [ ] CHANGELOG documents the consistency change, AFK withholding, redesign, and update fix
- [ ] v2.9.0 published; 7 assets; notes match the tag; checksum verified
- [ ] `typeburn update` exercised against the real release
- [ ] `plans/260801-1747-result-responsive-layout/` closed with its supersession recorded

## Risk Assessment

**Risk:** the automated gate is green and the manual pass is skipped — the exact
path by which every defect in this plan shipped.
**Mitigation:** the manual pass is a success criterion on the *integration branch*
and its output goes in the PR body, so skipping it is visible rather than silent.

**Risk:** docs get updated from this plan's prose rather than from source,
recreating the drift.
**Mitigation:** step 8 requires verifying each claim against source. The layering
sentences in particular must be written from `go list -deps` output.

**Risk:** a stable tag is immutable.
**Mitigation:** verify published assets before calling the release done; fix
forward with v2.9.1.

**Risk:** users read changed consistency scores as a regression.
**Mitigation:** CHANGELOG states it is a correction, shows a worked before/after,
and confirms stored history is untouched.

## Rollback

The integration branch is the rollback mechanism for Half A — nothing reaches
`main` until it passes. After Half B, rollback is fix-forward only (v2.9.1); tags
and published artifacts cannot be withdrawn. Phase 8's branch-protection change
must be reverted via `gh api` using the pre-state recorded in phase-08.
