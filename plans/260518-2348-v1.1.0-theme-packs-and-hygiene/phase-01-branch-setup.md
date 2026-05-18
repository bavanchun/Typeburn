---
phase: 1
title: "Branch Setup"
status: pending
priority: P1
effort: "5m"
dependencies: []
---

# Phase 1: Branch Setup

## Overview
Create the protected-main feature branch all subsequent phases commit to.
Gate phase — no code, no commit of its own; just branch creation.

## Requirements
- Functional: feature branch exists locally, based on latest `main`.
- Non-functional: respects hard-enforced protected-main workflow (the local
  PreToolUse hook blocks commit/push on `main`).

## Architecture
Single shared branch `feat/v1.1.0-theme-packs-hygiene`. Parallel phases 2/3/4
each add their own commit on this branch (file-disjoint → no conflicts).

## Related Code Files
- Create: none
- Modify: none
- Delete: none

## Implementation Steps
1. `git -C /Users/vchun/Codes/My-projects/Typeburn fetch origin`
2. `git -C ... switch main && git -C ... pull --ff-only origin main`
3. `git -C ... switch -c feat/v1.1.0-theme-packs-hygiene`
4. Confirm `git rev-parse --abbrev-ref HEAD` == `feat/v1.1.0-theme-packs-hygiene`.

## Success Criteria
- [ ] On branch `feat/v1.1.0-theme-packs-hygiene`, up to date with `origin/main`.
- [ ] Working tree clean.

## Risk Assessment
- Hook blocks commits on main: expected; this phase deliberately creates the
  branch first. No commit happens here.
