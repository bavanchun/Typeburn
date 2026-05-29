---
phase: 3
title: "Hygiene Cleanup"
status: completed
priority: P3
effort: "1h"
dependencies: []
---

# Phase 3: Hygiene Cleanup

## Overview
Three low-risk hygiene items: fix a misleading comment, split `app/model.go` back under 200 LOC, and formally document the `effWPM` zero-sentinel as reviewed-and-accepted (no code/schema change).

## Requirements
- **completion.go comment:** comment must describe the actual position-based completion logic.
- **app/model.go:** < 200 LOC (currently 204), without changing behavior.
- **effWPM:** no functional/schema change; record the review decision so it isn't re-flagged.

## Architecture

### 3a. completion.go comment (`internal/typing/completion.go:~54`)
Logic counts a word complete when its trailing space position `i < len(typed)` — correct Monkeytype position-based progress. The inline comment ("we need typed[i] to exist and be a space") overstates the check (it does not verify the char IS a space). **Fix:** reword comment to "word counts complete once typing has advanced past its trailing-space position" — comment-only, zero logic change. Existing `completion`-related tests must stay green (no edit needed).

### 3b. app/model.go split (204 → <200 LOC)
`model.go` holds: `Screen` enum + const block (lines 16-26), `Model` struct (28-68), `New` (70-86), `Init` (88), `Update` (92-...). Sibling files already exist (`model_key_handler.go`, `routing.go`, `model_view.go`, etc.). **Fix (KISS):** move the `Screen` type + its `const (...)` enum into a new tiny `internal/app/screen.go`. That removes ~12 lines from model.go → under 200, with no behavior change and a self-documenting filename. Do NOT reshuffle `Update`/routing.

### 3c. effWPM — documented, NOT changed (`internal/storage/new_best.go:41`)
`if r.NetWPM == 0 { return float64(r.WPM) }` is a legacy-compat fallback (records pre-`NetWPM` unmarshal it as 0.0). A real run of all-wrong chars also yields NetWPM 0.0, but `WPM` is then 0 too, so the returned value is correct. "Hardening" to distinguish the two would require a presence flag → a **storage schema change for a non-bug** (YAGNI, risks serialization). **Decision: keep as-is.** Add one clarifying sentence to the existing comment noting the 0.0-vs-legacy ambiguity is benign because both fields are 0 in that case. Update `docs/project-roadmap.md` Known-Limitations note if warranted.

## Related Code Files
- Modify: `internal/typing/completion.go` (comment only)
- Create: `internal/app/screen.go` (move `Screen` enum + const)
- Modify: `internal/app/model.go` (remove moved enum)
- Modify: `internal/storage/new_best.go` (comment only)
- Read for context: existing `internal/app/*.go` to confirm `Screen` references resolve across the package (same package, no import change needed)

## Implementation Steps
1. Reword `completion.go` comment; run `go test ./internal/typing/ -race`.
2. Create `internal/app/screen.go` with `package app`, move `type Screen int` + the `const (...)` block; delete from `model.go`. Build + `go test ./internal/app/ -race`.
3. Confirm `wc -l internal/app/model.go` < 200.
4. Add clarifying sentence to `effWPM` comment; no logic change.
5. Full `go test ./... -race`; `gofmt -l .` empty; `go vet ./...`.

## Success Criteria
- [ ] `completion.go` comment matches behavior; typing tests green.
- [ ] `internal/app/model.go` < 200 LOC; `screen.go` holds the enum; app tests green; behavior unchanged.
- [ ] `effWPM` comment clarified; no schema/logic change; storage tests green.
- [ ] Full `-race` green; gofmt empty; vet clean.

## Risk Assessment
- **Low.** Comment edits are zero-risk. The enum move is a same-package relocation (no import cycle, no API change). effWPM intentionally untouched to avoid a risky storage-format change — this is the brutally-honest YAGNI call; surfaced to user, who approved "all" with this caveat.
