# Contributing to Typeburn

Thanks for your interest in improving Typeburn.

## Prerequisites

- **Go 1.25+** (the module targets `go 1.25.0`).
- **GoReleaser `v2.15.4`** — the **exact pinned version**. Install with:

  ```sh
  go install github.com/goreleaser/goreleaser/v2@v2.15.4
  ```

  This same exact version is used by `.github/workflows/release.yml` and is the
  version `make snapshot` is validated against. Local dry-run and CI must use
  the identical version so results are byte-for-byte comparable. Bump the pin
  deliberately in `.goreleaser.yaml`, `release.yml`, and this file together —
  never float it (e.g. never `~> v2`).

## Development workflow

```sh
make build       # → ./bin/typeburn (ldflags-stamped)
make run         # go run ./cmd/typeburn
make test        # go test ./...
make test-race   # go test ./... -race -count=1   (the CI gate)
make lint        # gofmt -l check + go vet
make size-check  # build + binary size cap
make fmt         # gofmt -w .
make version     # build, then print the resolved version banner
make clean       # remove ./bin/
```

All of `build`, `vet`, `gofmt -l` (empty), and `go test ./... -race -count=1`
must pass before a change is merged — these are exactly what CI enforces.
`make lint` also rejects `os.Exit` inside `internal/cli/notui` so raw-mode
terminal restoration is not bypassed.

## Dependency policy

Allowed dependency families are stdlib, `charm.land/*`,
`github.com/charmbracelet/*`, `github.com/spf13/cobra`, and `golang.org/x/*`.
Anything else needs explicit user approval and a short rationale in the PR.

Pinned CLI deps as of v2.0.0: cobra `v1.10.2`, fang `v1.0.0`,
`golang.org/x/term v0.43.0`. Bump them deliberately, not with `@latest`.

## Keystroke schema versioning

`typeburn replay` accepts `schema_version: 1` logs using
`typing.Keystroke` JSON tags. Adding a `Keystroke` field is non-breaking only
when old logs decode safely via a zero value or `omitempty`. Renaming/removing a
field or changing semantics requires a new schema version and explicit migration
handling in `internal/cli/cmd_replay.go`.

## Branch & PR workflow (protected main)

`main` is protected — **no direct commits or pushes**. Enforced by GitHub
branch protection (PR required, `ci.yml` must pass, linear history, admins
included) and squash-only merges with automatic branch deletion.

1. Branch off `main`: `feat/…`, `fix/…`, or `chore/…`.
2. Commit on the branch; push it; open a PR to `main`.
3. CI green → **squash-merge** the PR.
4. Tags are cut on `main` only after the PR merges (see *Releasing* below).

- **Conventional Commits**: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`,
  `build:`, `ci:`, `chore:` (with optional scope). Use `!` / `BREAKING CHANGE:`
  for breaking changes.
- **No AI / assistant references** in commit messages or PR descriptions.
- Update `CHANGELOG.md` under `[Unreleased]` for any user-visible change.
- One logical change per PR; keep the diff focused.

## Releasing

Releases are fix-forward and PR-based:

1. Land all release changes via a branch → PR → squash-merge into `main`.
2. From the merged `main` commit, run the disposable-tag dry-run (below),
   verify, delete it.
3. `git tag -a vX.Y.Z <merged-SHA> -m "Typeburn vX.Y.Z"` then
   `git push origin vX.Y.Z` (separate push — never `git push --follow-tags`).
4. `release.yml` self-gates (`ci.yml` does not run on tags) and publishes.

Never delete-and-re-tag a version that reached the module proxy/sumdb (it is
append-only — the version becomes permanently uninstallable). Fix forward to
the next patch instead.

## Testing a release

Two distinct levels — know which one you are running:

1. **`make snapshot`** — builds + archives into `dist/` locally. Proves the
   **build/archive/ldflags path only**. It does **not** exercise publishing,
   `GITHUB_TOKEN` auth, release notes, or asset upload.
2. **Disposable-tag publish dry-run** — push a throwaway pre-release tag (e.g.
   `v0.0.0-rc.test`) so `release.yml` runs the full `test` + publish path
   against a real GitHub Release, verify the assets/checksums/notes, then
   delete the release and the tag. This is the only thing that proves the
   actual publish path. Never delete-and-re-tag a *real* version that reached
   the module proxy — fix forward to the next patch instead.

## Release notes

`CHANGELOG.md` is the single source of truth for release notes. The release
pipeline does **not** auto-generate notes from git history (the history has no
conventional `feat:`/`fix:` commits, so it would be empty). The `[1.0.0]`-style
section is extracted to `.github/release-notes.md` and passed to GoReleaser via
`--release-notes`.

## Homebrew (enabled — cask via the tap)

Users install with:

```sh
brew install bavanchun/tap-typeburn/typeburn
```

This is a **cask** (not a formula): it wraps the prebuilt release archive, so
no user Go/Xcode toolchain is needed. GoReleaser's `homebrew_casks:` block
(`.goreleaser.yaml`) commits the cask `.rb` into the tap repo
`bavanchun/homebrew-tap-typeburn` under `Casks/`. The cask is **not** a GitHub
release asset — the release-asset count stays 7 (6 archives + `checksums.txt`).

Wiring:

1. Tap repo `bavanchun/homebrew-tap-typeburn` (provisioned).
2. A **fine-grained** PAT scoped to **only** that tap repo with
   `Contents: read and write`, shortest viable expiry. **Never** a classic or
   org-wide PAT. Stored as the repo secret `HOMEBREW_TAP_TOKEN`
   (`gh secret set HOMEBREW_TAP_TOKEN -R bavanchun/Typeburn`).
3. `.goreleaser.yaml` `homebrew_casks:` references
   `token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"` (the default `GITHUB_TOKEN` cannot
   push cross-repo). `release.yml` injects `HOMEBREW_TAP_TOKEN` at the
   GoReleaser **step `env:` only** — never job-level; job `permissions:`
   stay scoped to the source repo.
4. `release.prerelease: auto` + cask `skip_upload: "auto"`: a prerelease/`-rc`
   tag (e.g. the disposable dry-run `v0.0.0-rc.test`) is flagged prerelease
   and commits **nothing** to the tap.

### Tap rollback runbook (a bad cask breaks `brew upgrade` for everyone)

A broken `Casks/typeburn.rb` in the tap repo affects every user's next
`brew install`/`brew upgrade`. The CI PAT is intentionally too narrow to
self-heal interactively, so rollback is a **manual human action**:

1. With a HUMAN GitHub credential (NOT the CI PAT), in
   `bavanchun/homebrew-tap-typeburn`:
   `git revert <bad commit touching Casks/typeburn.rb>` and push.
   This restores the last-good cask immediately for all users.
2. Fix the root cause in this repo, then ship a **fix-forward** patch release
   (a new tag) — never delete-and-re-tag a shipped version (it has reached
   the module proxy / Homebrew users).
3. The next stable release's GoReleaser run overwrites the cask cleanly.

(There is deliberately no commented-out alternate schema in
`.goreleaser.yaml` — dead schema rots across GoReleaser versions; the live
block plus this prose is the spec.)
