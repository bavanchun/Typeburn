# PM Report — Distribution v1.5 On-ramp — COMPLETE

Date: 2026-05-20 | Plan: `20260519-distribution-v1.5-onramp` | Status: **completed**

## Outcome

**v1.5.0 shipped** — tag `598f7ec` → commit `ac229c0`. All 4 phases
delivered, verified, code-reviewed (0 Critical/High). Zero durable
artifact from the dry-run. Both install channels live + clean-container
verified.

## Phase completion

| Phase | Status | Evidence |
|-------|--------|----------|
| P1 Go floor 1.25 + lockstep | ✅ 6/6 todo, 4/4 success | suite + `goreleaser build` green under installed go1.25.10 (11 pkgs, oracle-matched); 5-file/6-occurrence lockstep; README caveats verbatim |
| P2 install.sh + GoReleaser determinism | ✅ 6/6 todo, 5/5 success | `.goreleaser.yaml` pinned `typeburn`; 14/14 offline harness (RED→GREEN); shellcheck clean 0.10.0+0.11.0+CI; CI `installer` job |
| P3 Homebrew cask | ✅ 8/8 todo, 5/5 success | `homebrew_casks:`+`prerelease:auto`+`skip_upload:auto`; before-hook removed; token step-env-only; G0/G1 user-completed |
| P4 docs sync + safe dry-run | ✅ 6/6 todo, 6/6 success | C1 dry-run all invariants + teardown; v1.5.0 published (7 assets, 1106-char notes, cask `5c42971`); both channels clean-container |

Sync-back: all 4 phase files frontmatter `completed`; 26/26 todo + 20/20
success boxes checked; plan.md `status: completed` + completion section.
0 unmapped tasks.

## Delivery artifacts

- PR #12 (feature, squash `412539a`) + PR #13 (doc sync, squash `ac229c0`)
- Release: https://github.com/bavanchun/Typeburn/releases/tag/v1.5.0
  (prerelease=false, 7 assets, deterministic `typeburn_1.5.0_*` names)
- Tap: `bavanchun/homebrew-tap-typeburn` `Casks/typeburn.rb` (commit `5c42971`)
- New install channels: `curl|sh` installer, `brew install
  bavanchun/tap-typeburn/typeburn`; Go floor 1.26.2→1.25.0

## Verification ledger

- Go race suite 11/11 under go1.25.10 (oracle-matched)
- install.sh: 14/14 offline harness; real-binary E2E (host) + clean
  Alpine (no Go, mkdir -p proven) + real-v1.5.0 one-liner
- `brew install` clean homebrew/brew container → `typeburn v1.5.0`
- Dry-run C1: prerelease-flagged, 7 assets, ZERO tap commit,
  `releases/latest` unaffected, full teardown (tap byte-identical)
- code-reviewer: 0 Critical/High; M-1 + L-1 applied

## Docs impact

Handled — PR #13 synced `docs/system-architecture.md`,
`docs/project-roadmap.md`, `docs/codebase-summary.md`, `CHANGELOG.md`,
`.github/release-notes.md`. No further docs action required.

## Outstanding

- **Finalize chore PR pending**: untracked `plans/20260519-distribution-
  v1.5-onramp/` (plan + reports) to land via a separate `chore/...` PR
  (protected-main; cannot commit to main directly).
- Deferred (out of v1.5 scope, user decision): independent signing of
  `checksums.txt` (cosign/minisign) — tracked in plan Validation Log.

## Unresolved questions

None.
