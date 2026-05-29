---
phase: 4
title: "Docs and Verification"
status: completed
priority: P3
effort: "1h"
dependencies: [1, 2, 3]
---

# Phase 4: Docs and Verification

## Overview

Update source-of-truth docs and run the full CI gate. Closes the feature out
for a PR.

## Requirements

- **Functional:** docs reflect the new heatmap; CI gate fully green.
- **Non-functional:** docs concise, consistent with existing style.

## Architecture

Docs are source of truth (per CLAUDE.md). Touch the two that describe metrics
behavior + release history. No code architecture change in this phase.

## Related Code Files

- Modify: `docs/codebase-summary.md` (metrics package entry — add `KeyHeatmap`
  + `Result.KeyMisses`; note Result/CLI surfacing)
- Modify: `docs/project-roadmap.md` (add a shipped entry under Post-1.0 once
  released, or a "Next" → in-progress note pre-release)
- (No CLAUDE.md change required — no new constraint or layering rule introduced.)

## Implementation Steps

1. Update `docs/codebase-summary.md` metrics section: document `KeyHeatmap`,
   the `KeyMiss` type, and the `Result.KeyMisses` field; mention it surfaces on
   the Result screen and in CLI `key_misses`.
2. Update `docs/project-roadmap.md` with the heatmap entry (version TBD at ship).
3. Run the full CI gate locally (exactly what `ci.yml` enforces):
   - `go test ./... -race -count=1` → GREEN
   - `go vet ./...` → clean
   - `gofmt -l .` → empty
   - `make size-check` → within binary size cap
4. `make build && make version` sanity check (binary builds, banner resolves).

## Success Criteria

- [ ] `docs/codebase-summary.md` + `docs/project-roadmap.md` updated.
- [ ] `go test ./... -race -count=1` GREEN.
- [ ] `go vet ./...` clean; `gofmt -l .` empty.
- [ ] `make size-check` passes.
- [ ] Branch + PR ready (feat/key-error-heatmap → PR → squash-merge per Git
      Workflow in CLAUDE.md; never commit to main directly).

## Risk Assessment

- **Low.** Docs + verification only. The size-check is the one gate that could
  surprise — the feature adds little code, but confirm the binary stays under
  the cap (it should comfortably).

## Notes

- **Git:** branch `feat/key-error-heatmap`, conventional commits, no AI refs,
  PR to protected `main`, squash-merge. Tag only after merge if releasing.
- **Out of scope (future):** lifetime/aggregate heatmap across history,
  per-key error-*rate* coloring, finger/row grouping, configurable N, ASCII
  space-glyph fallback.
