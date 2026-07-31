---
phase: 5
title: "Verify And Docs Sync"
status: in-progress
priority: P2
effort: "2h"
dependencies: [4]
---

# Phase 5: Verify And Docs Sync

## Overview

Run the full CI gate, verify the feature against a real release download, and
update the documentation surfaces that this change actually invalidates.

## Requirements

- Functional: docs describe the new behavior and the new dependency accurately.
- Non-functional: `ci.yml`'s three checks green; binary under cap; no doc claim
  unverified against source.

## Architecture

Docs impact is real here (new user-visible behavior, new dependency, changed
internal contract), so per `documentation-management.md` the evergreen docs get
updated — but only the smallest owning surfaces:

| Doc | Why it changes |
|---|---|
| `docs/codebase-summary.md` | new `internal/cli/updateui` package; `internal/update` progress contract |
| `docs/system-architecture.md` | `typeburn update` now runs a second, inline Bubble Tea program outside `internal/app` |
| `docs/project-roadmap.md` | new shipped entry; the v2.3.0 "deferred follow-ups" list is unchanged (signing/delta updates still deferred) |
| `CLAUDE.md` | allowed-dependency line already covers `charm.land/*`; add `bubbles` to the architecture note describing the CLI/TUI seam |
| `CHANGELOG.md` | user-visible change |

`README.md` gets a line only if it documents `typeburn update` output — check
before editing rather than assuming.

This work supersedes item 2 of the completed
`plans/20260530-update-ux-polish/` plan (the flat `downloading…/verifying…/
installing…` lines). That plan stays `completed`; note the supersession in the
roadmap entry rather than editing a closed plan.

## Related Code Files

- Modify: `docs/codebase-summary.md`, `docs/system-architecture.md`,
  `docs/project-roadmap.md`, `CHANGELOG.md`, `CLAUDE.md`
- Verify only: `README.md`, `CONTRIBUTING.md`

## Implementation Steps

1. Full gate — all three are exactly what CI enforces:
   - `go test ./... -race -count=1`
   - `go vet ./...`
   - `gofmt -l .` (must print nothing)
2. `make lint && make size-check && make build`; record the final binary size
   and compare against the Phase 1 baseline.
3. Manual verification on a real terminal, all four paths:
   - default colored TTY run
   - `NO_COLOR=1` run — same layout, no color
   - piped run (`typeburn update --yes | cat`) — plain lines
   - narrow terminal (< 56 cols) — plain lines
4. Cancellation check: `ctrl+c` mid-download, then confirm the binary's
   directory holds no `typeburn_*.tar.gz`, `checksums.txt`, or extracted
   `typeburn` leftover.
5. Update the docs in the table above. Verify each claim against source before
   writing it.
6. Confirm no plan-ID, phase number, or audit label leaked into any code
   comment, test name, or commit message (`review-audit-self-decision.md`).

## Success Criteria

- [x] `go test ./... -race -count=1` green
- [x] `go vet ./...` clean
- [x] `gofmt -l .` empty
- [x] `make size-check` passes; final size recorded in the roadmap entry (9,002,530)
- [ ] All four manual paths verified and their output pasted into the PR body
      — **OUTSTANDING.** No pty in the implementation environment; must be run
      by hand before merge.
- [ ] Cancellation leaves zero leftover files — **OUTSTANDING as an executed
      check.** `stopApply` now waits for `Apply` to unwind so its `defer
      release()` / `defer cleanup()` run, and that ordering is asserted by
      `TestStopApply_WaitsForTheUpdateToUnwind`, but no real interrupted
      download has been run against a real install directory.
- [x] Docs updated and every claim traced to source
- [x] No plan/phase identifiers in code, tests, or commit messages

## Risk Assessment

**Risk:** docs drift — describing intended rather than shipped behavior.
**Mitigation:** each doc edit cites the file and symbol it describes; the
reviewer checks the citation, not the prose.

**Risk:** the manual verification is skipped because the automated gate is
green. The automated tests cannot prove the animation actually looks right in a
real terminal — that is precisely the part under test here.
**Mitigation:** the four manual paths are success criteria, and their captured
output is required in the PR body.
