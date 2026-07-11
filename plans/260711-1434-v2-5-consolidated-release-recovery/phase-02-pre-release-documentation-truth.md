---
phase: 2
title: Pre-Release Documentation Truth
status: completed
priority: P1
effort: 3h
dependencies:
  - 1
---

# Phase 2: Pre-Release Documentation Truth

## Overview

Reconcile evergreen docs and CHANGELOG with runtime and remote truth while the
release is still pending. Expand PR #57 with the verified code/docs scope;
leave `.github/release-notes.md` at the latest published `v2.4.1`.

## Context Links

- Plan: [plan.md](./plan.md)
- Research: [Documentation truth](./research/documentation-and-release-truth.md)
- Runtime authorities: `internal/theme/theme.go`, `internal/config/settings.go`,
  `internal/ui/settings_rows.go`, `internal/cli/cmd_config.go`

## Requirements

- Document 8 selectable themes and describe `mono` as grayscale color.
- Describe `NO_COLOR` separately as attribute-only rendering.
- Document 8 persisted/CLI keys vs 7 TUI Settings rows; `update_check` is CLI-only.
- Correct current architecture, CLI, runner and screen descriptions.
- State public stable `v2.4.1`; combined `v2.5.0` is upcoming/unreleased.
- Move premature Strict `v2.5.0` CHANGELOG section back under `Unreleased`, add
  punctuation/numbers and Code-label/best-marker fixes, and compare from v2.4.1.
- Do not edit release notes or any historical journal.

## Architecture

Use a three-state documentation model:

```text
Integration: stable v2.4.1; v2.5.0 content under Unreleased
Release prep: cut v2.5.0 section + exact release-notes artifact
Published: remote tag/release/workflow evidence proves v2.5.0
```

Evergreen docs derive names/fields from code authorities. Historical v1 sections
remain historical; avoid blind global replacements.

### Dependency Map

```text
Phase 1 verified behavior ─> docs/CHANGELOG fixed behavior
runtime theme/config/UI ─> evergreen docs
remote v2.4.1 truth ─> pre-release roadmap state
Phase 2 complete ─> PR #57 merge gate
```

## File Inventory

| Action | File | Required reconciliation | Authority |
|---|---|---|---|
| Modify | `README.md` | 8 themes; mono vs NO_COLOR | `theme.go`, `mono-theme.go` |
| Modify | `docs/cli-reference.md` | 8 keys, values/defaults/scope | `cmd_config.go`, `settings.go` |
| Modify | `docs/system-architecture.md` | Full Settings schema and surfaces | runtime structs |
| Modify | `docs/codebase-summary.md` | 7 rows/8 keys, themes, runner signature | runtime packages |
| Modify | `docs/project-overview-pdr.md` | Current 6 screens/4 modes/settings/themes | app/UI/theme |
| Modify | `docs/project-roadmap.md` | Combined upcoming v2.5; no phantom v2.6 | remote tags/releases |
| Modify | `CHANGELOG.md` | Restore Unreleased truth | release policy |
| Modify | `internal/ui/settings_rows.go` | Correct stale source comments only | current rows/themes |
| Modify | `internal/ui/screen_settings.go` | Correct selection-range comment only | current row constants |
| Modify | `internal/ui/screen_settings_view.go` | Correct row-count comment only | current rows |
| No change | `.github/release-notes.md` | Must remain v2.4.1 | latest release |
| Exclude | `docs/journals/**` | Historical/immutable; user journal untouched | scope decision |

## Function and Interface Checklist

- [ ] Verify `theme.Available()` contains eight exact names.
- [ ] Verify `config.Settings` and CLI expose eight persisted keys.
- [ ] Verify Settings screen exposes seven rows and omits `update_check`.
- [ ] Verify CLI accepts Code default mode although TUI row cycles three modes.
- [ ] Verify runner docs use current strict/punctuation/numbers signature.
- [ ] Keep release notes unchanged in integration PR.

## Test Scenario Matrix

| Priority | Validation | Expected |
|---|---|---|
| Critical | Roadmap/CHANGELOG release state | No shipped v2.5/v2.6 claim; Unreleased from v2.4.1 |
| Critical | Journal path diff/status | No content/staging change |
| High | `typeburn config list --json` | Exactly 8 keys |
| High | Theme authority vs docs | Eight names; mono grayscale; NO_COLOR attribute-only |
| High | TUI settings docs | Seven named rows; CLI-only update check explicit |
| Medium | Historical v1 prose | Preserved or clearly labelled historical |

## Implementation Steps

1. Capture code authority outputs and remote release truth in command output.
2. Update named evergreen documents; avoid global count-only replacement.
3. Normalize CHANGELOG to one honest `Unreleased` section.
4. Correct source comments without changing behavior.
5. Run docs drift searches excluding `docs/journals/**` and manually classify
   legitimate historical release text.
6. Run focused package tests plus full gates.
7. Stage explicit files only; assert release notes and journal are unstaged.
8. End when code/docs commits are pushed; Phase 3 exclusively owns PR metadata,
   final checks, review and merge readiness.

## Validation Commands

```sh
go test ./internal/config ./internal/cli ./internal/theme ./internal/ui -count=1
go run . config list --json
rg -n 'v2\.6\.0.*(stable|current|shipped)|mono.*attribute-only|4 settings only' \
  README.md CHANGELOG.md docs --glob '!docs/journals/**'
git diff -- .github/release-notes.md
make lint && make test-race && make size-check
```

## Success Criteria

- [ ] Every named drift is reconciled against a code authority.
- [ ] Pre-release docs cannot be mistaken for a published v2.5/v2.6 release.
- [ ] CHANGELOG has Strict, punctuation/numbers, Code label and marker fixes under Unreleased.
- [ ] `.github/release-notes.md` remains the v2.4.1 artifact.
- [ ] Code/docs changes are pushed and ready for Phase 3 PR reconciliation.
- [ ] Journal remains untouched and untracked.

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| Calling mono attribute-only | User-facing false docs | Separate grayscale and NO_COLOR contracts |
| Blindly replacing historical v1 facts | Corrupt release history | Manual section classification |
| Updating release notes early | Latest-release artifact lies | Hard no-change assertion |
| Counts drift again | Repeat debt | Named lists plus authority links |
| Untracked journal content changes invisibly to git diff | User data modified | Baseline SHA-256 + untracked assertion |

## Security Considerations

Documentation must retain the unsigned-release trust boundary. Do not imply
checksums protect against a compromised release host.

## Unresolved Questions

None.
