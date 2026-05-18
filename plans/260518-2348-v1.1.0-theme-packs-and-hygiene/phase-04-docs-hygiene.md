---
phase: 4
title: "Docs Hygiene"
status: pending
priority: P2
effort: "1.5h"
dependencies: [1]
---

# Phase 4: Docs Hygiene

## Overview
Fix stale/false statements now that the app is publicly released; correct the
misleading word-wrap comment; reconcile the core-import architecture rule with
reality. Doc + one comment only — no behavior change. One commit.

## Requirements
- Functional: no doc claims a pre-v1.0.0 state; the core-import rule matches
  actual code; the word-wrap comment describes what the code does.
- Non-functional: zero code-behavior change (comment-only in `.go`);
  `gofmt`/`vet`/`test -race` still green (sanity, nothing should change).

## Architecture
Pure documentation/comment edits. The core-import rule is **doc-vs-reality
drift** (audit finding #2): `config/keymap.go:3` imports bubbletea,
`theme/mono-theme.go:6` imports lipgloss. Per the locked decision, **relax the
doc to match reality** (no second consumer ⇒ YAGNI; do not refactor working
code). Word-wrap: `word_stream_renderer.go:84-92` comment claims a scan-back
"word-aware" wrap; impl hard-wraps at the cell boundary. Correct the comment
to state the actual behavior (char-cell hard-wrap with post-space flush);
keep CJK `cellW:=1` deferral note pointing to roadmap m5.

## Related Code Files
- Modify: `README.md` (line ~9 stale badge note; any other pre-tag wording),
  `docs/project-roadmap.md` (badge/pre-1.0 notes; mark v1.1.0 scope),
  `docs/system-architecture.md` (core-import rule wording),
  `CLAUDE.md` (Architecture section "no bubbletea/lipgloss imports" → reflect
  that `config` binds tea key types and `theme` returns lipgloss styles
  by design; reusability is aspirational, not enforced),
  `CHANGELOG.md` (author one `[Unreleased]` block covering ALL of v1.1.0's
  user-visible changes: theme packs, persistence-failure notice, doc
  corrections — Phase 4 is the sole parallel-set owner of this file; Phases
  2 & 3 must NOT touch it),
  `internal/ui/word_stream_renderer.go` (comment lines ~84-92, 124-126 only)
- Create / Delete: none

## Implementation Steps
1. `README.md:9`: remove/replace "Release/pkg.go.dev badges 404 until the
   first tag is published — expected pre-`v1.0.0`." with accurate current
   wording (badges resolve; note the ~1h proxy lag already documented at
   lines 39-41 stays). Scan the rest of README for other pre-tag phrasing.
2. `docs/project-roadmap.md`: drop pre-1.0 badge caveats; add a short
   v1.1.0 entry (theme packs + hygiene) under the post-1.0 section; keep the
   parent-dir-fsync and CJK items as **accepted/deferred** (do NOT reopen —
   roadmap-accepted, no new data).
3. `docs/system-architecture.md` + `CLAUDE.md`: reword the "core packages
   must not import Bubble Tea/Lip Gloss" rule to the truthful invariant:
   pure-logic packages (`typing`, `metrics`, `words`, `storage`, `version`)
   stay UI-free; `config` may reference tea key types for keymaps and
   `theme` returns lipgloss styles by design — these are the styling/input
   boundary, not reusable-core. State it as intentional, not a violation.
4. `word_stream_renderer.go`: rewrite the doc comment to describe actual
   behavior — "hard-wrap at `width` cells; a word longer than the line is
   split; a trailing space at/after the boundary flushes the line" — and the
   `cellW := 1` comment to reference roadmap m5 (CJK deferral) without
   claiming word-awareness. **No logic edits.**
5. `CHANGELOG.md`: add a single `[Unreleased]` section (Keep a Changelog
   style) with the v1.1.0 user-visible changes — Added: 6 theme packs
   (Solarized Dark/Light, Dracula, Nord, Gruvbox Dark/Light); Added:
   non-blocking notice on persistence failure; Fixed/Docs: post-1.0 status,
   core-import rule, word-wrap comment. Phase 6 later renames this to
   `[1.1.0]`.
6. `make fmt && make lint && make test-race` (must be unchanged-green —
   proves the comment edit is behavior-neutral).
7. Commit: `docs: correct post-1.0 status, core-import rule, word-wrap
   comment; changelog [Unreleased] for v1.1.0`.

## Success Criteria
- [ ] No doc states a pre-v1.0.0 / "until first tag" condition as current.
- [ ] Architecture rule matches actual imports (no false "violation").
- [ ] Word-wrap comment matches code behavior; CJK note → m5.
- [ ] `CHANGELOG.md` has one `[Unreleased]` block covering all 3 phases'
  user-visible changes.
- [ ] `gofmt`/`vet`/`test -race` green and unchanged (behavior-neutral).
- [ ] One commit on the feature branch.

## Risk Assessment
- **Accidental logic edit in word_stream_renderer.go:** restrict to comment
  lines; step 5 green-unchanged proves neutrality.
- **Reopening accepted decisions:** explicitly keep parent-dir fsync + CJK as
  accepted/deferred per review-audit discipline (no new data → no reversal).
- **CLAUDE.md is user-owned guidance:** reword the technical invariant only;
  do not alter workflow/rules sections.
