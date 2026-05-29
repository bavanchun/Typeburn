---
phase: 3
title: Surface release URL + docs
status: completed
priority: P2
effort: 1h
dependencies:
  - 2
---

# Phase 3: Surface release URL + docs

## Overview

Surface `Release notes: <url>` in `typeburn update` output so users see what
changed, mirroring the exact wording `version --check-update` already uses.
`Result.ReleaseURL` already exists and is repo-guarded — no new fetching. Then
update docs and run the full CI gate.

## Requirements

- Functional: when an update is available, both `update` and `update --check`
  print `Release notes: <ReleaseURL>` (only when non-empty).
- Non-functional: wording matches `version --check-update` (`Release notes: %s`);
  empty/guarded-away URL prints no broken line.

## Architecture

- **`cmd_update.go`:**
  - `reportCheck` upgrade-available branch (`--check`): append
    `Release notes: <result.ReleaseURL>` when non-empty (matches
    `renderVersionCheckHuman`).
  - Install path: print the release-notes line before the confirm prompt (so the
    user can decide informed). Skip the line when `ReleaseURL == ""`.
  - The managed-install message may also append the notes line (optional; keep if
    it doesn't push the file past 200 LOC).
- **In-app hint (Phase 1 file):** intentionally NOT changed — the footer is one
  width-capped muted line; a URL would overflow. Release URL is CLI-only. Document
  this decision.
- **Wording source of truth:** `cmd_version.go:renderVersionCheckHuman` uses
  `Release notes: %s` — reuse verbatim for consistency.

## Related Code Files

- Modify: `internal/cli/cmd_update.go`
- Modify: `internal/cli/cmd_update_test.go`
- Modify: `CHANGELOG.md` (`[Unreleased]` → add the three UX entries)
- Modify: `README.md` / usage docs IF they quote `update` output (verify first)
- Modify: `docs/codebase-summary.md` if update-UX behavior described there

## Implementation Steps (TDD)

1. **Red:** in `cmd_update_test.go`, assert that with a stubbed
   `UpgradeAvailable` result carrying a `ReleaseURL`, both `update` (install path,
   pre-confirm) and `update --check` stdout contain `Release notes: <url>`. Add a
   case asserting NO `Release notes:` line when `ReleaseURL == ""`.
2. **Green:** add the conditional `Release notes:` prints in `reportCheck` and the
   install path of `runUpdate`.
3. Run `go test ./internal/cli/ -race -count=1` → green.
4. **Docs:** add `CHANGELOG.md` `[Unreleased]` entries for the hint fix, progress
   feedback, and release-notes surfacing. Grep README/docs for quoted `update`
   output and sync if present.
5. **Full CI gate:** `go test ./... -race -count=1`, `go vet ./...`,
   `gofmt -l .` (empty), `make size-check` (exit 0).

## Success Criteria

- [ ] `update` and `update --check` print `Release notes: <url>` when available.
- [ ] No release-notes line when `ReleaseURL` is empty.
- [ ] Wording matches `version --check-update`.
- [ ] `CHANGELOG.md` `[Unreleased]` updated; docs synced if affected.
- [ ] Full CI gate green; binary size-check passes.

## Risk Assessment

- Low. Additive output only; guarded on non-empty URL so no broken line.
- `ReleaseURL` is already repo-prefix-guarded in `check.go` — no SSRF/arbitrary
  URL risk introduced.
- Confirm the asserted `Release notes:` lines don't collide with Phase 2's stage
  lines in the same stdout buffer (order: notes line before confirm, stage lines
  after confirm).
