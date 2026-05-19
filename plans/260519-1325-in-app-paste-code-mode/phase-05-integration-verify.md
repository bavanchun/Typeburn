---
phase: 5
title: "Integration Verify"
status: completed
priority: P1
effort: "1h"
dependencies: [4]
---

# Phase 5: Integration Verify

## Overview
Prove the whole module is green and regression-free before the release PR.
Read + test only (fixup commit only if a defect is found).

## Requirements
- Functional: phases 2–4 commits present; full CI gate green together;
  manual smoke of the in-app paste flow + the unchanged `--text` flow.
- Non-functional: no regression; codetext `Load` + Typing PasteMsg + all
  goldens unchanged; coverage ≥ baseline.

## Architecture
Sequential TDD phases each committed; this joins + validates. Mandatory
subagents (per cook): `tester` (full suite + coverage), `code-reviewer`
(diff vs main against acceptance criteria + the brainstorm decisions).
Note the v1.2.0 code-reviewer transient-auth quirk: it is read-only → if it
fails with "Not logged in" simply retry; never skip the gate.

## Related Code Files
- Modify: none (defect fixup only, on the owning phase's files)

## Implementation Steps
1. Confirm 3 impl commits on `feat/v1.3.0-in-app-paste`.
2. `make fmt` (no diff), `make lint`, `make test-race` (CI gate) — 100%.
3. Coverage: codetext / ui / app ≥ baselines (codetext ~90%+, ui ~86%+,
   typing ~95%); report deltas.
4. Manual smoke (`make build`):
   - No `--text`: Home→Tab to Code→Enter→paste screen; paste a snippet
     (terminal bracketed paste) → returns to Home, Code enabled → Enter →
     Code test runs on it; finishes → History lists it, not ★.
   - Invalid: paste empty / a huge (>cap) blob / binary → reason shown,
     stays on paste screen; a good paste then recovers; esc → Home.
   - `--text file`: unchanged (Code enabled, Enter starts; paste screen not
     reached for that run).
   - time/words/quote unaffected; paste while typing a normal test still
     feeds the engine (Typing PasteMsg branch intact).
5. Side-effect sweep: `git diff main...HEAD --stat`; confirm
   `internal/codetext` `Load` path + its prior tests unchanged in behaviour,
   Typing PasteMsg branch intact, no unintended exported-sig change beyond
   the intended `codetext.Normalize` addition + any documented internal ones.
6. Spawn `tester` then `code-reviewer` (parallel, read-only) with the
   brainstorm decisions + acceptance criteria as context. Address
   critical/high; observational → note & proceed.
7. Fixup on the owning phase's files only if needed; re-run step 2.

## Success Criteria
- [ ] `gofmt -l` empty, `go vet` clean, `go test ./... -race -count=1` 100%.
- [ ] Coverage ≥ baseline.
- [ ] Manual smoke (paste valid/invalid/esc + `--text` + normal modes) pass.
- [ ] `Load` behaviour + Typing PasteMsg + goldens unchanged.
- [ ] code-reviewer: 0 critical (or all addressed).

## Risk Assessment
- Bracketed paste not firing in some terminal during manual smoke — note as
  environment-specific; the routed-PasteMsg unit test is the deterministic
  proof; document residual terminal-dependent risk.
- Chunked-paste behaviour — verify the "last PasteMsg wins" decision holds in
  the real terminal; if a real terminal chunks, record as a known limitation
  for v1.3.x (do not silently change scope).
- Coverage dip from new screen branches — add tests on the owning phase.
