---
phase: 1
title: "Lock module and command contract"
status: done
effort: "M"
---

# Phase 1: Lock module and command contract

## Objective

Encode the failed v2.5.0 install surface and exact lowercase replacement as
tests before changing production code.

## File Inventory

| File | Action | Contract |
|---|---|---|
| `internal/update/selfpath_test.go` | modify | exact Go-install detection/advice |
| `internal/cli/cmd_version_test.go` | modify | exact update command |
| `internal/update/selfpath.go` | inspect | implementation boundary |
| `internal/cli/cmd_version.go` | inspect | user-facing update boundary |

## Implementation Steps

1. Add lowercase `typeburn` classification cases as green regression coverage.
2. Require exact advice: `go install github.com/bavanchun/Typeburn/v2/cmd/typeburn@latest`.
3. Strengthen update-available coverage to assert the same command.
4. Record the focused command/output; only exact old-command assertions are the
   intentional red failures because filename classification is directory-based.

## Function / Interface Checklist

- `instructionFor(InstallGo)` owns the canonical command.
- Go fixtures use lowercase `typeburn`; archive/Homebrew behavior is unchanged.
- Version update output agrees exactly; repository URLs remain unchanged.

## Test Matrix

| Scenario | Expected |
|---|---|
| Go-installed lowercase binary | exact `/v2/cmd/typeburn@latest` advice |
| Archive/Homebrew | existing advice unchanged |
| Update available | canonical command rendered |
| No update/error | existing behavior unchanged |

## Dependencies

- Input: user's confirmed `/v2` plus v2.5.1 fix-forward decision.
- Blocks Phase 2; tests stay red until that phase.

## Success Criteria

- [x] Tests lock module path, command path, binary case, and `@latest`.
- [x] Expected red failure is recorded; production code is untouched.
- [x] Commit: merged into `fix(module): migrate Typeburn to the v2 module path` (ef36f6e).
