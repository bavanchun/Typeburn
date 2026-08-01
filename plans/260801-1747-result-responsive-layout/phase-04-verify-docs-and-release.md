---
phase: 4
title: "Verify, Docs, And Release"
status: pending
priority: P2
effort: "2h"
dependencies: [3]
---

# Phase 4: Verify, Docs, And Release

## Overview

Run the CI gate, look at the screen on a real terminal at several widths, sync
the docs, and ship v2.8.0 — which is also the release that finally exercises the
animated updater added in v2.7.0.

## Requirements

- Functional: docs describe the shipped layout and the de-duplication.
- Non-functional: `ci.yml`'s three checks green; binary under cap.

## Architecture

### The manual check is the point

This is a visual change. The automated tests prove structural invariants — line
counts, widths, `NO_COLOR` parity, reveal stability — none of which is the same
as the screen looking right. v2.6.0 shipped a Result redesign whose
wide-terminal degradation was only noticed when a screenshot was taken, which is
precisely the failure mode a green test suite cannot catch.

Look at it at 80, 120, and full-screen, colored and `NO_COLOR`.

### Docs impact

| Doc | Why |
|---|---|
| `docs/codebase-summary.md` | new `result_layout.go`; graph stretching; grid contents |
| `docs/project-roadmap.md` | v2.8.0 ship entry; current-stable line |
| `CHANGELOG.md` | user-visible layout change **and** the de-duplication |

`docs/design-guidelines.md` gets the width-cap rule if that file exists — check
before assuming it does.

### Release

Standard runbook: release-prep PR (CHANGELOG roll, `.github/release-notes.md`,
roadmap), squash-merge, then annotate and push `v2.8.0` on the merged commit in
a separate push.

**Bonus verification available here, once only:** the installed binary is
v2.7.0, which contains the animated updater. Running `typeburn update` against
the published v2.8.0 is the first and only chance to watch that feature work
against a real download — a self-updater always runs the *old* binary's update
code, so v2.7.0's animation could not be observed when v2.7.0 itself was
installed. Worth doing deliberately rather than incidentally.

## Related Code Files

- Modify: `docs/codebase-summary.md`, `docs/project-roadmap.md`, `CHANGELOG.md`,
  `.github/release-notes.md`
- Verify only: `README.md`, `docs/design-guidelines.md`

## Implementation Steps

1. `go test ./... -race -count=1`, `go vet ./...`, `gofmt -l .` (must be empty).
2. `make lint && make size-check && make build`.
3. Manual pass on a real terminal: 80 / 120 / full-screen, each with and without
   `NO_COLOR`. Also run a **long** test (100 words) so the downsampling path is
   seen, not just the stretched one.
4. Update the docs above; verify each claim against source.
5. Release-prep PR → CI green → squash-merge.
6. Tag `v2.8.0` annotated on the merged commit; push the tag separately.
7. Verify the published release: 7 assets, notes non-empty, checksum of one
   archive verified against `checksums.txt`.
8. Run `typeburn update` from the installed v2.7.0 binary and watch the animated
   block against a real download. Capture what it looks like.

## Success Criteria

- [ ] `go test ./... -race -count=1` green
- [ ] `go vet ./...` clean; `gofmt -l .` empty
- [ ] `make size-check` passes
- [ ] Manual pass at 3 widths × 2 color modes, plus one long run
- [ ] Docs updated with claims traced to source
- [ ] v2.8.0 published; 7 assets; checksum verified
- [ ] Animated updater observed working against a real download

## Risk Assessment

**Risk:** the automated gate is green and the manual look gets skipped — the
same path by which the current defect shipped.
**Mitigation:** the manual pass is a success criterion, and its output goes in
the PR body.

**Risk:** a stable tag is immutable; the Go module proxy and sumdb are
append-only. A defect in v2.8.0 is fixed forward with v2.8.1, never by
re-tagging.
**Mitigation:** verify the published assets before calling the release done.
