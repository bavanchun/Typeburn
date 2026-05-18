---
phase: 5
title: "Integration Verify"
status: pending
priority: P1
effort: "1h"
dependencies: [2, 3, 4]
---

# Phase 5: Integration Verify

## Overview
Join the three parallel phases on the feature branch; prove the whole module
is green and side-effect-free before opening the PR. Read + test only — no
new feature commit (a fixup commit only if a defect is found).

## Requirements
- Functional: all of phase 2/3/4 commits present on
  `feat/v1.1.0-theme-packs-hygiene`; full suite green together.
- Non-functional: no regression in any package; CI-equivalent gates pass.

## Architecture
File ownership across phases 2/3/4 is disjoint (theme/config vs app/ui-notice
vs docs/word-wrap-comment) → commits land sequentially with no merge
conflict. This phase runs the exact CI gate locally and a manual smoke of
every theme on every screen.

## Related Code Files
- Modify: none (defect fixup only, on the owning phase's files, if needed)

## Implementation Steps
1. Confirm 3 commits on branch (theme packs, persistence notice, docs).
2. `make fmt` (no diff), `make lint` (`gofmt -l` empty + `go vet ./...`),
   `make test-race` (`go test ./... -race -count=1`) — the exact CI gate.
3. Coverage check: `go test ./internal/theme/ ./internal/config/
   ./internal/app/ ./internal/ui/ -cover` ≥ pre-change baselines
   (theme was 100%, ui 87.4%).
4. Manual smoke (`make run`): cycle all 8 themes in Settings; for each,
   visit Home/Typing/Result/History; confirm legibility (esp. the 2 light
   themes) and that the picker lists 8.
5. Force a persistence failure (unwritable XDG dir) once in a real run;
   confirm the notice appears, input still works, notice clears on key.
6. Side-effect sweep: `git diff main...HEAD --stat` — confirm only expected
   files changed; no public API/signature change except the intended
   `AppendHistory` error now consumed (callers unchanged externally).
7. If any defect: fix on the owning phase's files, amend/add a fixup commit
   with a clear message, re-run step 2.
8. Spawn `code-reviewer` (done by cook): acceptance criteria met, no
   regression in touchpoints, no unintended contract change, patterns
   followed, no new lint/type errors.

## Success Criteria
- [ ] `gofmt -l` empty, `go vet ./...` clean, `go test ./... -race -count=1`
  100% pass.
- [ ] Coverage ≥ baseline for theme/config/app/ui.
- [ ] All 8 themes legible on all screens (manual).
- [ ] Persistence notice verified in a real run.
- [ ] `git diff` shows only intended files; no unintended contract change.
- [ ] code-reviewer: no critical findings (or all addressed).

## Risk Assessment
- **Parallel commit ordering:** disjoint ownership makes order irrelevant;
  if a phase touched a shared file unexpectedly, this gate catches it via
  the diff-stat sweep (step 6).
- **Coverage regression from new untested code:** step 3 enforces baseline;
  add tests on the owning phase if below.
