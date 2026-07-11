---
phase: 3
title: "Reconcile install documentation and release contracts"
status: done
effort: "M"
---

# Phase 3: Reconcile install documentation and release contracts

## Objective

Align user and developer truth with `/v2/cmd/typeburn`, preserving history and repository URLs.

## File Inventory

| Files | Required update |
|---|---|
| `README.md` | badge, install, binary case, source commands, updater advice |
| `CLAUDE.md`, `CONTRIBUTING.md` | module and entrypoint truth |
| `docs/code-standards.md`, `docs/codebase-summary.md` | tree and commands |
| `docs/system-architecture.md`, `docs/cli-reference.md` | architecture and install paths |
| `docs/project-roadmap.md` | corrective release truth |
| `docs/project-overview-pdr.md` | modify only if live inspection proves drift |
| `CHANGELOG.md`, `.github/release-notes.md` | exact v2.5.1 correction |

## Implementation Steps

1. Use `/v2/cmd/typeburn` for Go install and `./cmd/typeburn` for source commands.
2. Replace root-main descriptions with `cmd/typeburn/main.go`.
3. Add v2.5.1 Fixed notes; retain the v2.5.0 section and link.
4. Target scans to README, contributor docs, current docs, and runtime update
   code/tests; explicitly exclude historical changelog sections, plans, journals.
5. Reject `/v2` inserted into GitHub release/raw/security URLs and positively
   verify updater API/download URLs remain repository-root paths.
6. Update the disposable-release runbook to state the unique v0 tag is
   archive-only and intentionally cannot prove the `/v2` module channel.

## Content Contract Checklist

- Examples use `@latest` and `@v2.5.1` appropriately.
- Installed binary is lowercase `typeburn`.
- v2.5.0 is never described as module-channel repaired.

## Test Matrix

| Scan | Expected |
|---|---|
| Old install command | zero current actionable hits |
| Root source command | zero stale current guidance |
| GitHub URLs | repository-root URLs retain no `/v2` |
| Notes/changelog | version, date, links, correction agree |

## Dependencies

- Requires Phase 2 paths; blocks release preparation.

## Success Criteria

- [x] Current install, compile, run, and update instructions agree exactly.
- [x] Historical truth and non-module GitHub URLs remain intact.
- [x] Commits: `fix(docs): correct stale install tags, roadmap truth, entrypoint
  description, plan checkboxes` (`21beeded`) and follow-up PDR reconciliation.
