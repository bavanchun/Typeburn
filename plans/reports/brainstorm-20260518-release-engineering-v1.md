# Brainstorm Summary — Tag v1.0.0 + Professional Release Engineering

**Date:** 2026-05-18 · **Status:** Approved → handoff to `/ck:plan --tdd`
**Repo:** `github.com/bavanchun/Typeburn` · **HEAD at brainstorm:** `61a4afd` · branch `main`

## Problem Statement

Typeburn v1.0 is feature-complete (10 phases, all tests green, clean tree) but **never actually
released**: no git tags, no version in binary, CI is test-only, roadmap promises GitHub Release
binaries + `go install` that don't exist. Goal: cut a real `v1.0.0` and add professional release
engineering + repo hygiene.

## Scout Findings (codebase context)

- No git tags (`git tag -l` empty). Remote `git@github.com:bavanchun/Typeburn.git`.
- No version var / `--version` flag / ldflags point in `main.go`.
- `.github/workflows/ci.yml` builds+vets+tests only; no release workflow, no GoReleaser.
- `Makefile`: build/run/test/lint/fmt; no release/version targets.
- Present: LICENSE, README, full `docs/`, .editorconfig, .gitignore.
- Missing: CHANGELOG, CONTRIBUTING, issue/PR templates, README badges.

## Locked Requirements

- **Version number:** `v1.0.0` (matches roadmap "v1.0 SHIPPED"; M2 → v1.1.0).
- **Version mechanism:** Hybrid — `internal/version` vars overridable via `-ldflags -X`,
  fallback to `debug.ReadBuildInfo()` when empty. Correct version for BOTH GoReleaser binaries
  (ldflags) AND `go install @v1.0.0` (build info).
- **Scope (all selected):** GoReleaser + release CI · version flag + Makefile/CI wiring ·
  CHANGELOG.md · repo hygiene (badges, install, CONTRIBUTING, issue/PR templates).
- **Distribution:** GitHub Release binaries + `go install` LIVE at v1.0.0; **Homebrew tap
  config written but DISABLED** (commented + TODO) until tap repo + `HOMEBREW_TAP_GITHUB_TOKEN`
  provisioned.
- **Acceptance criteria:**
  - `git tag` shows `v1.0.0`; `go install github.com/bavanchun/Typeburn@v1.0.0` → `typeburn --version` prints `v1.0.0`.
  - Tag push triggers release workflow → GitHub Release w/ linux/darwin/windows × amd64/arm64 archives + `checksums.txt`.
  - GoReleaser binary `--version` prints `v1.0.0` (not `(devel)`).
  - `make build`, `ci.yml`, `go test ./...` stay green; ci.yml unmodified.
  - README renders badges; CHANGELOG has `[1.0.0]` section.
- **Out of scope:** real brew publishing, code signing/notarization, Docker images, deb/rpm/apk,
  SBOM/cosign, M2 code fix, any feature work.
- **Constraints:** Go 1.26, module path fixed, files <200 lines kebab-case, don't break ci.yml/tests,
  conventional commits, no AI refs.

## Evaluated Approaches

| # | Approach | Pros | Cons | Verdict |
|---|----------|------|------|---------|
| A | GoReleaser pipeline (one config, action on tag) | Industry standard; declarative matrix build, archives, checksums, notes, brew | Adds goreleaser-action dep; config learning curve | ✅ Chosen |
| B | Hand-rolled GHA matrix + `gh release create` | No extra tool | ~150 lines brittle YAML; manual checksums/notes; reinvents GoReleaser | ❌ |
| C | GoReleaser + nfpm/docker/cosign | Maximal distribution | Scope creep, YAGNI for v1.0 TUI | ❌ |

## Recommended Solution (Approach A) — Deliverables

1. `internal/version/version.go` (~40 LOC): `Version/Commit/Date` vars + `Resolve()` build-info fallback.
2. `main.go`: stdlib `flag` `--version`/`-v` → print version block, exit before `tea.NewProgram`.
3. `.goreleaser.yaml`: linux/darwin/windows × amd64/arm64; `-trimpath -ldflags "-s -w -X .../internal/version.*"`;
   tar.gz/zip archives; sha256 checksums; git changelog; GitHub release; `brews:` block commented + TODO.
4. `.github/workflows/release.yml` (new; `ci.yml` untouched): trigger `push tags v*`, `fetch-depth:0`,
   setup-go 1.26, `goreleaser/goreleaser-action@v6`, `GITHUB_TOKEN`.
5. `Makefile`: `VERSION := git describe --tags --always --dirty`; ldflags in `build`; `version`, `snapshot`, `release` targets.
6. `CHANGELOG.md`: Keep a Changelog; `[1.0.0] - 2026-05-18` from roadmap+git; `[Unreleased]` = M2 backlog.
7. Repo hygiene: README badges (CI/release/Go/license/pkg.go.dev) + Install section; `CONTRIBUTING.md`;
   `.github/ISSUE_TEMPLATE/{bug,feature}.yml` (YAML forms) + `PULL_REQUEST_TEMPLATE.md`; roadmap note actual tag.

## Implementation Considerations & Risks

- **Ordering (critical):** tag is the LAST step. All files committed+pushed first — GoReleaser builds
  the tagged commit; tagging a commit missing config = broken public release.
- **Mitigation:** `goreleaser check` + `make snapshot` (dry-run, no publish) locally before pushing real tag.
- **Verified mechanism:** `go install @v1.0.0` stamps `debug.BuildInfo.Main.Version` = `v1.0.0`,
  so hybrid fallback returns correct version without ldflags.
- **Brew tension reconciled:** "all now" + "keep disabled" → pipeline built, brew block inert until provisioned.

## Success Metrics

- GitHub Release page exists for `v1.0.0` with 6+ archives + checksums.txt.
- `typeburn --version` correct across: GoReleaser binary, `go install @v1.0.0`, `make build`.
- CI green; tests green; no regression to existing workflow.

## Next Steps

1. `/ck:plan --tdd` using this report as source (TDD focus: `internal/version` Resolve() + flag behavior;
   config/YAML deliverables verified via `goreleaser check`/snapshot, not unit tests).
2. Implement → snapshot dry-run → push files → push `v1.0.0` tag → verify Release.
3. `/ck:journal`.

## Unresolved Questions

- pkg.go.dev badge valid only after tag pushed to proxy — acceptable (resolves on release).
- Windows/arm64 build: included by default; drop if it complicates archives (low risk).
- None blocking.
