---
phase: 3
title: "Prune stale branches"
status: completed
effort: "S"
---

# Phase 3: Prune stale branches

## Overview

Delete the two remaining stale branches now that their value (if any) is preserved:
- `feat/v1.1.0-theme-packs-hygiene` — v1.1.0 already shipped; its remote is already `gone`. Delete the dangling local branch.
- `quiet-make-recipe-echo` — its only useful change was salvaged in Phase 2. Delete local + remote.

(`chore/agent-config` is deleted in Phase 1, not here.) Reversible via recorded SHAs.

## Requirements
- Functional: both branches removed; working set is just `main` (+ origin/main).
- Non-functional: do not run before Phase 2's salvage PR is merged (or the change explicitly abandoned).

## Dependencies
- **Blocked by Phase 2.** Deleting `quiet-make-recipe-echo` is only safe after `@go build` is merged to `main`.

## Implementation Steps

1. Pre-flight — confirm Phase 2 landed and you are on `main`:
   ```sh
   git switch main && git pull --ff-only
   grep -n '@go build' Makefile   # must match → salvage merged
   ```
   If no match, STOP — Phase 2 not complete.

2. Delete the shipped-feature local branch (remote already gone):
   ```sh
   git branch -D feat/v1.1.0-theme-packs-hygiene
   ```

3. Delete the salvaged stale branch, remote then local:
   ```sh
   git push origin --delete quiet-make-recipe-echo
   git branch -D quiet-make-recipe-echo
   ```

4. Prune dangling remote-tracking refs and verify final state:
   ```sh
   git fetch --prune
   git branch        # expect only: * main
   git branch -r     # expect only: origin/HEAD -> origin/main, origin/main
   ```

## Success Criteria

- [x] `feat/v1.1.0-theme-packs-hygiene` gone from `git branch`.
- [x] `quiet-make-recipe-echo` gone from `git branch` and `git branch -r`.
- [x] `git branch` lists only `main`.
- [x] `git branch -r` lists only `origin/HEAD` and `origin/main`.

## Risk Assessment

- **Risk:** deleting `quiet-make-recipe-echo` before salvage merged → losing the change. **Mitigation:** Step 1 `grep '@go build'` gate; SHA `a9f21b7` recorded.
- **Risk:** unintended branch deletion. **Mitigation:** delete only the two named refs; verify final `git branch` output matches expected.
