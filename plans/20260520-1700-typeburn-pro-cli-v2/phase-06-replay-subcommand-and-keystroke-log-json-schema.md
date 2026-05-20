---
phase: 6
title: "replay subcommand and keystroke-log JSON schema"
status: completed
priority: P2
effort: "3h"
dependencies: [2]
---

# Phase 6: `replay` subcommand and keystroke-log JSON schema

## Overview

`typeburn replay <log.json> [--json]` reads a keystroke log + mode + endMs
from disk and runs `metrics.Compute` to produce deterministic output. No TTY
required. Lock the on-disk schema at `schema_version: 1` for forward-compat.

## Requirements

- Functional:
  - `typeburn replay testdata/sample-keystroke-log.json` prints a metrics table
  - `--json` prints metrics JSON
  - Missing file → exit 2 with clear message
  - Malformed JSON → exit 2
  - Wrong schema_version → exit 2 with "unsupported schema version"
- Non-functional: identical output across runs (no clock, no randomness)

## Architecture

```
internal/cli/
  cmd_replay.go
  cmd_replay_test.go
testdata/
  sample-keystroke-log.json   # ~30 keystrokes, words mode
```

Wire schema (locked v1):

```json
{
  "schema_version": 1,
  "mode": "words",
  "end_ms": 30000,
  "log": [
    {"time_ms": 0, "typed": 104, "target": 104, "correct": true},
    {"time_ms": 120, "typed": 101, "target": 101, "correct": true}
  ]
}
```

Add JSON tags to `internal/typing/engine.go:Keystroke`:

```go
type Keystroke struct {
    TimeMs  int64 `json:"time_ms"`
    Typed   rune  `json:"typed"`
    Target  rune  `json:"target"`
    Correct bool  `json:"correct"`
}
```

This is back-compatible: nothing currently marshals `Keystroke` to JSON.

`cmd_replay.go` decodes wrapper, validates `schema_version == 1`,
calls `metrics.Compute(log, mode, endMs)`, prints via `output.Render`.

## Related Code Files

- Create: `internal/cli/cmd_replay.go`, `internal/cli/cmd_replay_test.go`
- Create: `testdata/sample-keystroke-log.json`
- Modify: `internal/typing/engine.go` (add json tags to `Keystroke`)
- Modify: `internal/typing/engine_test.go` if it asserts struct equality (back-compat check)

## Implementation Steps

1. Add JSON tags to `Keystroke`. Run `go test ./internal/typing/ ./internal/metrics/` — must still pass.
2. Define `ReplayInput` wrapper in `cmd_replay.go`. Constants for `schemaVersionV1 = 1`.
3. Decode + validate (`schema_version`, `mode` ∈ known set, `end_ms >= 0`, `len(log) > 0`).
4. Call `metrics.Compute(input.Log, config.Mode(input.Mode), input.EndMs)`.
5. Render via `output.Render` (Phase 4 helpers).
6. Craft `testdata/sample-keystroke-log.json` by hand: simulate typing "hello world" in 5 seconds (rough timings); commit as fixture.
7. Tests:
   - Happy path → expected WPM/accuracy (computed once and asserted)
   - Missing file → exit 2
   - Malformed JSON → exit 2
   - schema_version=2 → exit 2 with "unsupported schema version"
8. Manual smoke against the fixture.

## Success Criteria

- [ ] Replay of the fixture produces stable metrics across 10 runs
- [ ] Schema version mismatch produces clear error
- [ ] `--json` output is valid JSON
- [ ] `Keystroke` JSON marshaling round-trips (test)
- [ ] All existing tests pass

## Risk Assessment

- **Risk:** Adding JSON tags changes binary marshaling of any caller.
  **Mitigation:** No current caller marshals `Keystroke`; grep confirms. Tests round-trip.
- **Risk:** Wire format locked too early.
  **Mitigation:** `schema_version` field is the lever; future versions add fields and validate.
- **Risk:** `metrics.Compute` returns 100% accuracy on zero-keystrokes → confusing replay output.
  **Mitigation:** Validate `len(log) > 0` before compute; helpful error if empty.
