---
phase: 1
title: Benchmark Baseline
status: completed
priority: P1
effort: 1.5h
dependencies: []
---

# Phase 1: Benchmark Baseline

## Context Links

- Audit targets: `internal/typing`, `internal/metrics`, `internal/ui`,
  `internal/storage`
- Existing test strength: `typing` 96.2%, `metrics` 94.0%, `ui` 87.3%
- Current gap: no `Benchmark*` functions found by `rg 'func Benchmark'`

## Overview

Create repeatable Go benchmarks for the audited hot paths. This phase must not
change production behavior. It creates the data gate for later work.

## Requirements

- Functional: add benchmark functions only; no app behavior change.
- Non-functional: benchmark data covers realistic and worst-case sizes, including
  Code mode 10k runes.
- Process: run baseline benchmarks before any optimization/refactor.

## Architecture

Benchmarks stay in package-local `_test.go` files so they can use existing
constructors and test helpers without adding public API. Use deterministic test
data and fixed timestamps. Avoid golden output comparison inside benchmarks.

## File Inventory

| File | Action | Rough size | Test impact |
|---|---:|---:|---|
| `internal/typing/engine_benchmark_test.go` | Create | <120 LOC | Engine apply/log/states |
| `internal/metrics/metrics_benchmark_test.go` | Create | <140 LOC | Compute/live WPM/key heatmap |
| `internal/ui/render_benchmark_test.go` | Create | <180 LOC | Word/code stream + TypingModel view |
| `internal/storage/history_benchmark_test.go` | Create | <100 LOC | Best buckets/history append helpers |

## Test Scenario Matrix

| Scenario | Criticality | Benchmark |
|---|---|---|
| 100-word normal typing | High | `BenchmarkEngineApplyWords100` |
| Code 10k sequential paste-like input | Critical | `BenchmarkEngineApplyCode10k` |
| `Engine.States()` over 10k runes | Critical | `BenchmarkEngineStatesCode10k` |
| `metrics.Compute()` over 10k log | High | `BenchmarkMetricsComputeCode10k` |
| `metrics.LiveWPM()` over 10k log | Medium | `BenchmarkLiveWPMCode10k` |
| Word stream render | High | `BenchmarkRenderWordStreamWords100` |
| Code stream render 10k | Critical | `BenchmarkRenderCodeStreamCode10k` |
| Full typing view Code 10k | Critical | `BenchmarkTypingViewCode10k` |
| Best bucket over 200 records | Medium | `BenchmarkBestBucketHistory200` |

## Dependency Map

```text
typing benchmarks -> internal/typing only
metrics benchmarks -> typing.Keystroke + metrics
ui benchmarks -> typing.Engine + ui renderers + theme/config
storage benchmarks -> storage.Record helpers
```

## Implementation Steps

1. Create deterministic target builders:
   - words text around 100 words
   - code text near 10k runes with newlines/tabs
   - timed logs with fixed millisecond increments
2. Add `internal/typing` benchmarks for `Apply`, `Backspace`, `Log`, `States`.
3. Add `internal/metrics` benchmarks for `Compute`, `LiveWPM`, `KeyHeatmap`.
4. Add `internal/ui` benchmarks for `RenderWordStream`, `RenderCodeStream`,
   and full `TypingModel.View`.
5. Add `internal/storage` benchmark for current best-bucket/new-best path.
6. Run:
   ```sh
   go test ./internal/typing ./internal/metrics ./internal/ui ./internal/storage -bench . -benchmem
   ```
7. Save benchmark output in the PR description or a plan report, not as brittle
   hardcoded thresholds.

## Success Criteria

- [ ] `rg 'func Benchmark' internal` finds benchmark coverage for all 4 areas.
- [ ] Benchmark command runs successfully.
- [ ] Benchmarks do not require network, terminal TTY, env-specific files, or
      external tools.
- [ ] No production code changed in this phase.
- [ ] Phase 2 decision is made from measured data.

## Risk Assessment

- Risk: noisy benchmark numbers. Mitigation: focus on relative before/after and
  allocations, not exact CI thresholds.
- Risk: benchmark helpers grow files over 200 LOC. Mitigation: split by package
  and keep builders local.
- Risk: benchmark accidentally mutates shared state. Mitigation: create fresh
  engine/model inside each benchmark iteration.
