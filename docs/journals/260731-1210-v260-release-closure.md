# v2.6.0 Release Closure — Journal

**Date:** 2026-07-31
**Plan:** `plans/260731-1004-v260-release-closure/` (3 phases, completed)
**Release:** https://github.com/bavanchun/Typeburn/releases/tag/v2.6.0

## What happened

Closed the Result-redesign delivery loop by cutting v2.6.0 per the
CONTRIBUTING runbook, same-day and defect-free:

1. **Release-prep PR #62** — rolled CHANGELOG `[Unreleased]` → `[2.6.0]`,
   extracted `.github/release-notes.md` verbatim (diff-verified in-script,
   not by eye), marked the roadmap entry shipped, and swept in the stray
   v2.5.0 journal that had sat untracked since 2026-07-02. Squash-merged
   as `5858058f`.
2. **Dry-run → tag** — `v0.0.0-rc.test` on `5858058f` published 7 assets
   with correct 2.6.0 notes (prerelease-flagged as designed); deleted with
   `--cleanup-tag`; annotated `v2.6.0` pushed alone on the identical SHA;
   `release.yml` green; release is latest, not prerelease.
3. **Channel verification (all four + Homebrew)** —
   - Archive: darwin_arm64 tarball sha256 OK against checksums.txt, binary
     reports `v2.6.0 (5858058, …)`.
   - `go install …@v2.6.0`: worked immediately — no proxy lag this time
     (v2.5.x had ~1 h).
   - `typeburn update --check` from a real v2.5.1 binary: sees v2.6.0.
   - `install.sh` into temp BIN_DIR: sha verified, v2.6.0 installed.
   - Homebrew cask: GoReleaser had already regenerated `Casks/typeburn.rb`
     at 2.6.0 with shas matching checksums.txt — no manual bump needed.

## Notes for next release

- The dry-run/tag-same-SHA discipline paid for itself in confidence and cost
  ~10 minutes total; keep it.
- `gh release view` has no `isLatest` field — use `gh release list` to
  confirm the Latest marker.
- Go proxy ingest was instant this release; keep the ~1 h tolerance in the
  runbook anyway.

## Loop state

Delivery loop for the Result redesign is fully closed: brainstorm → plan →
implement → review → ship → merge → release → verify. The next item is
strategic, not technical: pick the user-feedback channel (roadmap "Next",
item 1) before starting any new feature.
