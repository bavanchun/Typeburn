---
title: Architecture Performance Hardening
description: >-
  Measure Typeburn typing/render hot paths, fix only proven performance issues,
  clean pure-core layering, dedupe best-bucket logic, and sync docs.
status: completed
priority: P2
effort: 6h
branch: fix/roadmap-v2-4-current-stable
tags:
  - refactor
  - performance
  - tech-debt
blockedBy: []
blocks: []
created: '2026-06-04T06:54:53.850Z'
createdBy: 'ck:plan'
source: skill
---

# Architecture Performance Hardening

## Overview

This plan turns the architecture/performance audit into a controlled hardening
track. Current app health is good: `go test ./...`, `go test ./... -race
-count=1`, `go vet ./...`, and `gofmt -l .` were clean during audit. Core
coverage is strong (`typing` 96.2%, `metrics` 94.0%, `ui` 87.3%).

Do not optimize blind. First establish benchmark data. Then apply only small,
evidence-backed changes: reduce typing-screen log replay/copy cost if measured,
move mode-only dependencies out of `config`, dedupe best-bucket logic, and
update docs to match source.

## Scope Challenge

- Existing code: `metrics.LiveWPM`, `typing.Engine.Log`, `typing.Engine.States`,
  stream renderers, and storage best helpers already solve most behavior.
- Minimum changes: add benchmarks; add an engine snapshot only if benchmarks
  show render/log replay cost; split mode definitions; expose reusable best
  helpers; sync docs.
- Complexity: expected 12-18 touched files, 1 small new package, 5 phases. This
  is justified because layering and benchmark work touch multiple packages.
- Selected scope: HOLD. No feature expansion, no renderer rewrite, no new deps.

## Audit Baseline

| Area | Current evidence | Risk |
|---|---|---|
| CI gate | `go test`, race, vet, gofmt clean | Low |
| Benchmarks | `go test ./... -run '^$' -bench . -benchmem` has no benchmark funcs | Medium |
| Typing render | `TypingModel.View()` calls `States()` and `Log()` then replays via `typedFromLog()` | Medium |
| Live WPM | Tick path calls `m.eng.Log()` then `metrics.LiveWPM()` every ~250ms | Medium |
| Core mode deps | `typing`/`metrics`/`words` import `config` only for `Mode`; `config` also owns Bubble Tea key types by repo decision | Important architecture debt |
| Best marker | `storage` and `ui` duplicate effective-WPM/bucket logic | Medium maintenance risk |
| Docs | `docs/system-architecture.md` claims `config` has zero Bubble Tea imports | Medium drift |

## Design Decisions

- Measurement-first: Phase 2 optimization is gated by Phase 1 benchmark data.
- Keep Bubble Tea Elm architecture unchanged.
- Keep dependencies unchanged. No external benchmark/perf libs.
- New domain package allowed: `internal/mode` for `Mode` and `LengthsFor`.
- Keep `config.Keymap` in `internal/config` unless user explicitly approves a
  larger boundary change; `CLAUDE.md` says that Bubble Tea key binding import is
  deliberate.
- Do not change persisted JSON shape unless tests prove unavoidable.
- No plan/finding IDs in code comments, migration names, or tests.
- Keep every Go file under 200 LOC; split benchmark files by package if needed.

## Cross-Plan Dependencies

None. All project `plans/*/plan.md` are `status: completed`. The only deferred
phase in old plans is Obsidian hook work, which does not overlap Typeburn Go
architecture/performance.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Benchmark Baseline](./phase-01-benchmark-baseline.md) | Completed |
| 2 | [Typing Snapshot Optimization](./phase-02-typing-snapshot-optimization.md) | Completed |
| 3 | [Mode Layering Refactor](./phase-03-mode-layering-refactor.md) | Completed |
| 4 | [Best Bucket DRY](./phase-04-best-bucket-dry.md) | Completed |
| 5 | [Docs and Verification](./phase-05-docs-and-verification.md) | Completed |

## Dependencies

- Phase 1 is the data gate.
- Phase 2 depends on Phase 1. It may be skipped or reduced if benchmark data
  shows no meaningful typing/render issue.
- Phase 3 can run after Phase 1 and before/after Phase 2, but should land before
  Phase 5 docs sync.
- Phase 4 can run after Phase 1 and is independent of Phase 2/3 except final
  verification.
- Phase 5 depends on Phases 1-4.

```text
P1 Benchmark Baseline
├── P2 Typing Snapshot Optimization (conditional)
├── P3 Mode Layering Refactor
└── P4 Best Bucket DRY
    └── P5 Docs and Verification
```

## Success Criteria

- Benchmarks exist and run with `go test ./internal/typing ./internal/metrics
  ./internal/ui ./internal/storage -bench . -benchmem`.
- If Phase 2 changes code, benchmark output shows lower allocations and/or time
  for the Code-mode 10k render case without behavior regressions.
- `internal/typing`, `internal/metrics`, `internal/words`, and `internal/runner`
  no longer import `internal/config` just to use `Mode`.
- Best marker logic has one source of truth for bucket key/effective WPM.
- Docs match source architecture.
- Final gates pass: `go test ./... -race -count=1`, `go vet ./...`, empty
  `gofmt -l .`, and benchmark command completes.

## Out of Scope

- New typing modes, UI features, history schema migration, renderer
  virtualization, async rendering, terminal telemetry, new dependencies.
- Changing Bubble Tea, Lip Gloss, cobra/fang, update/self-update behavior.

## Red-Team Notes

- Do not use benchmark numbers as flaky CI assertions. Record baseline in
  comments/docs or compare manually during review.
- Do not split `Mode` in a way that creates import cycles between `mode`,
  `config`, `runner`, and `ui`.
- Do not move `config.Keymap` out of `config` in this plan; that would reverse
  the current repo rule without explicit user approval.
- Do not expose mutable engine internals. Any snapshot API must copy or return
  immutable data by contract.
- Do not “fix” the accepted legacy `NetWPM==0` ambiguity with a schema change.
