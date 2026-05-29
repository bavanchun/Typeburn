# typeburn update: Self-Update Command Implementation

**Date**: 2026-05-30 11:30
**Severity**: Medium (feature delivery, no production incidents)
**Component**: CLI / Update Infrastructure
**Status**: Resolved (shipped PR #33, commit 365bc4a7)

## What Happened

Shipped `typeburn update` — a self-hosted binary updater that replaces the running executable in-place using only stdlib, no third-party self-update libraries. Command validates against published checksums over HTTPS, detects package-manager installs (Homebrew, `go install`), and guards against downgrades. Flags: `--check` (report only), `--yes` (skip prompt). Full test coverage; all CI gates green (gofmt, go vet, go test -race, size check).

## The Brutal Truth

The plan red-team process and post-implementation code review operated on *completely different threat models* — and we needed both. Code review caught two real defects the adversarial planning process missed entirely: one Critical (asset naming bug), one High (HTTP redirect not validated). Tests that hold a constant variable masked the upgrade-path defect perfectly. Plan-phase testing is not a substitute for concrete post-code review against real integration values.

This is humbling. Comprehensive planning + adversarial red-team felt complete, but it wasn't until the actual code was written and reviewed that the wiring-level bugs surfaced. Lesson: review and red-team are orthogonal threat models.

## Technical Details

**Architecture:**
- Pure logic in `internal/update`: `Check()` resolves latest GitHub release (forced fetch), `Download()` streams to temp, `Verify()` checks SHA-256 against published `checksums.txt` over HTTPS, `Replace()` atomically swaps the binary.
- CLI wiring in `internal/cli/cmd_update.go`: parses flags, guards against stdin-prompt when non-interactive, detects managed installs via symlink inspection.
- Strict layering: zero UI dependencies in `internal/update`; all screen logic isolated in CLI layer.
- All Go files <200 LOC; dependency budget strictly honored (stdlib only).

**Trust Model:**
- Checksum-only integrity (binaries unsigned by design, v1 scope). Same trust model as `curl install.sh | sh` — detects corruption, not a compromised host. Not silently reversed during review; this is user-confirmed threat scope.

**Platform Split:**
- Unix: atomic rename over running executable (process retains old inode, new code runs on next exec).
- Windows: move-aside + rollback + crash-recovery guard.

## What We Tried

1. **Adversarial Plan Review**: 15 findings (4 Critical, 9 High, 2 Medium) accepted and folded into phases before any code written.
   - Design-level risks caught: TOCTOU/symlink race on replace, EXDEV isolation (extract temp in target dir, not OS temp), Result.Latest v-prefix ambiguity, non-tty stdin handling, downgrade guard.
   - All integrated into phase design before implementation.

2. **Post-Code Review**: Code-reviewer ran fresh against final implementation.
   - **Critical Defect C1 Found**: `Apply()` built asset name from `current` version instead of `target` tag. Real upgrade 2.2.0→v2.3.0 would request `typeburn_2.2.0_…` under tag `v2.3.0` → 404.
     - Root cause: redundant version parameter that was never meant to vary. Fix: dropped param; asset name now derives from target tag (prevents recurrence by construction). Regression test asserts genuine current≠target swap.
   - **High Defect H1 Found**: redirect allowlist checked host but not scheme. A 302 to cleartext HTTP on github.com host would be followed → checksum anchor undermined.
     - Fix: refuse non-https redirects (loopback exempt for httptest).

3. **Test Coverage Gaps Exposed**:
   - Table-driven tests holding version constant masked the upgrade-path bug perfectly. Variable held, variable not exposed.
   - Regression tests added post-fix verify the actual hazard (current version ≠ target version in real upgrade scenario).

## Root Cause Analysis

**C1 (Asset Naming):** Design review focused on attack surface and concurrency primitives; nobody ran a mental trace of the asset-name derivation against actual GitHub API values. This is a "integration realism" gap — red-team scenarios are strategic (can I race? can I symlink?), not tactical (does the download URL match the tag?).

**H1 (HTTP Redirect):** Host-only validation was "probably fine" in design; code review caught that HTTP on a valid host defeats the checksum trust anchor. Scheme validation is not a security feature in isolation — it's a pre-condition for the checksum check to matter.

**Test Masking:** Constant-version tests meant the variance (2.2.0→v2.3.0) that triggers the bug never ran. This is classic: tests that don't exercise the variable's range hide the variable's bug.

## Lessons Learned

1. **Adversarial planning and post-code review are orthogonal, not redundant.** Plan red-team operates at design/threat level (concurrency, symlinks, trust boundaries). Code review operates at integration level (does the download URL match the live API response?). Both needed; neither alone was sufficient. Plan finds "can you race a file?"; review finds "what do you request from GitHub?".

2. **Tests hiding their own bugs.** A test that holds a variable constant can't expose that variable's range-dependent bug. Regression tests must actively exercise the variance that caused the original defect — not just assert "current behavior is correct" — or they won't catch the next person's off-by-one in the same code.

3. **Managed install detection is not self-evident.** Symlink inspection, executable path whitelisting, and the mapping to upgrade command (Homebrew → `brew upgrade`, go install → `go install -u`) is platform-specific knowledge. Good that code review forced this to be explicit and tested.

4. **Checksum validation is a pre-condition, not sufficient.** If the download URL doesn't match reality, or if redirects can bypass HTTPS, the checksum becomes irrelevant. These aren't secrets — they're the wiring that makes checksums matter.

## Next Steps

- **Merged and shipped.** All CI green; PR squash-merged to main; commit 365bc4a7 now in main branch.
- **Release notes:** document `--yes` and `--check` flags, warn against `curl | sh` pattern upgrades (encourage `typeburn update` instead), note managed-install detection.
- **Maintenance:** monitor upgrade telemetry (success vs. failure rates, platform breakdown) when available. If we see unexpected 404s on real users, check if Result.Latest parsing has regressed.
- **Debt note:** Windows rollback + crash-recovery is non-trivial; consider simplification if we ever drop Windows support, but for now it's worth the complexity to avoid a broken executable on failed upgrade.

## Unresolved Questions

None — both defects fixed and merged. All tests green. CI gate passed. Ready for v2.2.0 release.
