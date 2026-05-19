---
phase: 1
title: "WS1 Go Floor Confirm + Lockstep"
status: completed
priority: P2
effort: "1-2h"
dependencies: []
---

# Phase 1: WS1 Go Floor Confirm + Lockstep

## Overview

The Go floor is **not an open question** — it is verified bounded at **1.25**
by direct deps. This phase confirms 1.25 builds+tests green under the *actual*
1.25 toolchain (incl. the GoReleaser before-hook path), then propagates 1.25 to
all lockstep sites. Runs in parallel with Phase 2. Marginal reach (1.26.2→1.25);
hygiene, not the headline.

## Requirements
- Functional: `go.mod go 1.25.x`, full `make test-race` + `go vet` + `gofmt`
  green under an installed Go 1.25 toolchain; `goreleaser build --snapshot`
  green under that same toolchain.
- Non-functional: zero app behavior change; all 5 lockstep files consistent;
  existing README caveats preserved verbatim.

## Key Insight (red-team C2, verified evidence)
- `~/go/pkg/mod/charm.land/bubbletea/v2@v2.0.6/go.mod:5` → `go 1.25.0`
- `~/go/pkg/mod/charm.land/lipgloss/v2@v2.0.3/go.mod:5` → `go 1.25.0`
- `go list -m -f '{{.GoVersion}}' all` → max 1.25.0.
Go's module rule: effective floor = max `go` directive in the transitive
graph. **1.25 is the floor. No candidate below 1.25 can build.** Do not
"binary-search from 1.22" — that is wasted effort on impossible values.

## Architecture
Confirmation, not exploration. The trap (red-team H7): editing `go.mod` to
`1.25` while a *newer* toolchain (1.26.x) is locally installed will false-green
— Go is backward compatible and `make snapshot` uses the dev's installed
toolchain. Validation MUST run under an actually-installed Go 1.25 via
`golang.org/dl/go1.25`, exercising the exact `release.yml` path (incl.
GoReleaser `before.hooks`).

## Related Code Files
- Modify: `go.mod` (`go 1.26.2` → `go 1.25.x`)
- Modify: `.github/workflows/ci.yml` (`go-version` → `1.25.x`)
- Modify: `.github/workflows/release.yml` (×2 `go-version: "1.26.x"` → `1.25.x`)
- Modify: `CONTRIBUTING.md` (toolchain reference)
- Modify: `README.md:25` ("Requires **Go 1.26+**" → "Go 1.25+") — **preserve
  verbatim** the case-sensitive module-path warning (`README.md:33-35`) and the
  Go-proxy-lag caveat (`README.md:37-39`); change only the numeral (red-team M2)

## Implementation Steps (TDD: existing suite is the regression oracle)
1. **Baseline:** `make test-race` on current 1.26.2 → record PASS set (oracle).
2. **Install candidate toolchain:** `go install golang.org/dl/go1.25@latest &&
   go1.25 download`. (If a precise patch is needed, use the latest 1.25.x.)
3. **Confirm under 1.25:** set `go.mod` `go 1.25.0`; run with the candidate
   toolchain explicitly: `GOFLAGS=-mod=readonly go1.25 build ./...`,
   `go1.25 test ./... -race -count=1`, `go1.25 vet ./...`,
   `gofmt -l .` → all green (must match step-1 oracle set).
4. **Exercise the release path under 1.25:** run `goreleaser build --snapshot
   --clean` with Go 1.25 active (this is the exact `release.yml` +
   `.goreleaser.yaml before.hooks` path; `make snapshot` under the dev's 1.26
   is NOT sufficient validation — red-team H7).
5. **Lockstep propagation:** apply `1.25.x` to all 5 files / 6 occurrences.
   Workflows use `go-version: "1.25.x"`. README: numeral only; diff-review that
   the two caveats survive untouched.
6. **Audit:** `grep -rn "1\.2[0-9]" go.mod .github CONTRIBUTING.md README.md`
   → single consistent floor everywhere. (P4 re-audits.)

## Todo List
- [x] Baseline `make test-race` PASS recorded on 1.26.2 (11 pkgs oracle)
- [x] Go 1.25 toolchain installed via `golang.org/dl/go1.25.10`
- [x] build+test+vet+gofmt green under go1.25.10 (matches oracle)
- [x] `goreleaser build --snapshot` green under go1.25.10
- [x] 5 files / 6 occurrences set to 1.25; README caveats verbatim-preserved
- [x] grep audit single consistent floor

## Success Criteria
- [x] Full suite green under an installed Go 1.25 toolchain (go1.25.10)
- [x] `goreleaser build --snapshot` green under Go 1.25
- [x] All 5 lockstep files consistent at 1.25
- [x] README case-sensitivity + proxy-lag caveats unchanged (verified P4)

## Risk Assessment
- **False-green via newer local toolchain** (H7) → mandatory `go1.25`
  explicit-binary validation incl. `goreleaser build --snapshot`.
- **Lockstep drift** → grep audit in step 6 + P4 re-audit; a missed site =
  release fails at tag time.
- **Caveat loss in README rewrite** (M2) → numeral-only edit + diff review;
  P4 asserts both caveats still present.
- **Over-investment** → this is confirm-not-spike; cap ~1-2h. Reach comes from
  P2/P3, not here.
