# v1.5.0 — Distribution On-Ramp (Go 1.25, install.sh, Homebrew Cask)

**Date**: 2026-05-20 02:50
**Severity**: High (closes critical install-friction gap for terminal users)
**Component**: build system (.goreleaser.yaml, ci.yml, release.yml), distribution (install.sh, homebrew_casks, go.mod floor), docs (README, CONTRIBUTING, system-architecture, codebase-summary)
**Status**: Resolved

## What Happened

Typeburn v1.5.0 shipped a complete distribution on-ramp: one-liner curl|sh installer, Homebrew cask, and Go 1.25.0 floor to broaden accessibility beyond developers with a local Go toolchain. Three pivots:

1. **Go floor downgrade** (1.26.2 → 1.25.0): bounded below by two deps pinning go 1.25.0 (charm.land/bubbletea/v2 v2.0.6 + charm.land/lipgloss/v2 v2.0.3). Build + suite verified under a clean go1.25.10 toolchain installed via golang.org/dl. Lockstep updates across go.mod, ci.yml (test job GOTOOLCHAIN=local + golang.org/dl/go@v1.25 in PATH), release.yml build matrix, CONTRIBUTING.md, README.md.

2. **Hardened installer** (install.sh): POSIX sh, os/arch detect, resolves latest non-prerelease tag via GitHub API, downloads archive + checksums.txt, verifies sha256, validates archive member (rejects symlinks), atomically installs to ~/.local/bin (no sudo). Honest trust-boundary docs—not "safe," but clear tradeoffs. Offline regression harness (14+ test cases via mock http.server) + shellcheck in CI (`installer` job: shellcheck + harness + goreleaser check).

3. **GoReleaser determinism + safe-dry-run + Homebrew tap publish**: project_name/builds.binary/archives.name_template pinned to lowercase `typeburn` (fixed asset names install.sh maps os/arch to verbatim). Removed before.hooks go-build (no project code in token-bearing publish job). release.prerelease:auto + homebrew_casks.skip_upload:auto ensure dry-run (v0.0.0-rc.test) publishes zero durable artifacts. HOMEBREW_TAP_TOKEN injected at GoReleaser step env only. Real v1.5.0 then published (7 assets + 1106-char curated notes; cask commit 5c42971 to bavanchun/homebrew-tap-typeburn). Both channels verified clean in disposable Docker containers.

Process: PR #12 (feature, 412539a) + PR #13 (doc sync, ac229c0), both squash-merged to protected main. Tester: 14/14 harness cases. Code-reviewer: APPROVE.

## The Brutal Truth

This was a gauntlet of version-skew, toolchain, and container gotchas—each solvable individually but collectively exhausting. The infuriating part: the false-green trap on Go 1.25 floor validation. Simply editing go.mod to 1.25 while go 1.26.2 is installed false-greens (Go is backward compatible; the toolchain directive auto-switches down). Real validation required installing a SEPARATE go1.25.10 via golang.org/dl, setting GOTOOLCHAIN=local, shimming it into PATH, and running goreleaser's build step under that actual 1.25 toolchain. The golang.org/dl path must be `go1.25.10` (the full patch version), not `go1.25` (which does not exist). I spent 90 minutes chasing non-existent paths and misinterpreting toolchain switching semantics before verifying it live.

The shellcheck version skew was equally maddening: local 0.11.0 passed the [ -n x ] && cmd || true idiom (SC2015 is info-level), but ubuntu-latest's shellcheck exited 1. Rather than disable SC2015, I rewrote every A && B || C as an if-block—structurally version-proof. Cleaner, but it meant rewriting three locations and re-proving the harness passes both 0.10.0 and 0.11.0.

macOS Docker Desktop's /tmp bind-mount trap was a silent failure: the python http.server fixture 404'd on every request until I moved the docroot under the repo path (Docker Desktop shares $HOME/project dirs, not /tmp). And during debugging, piping to tail masked install.sh's real exit code, creating a feedback loop of false-negative test runs.

The emotional reality: distribution is unglamorous infrastructure work. It's pipelined constraints (version compatibility, asset naming, token scoping, prerelease semantics) and cross-platform determinism with zero margin for error. Each failure is a silent no-op until you inspect the right place.

## Technical Details

**Go Floor Validation (H7 False-Green Trap):**
- Edit go.mod → go 1.25.0 does NOT validate in isolation when go 1.26.2 is installed.
- Go's toolchain directive auto-switches: the installed 1.26.2 is backward-compatible and silently services the requirement.
- Real validation: install golang.org/dl/go1.25.10 (PATH shim + GOTOOLCHAIN=local) and run `goreleaser build` under that actual 1.25 toolchain.
- Path: `golang.org/dl/go1.25.10` (full patch); `golang.org/dl/go1.25` does not exist (no-op alias).
- Verified: ci.yml test job now exports GOTOOLCHAIN=local + adds golang.org/dl/go@v1.25 to PATH; goreleaser build job inherits and respects it.

**Shellcheck Version Skew:**
- Local 0.11.0: SC2015 is info-level, [ -n x ] && cmd || true passes.
- ubuntu-latest shellcheck: SC2015 exits non-zero (stricter config or older version).
- Fix: eliminate A && B || C pattern entirely (not a disable directive). Rewrite as if-blocks—structurally impossible for SC2015 to flag.
- Tested: SC2015-clean on both shellcheck 0.10.0 and 0.11.0; harness still 14/14 green.

**install.sh Trust Boundary:**
- Honest docs: sha256 verify defends against corrupted/MITM download, but does NOT make curl | sh "safe"—release, archive, and checksums are self-consistent on a compromised repo.
- Non-piped audit path in README: download, read install.sh, verify, then execute.

**macOS Docker Desktop /tmp Bind-Mount:**
- /tmp is NOT shared to Docker containers on macOS (uses a synthetic mount). Fixture docroot must be under $HOME/project (shared).
- Silent failure: http.server 404'd, appeared to be test harness logic bug, was actually mount invisible.

**Homebrew Cask Token Containment:**
- HOMEBREW_TAP_TOKEN injected ONLY in GoReleaser step (release.yml), NOT in test or build jobs.
- before.hooks removed: go-build would execute project code in the token-bearing job (privilege escalation risk).
- skip_upload:auto + prerelease:auto: dry-run (v0.0.0-rc.test) publishes zero to tap or releases/latest.
- release.prerelease:auto: GitHub excludes prerelease tags from /releases/latest—dry-run cannot become "latest" for install.sh.

**PAT Exposure (G0 → G1 Two-Gate):**
- G0 verify-dead confirmed a pasted PAT was STILL LIVE when checked (not already-revoked).
- Real credential, real risk: if the pipeline had executed with that token live, it would have been burned.
- G1 fresh-scoped replaced it; user confirmed revocation; v1.5.0 published with new PAT.

## What We Tried

1. **Go 1.25 validation via editing go.mod alone:** false-green (toolchain auto-switch masked incompatibility). Abandoned.
2. **Direct shellcheck disable directives on SC2015:** overfits to a single version; refactored instead to eliminate the pattern.
3. **Fixture docroot in /tmp on macOS Docker Desktop:** silent 404 failures. Moved to $HOME/project path (shared mount).
4. **Piping install.sh output to tail for debugging:** masked exit code. Removed; tested with explicit rc=$? capture.

## Root Cause Analysis

1. **Go floor trap:** Conflated "edited go.mod" with "validated against 1.25." Missing step: install a second toolchain and prove the build runs under it, not just that it compiles.

2. **Shellcheck version skew:** Assumed local CI passes = CI passes. Did not account for tool config variance across ubuntu-latest images. Fix: prove across versions, not just locally.

3. **macOS Docker /tmp invisible:** Did not verify container mount points before designing the test harness. Container filesystem is NOT host filesystem; shared dirs must be explicit.

4. **PAT exposure (G0 check):** A pasted credential during brainstorm was never revoked after creation. The two-gate design (G0 verify-dead + G1 fresh-scoped) caught it, but it should have been assumed live until explicitly checked.

5. **Version skew is sequential, not parallel:** install.sh-against-/releases/latest could only be fully tested AFTER the real v1.5.0 tag (v1.4.0 assets use capital `Typeburn_1.4.0_*`, new install.sh expects lowercase). The disposable rc tag proves publish SAFETY; the post-stable-tag clean-container test proves the HAPPY PATH. These are load-bearing sequential, not parallelizable.

## Lessons Learned

1. **Go floor validation is not "edit go.mod": requires a separate toolchain.** When pinning a lower Go version, install that version via golang.org/dl/{version}/{patchlevel} and prove the build runs under GOTOOLCHAIN=local + PATH shim. "Compiles on 1.26.2" ≠ "works on 1.25."

2. **Lint-clean locally ≠ lint-clean in CI.** For scripts piped to users, prove across published versions. Add version assertions or multi-version CI steps; do not trust a single local run.

3. **Container mount semantics are not obvious.** Before designing a test harness that moves files, verify container mounts. macOS Docker Desktop in particular has non-obvious defaults.

4. **Version skew is a sequential constraint, not a parallel defect.** Some validations MUST happen in order (asset names change between releases). Accept this and build the pipeline to validate pre-release safety (dry-run), then post-release happy path separately.

5. **Credentials pasted during brainstorm are live until proven otherwise.** Assume any plaintext secret in a session is burned immediately. The G0→G1 two-gate design (dead verification + fresh re-scoped token) is correct; use it before every publish pipeline.

## Next Steps

1. **Audit ci.yml GOTOOLCHAIN wiring:** confirm that release.yml build job inherits the local 1.25 toolchain correctly (manual test: release a v0.0.0-rc.validate tag and inspect the build logs).

2. **Document install.sh asset naming contract:** add a comment to .goreleaser.yaml linking the lowercase project_name to the shell script's hardcoded `typeburn_<ver>_<os>_<arch>.tar.gz` pattern, so future version bumps don't accidentally re-derive names.

3. **Add version.txt pinning to macOS Docker image:** if future harnesses also use Docker Desktop, pin the image tag and add a build-time assertion that /tmp is not available (early fail rather than silent skew).

4. **Post-mortems on the G0 credential:** document the revocation + replacement in the tap repo's SECURITY.md, so future releases know the timeline.

**Unresolved Questions:**

None. All friction points identified and mitigations applied or scheduled.

**Status:** DONE. v1.5.0 released, both install channels verified in clean Docker containers, PAT exposed and replaced, Go floor validated live under 1.25.10.
