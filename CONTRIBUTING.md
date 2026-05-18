# Contributing to Typeburn

Thanks for your interest in improving Typeburn.

## Prerequisites

- **Go 1.26+** (the module targets `go 1.26.2`).
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
make run         # go run .
make test        # go test ./...
make test-race   # go test ./... -race -count=1   (the CI gate)
make lint        # gofmt -l check + go vet
make fmt         # gofmt -w .
make version     # build, then print the resolved version banner
make clean       # remove ./bin/
```

All of `build`, `vet`, `gofmt -l` (empty), and `go test ./... -race -count=1`
must pass before a change is merged — these are exactly what CI enforces.

## Commit & PR conventions

- **Conventional Commits**: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`,
  `build:`, `ci:`, `chore:` (with optional scope). Use `!` / `BREAKING CHANGE:`
  for breaking changes.
- **No AI / assistant references** in commit messages or PR descriptions.
- Update `CHANGELOG.md` under `[Unreleased]` for any user-visible change.
- One logical change per PR; keep the diff focused.

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

## Homebrew (planned — not yet enabled)

Homebrew distribution is intentionally deferred. To enable it later:

1. Provision a public tap repository, e.g. `bavanchun/homebrew-tap`.
2. Mint a **fine-grained** Personal Access Token scoped to **only** that tap
   repo with `Contents: write`, with an expiry. **Never** use a classic or
   org-wide PAT. Store it as a repository secret (e.g. `HOMEBREW_TAP_TOKEN`).
3. Add a `homebrew_casks:` block to `.goreleaser.yaml` referencing the tap and
   that secret, and wire the secret into the `goreleaser` job in `release.yml`.

(There is deliberately no commented-out `brews:`/`homebrew_casks:` block in
`.goreleaser.yaml` — dead schema rots across GoReleaser versions; this prose is
the spec instead.)
