---
phase: 10
title: "Tests teatest & CI"
status: pending
priority: P2
effort: ~4h
dependencies: [9]
---

# Phase 10: Tests teatest & CI

## Overview

Close coverage gaps with real tests, add teatest Update/render smoke per screen, run race detector on the engine, and ship CI (GitHub Actions ubuntu+macOS), a Makefile, README quickstart + keybind table, and `.editorconfig`. No mocks/fakes/cheats — real tests only.

Refs: researcher-01 §8 (teatest); design §8 (keybind table for README).

## Requirements

### Functional
- Pure unit coverage complete: metrics (all formulas + worked example + AFK), typing engine (states/backspace/completion), words (determinism/buckets), storage (round-trip, corrupt, rotation, XDG fallback, new-best).
- teatest smoke per screen (Home, Typing, Result, Settings, History): drive key input, assert Update transitions / rendered frame contains expected markers. Verify teatest module path (`github.com/charmbracelet/x/exp/teatest` or current) — escalate if unresolvable, do not mock instead.
- Race detector clean on engine/metrics (`-race`).
- GitHub Actions: matrix ubuntu-latest + macos-latest → `go build ./...`, `go vet ./...`, `gofmt -l` (fail if non-empty), `go test ./... -race`.
- Makefile targets: `build`, `run`, `test`, `lint`. README: quickstart + full keybind table (from design §8). `.editorconfig`.

### Non-functional
- Real tests only — no stubs to pass CI. Deterministic (seeded words, fixed timestamps). Files <200 lines.

## Architecture

teatest spins a real `tea.Program`; feed `tea.KeyPressMsg`, capture final/intermediate frame, assert substring (avoid brittle full golden where styling varies — use tolerant content asserts; optional golden behind `-update`).

```text
.github/workflows/ci.yml   matrix os:[ubuntu-latest,macos-latest], go 1.26
  steps: checkout → setup-go → build → vet → gofmt-check → test -race
Makefile: build|run|test|lint
README.md: what/install/quickstart + §8 keybind table
.editorconfig: go tabs, lf, trim trailing ws
```

## Related Code Files

Create:
- `.github/workflows/ci.yml`
- `Makefile`
- `README.md`
- `.editorconfig`
- `internal/ui/screen-home_test.go`, `screen-typing_test.go` (extend), `screen-result_test.go` (extend), `screen-settings_test.go`, `screen-history_test.go` (teatest)
- coverage-gap tests in existing `*_test.go` as needed

Modify:
- `go.mod`/`go.sum` (add verified teatest dep)
- existing test files to fill gaps

Delete: none.

## Implementation Steps

1. Verify teatest module path via `go get` (try `github.com/charmbracelet/x/exp/teatest`; if moved, resolve current path; escalate if unresolvable — never substitute mocks). Pin in go.mod.
2. Per-screen teatest: build screen model, send scripted keys (e.g. Home: tab→tab→enter starts; Typing: type word triggers states; Settings: ↓ then → cycles+persists to temp dir; History: g/G scroll), assert frame content / model state. Use temp XDG dirs (`t.Setenv`) for storage screens.
3. Coverage audit (`go test ./... -cover`): fill gaps in metrics/typing/words/storage to high coverage of pure logic (no arbitrary % gate, but every formula/branch + corrupt/rotation/fallback covered).
4. `go test ./internal/typing/... ./internal/metrics/... -race` clean.
5. `ci.yml`: matrix ubuntu+macos, Go 1.26, steps build/vet/gofmt-check/test -race.
6. `Makefile` (build/run/test/lint — lint = `go vet` + `gofmt -l` check), `.editorconfig`, `README.md` (quickstart, install, full keybind table from design §8).
7. Run CI locally-equivalent commands; ensure green; push triggers workflow on both OSes.

## Success Criteria

- [ ] teatest smoke passes for all 5 screens (real program, no mocks).
- [ ] All metric formulas (incl. worked example), engine states, words determinism, storage corrupt/rotation/XDG/new-best covered by real tests.
- [ ] `go test ./... -race` green locally and in CI.
- [ ] CI runs on ubuntu + macOS: build, vet, gofmt-check, test all pass.
- [ ] Makefile (build/run/test/lint), README (quickstart + §8 keybind table), `.editorconfig` present.
- [ ] teatest module path verified & pinned (or escalated, not mocked).

## Risk Assessment

| Risk | L×I | Mitigation |
|---|---|---|
| teatest module path moved/unstable | M×M | Verify via `go get`; escalate if unresolvable; tolerant content asserts not strict golden |
| Golden frames brittle across OS/term | M×M | Assert substrings/markers, not full styled golden; optional `-update` golden |
| Timing-dependent test flake (ticks) | M×M | Inject fixed timestamps into engine; avoid wall-clock in assertions |
| CI macOS runner slower/flaky | L×L | Same steps both OS; no OS-specific paths (XDG fallback tested) |

## Unresolved Questions

None — scope fully locked. (teatest exact import path is a verify-at-implementation step, not an open scope question.)
