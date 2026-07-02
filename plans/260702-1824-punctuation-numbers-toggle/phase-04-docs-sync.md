---
phase: 4
title: Docs Sync
status: completed
effort: small
priority: P3
dependencies:
  - 1
  - 2
  - 3
---

# Phase 4: Docs Sync

## Overview

Update user-facing docs to reflect the shipped feature: README features list +
Usage/Keybindings if any new interaction surface, and `docs/project-roadmap.md`
to move this from "recommended in brainstorm" to shipped, consistent with how
every prior feature (Strict Mode, Heatmap, Self-Update) is documented there.

## Requirements

- Functional:
  - `README.md`: add a bullet to the `## Features` list describing
    punctuation/numbers toggles (mirror the existing "Strict mode" bullet
    style at line ~20).
  - `README.md`: if `config set punctuation|numbers` is a new documented CLI
    surface, add example usage near the existing `typeburn config set theme
    nord` example (line ~101).
  - `docs/project-roadmap.md`: add a new shipped entry under "Shipped Post-1.0"
    following the exact style of the `v2.5.0 (Strict Mode)` entry (line ~250),
    with the correct next version number (confirm current version via
    `internal/version` or the latest git tag before writing — do not guess).
- Non-functional: no scope creep — do not rewrite unrelated doc sections.

## Architecture

N/A — documentation-only phase.

## Related Code Files

- Modify: `README.md`
- Modify: `docs/project-roadmap.md`
- Check (no edit unless behavior changed): `docs/code-standards.md`,
  `docs/system-architecture.md` — only touch if the transform introduced a
  new architectural pattern (it should not have, per phase 1's design note).

## Implementation Steps

1. Confirm current shipped version: `git tag --sort=-v:refname | head -1` or
   check `internal/version` for the latest resolved version, to pick the
   correct next version number for the roadmap entry.
2. Add README `## Features` bullet for punctuation/numbers toggles.
3. Add README CLI usage example if `config set punctuation|numbers` needs
   documentation (check if other boolean config keys like `strict_mode` or
   `update_check` already have a documented example — mirror that, or skip if
   the existing `config set theme nord` example is treated as sufficiently
   representative of the pattern).
4. Add `docs/project-roadmap.md` shipped entry, matching the terse style of
   prior entries (one paragraph, ship date, PR reference once known).
5. Verify no other doc references stale info (e.g. "4 settings only" design
   constraint at roadmap line ~183 — this line currently states "4 settings
   only" as a v1 design constraint; confirm whether it needs updating to
   reflect the actual current settings count, since Strict Mode already
   pushed past 4 and this phase adds 2 more — check if a correction is owed
   here rather than compounding the staleness).

## Success Criteria

- [x] README `## Features` list includes punctuation/numbers toggle bullet
- [x] `docs/project-roadmap.md` has a new shipped entry with correct version (v2.6.0)
- [x] No stale "4 settings only" claim left uncorrected (annotated as v1-scope/superseded)
- [x] No unrelated doc sections modified

## Risk Assessment

- **Risk:** Version number guessed wrong (roadmap entries are chronological
  and version-numbered).
  **Mitigation:** Step 1 confirms actual current version from git tags before
  writing, not assumption.
- **Risk:** Doc scope creep — rewriting sections beyond what changed.
  **Mitigation:** Explicit non-functional requirement above; touch only the
  listed files/sections.
