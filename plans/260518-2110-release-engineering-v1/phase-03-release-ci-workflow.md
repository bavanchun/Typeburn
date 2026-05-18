---
phase: 3
title: "Release CI workflow"
status: completed
priority: P1
effort: "1.5h"
dependencies: [2]
---

# Phase 3: Release CI workflow

## Overview

New `.github/workflows/release.yml`, tag-triggered, that **self-gates with its own `test`
job** (because `ci.yml` does NOT run on tag pushes — verified) then publishes via a
least-privilege, SHA-pinned GoReleaser job. Existing `ci.yml` stays byte-for-byte untouched.

Red-team applied: F2 (self-gating test job — tag ref has no ci.yml coverage), F1 (privileged
job runs no tests), F6 (concurrency group), F8 (SHA-pin actions + exact GoReleaser version),
F15 (job-level least-privilege perms + verify repo default), F7 (post-upload asset assertion).

## Requirements

- Functional:
  - Trigger: `on.push.tags: ['v*']` only. **Verified fact:** `ci.yml:3-7` is
    `push.branches:["**"]` + `pull_request` with NO `tags:` key → a tag push does NOT
    trigger `ci.yml`. Therefore the tagged commit has zero CI coverage unless `release.yml`
    provides it. (Do not restate "ci.yml unaffected" as reassurance — state the consequence.)
  - Job `test` (least privilege `permissions: { contents: read }`): checkout, setup-go 1.26,
    `go build ./... && go vet ./... && test -z "$(gofmt -l .)" && go test ./... -race -count=1`.
  - Job `goreleaser` `needs: [test]`, `permissions: { contents: write }`: checkout
    `fetch-depth: 0` + `git fetch --force --tags`, setup-go, run pinned GoReleaser with
    `--release-notes` pointing at the extracted `CHANGELOG.md [1.0.0]` section (Phase 4/5),
    `GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`.
  - Post-publish assertion step: `gh release view ${{ github.ref_name }} --json assets`
    → fail loudly if asset count != expected (6 archives + checksums.txt) so a partial
    upload does not silently leave a broken public release.
  - `concurrency: { group: release-${{ github.ref }}, cancel-in-progress: false }` so a
    re-run / accidental double tag never races two uploads to the same release.
  - **No workflow-level `permissions:`** — declared per job (least privilege). Every future
    job MUST declare its own `permissions`.
  - All third-party actions **SHA-pinned** with trailing version comment:
    `actions/checkout@<sha> # v4`, `actions/setup-go@<sha> # v5`,
    `goreleaser/goreleaser-action@<sha> # v6`. GoReleaser binary pinned to the **exact
    version** from Phase 2 (`version: 'v2.x.y'`, not `~> v2`).
- Non-functional: only auto `GITHUB_TOKEN`; no PAT (Homebrew deferred). Repo Settings →
  Actions → Workflow permissions confirmed **read-only default** (so any future omission
  fails closed) — verification noted in Phase 5 pre-flight.

## Architecture

```
git push origin v1.0.0  (tag)        ci.yml: NOT triggered (branches-only filter — verified)
   └─► release.yml  (tags: v*, concurrency: release-<ref>)
        job test       [contents: read]  build+vet+fmt+test -race   ← the real gate for tagged ref
        job goreleaser [contents: write] needs: test
              checkout(0)+tags → setup-go → goreleaser(pinned) release --clean
                 --release-notes CHANGELOG[1.0.0]
              → assert asset count == 7  (6 archives + checksums.txt)
```

## Related Code Files

- Create: `.github/workflows/release.yml`
- Verify-unchanged: `.github/workflows/ci.yml` (`git diff --exit-code` must be clean)

## Implementation Steps

1. Resolve SHAs for the three actions at their intended major (`gh api` or repo tag → commit);
   record `@<sha> # vN`.
2. Write `release.yml`:
   - `name: Release`, `on: { push: { tags: ['v*'] } }`,
     `concurrency: { group: release-${{ github.ref }}, cancel-in-progress: false }`
   - `jobs.test` `runs-on: ubuntu-latest` `permissions: { contents: read }`: SHA-pinned
     checkout + setup-go `1.26.x` cache; run build/vet/gofmt/`go test ./... -race -count=1`.
   - `jobs.goreleaser` `needs: [test]` `permissions: { contents: write }`: SHA-pinned
     checkout `fetch-depth: 0`; `git fetch --force --tags`; SHA-pinned setup-go;
     SHA-pinned `goreleaser-action` `with: { version: 'v2.x.y', args: release --clean
     --release-notes=.github/release-notes.md }` (notes file extracted from CHANGELOG in
     Phase 5), `env: { GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }} }`.
   - Final step: `gh release view ${{ github.ref_name }} --json assets --jq '.assets|length'`
     compared to expected; non-match → `exit 1`.
3. Lint: `actionlint` if available; else `python -c 'import yaml,sys;yaml.safe_load(open(sys.argv[1]))'`.
4. `git diff --exit-code .github/workflows/ci.yml` → clean (untouched).
5. Confirm GoReleaser pinned version in `release.yml` == the version validated by Phase 2
   `make snapshot` == the version documented in CONTRIBUTING (Phase 4). All three must match.

## Success Criteria

- [ ] `release.yml` triggers only on `v*`; documents (verified) that `ci.yml` does NOT fire on tags
- [ ] `test` job gates `goreleaser` job via `needs:`; `test` is `contents: read`, publish is `contents: write`
- [ ] No workflow-level `permissions:`; both jobs declare their own
- [ ] All actions SHA-pinned w/ version comment; GoReleaser pinned to exact `v2.x.y` (== Phase 2)
- [ ] `concurrency:` group present, `cancel-in-progress: false`
- [ ] Post-publish asset-count assertion step present (fails on partial upload)
- [ ] `checkout` `fetch-depth: 0` + `fetch --tags`
- [ ] YAML valid; `ci.yml` byte-identical (`git diff --exit-code` clean)

## Risk Assessment

- True publish path only runs on a real tag push — covered by Phase 5's **disposable
  pre-release tag dry-run** (not by snapshot). Residual: repo `GITHUB_TOKEN` policy /
  branch protection blocking release creation → Phase 5 pre-flight verifies repo default
  token = read-only and that `contents: write` job override is permitted.
- SHA pins go stale → acceptable; bump deliberately (note in CONTRIBUTING). Trades freshness
  for supply-chain integrity on the privileged binary pipeline (correct trade for releases).

## Next Steps

Phase 4 creates CHANGELOG.md (release-notes source), adds it to archive `files:`, SECURITY.md,
README case/proxy/unsigned notes, CONTRIBUTING (pinned-version + Homebrew TODO), templates.
