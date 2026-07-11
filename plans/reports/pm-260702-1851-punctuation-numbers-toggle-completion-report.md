# PM Report: Punctuation & Numbers Toggle — Completion

Date: 2026-07-02 | Plan: `plans/260702-1824-punctuation-numbers-toggle/` | Status: completed

## Summary

4/4 phases complete, all success criteria checked off. Post-implementation
code review found 4 issues (1 high, 2 medium, 1 low); all fixed with user
sign-off on 2 scope/behavior decisions. Full suite green.

## Phases

| Phase | Status | Notes |
|---|---|---|
| 1 Config & Generator (TDD) | ✅ | `ApplyOptions` transform, deterministic, token-count-preserving |
| 2 Wiring Call Sites (TDD) | ✅ | Validation caught 4 factual errors in original draft (call chain), corrected before implementation |
| 3 Settings UI & CLI (TDD) | ✅ | 2 new toggle rows + `config get/set punctuation\|numbers` |
| 4 Docs Sync | ✅ | README + roadmap (v2.6.0), corrected stale "4 settings only" line |

## Code Review Findings (all resolved)

| Severity | Finding | Resolution |
|---|---|---|
| HIGH | Quote-wrap spec'd but not implemented | Implemented + tested (user chose to implement) |
| MEDIUM | `applyNumbers` off-by-one (5-digit possible) | Fixed (`IntN(max-1)+1`) |
| MEDIUM | Capitalize-after-`;` undocumented deviation | Kept, locked in with test (user chose to keep) |
| LOW | Stale "5 rows" doc comment | Fixed |

## Verification

`go test ./... -race -count=1`, `gofmt -l .`, `go vet ./...` — all clean, both
before and after review fixes.

## Files Changed

20 files across `internal/config`, `internal/words`, `internal/runner`,
`internal/ui`, `internal/app`, `internal/cli`, `README.md`,
`docs/project-roadmap.md`. Full list in plan.md's implementation notes.

## Unresolved Questions

None.
