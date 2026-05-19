---
phase: 3
title: "WS3 Homebrew Cask via GoReleaser"
status: completed
priority: P1
effort: "4-6h"
dependencies: [2]
---

# Phase 3: WS3 Homebrew Cask via GoReleaser

## Overview

Add a GoReleaser `homebrew_casks:` block that commits a cask wrapping the
prebuilt release archive to `bavanchun/homebrew-tap-typeburn`, giving
`brew install bavanchun/tap-typeburn/typeburn`. Cask (not formula) per
`CONTRIBUTING.md:99` — no user Go toolchain. Must preserve every release
invariant and make the dry-run safe.

## Requirements
- Functional: a tagged **stable** release commits a working cask to the tap
  repo; `brew install bavanchun/tap-typeburn/typeburn` → runnable `typeburn`.
  A **prerelease/`-rc`** tag commits NOTHING to the tap.
- Non-functional: `assets==7` provably still holds (verified, not assumed);
  GoReleaser pinned `v2.15.4`; actions SHA-pinned; no dep code in the
  token-bearing job; PAT scoped to tap repo only.

## Key Insight (red-team H1)
`CONTRIBUTING.md:91-99` is the committed spec: `homebrew_casks:`, tap repo,
secret `HOMEBREW_TAP_TOKEN`. The original plan diverged (`brews:`,
`HOMEBREW_TAP_GITHUB_TOKEN`). Decision locked: **cask + `HOMEBREW_TAP_TOKEN` +
`homebrew-tap-typeburn`** (user-provisioned repo is authoritative). This phase
also rewrites `CONTRIBUTING.md:91-99` to match (tap repo name + final command).

## Architecture
Cask `binary` stanza points at the deterministic `typeburn` binary pinned in
P2 (`builds.binary: typeburn`) — no rename guesswork (red-team H2/C3 root-c
caused upstream). Token: `homebrew_casks.repository.token` MUST be explicitly
`{{ .Env.HOMEBREW_TAP_TOKEN }}` — default is the wrong (Typeburn-scoped)
`GITHUB_TOKEN` → cross-repo 403 (red-team H6). `skip_upload: auto` so
prerelease tags push nothing (red-team C1). TDD gate: `make snapshot` +
`goreleaser check` + GoReleaser-docs citation; P4 dry-run is the publish-path
proof.

## Related Code Files
- Modify: `.goreleaser.yaml` — add `homebrew_casks:` block; add
  `release.prerelease: auto`
- Modify: `.github/workflows/release.yml` — inject `HOMEBREW_TAP_TOKEN` into the
  **GoReleaser step `env:` only** (not job-level); **remove `go build ./...`
  from `.goreleaser.yaml before.hooks`** (the `test` job `release.yml:41-42`
  already builds — eliminates dep-code execution in the token-bearing job,
  red-team H6); keep `assets==7` assertion + permissions unchanged
- Modify: `CONTRIBUTING.md:91-99` (cask confirmed; tap repo
  `homebrew-tap-typeburn`; secret `HOMEBREW_TAP_TOKEN`; final brew command);
  add **tap-rollback runbook** (red-team H4)
- Modify: `README.md` (replace "Homebrew: planned" with real cask command)
- External (user): `bavanchun/homebrew-tap-typeburn` (done); fresh PAT secret

## Implementation Steps (TDD)
1. **Gate G0 — burned token confirmed DEAD (independent, no dep on G1):**
   verify the chat-pasted token returns 401 (`gh api -H "Authorization: token
   <old>" /user` → Bad credentials). Record timestamp. Phase BLOCKED until G0
   passes. (red-team H5)
2. **Gate G1 — fresh PAT provisioned:** fine-grained PAT scoped to
   `homebrew-tap-typeburn` ONLY, contents r/w; added via
   `gh secret set HOMEBREW_TAP_TOKEN -R bavanchun/Typeburn` (not echoed).
   Shortest viable expiry.
3. **Baseline:** `make snapshot` (post-P2 determinism) → record dist/ artifact
   set (6 archives + checksums.txt; no cask in the release-asset set).
4. **GoReleaser-docs verification:** via context7/docs-seeker, cite v2 docs
   confirming `homebrew_casks:` commits to the tap repo and does NOT add a
   GitHub release asset. Record citation in this phase (red-team H9 — no
   unverified invariant).
5. Add `homebrew_casks:` to `.goreleaser.yaml`: `name: typeburn`,
   `repository: { owner: bavanchun, name: homebrew-tap-typeburn, token:
   "{{ .Env.HOMEBREW_TAP_TOKEN }}" }`, `binary: typeburn`, `homepage`,
   `description`, `skip_upload: auto`, cask `test`/`zap` as appropriate. Add
   `release.prerelease: auto`. Remove `go build ./...` from `before.hooks`.
6. **Re-run `make snapshot` + `goreleaser check`** → assert: exactly 6
   archives + checksums.txt = 7 release assets, cask `.rb` generated for the
   tap (NOT in release-asset set), cask references `typeburn` binary.
7. Wire `HOMEBREW_TAP_TOKEN` into the GoReleaser step `env:` only; confirm
   `permissions:` blocks unchanged; confirm `before.hooks` no longer runs
   `go build`.
8. Docs: `CONTRIBUTING.md:91-99` rewrite + tap-rollback runbook; README real
   cask command (`brew install bavanchun/tap-typeburn/typeburn`).

## Todo List
- [x] G0 burned token confirmed dead + revoked (user-completed)
- [x] G1 fresh scoped PAT added as `HOMEBREW_TAP_TOKEN` (user-completed)
- [x] `make snapshot` baseline recorded
- [x] GoReleaser v2 docs citation recorded (cask → tap, not release asset)
- [x] `homebrew_casks:` + `prerelease: auto` added; `before.hooks` go-build removed
- [x] snapshot+check: 7 release assets, valid cask, `typeburn` binary
- [x] token wired at step env only; permissions/assertion unchanged
- [x] CONTRIBUTING rewrite + rollback runbook + README updated

## Success Criteria
- [x] `make snapshot`+`goreleaser check` green; release-asset count provably 7
- [x] Prerelease tag → `skip_upload: auto` pushed nothing (proven in P4 dry-run)
- [x] `brew install bavanchun/tap-typeburn/typeburn` runnable (clean-container)
- [x] `release.yml` job permission scope unchanged; no dep code in token job
- [x] `CONTRIBUTING.md` consistent with implementation; rollback runbook present

## Risk Assessment
- **PAT leak via dep code (H6)** → `before.hooks` go-build removed; token at
  step env only; fine-grained tap-only; short expiry; G0 ensures old token dead.
- **Wrong token default → cross-repo 403 (H6)** → explicit
  `token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"`; `make snapshot` does NOT prove
  auth — P4 dry-run does.
- **`assets==7` drift (H9)** → docs-cited behavior + snapshot pre-check +
  P4 hard gate; never rely on the post-publish assertion alone.
- **Bad cask breaks `brew upgrade` for all users (H4)** → tap-rollback runbook:
  `git revert` the bad `Casks/typeburn.rb` in `homebrew-tap-typeburn` (manual,
  needs a HUMAN credential — NOT the CI PAT), then fix-forward patch release.
- **Dry-run publishes real cask (C1)** → `skip_upload: auto` +
  `prerelease: auto`; P4 asserts zero tap commits on the `-rc` dry-run.
