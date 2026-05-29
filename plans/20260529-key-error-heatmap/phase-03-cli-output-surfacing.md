---
phase: 3
title: "CLI Output Surfacing"
status: completed
priority: P2
effort: "1h"
dependencies: [1]
---

# Phase 3: CLI Output Surfacing

## Overview

Expose the heatmap in CLI output: a `key_misses` array in JSON and top-key rows
in the table view. Flows automatically through `run --no-tui` and `replay` since
both build their output from `metrics.Result` via `newMetricOutput` /
`metricTableRows`.

## Requirements

- **Functional:**
  - `metricOutput` gains `KeyMisses []keyMissOutput` → JSON key `key_misses`,
    each `{ "key": "e", "misses": 4, "attempts": 31 }`.
  - Table output gains rows for the top keys, e.g. `most_missed_e | 4/31`.
  - Empty heatmap → `key_misses: []` (or omitted — pick one and assert it);
    table simply adds no rows.
- **Non-functional:**
  - Deterministic ordering (inherits Phase 1 sort) for stable JSON/table tests.
  - No change to existing JSON keys (additive only — preserves the CLI contract
    relied on by `run --no-tui --json` and `replay --json`).

## Architecture

`metric_output.go` is the single mapping layer (callers: `cmd_run_notui.go:27`,
`cmd_replay.go:50,53`). Extend it; both call sites benefit with no other edits.

```go
type keyMissOutput struct {
    Key      string `json:"key"`
    Misses   int    `json:"misses"`
    Attempts int    `json:"attempts"`
}

type metricOutput struct {
    ...
    DurationMs int64           `json:"duration_ms"`
    KeyMisses  []keyMissOutput `json:"key_misses"`   // NEW (last field)
}
```

`newMetricOutput` maps `r.KeyMisses` → `[]keyMissOutput` (using `Label` for `Key`).
`metricTableRows` appends top-N rows after `duration_ms` using a compact
`"misses/attempts"` value form. Cap table rows at top 5 to keep the table tight;
JSON carries the full list (still top-8 from Phase 1).

## Related Code Files

- Modify: `internal/cli/metric_output.go` (add `keyMissOutput`, struct field,
  mapping in `newMetricOutput`, rows in `metricTableRows`)
- Modify: `internal/cli/cmd_run_test.go` (assert `key_misses` in JSON;
  verify additive — existing keys unchanged)
- Possibly: `internal/cli/cmd_replay_test.go` if it snapshots metric output

## Implementation Steps

1. Add `keyMissOutput` struct + `KeyMisses` field to `metricOutput`.
2. Map in `newMetricOutput`: iterate `r.KeyMisses` → `keyMissOutput{Key: km.Label, ...}`.
3. Append top-5 rows in `metricTableRows` (`most_missed_<label>` → `n/attempts`).
4. Update `cmd_run_test.go`:
   - JSON contains `key_misses` with expected entries for a known-error input.
   - Existing metric keys still present and unchanged (contract guard).
5. Run `go test ./internal/cli/ -race -count=1`.

## Success Criteria

- [ ] `run --no-tui --json` includes `key_misses`; existing keys unchanged.
- [ ] `replay` JSON + table include heatmap data.
- [ ] Empty heatmap handled (empty array / no rows) and asserted.
- [ ] `go test ./internal/cli/ -race -count=1` GREEN; `gofmt`/`vet` clean.

## Risk Assessment

- **Low.** Purely additive to an output struct. The only real risk is breaking
  an existing JSON snapshot test — mitigated by the explicit "existing keys
  unchanged" assertion and by appending the new field last.
