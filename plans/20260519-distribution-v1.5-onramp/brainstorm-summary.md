# Brainstorm Summary — Distribution v1.5 (broadly-installable on-ramp)

Date: 2026-05-19
Status: approved (design + 2 gating decisions locked)

## Problem

Typeburn app quality is ship-ready (9/10 repo hygiene, 8/10 in-app UX). The
*on-ramp* is the weak link. Within its real audience (terminal users), install
fails for most:

- `go install` requires Go **1.26.2** (`go.mod`) — almost nobody has it.
- Only fallback: 5-step manual binary download + checksum verify.
- No package managers (`.goreleaser.yaml` = raw archives only; no brews/nfpms).
- Binary-name split: `Typeburn` (go install/release) vs `typeburn` (make).

"Ship to everyone" has an unfixable form-factor ceiling (a TUI inherently
excludes non-terminal users — that's not a defect). Target = "broadly
installable for terminal users", not literal everyone.

## Approaches evaluated

Three workstreams, sequenced by risk × reach.

### WS1 — Lower Go version floor (spike-first; gates messaging)
- Empirical spike, not a guaranteed change. Floor bounded below by
  `charm.land/bubbletea/v2` & `lipgloss/v2` min Go.
- Method: lower `go.mod` (try 1.22) → `make test-race` → check
  `go list -m -f '{{.GoVersion}}' all` → repeat until tests fail / dep forces floor.
- If it drops: **lockstep** update across 5 sites — `go.mod`, `ci.yml`,
  `release.yml` (×2 `go-version`), `CONTRIBUTING.md`, `README`.
- Risk: if bubbletea v2 pins ~1.25/1.26, gain is minimal → installer+brew carry reach.

### WS2 — `install.sh` one-line installer (lowest risk, independent)
- Static `install.sh` in repo root. OS/arch detect → map release archive →
  download latest → **mandatory checksums.txt verify** → extract →
  install to `~/.local/bin` (no sudo; warn if not on PATH).
- README one-liner + documented non-piped path for security-conscious users.
- Zero goreleaser/release.yml change → `assets == 7` assertion untouched.
- Out: Windows (sh-only; Windows keeps manual zip this round).

### WS3 — Homebrew tap (highest macOS reach, highest coordination)
- goreleaser `brews:` block → formula auto-committed to NEW repo
  `bavanchun/homebrew-tap` (user provisions).
- Needs scoped PAT secret `HOMEBREW_TAP_GITHUB_TOKEN` (default GITHUB_TOKEN
  can't push cross-repo). Scope: tap repo only, contents r/w.
- Formula → tap repo, NOT a release asset → `assets == 7` stays valid
  (verify first via `make snapshot`).
- `brew install bavanchun/tap/typeburn` (lowercase command).

## Decisions locked

- **Binary name:** brew + installer + `make` produce lowercase `typeburn`;
  `go install` keeps `Typeburn` (module path is case-sensitive,
  unchangeable without breaking module-path rename — accepted, YAGNI).
- **WS3:** included fully; user provisions tap repo + PAT.
- **install.sh target:** `~/.local/bin`, no sudo, warn if not on PATH.

## Scope boundary (OUT this round)

Scoop/winget, AUR, deb/rpm (nfpms), Nix, snap. Defer until demand.

## Build order

WS1 spike → WS2 installer (ship fast, independent) → WS3 Homebrew
(needs tap repo + PAT provisioned first).

## Acceptance criteria

- WS1: tests green on lowered floor across all 5 lockstep sites, OR documented
  "blocked at 1.X by bubbletea v2".
- WS2: `curl -fsSL …/install.sh | sh` installs runnable `typeburn` on
  macOS+Linux (amd64+arm64), checksum-verified, on PATH.
- WS3: `brew install bavanchun/tap/typeburn` runnable; release pipeline still
  asserts 7 assets and passes.

## Risks

- Lockstep drift in WS1 → broken release if any of 5 sites missed.
- WS3 PAT adds a credential to a deliberately least-privilege pipeline; must
  scope tightly to tap repo.
- Piped installer is a security smell; checksum verification is the mitigation
  and is non-optional.

## Unresolved questions

- WS1 outcome unknown until spike runs (does any code/dep actually need 1.26?).
- Confirm `make snapshot` shows brew formula does not count toward release assets
  (expected: stays 7) before first real tagged release with WS3.

## Security note (process)

A PAT was pasted in plaintext in chat during this session and must be revoked +
regenerated; it was deliberately excluded from this report and all files. Secret
must be added via GitHub Actions secrets / `gh secret set`, never committed.
