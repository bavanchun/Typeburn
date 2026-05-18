---
phase: 5
title: "Release execution & verification"
status: completed
priority: P1
effort: "1.5h"
dependencies: [4]
---

# Phase 5: Release execution & verification

## Overview

Cut the actual `v1.0.0`. Pre-flight gates, a **disposable pre-release tag dry-run of the full
publish path**, SHA-pinned tag on the exact CI-verified commit, then verify the published
release. Rollback is **fix-forward** (the sumdb is append-only — re-tagging a version that
reached the proxy poisons it permanently).

Red-team applied: F2/F6 (pin exact CI-green SHA, two separate pushes, forbid `--follow-tags`),
F3 (fix-forward rollback, never re-tag a proxied version), F10 (disposable-tag publish
dry-run — snapshot does NOT prove publish), F7 (partial-release detection + decision tree),
F11 (release notes from CHANGELOG.md), F15 (verify repo default token perms), F5 (proxy lag
is NOT a release blocker).

## Requirements

- Functional:
  - All phase-1–4 changes committed to `main` (conventional, no AI refs) and pushed.
  - **Repo Settings verified**: Actions → Workflow permissions = read-only default;
    `release.yml` `goreleaser` job `contents: write` override permitted; no branch
    protection blocking release creation by `GITHUB_TOKEN`.
  - **Disposable publish dry-run**: push a throwaway pre-release tag (e.g. `v0.0.0-rc.test`)
    → `release.yml` runs the full `test`+`goreleaser` publish path → verify assets +
    checksums + notes → `gh release delete` + delete tag locally & remotely. (Pre-release,
    unadvertised → negligible proxy fetch; safe to delete. Converts `v1.0.0` into an
    already-proven second run.)
  - Extract `CHANGELOG.md [1.0.0]` section → `.github/release-notes.md` (consumed by
    `release.yml` GoReleaser `--release-notes`).
  - Annotated tag `v1.0.0` created on the **exact** CI-verified SHA, pushed **separately**
    from the branch push. `git push --follow-tags` is **forbidden** (would double-fire).
  - GitHub Release `v1.0.0`: linux/darwin/windows × amd64/arm64 archives + `checksums.txt`
    + CHANGELOG-derived notes; post-publish asset-count assertion (Phase 3) GREEN.
  - `go install github.com/bavanchun/Typeburn@v1.0.0` → `typeburn --version` prints `v1.0.0`
    — **note:** may lag the module proxy up to ~1h; NOT a release blocker, retry/poll.
- Non-functional: zero broken public release; fix-forward rollback documented & primary.

## Architecture

```
PRE-FLIGHT (local): go test -race ; goreleaser check (no deprec) ; make snapshot (build/archive)
REPO CHECK: gh api repos/:owner/:repo/actions/permissions → read-only default confirmed
DRY-RUN:  git tag v0.0.0-rc.test ; git push origin v0.0.0-rc.test
            └─► release.yml full publish ─► verify assets/checksums/notes
            └─► gh release delete v0.0.0-rc.test --yes ; git push --delete origin v0.0.0-rc.test ; git tag -d
SHA PIN:  SHA=$(git rev-parse HEAD)  (must equal release.yml test-job-green commit)
PUSH:     git push origin main         (separate)
TAG:      git tag -a v1.0.0 $SHA -m … ; git push origin v1.0.0   (separate; NOT --follow-tags)
            └─► release.yml ─► GitHub Release v1.0.0 + asset-count assertion
VERIFY:   gh release view v1.0.0 --json assets ; go install …@v1.0.0 (poll ≤1h) ; --version==v1.0.0
```

## Related Code Files

- Create: `.github/release-notes.md` (extracted CHANGELOG `[1.0.0]`; committed for the action)
- Otherwise git refs + remote only.

## Implementation Steps

1. **Pre-flight (all pass before any push):** `go build ./... && go vet ./... &&
   test -z "$(gofmt -l .)"`; `go test ./... -race -count=1` GREEN; `goreleaser check`
   (no deprecation); `make snapshot` + snapshot binary `--version` sane.
2. **Repo perms check:** confirm Settings → Actions workflow permissions read-only;
   `contents: write` job override allowed; no blocking branch protection.
3. Extract `.github/release-notes.md` from `CHANGELOG.md [1.0.0]`.
4. Stage + commit phases 1–4 (conventional): `feat:` version pkg/flag; `build:` goreleaser+Makefile;
   `ci:` release workflow; `docs:` CHANGELOG/SECURITY/CONTRIBUTING/templates/README/roadmap +
   `.github/release-notes.md`.
5. `git push origin main`. Capture `SHA=$(git rev-parse HEAD)`.
6. **Disposable dry-run:** `git tag v0.0.0-rc.test $SHA && git push origin v0.0.0-rc.test`.
   Watch `release.yml` (`gh run watch`). Verify `gh release view v0.0.0-rc.test --json assets`
   = 7 assets + notes correct. Then **delete**: `gh release delete v0.0.0-rc.test --yes`;
   `git push --delete origin v0.0.0-rc.test`; `git tag -d v0.0.0-rc.test`.
7. Confirm the `release.yml` `test` job was GREEN for `$SHA` and `main` HEAD still == `$SHA`
   (no drift). If HEAD moved, re-evaluate — tag MUST be on the dry-run-proven SHA.
8. `git tag -a v1.0.0 $SHA -m "Typeburn v1.0.0"`; `git push origin v1.0.0`
   (**separate push; never `git push --follow-tags`**).
9. Watch `release.yml`; on completion the asset-count assertion (Phase 3) gates success.
10. **Verify:** `gh release view v1.0.0 --json assets --jq '.assets|length'` == expected;
    download one archive → `--version` == `v1.0.0`;
    `go install github.com/bavanchun/Typeburn@v1.0.0` (poll up to ~1h for proxy) →
    `typeburn --version` == `v1.0.0`.
11. Backfill real Release URL into `CHANGELOG.md`/roadmap; commit `docs: link v1.0.0 release`;
    push (this is the only post-tag commit; HEAD now moves past the tag — acceptable, tag is fixed).

## Success Criteria

- [ ] Pre-flight all GREEN; repo token perms verified read-only + write-override allowed
- [ ] Disposable `v0.0.0-rc.test` ran full publish path, verified, then fully deleted
- [ ] `v1.0.0` annotated tag on the exact dry-run-proven SHA; pushed separately (no `--follow-tags`)
- [ ] `release.yml` `test`+`goreleaser` GREEN; asset-count assertion passed
- [ ] Release has 6 archives + `checksums.txt` + CHANGELOG-derived notes
- [ ] `go install …@v1.0.0` → `typeburn --version` == `v1.0.0` (within proxy lag window)
- [ ] Release URL backfilled into CHANGELOG/roadmap

## Risk Assessment

- **sumdb is append-only / immutable.** ROLLBACK = **fix-forward to v1.0.1**. NEVER delete
  and re-tag `v1.0.0` once it reached the proxy/sumdb — that version becomes permanently
  uninstallable (`checksum mismatch`) for all users. Delete+retry of the *same* version is
  permitted ONLY for the disposable `v0.0.0-rc.test` (unadvertised, pre-proxy) or in the
  narrow window where the tag was pushed but `release.yml` has not run and zero `go install`
  occurred (rarely guaranteeable — prefer fix-forward).
- **Partial release** (job dies after N of 7 assets): the Phase 3 post-publish assertion
  fails the run. Detection: `gh release view v1.0.0 --json assets` count + spot checksum
  cross-check. Decision tree: (a) assets incomplete AND zero proxy fetch confirmed →
  `gh release delete v1.0.0` + delete tag + fix + retry SAME version; (b) any proxy fetch
  possible OR uncertain → **fix-forward v1.0.1**, leave v1.0.0 as a documented bad release.
- **Disposable dry-run is the real publish-path proof** — `make snapshot` proves only
  build/archive. The dry-run exercises `GITHUB_TOKEN`, `contents: write`, changelog/notes,
  checksum upload, and the assertion step on a deletable tag.
- Proxy lag on `go install` is expected (≤~1h) and is NOT a release failure — poll, don't roll back.

## Next Steps

`/ck:journal`. Optional follow-ups: enable Homebrew (provision tap repo + fine-grained PAT,
add `homebrew_casks:` block per CONTRIBUTING TODO), M2 precision fix → v1.1.0.
