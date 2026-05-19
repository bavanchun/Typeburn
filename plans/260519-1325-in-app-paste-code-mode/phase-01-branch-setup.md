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
Create the protected-main feature branch; commit plan artifacts. Gate phase.

## Requirements
- Functional: `feat/v1.3.0-in-app-paste` off latest `main`.
- Non-functional: respects hard-enforced protected-main.

## Architecture
Single shared branch; sequential per-phase commits.

## Related Code Files
- Create/Modify/Delete: none (code)

## Implementation Steps
1. `git fetch origin`; `git switch main`; `git pull --ff-only origin main`.
2. `git switch -c feat/v1.3.0-in-app-paste` as its OWN Bash call (the
   protect-main hook evaluates HEAD pre-exec — never chain `switch -c &&
   commit`).
3. Commit the plan dir: `docs(plan): v1.3.0 in-app paste plan + brainstorm`.

## Success Criteria
- [ ] On `feat/v1.3.0-in-app-paste`, up to date with origin/main.
- [ ] Plan artifacts committed.

## Risk Assessment
- Hook denies chained `switch -c && commit` (HEAD=main at hook time): run
  branch creation standalone. Known from v1.1.0/v1.2.0.
