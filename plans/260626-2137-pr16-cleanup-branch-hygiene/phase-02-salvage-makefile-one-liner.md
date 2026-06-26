---
phase: 2
title: "Salvage Makefile one-liner"
status: completed
effort: "S"
---

# Phase 2: Salvage Makefile one-liner

## Overview

The `quiet-make-recipe-echo` branch is a stale pre-v2 branch (merging it would delete ~32k lines of current source) and must not be merged. Its one genuine improvement is silencing the echoed build command by prefixing the `build:` recipe with `@`. Re-apply that single change fresh onto current `main` via a small `fix/` branch and PR. Do **not** cherry-pick from the stale branch (it carries obsolete context); make the edit by hand on a clean branch.

## Requirements
- Functional: `make build` no longer echoes the `go build …` line; still builds `./bin/typeburn`.
- Non-functional: one-line diff, conventional commit, PR through protected-main flow.

## Architecture

Current `Makefile` `build:` target (on `main`):
```make
build:
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY) .
```
Change the single recipe line to:
```make
build:
	@go build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY) .
```
That is the entire change. Do not touch `SIZE_LIMIT`, `.PHONY`, `size-check`, or `notui-noexit-check` — the stale branch removed those, which is wrong; keep them.

## Related Code Files
- Modify: `Makefile` (one line in the `build:` target)

## Implementation Steps

1. From up-to-date `main`, create the branch:
   ```sh
   git switch main && git pull --ff-only
   git switch -c fix/quiet-make-build-echo
   ```
2. Edit `Makefile`: prefix the `go build` recipe line under `build:` with `@` (see Architecture). Leave everything else untouched.
3. Verify the build is now silent and still works:
   ```sh
   make build        # should NOT print the "go build ..." command; should produce bin/typeburn
   ls -l bin/typeburn
   make lint         # gofmt + vet + guards still pass
   ```
4. Commit and open PR:
   ```sh
   git add Makefile
   git commit -m "build: silence echoed go build command in make build target"
   git push -u origin fix/quiet-make-build-echo
   gh pr create --base main --title "build: silence echoed go build command" \
     --body "Re-applies the one useful change from the obsolete quiet-make-recipe-echo branch onto current main: prefix the build recipe with @ so 'make build' does not echo the full go build invocation. No behavior change beyond output verbosity. Supersedes quiet-make-recipe-echo (to be deleted)."
   ```
5. Wait for CI green, then **squash-merge** (squash is the only enabled mode; branch auto-deletes on merge):
   ```sh
   gh pr checks --watch
   gh pr merge --squash --delete-branch
   ```

## Success Criteria

- [x] `fix/quiet-make-build-echo` PR opened against `main`.
- [x] CI green; PR squash-merged; its branch auto-deleted.
- [x] On merged `main`, `make build` runs without echoing the `go build` line and still emits `./bin/typeburn`.
- [x] No other Makefile targets changed.

## Risk Assessment

- **Risk:** pushing to `main` directly. **Mitigation:** PR-only flow; local hook + branch protection block direct push.
- **Risk:** `@` hides errors. **Mitigation:** `make build` still surfaces non-zero exit + stderr; only the command echo is suppressed, matching standard Make practice.
- **Risk:** accidentally importing the stale branch's other deletions. **Mitigation:** hand-edit on a fresh branch; never merge/cherry-pick `quiet-make-recipe-echo`.
