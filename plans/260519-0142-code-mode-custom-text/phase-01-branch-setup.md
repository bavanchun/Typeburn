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
Create the protected-main feature branch all phases commit to. Gate phase —
no code, no commit.

## Requirements
- Functional: `feat/v1.2.0-code-mode` exists off latest `main`.
- Non-functional: respects hard-enforced protected-main (local hook +
  branch protection).

## Architecture
Single shared branch; sequential per-phase commits.

## Related Code Files
- Create/Modify/Delete: none

## Implementation Steps
1. `git -C /Users/vchun/Codes/My-projects/Typeburn fetch origin`
2. `git switch main && git pull --ff-only origin main`
3. `git switch -c feat/v1.2.0-code-mode` (its OWN Bash call — the
   protect-main hook evaluates HEAD pre-exec; never chain `switch -c &&
   commit`).
4. Confirm branch + clean tree. Commit the plan dir as
   `docs(plan): v1.2.0 code mode plan + brainstorm`.

## Success Criteria
- [ ] On `feat/v1.2.0-code-mode`, up to date with origin/main.
- [ ] Plan artifacts committed on the branch.

## Risk Assessment
- Hook denies chained `switch -c && commit` (HEAD=main at hook time): run
  branch creation as a standalone step. Known from v1.1.0.
