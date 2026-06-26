---
title: "PR #16 disposition + branch hygiene"
description: "Close oversized agent-config PR #16, salvage one Makefile improvement, prune 3 stale branches"
status: completed
priority: P2
branch: "main"
tags: [git-hygiene, repo-maintenance, handoff]
blockedBy: []
blocks: []
created: "2026-06-26T14:43:45.451Z"
createdBy: "ck:plan"
source: skill
---

# PR #16 disposition + branch hygiene

## Overview

Operational cleanup, no Go source changes. Three independent units:

1. **Close PR #16** (`chore/agent-config`) — 4,008,092 additions / 26,111 files committing personal agent tooling (`.claude` skills, `.agent`/`.codex`/`.gemini` configs) that main's `.gitignore` already excludes. Decision: **close, do not merge**; delete branch local+remote.
2. **Salvage Makefile one-liner** — the stale `quiet-make-recipe-echo` branch is obsolete (pre-v2, would delete ~32k lines of current source) but holds one genuine improvement: `@`-prefix on `go build` to silence the echoed command. Re-apply that single change fresh on a new `fix/` branch via PR.
3. **Prune stale branches** — delete `feat/v1.1.0-theme-packs-hygiene` (local; remote already gone, v1.1.0 shipped) and `quiet-make-recipe-echo` (local+remote, after its one-liner is salvaged in phase 2).

**Why this is safe:** branch deletion is reversible — every deleted SHA is recorded below and recoverable from reflog (local) or GitHub's branch-restore / object retention (remote) for the retention window. No protected ref is touched; `main` is untouched except via the normal phase-2 PR.

## Recorded SHAs (restore points)

| Ref | SHA | Disposition |
|-----|-----|-------------|
| `chore/agent-config` (PR #16) | `1c6c0b9ed09e3d1edc7728d4d0c1a6eded4bdadf` | close PR, delete local+remote |
| `feat/v1.1.0-theme-packs-hygiene` | `9215f090b4f1152be6b0d921e04934a07a809c4e` | delete local (remote gone) |
| `quiet-make-recipe-echo` | `a9f21b7264e84a1c4a3395949dc8258a19baa926` | salvage 1 line, then delete local+remote |

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Close PR #16](./phase-01-close-pr-16.md) | Completed |
| 2 | [Salvage Makefile one-liner](./phase-02-salvage-makefile-one-liner.md) | Completed |
| 3 | [Prune stale branches](./phase-03-prune-stale-branches.md) | Completed |

## Dependencies

- Phases 1 and 2 are independent and may run in any order / parallel.
- **Phase 3 depends on Phase 2**: do not delete `quiet-make-recipe-echo` until its `@go build` change is merged (or explicitly confirmed dropped) in Phase 2. Deleting `chore/agent-config` is part of Phase 1, not Phase 3.
- Phase-2 PR merge follows the repo's protected-main rule (squash-merge only; CI green). The implementing agent must NOT push to `main` directly — local PreToolUse hook + GitHub branch protection both block it.

## Acceptance Criteria

- [x] PR #16 is `CLOSED` (not merged) with an explanatory comment.
- [x] `chore/agent-config` deleted local + remote.
- [x] A merged PR adds `@go build` to the Makefile `build` target on main; `make build` runs silently (no echoed `go build …` line) and still produces `./bin/typeburn`.
- [x] `feat/v1.1.0-theme-packs-hygiene` deleted local.
- [x] `quiet-make-recipe-echo` deleted local + remote.
- [x] `git branch` shows only `main` (plus any phase-2 fix branch until merged); `git branch -r` shows only `origin/main` (+ origin/HEAD).
- [x] Repo still clean: `go build ./...` ok, `make test` green (unchanged by this work).

## Constraints

- No Go source edits. Only: `gh` PR ops, `git` branch ops, one Makefile line.
- Conventional commits, no AI references. Per project CLAUDE.md, **do not** use `chore`/`docs` types for `.claude` file changes — N/A here since the Makefile change is a real `build:` change.
- Protected `main`: all merges via squash PR; never direct push.

## Handoff Notes

Self-contained for a fresh agent. Environment: CWD `/Users/vchun/Codes/My-projects/Typeburn`, branch `main`, macOS/zsh, `gh` authenticated, `ck` v4.5.0 available. Execute phases top-to-bottom; phase 3 after phase 2 merges.

## Open Questions

None.
