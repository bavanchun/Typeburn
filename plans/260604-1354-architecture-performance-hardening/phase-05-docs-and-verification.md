---
phase: 5
title: Docs and Verification
status: completed
priority: P1
effort: 1h
dependencies:
  - 1
  - 2
  - 3
  - 4
---

# Phase 5: Docs and Verification

## Context Links

- Docs source of truth: `docs/system-architecture.md`,
  `docs/codebase-summary.md`, `docs/code-standards.md`,
  `docs/project-roadmap.md`
- Required gates: `go test ./... -race -count=1`, `go vet ./...`, empty
  `gofmt -l .`

## Overview

Sync docs to the final architecture and run the full verification gate. This
phase closes the plan and prevents docs/source drift.

## Requirements

- Functional: docs accurately describe package layering, benchmark coverage,
  Mode ownership, and best-bucket ownership.
- Non-functional: full CI gate passes; benchmark command runs.
- Process: whole-plan consistency sweep before implementation handoff.

## Architecture

Docs must describe actual source:

```text
internal/mode     -> pure test mode enum/options
internal/config   -> settings, keymap, XDG paths; keymap is Bubble Tea boundary by repo rule
internal/typing   -> engine; imports mode for completion/progress
internal/metrics  -> metrics; imports mode/typing
internal/storage  -> history/settings persistence + best bucket helpers
internal/ui/app   -> Bubble Tea/Lip Gloss surfaces
```

## File Inventory

| File | Action | Rough size | Test impact |
|---|---:|---:|---|
| `docs/system-architecture.md` | Modify | docs | Fix config/mode layering |
| `docs/codebase-summary.md` | Modify | docs | Add `internal/mode`, benchmarks |
| `docs/code-standards.md` | Modify if needed | docs | Update pure-layer wording |
| `docs/project-roadmap.md` | Modify if needed | docs | Add completed hardening note |
| `plans/.../plan.md` | Review | docs | Consistency sweep |
| `plans/.../phase-*.md` | Review | docs | Consistency sweep |

## Test Scenario Matrix

| Scenario | Criticality | Command |
|---|---|---|
| Full race suite | Critical | `go test ./... -race -count=1` |
| Vet clean | Critical | `go vet ./...` |
| Format clean | Critical | `gofmt -l .` |
| Benchmark suite runs | High | `go test ./internal/typing ./internal/metrics ./internal/ui ./internal/storage -bench . -benchmem` |
| Mode-only imports clean | High | `go list -f ... ./internal/typing ./internal/metrics ./internal/words ./internal/runner | rg 'internal/config'` |
| Docs no stale `config.Mode` ownership claims | Medium | `rg 'config\\.Mode|Zero Bubble Tea Imports|Benchmark' docs` |

## Dependency Map

```text
P2/P3/P4 final source state -> docs sync -> full verification -> implementation handoff
```

## Implementation Steps

1. Update `docs/system-architecture.md`:
   - add `internal/mode`
   - clarify `internal/config` has Bubble Tea keymap boundary
   - remove stale zero-Bubble-Tea claim for `config`
2. Update `docs/codebase-summary.md`:
   - add package summary for `internal/mode`
   - note benchmark files and intended use
   - update best-bucket ownership
3. Update `docs/code-standards.md` only if wording conflicts with source.
4. Update `docs/project-roadmap.md` with a concise hardening entry if this work
   is release-significant.
5. Run stale-term sweep:
   ```sh
   rg 'config\\.Mode|Zero Bubble Tea Imports|typedFromLog|effWPM|histBucketKey' docs internal plans/260604-1354-architecture-performance-hardening
   ```
6. Run full gates:
   ```sh
   go test ./... -race -count=1
   go vet ./...
   gofmt -l .
   go test ./internal/typing ./internal/metrics ./internal/ui ./internal/storage -bench . -benchmem
   go list -f '{{.ImportPath}} {{join .Imports " "}}' ./internal/typing ./internal/metrics ./internal/words ./internal/runner | rg 'internal/config'
   ```
7. Re-read `plan.md` and every `phase-*.md`; remove stale assumptions created
   by red-team/validation or implementation decisions.

## Success Criteria

- [ ] Docs match final package architecture.
- [ ] No stale docs claim that `config` is pure/zero Bubble Tea if keymap remains
      there.
- [ ] Full race/vet/gofmt gates pass.
- [ ] Benchmark command runs and output is available for review.
- [ ] Plan files have no contradictions after consistency sweep.

## Risk Assessment

- Risk: docs overpromise performance gains. Mitigation: state measured results
  only; no invented thresholds.
- Risk: stale terms remain in journals. Mitigation: evergreen docs matter most;
  old journals may preserve history and do not need rewriting.
- Risk: plan files reference rejected decisions. Mitigation: whole-plan
  consistency sweep before handoff.
