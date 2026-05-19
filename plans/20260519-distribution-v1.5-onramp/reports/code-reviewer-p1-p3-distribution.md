# Code Review — Distribution v1.5 On-ramp (P1–P3)

Branch: `feat/distribution-v1.5-onramp` · Reviewer pass: adversarial release-eng
Scope: go.mod, ci.yml, release.yml, .goreleaser.yaml, install.sh,
scripts/test-install-sh.sh, README.md, CONTRIBUTING.md. (plans/ and
internal/*.go excluded per instruction.)

## Verification performed (empirical, not assumed)

- `shellcheck install.sh scripts/test-install-sh.sh` → CLEAN (0 warnings).
- `./scripts/test-install-sh.sh` → 14/14 PASS (matches claimed harness state).
- GoReleaser `{{ .Version }}` strips leading `v` (docs-confirmed). install.sh
  builds `ver="${tag#v}"` → archive `typeburn_<ver>_<os>_<arch>.tar.gz`.
  name_template `typeburn_{{ .Version }}_{{ .Os }}_{{ .Arch }}` is **consistent**
  with what install.sh resolves. Harness case (a) proves the os/arch table.
- `homebrew_casks` schema (v2.15): `binaries:` (plural), `directory:`,
  `repository.token:`, `skip_upload: auto` — all field names in
  `.goreleaser.yaml` are schema-correct. Cask commits to tap repo, NOT a
  release asset → asset count stays 7. Confirmed against GoReleaser v2 docs.
- go.mod `go 1.25.0`; ci.yml `1.25.x`; release.yml `1.25.x` ×2;
  CONTRIBUTING.md `Go 1.25+` / `go 1.25.0`; README `Go 1.25+`. Lockstep holds.
- README go-install case-sensitivity caveat (capital `T`) and ~1h proxy-lag
  caveat both present verbatim (README.md:57-63). M2 satisfied.

## Critical

None.

## High

None blocking. All AC items 1–5 verified satisfied. The token blast-radius
design (HIGH-risk area by mandate) is correctly contained — see Positive.

## Medium

**M-1 — `*..*` path-traversal glob is over-broad and structurally redundant.**
`install.sh:127-131`. The pre-extract guard rejects any tar entry matching
`/*|*..*`. `*..*` matches legitimate names containing a literal `..`
(`README..md`, `v1..2`). It does not cause a false *install* (extraction at
:134 only ever extracts the literal member `typeburn`, so traversal entries
are never written regardless), but a benign archive carrying an incidental
`..` filename would abort a valid install. Severity is Medium not High because
GoReleaser-produced archives never contain such names and the guard is
defense-in-depth over an already-safe single-member extraction.
Fix (optional, hardening-quality): anchor to true traversal only —
`case "$entry" in /*|../*|*/../*|*/..) echo BADPATH; break ;; esac`. Leave the
extraction-by-explicit-member as the primary control.

## Low

**L-1 — `VERSION=` override skips the GitHub API `"prerelease": true` JSON
check.** `install.sh:86-100`. When `VERSION=` is set the code jumps straight to
the string-pattern `is_prerelease` guard; the API boolean check (`:92`) only
runs on the resolved-latest path. A tag flagged prerelease on GitHub but whose
name lacks `-rc/-test/-alpha/...` (e.g. a hand-pinned `VERSION=v1.5.0` that the
maintainer marked prerelease) would not be refused. This is acceptable per the
phase-02 spec (VERSION= is explicit operator intent; harness case (f) proves
the string guard still rejects `VERSION=v0.0.0-rc.test`). Documenting in the
`VERSION` env comment that the override trusts the operator would close the
expectation gap. No code change required.

**L-2 — `cp` then `chmod 0755` leaves a sub-second umask-dependent perms window
on the staged temp file.** `install.sh:143-144`. The staged file is a
PID-suffixed dotfile in user-owned `$BIN_DIR`, not yet the final binary, and
`mv -f` is the atomic publish step — exploitability is effectively nil.
Reorder to `cp` → `chmod 0755` → `mv` is already the order used; no action
needed. Noted only for completeness.

**L-3 — Informational: `homebrew_casks` has no explicit `ids:`/platform
filter** while `builds:` produces windows `zip` archives. GoReleaser v2 docs
recommend an `ids:` filter on multi-platform builds; Homebrew casks are
macOS-only by Homebrew design and GoReleaser selects the darwin archives
automatically. The brief's verified-fact (`make snapshot` yields a valid cask
+ exactly 7 assets) empirically confirms GoReleaser handles the windows
archives without erroring here. Adding `ids:` would make archive selection
explicit and future-proof against build-matrix changes, but is NOT required
for v1.5 correctness.

## Acceptance Criteria — verdict

1. install.sh hardening — **PASS.** POSIX `sh`; `set -eu`; per-download
   explicit `|| return 1` + `[ -s ]` size guard (no pipefail reliance);
   `mktemp -d` + `chmod 700` + `trap cleanup EXIT INT TERM HUP`; prerelease
   refusal on both API-resolved (`:92` JSON bool) and string (`:98`
   `is_prerelease`) paths incl. VERSION= override (harness f); sha256 verified
   before any write (`:121`); symlink/non-regular/path rejected (`:131,137,138`);
   atomic `mv -f` same-fs as final step (`:145`); harness (h) proves prior
   binary byte-identical on failure; `mkdir -p "$BIN_DIR"` (`:141`); PATH
   membership + precedence warn (`:150-164`); README trust-boundary is honest
   — explicitly states it does NOT make `curl|sh` safe (README.md:35-49,
   install.sh:6-12).
2. .goreleaser.yaml — **PASS.** name_template + `builds.binary: typeburn`
   consistent with install.sh archive construction and cask `binaries:
   [typeburn]`. `token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"` explicit (NOT default
   GITHUB_TOKEN) at `:108`. `skip_upload: "auto"` (`:115`) + `release.prerelease:
   auto` (`:92`) present. Cask commits to tap, not a release asset → 7 holds
   (release.yml:88 assertion unchanged; docs-confirmed cask ≠ asset).
3. release.yml — **PASS.** `HOMEBREW_TAP_TOKEN` at GoReleaser step `env:` ONLY
   (`:80`), never job-level; `permissions:` blocks unchanged
   (test: `contents: read`; goreleaser: `contents: write`); `before.hooks`
   removed from .goreleaser.yaml so no `go build ./...` runs in the
   token-bearing job (test job at release.yml:41 builds instead).
4. No pipeline-invariant regression — **PASS.** assets==7 assertion intact;
   actions still SHA-pinned; GoReleaser still `v2.15.4` (release.yml:71,
   ci.yml:62, CONTRIBUTING.md:11); `--release-notes=.github/release-notes.md`
   path intact; changelog `exclude: [".*"]` filter intact.
5. No new lint/build error; Go-version lockstep single 1.25 across all 6
   occurrences; README go-install case-sensitivity + proxy-lag caveats
   verbatim — **PASS.**

## Positive observations

- Token blast-radius containment is textbook: step-scoped env, explicit
  tap-only PAT template, `before.hooks` go-build deleted so no project/test
  code executes beside the cross-repo token, job permissions deliberately
  left source-repo-scoped. The .goreleaser.yaml inline comments (`:16-21`,
  `:105-108`) correctly explain *why* (stable invariant), not plan-origin —
  compliant with the no-plan-refs-in-code rule.
- Safe-dry-run is defense-in-depth, not single-point: `release.prerelease:
  auto` (GitHub excludes from /latest) + cask `skip_upload: auto` +
  install.sh dual prerelease guard. Each layer independently blocks the
  disposable `v*` tag from poisoning users.
- Harness asserts BOTH exit status AND filesystem state per case, including
  the byte-identical prior-binary invariant (case h) — this is the right
  property to test for an atomic installer.
- `-X ...Version=v{{ .Version }}` re-adds the stripped `v` so the release
  binary banner matches the `go install` debug.ReadBuildInfo tag — correct,
  non-obvious detail handled.

## Unresolved questions

None blocking. G0/G1 PAT provisioning is the documented out-of-scope human
gate. M-1 and L-3 are hardening suggestions, not merge blockers.

**Status:** DONE_WITH_CONCERNS
**Summary:** All five acceptance criteria verified PASS empirically
(shellcheck clean, 14/14 harness, docs-confirmed GoReleaser semantics, lockstep
intact); zero Critical/High. One Medium (over-broad `..` traversal glob,
structurally redundant) and three Low (VERSION= bypass-by-design, perms
window, missing-but-tolerated cask `ids:` filter) are non-blocking hardening
notes.
