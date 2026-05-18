---
phase: 3
title: "Integration regression and release gate"
status: completed
priority: P1
effort: "1h"
dependencies: [1, 2]
---

# Phase 3: Integration regression and release gate

## Overview

The only point where Phase 1 + Phase 2 changes compile together. Run the full
regression + the proven v1.0.0 release pipeline checks, sync docs/CHANGELOG,
then — **only on explicit user approval** — cut `v1.0.1` fix-forward. Blocked
by Phases 1 and 2.

## Requirements

- Functional: tree builds with both changes; full `-race` suite GREEN;
  `goreleaser check` clean (no deprecation); `make snapshot` produces the
  6-archive + checksums set; CHANGELOG/roadmap updated for v1.0.1.
- Non-functional: no regression to v1.0.0 behavior outside M2/m4; release is
  fix-forward (never re-tag v1.0.0); tag only with user go-ahead.

## Architecture

```
merge P1+P2 on main ─► build+vet+gofmt ─► go test ./... -race -count=1
   └─► goreleaser check (no deprec) ─► make snapshot (binary --version sane)
   └─► CHANGELOG [1.0.1] + roadmap M2/m4 status ─► commit ─► push main
   └─► [USER APPROVAL GATE] ─► disposable v0.0.0-rc.test dry-run ─► verify
        ─► delete dry-run ─► tag v1.0.1 on proven SHA ─► release.yml ─► verify
```

## Related Code Files

- Modify: `CHANGELOG.md` (new `[1.0.1]` section: M2 fixed, MissedChars removed;
  move items out of `[Unreleased]`; add compare/tag links)
- Modify: `docs/project-roadmap.md` (M2 → ✅ Shipped v1.0.1; m4 → ✅ Removed v1.0.1)
- Modify: `.github/release-notes.md` (extract the new `[1.0.1]` section)
- No source changes here (integration/verification only)

## Implementation Steps

1. Confirm Phase 1 & 2 commits are both on `main` and the tree is clean.
2. `go build ./... && go vet ./... && test -z "$(gofmt -l .)"`;
   `go test ./... -race -count=1` → GREEN (whole repo, both changes together).
3. `goreleaser check` (grep-gate: no `deprecat`); `make snapshot`; extract a
   snapshot binary and confirm `--version` banner is sane (`vX` form).
4. Update `CHANGELOG.md`: add `## [1.0.1] - <date>` with
   `### Fixed` (sub-WPM new-best precision) and `### Removed` (always-zero
   MissedChars stat); refresh `[Unreleased]`/compare links.
5. Update `docs/project-roadmap.md` M2 and m4 status lines.
6. Regenerate `.github/release-notes.md` from the `[1.0.1]` section
   (same awk extraction used for v1.0.0; drop trailing blanks; append the
   `[1.0.1]:` link def).
7. **Commit:** `git commit -m "docs: changelog and roadmap for v1.0.1"`; then
   `git push origin main`; capture `SHA=$(git rev-parse HEAD)`.
8. **USER APPROVAL GATE** — releasing is irreversible (append-only sumdb).
   Present the SHA + planned tag and ask the user to approve cutting `v1.0.1`
   (or stop here with `main` updated). Do not tag without explicit approval.
9. On approval, reuse the v1.0.0 procedure exactly:
   disposable `v0.0.0-rc.test` on `$SHA` → watch `release.yml` → verify
   7 assets + non-empty notes + checksum cross-check → `gh release delete
   v0.0.0-rc.test --yes --cleanup-tag` → confirm HEAD still `$SHA` →
   `git tag -a v1.0.1 $SHA -m "Typeburn v1.0.1"` → `git push origin v1.0.1`
   (separate push, never `--follow-tags`) → watch `release.yml` →
   asset-count assertion GREEN.
10. Verify: `gh release view v1.0.1` 7 assets + notes; downloaded binary
    `--version` == `v1.0.1`; `go install …@v1.0.1` (poll ≤~1h, not a blocker).
11. Run `/ck:journal`.

## Success Criteria

- [ ] Both fixes compiled together; full `-race` suite GREEN; no v1.0.0 regression
- [ ] `goreleaser check` clean; `make snapshot` 6 archives + checksums
- [ ] CHANGELOG `[1.0.1]` + roadmap + `.github/release-notes.md` updated & committed
- [ ] `main` pushed; user explicitly approved the tag before any tag push
- [ ] (if approved) disposable dry-run verified then deleted; `v1.0.1` on proven SHA
- [ ] `release.yml` GREEN; release has 7 assets + notes; `--version` == `v1.0.1`

## Risk Assessment

- Parallel package breakage surfaces only here → step 2 is the hard gate;
  if build fails, bisect by phase, return to the owning phase, do not patch
  across ownership in this phase.
- Irreversible release → explicit user-approval gate (step 8) + disposable
  dry-run (step 9) before the real tag; fix-forward only, never re-tag v1.0.0.
- Proxy lag on `go install` is expected (≤~1h), not a release failure.
