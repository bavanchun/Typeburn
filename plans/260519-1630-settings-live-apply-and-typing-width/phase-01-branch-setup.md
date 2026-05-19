---
phase: 1
title: Branch Setup
status: completed
priority: P1
effort: 15m
dependencies: []
---

# Phase 1: Branch Setup

## Overview

Create the feature branch off a clean, up-to-date `main` and confirm the
pre-change test gate is fully green so later RED phases are unambiguous.

## Requirements
- Functional: branch `fix/v1.4.0-settings-live-apply-and-typing-width` exists
  off latest `main`; no source changes committed in this phase.
- Non-functional: protected-main respected (never commit to `main`).

## Architecture

Protected-main flow (`typeburn-release-runbook`): topic branch → per-phase
commits → PR → squash-merge → tag on merged SHA. The protect-main PreToolUse
hook evaluates HEAD before the command runs, so `git switch -c` MUST be its
own Bash call, then commits follow in later calls.

## Related Code Files
- Create: none
- Modify: none
- Delete: none

## Implementation Steps
1. `cd /Users/vchun/Codes/My-projects/Typeburn` (cwd resets between Bash
   calls in this env — prefix every git/gh call).
2. `git switch main && git pull --ff-only` (own call).
3. `git switch -c fix/v1.4.0-settings-live-apply-and-typing-width` (own call,
   no chained commit).
4. Baseline gate: `gofmt -l .` (clean), `go vet ./...` (clean),
   `go test ./... -race -count=1` (all pass), `make build` (success).
5. Record baseline: package count, codetext/ui coverage %, that
   `phase09_polish_test.go` and `screen_typing_test.go` are GREEN (they will
   be intentionally edited in Phase 3).

## Success Criteria
- [ ] On branch `fix/v1.4.0-settings-live-apply-and-typing-width`, `main` untouched
- [ ] `gofmt`/`vet`/`-race`/`make build` all green at baseline
- [ ] Baseline metrics recorded for later comparison

## Risk Assessment
- Hook denies chained `switch -c && commit` → run `switch -c` standalone.
- Stale `main` → `git pull --ff-only` before branching.

## Next Steps
Phase 2 (Settings live-apply refactor) — highest severity first.
