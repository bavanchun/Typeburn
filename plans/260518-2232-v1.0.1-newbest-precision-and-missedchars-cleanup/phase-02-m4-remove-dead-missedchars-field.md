---
phase: 2
title: "m4 remove dead MissedChars field"
status: completed
priority: P2
effort: "0.5h"
dependencies: []
---

# Phase 2: m4 remove dead MissedChars field

## Overview

`metrics.Result.MissedChars` is hard-coded to `0` (`compute.go:78` — the
package has no target text to compute it from) and rendered as `missed 0` on
every result screen. Remove the field, its dead computation, and its display.
Pure deletion — no behavior change to any real metric. Runs in parallel with
Phase 1 (disjoint file set).

## Requirements

- Functional: `Result` no longer has `MissedChars`; result screen no longer
  shows a `missed` stat; nothing else changes (WPM/accuracy/consistency
  untouched — `MissedChars` was explicitly informational, `compute.go:76`).
- Non-functional: no dangling references anywhere (`grep MissedChars` empty
  post-edit); build/vet/gofmt/tests GREEN; files stay < 200 LOC.

## Architecture

```
compute.go:  remove field decl (l.20), dead `missed:=0` block + comment
             (l.72–78), and `MissedChars: missed,` (l.120)
screen_result_view.go: remove the "missed" label+value line (l.147)
*_test.go:   drop `MissedChars: 0,` struct-literal lines
```

No replacement stat is added (YAGNI — users never saw a meaningful value;
adding real missed-char tracking is a separate, larger change deferred to a
future minor, already noted in roadmap).

## Related Code Files

- Modify: `internal/metrics/compute.go` (remove field, dead calc, return entry)
- Modify: `internal/metrics/compute_test.go` (drop refs if any — verify)
- Modify: `internal/ui/screen_result_view.go` (remove the `missed` stat line ~147)
- Modify: `internal/ui/screen_result_test.go` (remove `MissedChars: 0,` l.25)
- Modify: `internal/ui/test_helpers_test.go` (remove `MissedChars: 0,` l.16)

## Implementation Steps

1. `compute.go`: delete the `MissedChars int` struct field; delete the
   `missed := 0` statement and its preceding 5-line explanatory comment
   (the `// Missed = ...` / `// MissedChars is informational ...` block);
   delete `MissedChars: missed,` from the returned `Result{...}`. Keep the
   `correct, incorrect, extra` counting loop (still used).
2. `screen_result_view.go`: delete the line rendering
   `labelStyle.Render("missed") + " " + valueStyle.Render(...MissedChars)`.
   Check the surrounding slice/join: ensure no trailing separator or empty
   stat cell is left (adjust the adjacent join so the stat row stays balanced).
3. Remove `MissedChars: 0,` lines from `screen_result_test.go` (l.25) and
   `test_helpers_test.go` (l.16). Grep `compute_test.go` and remove any ref.
4. Repo-wide guard: `grep -rn MissedChars .` (excluding `plans/`) → **empty**.
5. `go build ./... && go vet ./... && test -z "$(gofmt -l .)"`.
6. `go test ./internal/metrics/ ./internal/ui/ -race -count=1` GREEN. If a
   result-screen teatest golden contained the `missed` line, regenerate that
   golden (this is an intended, correct output change — confirm the diff only
   drops the `missed` cell, nothing else).
7. **Commit (this phase's deliverable):**
   `git add internal/metrics/compute.go internal/metrics/compute_test.go internal/ui/screen_result_view.go internal/ui/screen_result_test.go internal/ui/test_helpers_test.go [regenerated goldens]`
   then `git commit -m "refactor: remove always-zero MissedChars from metrics result and result screen"`
   (no AI refs; do not push — Phase 3 owns push/release).

## Success Criteria

- [ ] `MissedChars` removed from `Result`, computation, and result screen
- [ ] repo-wide `grep MissedChars` (excl. `plans/`) is empty
- [ ] result stat row still renders cleanly (no empty/dangling cell)
- [ ] metrics/ui tests GREEN with `-race`; any golden change is only the dropped cell
- [ ] build/vet/gofmt clean
- [ ] exactly one `refactor:` commit, only the owned files, not pushed

## Risk Assessment

- Result-row layout breakage if a separator is left → step 2 explicitly checks
  the join/balance; covered by the result-screen teatest.
- Hidden test reference → step 4 repo-wide grep gate is the backstop.
- Zero metric-correctness risk: `MissedChars` provably never influenced
  WPM/accuracy/consistency (`compute.go:76` states it; deletion is inert).
