---
title: "Code Mode / Custom Text Input (v1.2.0)"
description: "ModeCode: type CLI-supplied text/code with full-literal whitespace; new codetext loader + isolated code renderer; TDD per phase, protected-main PR flow"
status: pending
priority: P2
branch: "feat/v1.2.0-code-mode"
tags: [feature, mode, tdd, release]
blockedBy: []
blocks: []
created: "2026-05-18T18:44:04.961Z"
createdBy: "ck:plan"
source: skill
---

# Code Mode / Custom Text Input (v1.2.0)

## Overview

New `code` test mode: practice typing on CLI-supplied text/code
(`typeburn --text <file>` | `--text -`) with **full-literal whitespace**
(Enter→`\n`, Tab→`\t`, tab=2-col visual, every space/indent typed). Engine
reuses Quote's exact-match completion; a new isolated multi-line renderer +
viewport handles layout; a new pure `internal/codetext` package does the
I/O+normalization. Saved to history, excluded from ★. Semver minor →
**v1.2.0**.

Source of truth: [brainstorm-summary.md](./brainstorm-summary.md) (all
input/whitespace/persistence/Home/renderer/loader decisions locked).

## Execution model

`--tdd`: every implementation phase is **tests-first** — write failing tests
that pin the new behaviour (and lock adjacent existing behaviour) → see red →
implement → green. Protected-main: feature branch
`feat/v1.2.0-code-mode` → per-phase commits → PR → squash-merge → tag on
merged SHA. Sequential (the renderer/engine/Home chain has real deps); no
parallel groups.

| Phase | Name | Status | Depends | TDD focus |
|-------|------|--------|---------|-----------|
| 1 | [Branch Setup](./phase-01-branch-setup.md) | Pending | — | gate, no commit |
| 2 | [Codetext Loader](./phase-02-codetext-loader.md) | Pending | 1 | normalize/size/stdin tests → impl |
| 3 | [ModeCode + Completion Seam](./phase-03-modecode-completion-seam.md) | Pending | 1 | completion+LengthsFor+Normalize+sync tests → impl |
| 4 | [Code Renderer + Viewport](./phase-04-code-renderer-viewport.md) | Pending | 1 | literal `\n`/`\t` + scroll-follow tests → impl |
| 5 | [Wiring + Home + History](./phase-05-wiring-home-history.md) | Pending | 2,3,4 | decide()/Home-disabled/IsNewBest-excludes-code tests → impl |
| 6 | [Integration Verify](./phase-06-integration-verify.md) | Pending | 5 | full -race, goldens unchanged, review |
| 7 | [Release v1.2.0](./phase-07-release-v1-2-0.md) | Pending | 6 | CHANGELOG/PR/dry-run/tag |

**Dependency:** 1 → 2,3,4 (sequential, each tests-first) → 5 → 6 → 7.

## Key locked decisions

- `--text <file>`|`-`(stdin) now; in-TUI paste deferred but **not
  precluded** (text-source is an injected string; Home forks on availability).
- Full literal whitespace; tab visual = **2 cols**; user types all
  indentation; trim one trailing `\n`; CRLF→LF; strip BOM; size cap
  ~10k runes / ~500 lines → graceful notice (no panic).
- Code runs **saved to history, never ★** (`IsNewBest` excludes
  `Mode=="code"`).
- Code **always** in Home cycle; no `--text` → shown disabled + hint, Enter
  no-op; **no length selector** for Code.
- Renderer **Approach A**: new `internal/ui/code_stream_renderer.go` +
  viewport; `word_stream_renderer.go` + its golden tests untouched.
- Loader in new pure `internal/codetext` (keeps `words`/`typing` I/O-free).
- Mode-seam duplication (LengthsFor/Normalize/modeOrder/Record) guarded by a
  sync test, same discipline as the theme work.

## Out of scope

In-TUI paste/edit, syntax highlighting, snippet library, language
detection/per-lang stats, elastic tab stops, AFKTrim for code.

## Dependencies

No cross-plan deps — prior plans completed; v1.1.0 shipped (its plan.md
top-level status frontmatter is stale-cosmetic only).
