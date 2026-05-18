---
phase: 2
title: "GoReleaser config & Makefile wiring"
status: pending
priority: P1
effort: "2h"
dependencies: [1]
---

# Phase 2: GoReleaser config & Makefile wiring

## Overview

Declarative cross-platform release pipeline via `.goreleaser.yaml`, plus Makefile targets that
inject Phase-1 ldflags for local builds and dry-run the build/archive path. Verification is
`goreleaser check` + `make snapshot` — which proves **builds + archives ONLY**, not the
publish/auth/changelog path (that is honestly out of snapshot scope; covered in Phase 5).

Red-team applied: F1 (no `go mod tidy`/`go test` in before.hooks), F4 (archive `files:`
omits CHANGELOG.md until Phase 4), F8 (pin exact GoReleaser version), F9 (plural `formats:`;
no dead `brews:` block), F10 (honest snapshot scope), F11 (curated CHANGELOG = release notes).

## Requirements

- Functional:
  - `.goreleaser.yaml` (GoReleaser v2 schema, `version: 2`) builds linux/darwin/windows ×
    amd64/arm64, `-trimpath`, ldflags
    `-s -w -X .../internal/version.{Version,Commit,Date}={{.Version}}/{{.Commit}}/{{.CommitDate}}`.
  - `before.hooks`: **`go build ./...` only** — NO `go mod tidy` (mutates go.sum, needs
    network), NO `go test ./...` (arbitrary code; tests run in Phase 3 release `test` job,
    not the privileged publish job). Set `GOFLAGS=-mod=readonly` in build env so any module
    drift fails loudly instead of silently mutating.
  - Archives: **plural** `formats: ["tar.gz"]`; `format_overrides: [{goos: windows, formats: ["zip"]}]`
    (singular `format` is deprecated/removed in newer v2). `files:` = `README.md`, `LICENSE`
    **only** (CHANGELOG.md does not exist until Phase 4 — added to this list there).
  - `checksum`: sha256 → `checksums.txt`.
  - `changelog`: **`disable: true`** for this release — repo history has zero `feat:`/`fix:`
    commits, so git-derived notes would be empty/misleading. Release notes come from curated
    `CHANGELOG.md` (Phase 4) via Phase 5 `--release-notes`.
  - `release`: github, `draft: false`.
  - **No `brews:` block at all** (deprecated/removed schema; commenting dead schema adds
    negative value). Homebrew provisioning instructions live as prose TODO in `CONTRIBUTING.md`
    (Phase 4).
  - Makefile: `VERSION`/`COMMIT`/`DATE` from git; `LDFLAGS`; `build` uses LDFLAGS;
    targets `version`, `snapshot`, `release`.
- Non-functional: GoReleaser pinned to an **exact version** (e.g. `v2.x.y`); that exact
  version recorded in `CONTRIBUTING.md` (Phase 4) and used identically by CI (Phase 3) so
  local dry-run == CI byte-for-byte.

## Architecture

```
git describe ─► Makefile VERSION,COMMIT,DATE ─► LDFLAGS ─► go build (-X internal/version.*)
.goreleaser.yaml builds.ldflags ─► same -X targets ─► dist/ archives + checksums.txt
make snapshot  ─► dist/ (builds+archives ONLY — no release/auth/changelog) 
make release   ─► full publish (CI only, Phase 3)
```

ldflags target `github.com/bavanchun/Typeburn/internal/version` (capital T, matches `go.mod:1`).

## Related Code Files

- Create: `.goreleaser.yaml`
- Modify: `Makefile` (VERSION/LDFLAGS + version/snapshot/release; update `build`)

## Implementation Steps

1. Install the **exact** GoReleaser version to pin (record the version string):
   `go install github.com/goreleaser/goreleaser/v2@v2.x.y`. Note it for Phase 3/4.
2. Write `.goreleaser.yaml`: `version: 2`; `before: { hooks: ["go build ./..."] }`;
   `builds` (env `CGO_ENABLED=0`, `GOFLAGS=-mod=readonly`; goos/goarch matrix; ldflags as above);
   `archives` (`formats: ["tar.gz"]`, `format_overrides` windows `formats: ["zip"]`,
   `files: ["README.md", "LICENSE"]`); `checksum`; `changelog: { disable: true }`;
   `release: { github: {owner: bavanchun, name: Typeburn}, draft: false }`. No `brews:`.
3. `goreleaser check` → valid. Also `goreleaser check 2>&1 | grep -i deprecat` → expect empty
   (fail the step if any deprecation string appears — exit-code-only is insufficient).
4. Edit `Makefile`: add
   `VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)`,
   `COMMIT := $(shell git rev-parse --short HEAD)`, `DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)`,
   `LDFLAGS := -s -w -X github.com/bavanchun/Typeburn/internal/version.Version=$(VERSION) -X ...Commit=$(COMMIT) -X ...Date=$(DATE)`;
   `build`: `go build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY) .`;
   add `.PHONY` + `version` (build then `@$(BIN_DIR)/$(BINARY) --version`), `snapshot`
   (`goreleaser release --snapshot --clean`), `release` (`goreleaser release --clean`).
5. `make build && ./bin/typeburn --version` → injected git-describe version (not `dev`).
6. `make snapshot` → `dist/` archives + `checksums.txt`; run a snapshot binary `--version`
   → real version, NOT `(devel)`. (This proves build+archive+ldflags ONLY — publish path
   is Phase 5's disposable-tag dry-run.)
7. `gofmt`/`vet`/`go test ./...` GREEN (before.hooks is build-only, won't run tests).

## Success Criteria

- [ ] `goreleaser check` passes AND emits no deprecation warnings (grep gate)
- [ ] `.goreleaser.yaml` uses plural `formats:`; archive `files:` = README+LICENSE only; no `brews:`
- [ ] `before.hooks` is `go build ./...` only (no tidy, no test); `GOFLAGS=-mod=readonly` set
- [ ] `changelog.disable: true` (notes deferred to CHANGELOG.md)
- [ ] GoReleaser pinned to exact version; version string recorded for Phase 3/4
- [ ] `make build` → `./bin/typeburn --version` shows git-describe value
- [ ] `make snapshot` → dist/ archives + checksums.txt; snapshot binary `--version` != `(devel)`
- [ ] tests/vet/fmt GREEN

## Risk Assessment

- GoReleaser v2 minor schema churn (singular→plural, `brews` removal) → mitigated by exact
  version pin + deprecation-grep gate (exit code alone passes deprecated config).
- Snapshot proves only build/archive; publish/auth/changelog UNtested here — explicitly
  deferred to Phase 5 disposable-tag dry-run (no false "exact pipeline" claim).
- Archive referencing CHANGELOG.md before it exists would fail `make snapshot`; avoided by
  README+LICENSE-only `files:` now, CHANGELOG added in Phase 4.

## Next Steps

Phase 3 adds the tag-triggered CI with a self-gating `test` job + SHA-pinned actions.
