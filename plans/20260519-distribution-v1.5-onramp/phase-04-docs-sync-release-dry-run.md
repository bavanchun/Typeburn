---
phase: 4
title: "Docs Sync + Safe Release Dry-Run"
status: completed
priority: P1
effort: "3-4h"
dependencies: [3]
---

# Phase 4: Docs Sync + Safe Release Dry-Run

## Overview

Integration gate and **hard prerequisite for the real v1.5 tag**. Verify
lockstep, sync docs, then prove the full publish path via a **safe** disposable
prerelease-tag dry-run that publishes NOTHING durable to users.

## Requirements
- Functional: a disposable `-rc.test` tag exercises the publish job, is flagged
  prerelease (excluded from `releases/latest`), pushes NO cask to the tap, then
  is fully torn down.
- Non-functional: every release invariant green; docs reflect reality; no real
  tag until this phase passes.

## Key Insight (red-team C1 — the load-bearing fix)
`release.yml:14-17` triggers on `v*`; the disposable tag `v0.0.0-rc.test`
(`CONTRIBUTING.md:76-79`) matches it; `.goreleaser.yaml:71 draft: false`.
Without P3's `release.prerelease: auto` + cask `skip_upload: auto`, this
"dry-run" would publish a REAL public release + live cask. P4 must AFFIRMATIVELY
ASSERT those mitigations fired — not assume them.

## Architecture
End-to-end verification only — no feature code. Pre-tag static gate
(`goreleaser release --snapshot` asset-count assertion) catches drift BEFORE
any tag. The disposable prerelease tag then proves the privileged auth/push
path that snapshot cannot.

## Related Code Files
- Modify: `docs/codebase-summary.md`, `docs/system-architecture.md`,
  `docs/deployment-guide.md` (install channels, Go 1.25 floor, cask)
- Modify: `docs/project-roadmap.md` / `CHANGELOG.md` (v1.5 entry)
- Verify (no edit expected): 5 lockstep files, `release.yml`, `.goreleaser.yaml`

## Implementation Steps
1. **Pre-tag static gate:** `goreleaser release --snapshot --clean` +
   `goreleaser check`; assert dist/ = exactly 6 archives + `checksums.txt`
   (would-be 7 release assets), cask `.rb` present for tap but NOT in the
   release-asset set. FAIL here blocks everything (red-team H9).
2. **Lockstep audit:** `grep -rn "1\.2[0-9]" go.mod .github CONTRIBUTING.md
   README.md` → single consistent `1.25` (or documented value). Assert
   `README.md:33-39` case-sensitivity + proxy-lag caveats still present
   (red-team M2).
3. **Whole-plan/doc wording reconcile:** README/CONTRIBUTING/docs install
   channels (go install caveat, install.sh one-liner + honest trust boundary,
   `brew install bavanchun/tap-typeburn/typeburn`) match actual P1-P3 outcomes.
   No stale `brews:`/`HOMEBREW_TAP_GITHUB_TOKEN`/"Homebrew: planned".
4. **Docs sync:** `docs/` deployment+architecture+roadmap + CHANGELOG v1.5.
5. **Safe disposable dry-run:** push `v0.0.0-rc.test`. Assert ALL of:
   (a) release job green incl. `assets==7`;
   (b) GitHub release flagged **prerelease** (NOT in `releases/latest`);
   (c) `homebrew-tap-typeburn` received **ZERO commit** (skip_upload:auto);
   (d) `install.sh` against `releases/latest` does NOT resolve the rc tag.
   Then full teardown (delete tag + release; verify tap repo clean) per a
   scripted checklist.
6. **Live verify (clean container, mandatory):** in a fresh container (no
   prior Go/PATH) run BOTH the `install.sh` one-liner and (post a real
   stable tag, or a controlled tap test) `brew install
   bavanchun/tap-typeburn/typeburn` → runnable `typeburn`. `~/.local/bin`
   absent by default here — proves P2 `mkdir -p`.

## Todo List
- [x] Pre-tag `goreleaser snapshot`+`check` asset-count gate passes (7)
- [x] Lockstep grep audit consistent; README caveats present
- [x] Doc wording reconciled (no stale brews/secret/planned)
- [x] `docs/` + CHANGELOG + release-notes synced for v1.5 (PR #13)
- [x] Safe dry-run: prerelease flagged, ZERO tap commit, latest unaffected, torn down
- [x] Clean-container brew + install.sh both runnable (real v1.5.0)

## Success Criteria
- [x] Pre-tag static gate proves 7 release assets, cask not an asset
- [x] All 5 Go-version sites consistent at 1.25
- [x] Dry-run: prerelease-flagged, tap untouched, `releases/latest` unaffected,
      fully cleaned up — zero durable user-visible artifact
- [x] Both install channels verified on a clean container (real v1.5.0)
- [x] `docs/` + CHANGELOG accurate for v1.5

## Risk Assessment
- **Dry-run leaks a real release/cask (C1)** → success criteria explicitly
  assert prerelease flag + zero tap commit; if either fails, STOP, do not tag.
- **Dry-run litter** → scripted teardown + post-run tap/releases clean check.
- **Clean-env test skipped** → not allowed; container/VM mandatory (local
  machine hides `mkdir -p`/PATH failures).
- **Lockstep miss surfaces here** → acceptable (this is the gate); fix +
  re-audit; never tag real v1.5 with drift.
