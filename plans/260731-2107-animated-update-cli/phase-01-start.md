---
phase: 1
title: "Dependency Baseline"
status: pending
priority: P1
effort: "0.5h"
dependencies: []
---

# Phase 1: Dependency Baseline

## Overview

Add `charm.land/bubbles/v2` to the module and prove the binary stays under the
size cap before any UI code is written, so the size risk is closed at the
cheapest possible point rather than discovered at the end.

## Requirements

- Functional: `charm.land/bubbles/v2` resolvable; `progress` and `spinner`
  importable.
- Non-functional: stripped binary stays under `SIZE_LIMIT` (10,485,760 bytes)
  with real headroom, not a squeaker.

## Architecture

`bubbles/v2` is covered by the existing dependency allowlist in `CLAUDE.md`
("stdlib, `charm.land/*`, `github.com/charmbracelet/*`, cobra,
`golang.org/x/*`"), so no per-dependency user approval is needed. It pulls
`github.com/charmbracelet/harmonica` (spring physics behind
`progress.SetSpringOptions`) as its only substantial new transitive dependency;
`bubbletea` and `lipgloss` are already linked.

Measured on darwin/arm64 with `-trimpath -ldflags '-s -w'` before this plan was
written, using a throwaway package that force-links both components:

| Build | Bytes |
|---|---|
| baseline (`main`) | 8,934,370 |
| baseline + `bubbles/progress` + `bubbles/spinner` | 8,951,522 |
| delta | **+17,152** |
| headroom remaining under cap | 1,534,238 |

The cap is not a constraint on this design. This phase re-measures in the real
tree to confirm.

## Related Code Files

- Modify: `go.mod`, `go.sum`

## Implementation Steps

1. `go get charm.land/bubbles/v2@v2.1.1`
2. `go mod tidy`
3. Confirm `go.mod` lists `charm.land/bubbles/v2` in the direct-require block
   and `github.com/charmbracelet/harmonica` as indirect.
4. `make build && make size-check` — record the actual byte count.
5. `make lint && make test` — nothing should change yet.

## Success Criteria

- [ ] `go.mod` requires `charm.land/bubbles/v2 v2.1.1` directly
- [ ] `make size-check` passes
- [ ] Recorded binary size within ~50 KB of the 8,951,522 measurement
- [ ] `make lint` and `make test` unchanged and green

## Risk Assessment

**Risk:** an unrelated transitive bump (e.g. `bubbles` pinning a newer
`ultraviolet` or `lipgloss`) drags in size or behavior changes.
**Mitigation:** inspect the `go.mod` diff before committing. If `go mod tidy`
upgrades `bubbletea`/`lipgloss` beyond the current pins, stop and report the
version deltas rather than accepting them silently — the TUI has `teatest`
golden files that a lipgloss bump can invalidate.
