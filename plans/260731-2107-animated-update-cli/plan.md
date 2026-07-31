---
title: "Animated Update CLI"
description: >-
  Replace the flat text output of `typeburn update` with an inline Bubble Tea
  progress block (bubbles/progress + bubbles/spinner) driven by real byte-level
  download progress, with a plain-text fallback on non-TTY and narrow terminals.
status: in-progress
priority: P2
effort: "1.5d"
branch: feat/animated-update-cli
tags: [cli, ui, update]
blockedBy: []
blocks: []
created: 2026-07-31
---

# Animated Update CLI

## Overview

`typeburn update` currently prints flat lines and blocks silently through a
~4 MB download (`internal/cli/cmd_update.go:126-132`):

```
updating v2.5.1 → v2.6.0 ...
  downloading...
  verifying...
  installing...
updated v2.5.1 → v2.6.0. restart typeburn to use the new version.
```

The TUI shipped a full motion layer in v2.4.1; the updater is the last static
surface. This plan gives it a framed, animated progress block built on Charm's
own components, with real percentages rather than a spinner pretending to know.

Selected from three prototyped layouts (checklist / single-focus / boxed) — the
boxed variant was chosen. A working prototype of all three was built and run
before this plan; the layout, the two `NO_COLOR` traps, and the binary-size
delta below are measurements from it, not estimates.

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | Framed animated progress block on a capable TTY | P1 |
| 2 | Real byte-level download progress, never a fabricated percentage | P1 |
| 3 | Plain-text fallback preserved exactly for non-TTY, narrow, and scripted use | P1 |
| 4 | `NO_COLOR`/`mono` layout-identical, attributes only | P1 |
| 5 | Cancellation that can never corrupt the binary swap | P1 |
| 6 | Stay inside the 10 MiB binary cap | P2 |

## Non-goals

- Animating `update --check` (detect-only, exits immediately).
- Changing the trust model — checksums over HTTPS, unsigned binaries, per
  `SECURITY.md`.
- Adding `huh` for the confirmation prompt: it would force a rewrite of the
  `isInteractive` seam and the confirm/cancel tests for negligible gain.
- Bringing Bubble Tea into any other CLI subcommand.
- Reflowing or responsively resizing the box — fixed width, degrade instead.

## Key decisions (settled during brainstorm)

**Use Charm components rather than hand-rolling.** `bubbles/progress` v2.1.1
already provides spring-smoothed fill (`SetSpringOptions`, via `harmonica`),
gradient blends (`WithColors`/`WithColorFunc`), and custom fill characters —
exactly what a hand-rolled ANSI repainter would have had to reimplement on top
of `internal/anim`. `CLAUDE.md`'s allowlist already covers `charm.land/*`, so no
new dependency approval is required.

**Binary size is not a constraint.** Measured before planning, darwin/arm64,
`-trimpath -ldflags '-s -w'`:

| Build | Bytes |
|---|---|
| baseline | 8,934,370 |
| + `bubbles/progress` + `bubbles/spinner` | 8,951,522 |
| delta | **+17,152** |
| headroom under the 10,485,760 cap | 1,534,238 |

**Progress is state, not an event stream.** The `Apply` goroutine writes the
latest `Progress` under a mutex; the render loop polls it at 40 ms. This avoids
both channel backpressure on the download and dropped stage transitions.

**The install stage is not cancellable.** `replaceBinary` is the atomic swap the
whole updater's safety rests on, so `ctrl+c` is refused from `StageInstalling`
onward.

## Phases

| # | Phase | Status | Depends on |
|---|-------|--------|------------|
| 1 | [Dependency Baseline](./phase-01-start.md) | Completed | — |
| 2 | [Byte-Level Progress Contract](./phase-02-byte-level-progress-contract.md) | Completed | 1 |
| 3 | [Boxed Update Renderer](./phase-03-boxed-update-renderer.md) | Completed | 2 |
| 4 | [Wire Into Update Command](./phase-04-wire-into-update-command.md) | Completed | 3 |
| 5 | [Verify And Docs Sync](./phase-05-verify-and-docs-sync.md) | In progress | 4 |

Phases are strictly sequential: 2 defines the data 3 renders, and 4 is the only
phase that changes user-visible behavior. Phases 1–3 are individually shippable
without altering any output.

## Target output

```
╭── typeburn update ─────────────────────────────╮
│                                                │
│  v2.5.1   →   v2.6.0     4.3 MB                │
│                                                │
│  ✓  checksums               64 KB              │
│  ⠋  downloading  ████████░░░░░░░  52%          │
│  ·  verifying                                  │
│  ·  installing                                 │
│                                                │
╰────────────────────────────────────────────────╯
```

## Success Criteria

- [ ] `go test ./... -race -count=1`, `go vet ./...`, and empty `gofmt -l .` —
      the exact CI gate
- [ ] `make size-check` passes
- [ ] Colored and `NO_COLOR` frames identical in line count and per-line width;
      `NO_COLOR` emits no color SGR
- [ ] Plain path output matches a full-string golden, differing from `main` only
      by the added `  checksums...` line
- [ ] All four runtime paths manually verified: colored TTY, `NO_COLOR`, piped,
      narrow terminal — **outstanding**, see phase 4
- [ ] `ctrl+c` mid-download leaves zero leftover files; cancellation refused
      during install
- [ ] Every touched Go file under 200 LOC
- [ ] `docs/` updated with claims traced to source

## Constraints

- Protected `main`: branch → PR → green `ci.yml` → squash-merge
  (`CLAUDE.md`, Git Workflow). Never commit directly to `main`.
- Pure-logic packages (`typing`, `metrics`, `words`, `codetext`, `storage`,
  `version`) stay UI-free — this plan touches none of them.
- Colors come from `internal/theme` Roles only; no hex literals in UI code.
- Release engineering files are out of scope; `ci.yml` stays byte-identical.

## Relationship to prior work

Supersedes item 2 of the completed `plans/20260530-update-ux-polish/` (the flat
stage lines shipped in v2.4.0). That plan remains `completed`; the supersession
is recorded in the roadmap entry, not by editing a closed plan.

## Open questions

None. The three items that could have blocked — dependency approval, binary-size
headroom, and layout choice — were each resolved with evidence before this plan
was written.

<!-- slug: animated-update-cli -->
