---
phase: 6
title: "Integration Verify"
status: pending
priority: P1
effort: "1.5h"
dependencies: [5]
---

# Phase 6: Integration Verify

## Overview
Prove the whole module is green, regression-free, and the goldens for
time/words/quote are unchanged before the release PR. Read + test only
(fixup commit only if a defect is found).

## Requirements
- Functional: phases 2–5 commits present; full CI gate green together;
  manual smoke of a real `--text` file AND `--text -` stdin.
- Non-functional: no regression in any package; word_stream goldens
  byte-identical; coverage ≥ baseline.

## Architecture
Sequential TDD phases each committed; this joins + validates. Mandatory
subagents (per cook): `tester` (full suite+coverage), `code-reviewer` (diff
vs main against acceptance criteria + the brainstorm decisions).

## Related Code Files
- Modify: none (defect fixup only, on the owning phase's files)

## Implementation Steps
1. Confirm 4 impl commits on `feat/v1.2.0-code-mode`.
2. `make fmt` (no diff), `make lint`, `make test-race` (the CI gate) — 100%.
3. Coverage: theme/config/app/ui/typing + new `codetext` ≥ baselines
   (theme 100%, ui ~87%, typing ~95%, metrics ~93%); `codetext` high.
4. Manual smoke (`make build`): `./bin/typeburn --text <a real .go file>` →
   Code selectable → multi-line literal render, 2-col tabs, Tab/Enter match,
   viewport scrolls, completion on exact match, result shows, History lists
   it, not ★. `cat file | ./bin/typeburn --text -` works. No `--text` →
   Code disabled+hint, other modes unaffected. Oversized/empty/binary file
   → Code disabled with reason, no crash.
5. Side-effect sweep: `git diff main...HEAD --stat`; confirm
   `word_stream_renderer.go` + its goldens unchanged (or only the
   behaviour-neutral shared-styler extraction with goldens green); no
   `words`/`typing` new I/O import; no unintended exported-signature change.
6. Spawn `tester` then `code-reviewer` (parallel, read-only) with the
   brainstorm decisions + acceptance criteria as context. Address any
   critical/high; observational → note & proceed.
7. Fixup on the owning phase's files only if needed; re-run step 2.

## Success Criteria
- [ ] `gofmt -l` empty, `go vet` clean, `go test ./... -race -count=1` 100%.
- [ ] Coverage ≥ baseline; `codetext` well-covered.
- [ ] Manual smoke (file + stdin + no-text + bad-input) all pass.
- [ ] Goldens for time/words/quote unchanged; no pure-core I/O leak.
- [ ] code-reviewer: 0 critical (or all addressed).

## Risk Assessment
- Golden drift from a shared-styler extraction — step 5 catches; revert to
  duplicated styler if goldens move.
- Coverage regression from new untested branches (viewport edges, loader
  errors) — step 3 enforces; add tests on the owning phase.
- Manual-smoke-only items (real terminal viewport scroll) can't be unit-
  proven fully — call out residual visual risk explicitly in the report.
