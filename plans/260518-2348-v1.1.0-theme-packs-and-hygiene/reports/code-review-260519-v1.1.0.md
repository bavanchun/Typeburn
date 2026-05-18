# Code Review — v1.1.0 theme packs + hygiene

Branch: `feat/v1.1.0-theme-packs-hygiene` vs `origin/main`
Scope: 3 impl commits (f4b93b1, 56a388f, 7b5ed44); planning commit 0d63728 ignored.
Reviewer: code-reviewer (staff prod-readiness pass)
Date: 2026-05-19

## Summary

Tight, well-scoped change. Build clean, `go vet` clean, `gofmt` clean, full
`go test ./...` green (app/config/metrics/storage/theme/typing/ui all ok). No
exported-signature breakage, no business-logic regression in the touched
hot paths, no security/concurrency concerns (single-threaded Bubble Tea model,
value-receiver state, no new I/O surfaces, no external input). All six
acceptance criteria (a)-(f) verified PASS. Zero critical issues.

## Acceptance Criteria Verification

### (a) Theme correctness — PASS
- 16 roles defined in `roles.go:12-27` (RoleBg..RoleBorderFocus, sentinel
  `roleCount`). All 6 new theme files map exactly those 16 keys — verified by
  inspection of dracula/nord/solarized-{dark,light}/gruvbox-{dark,light}-theme.go.
- `Load` unknown → `default` via `switch ... default: return Default()`
  (`theme.go:59-61`).
- NO_COLOR short-circuits before the name switch: `theme.go:41-43`
  `if noColor { return noColorTheme() }` precedes `switch name`. Confirmed
  Load(name,true).Name()=="no-color" asserted in `theme_test.go`
  TestPacks_NameRoundTripAndAllRolesResolve.
- `TestPacks_NameRoundTripAndAllRolesResolve` + `palette_luminance_test.go`
  give exhaustive non-nil-per-role + non-inverted-palette coverage.

### (b) Core layering / DRY sync — PASS
- `config/settings.go` does NOT import `internal/theme`; the accepted set is
  intentionally duplicated in `Normalize()` (`settings.go:51-58`) with an
  explicit comment stating the layering rule.
- `theme_available_sync_test.go` is in external pkg `config_test`, imports
  both `config` and `theme`, iterates `theme.Available()` and asserts each
  survives `Normalize()`. This genuinely catches divergence: adding a theme to
  `Available()` but not the switch makes `Normalize()` reset it → test fails
  with a precise message. Reverse drift (extra name in switch not in
  Available) is not caught, but that direction is harmless (a non-selectable
  name a user can't reach via UI) — acceptable, noted Low below.

### (c) Persistence notice — PASS
- Non-blocking: `persistErr` is a plain string field set at the failing
  persist site; nothing waits on it. Cleared unconditionally on the next
  keypress (`model_key_handler.go:24`) *before* any routing, and the keystroke
  still proceeds (value receiver — cleared copy propagates). Verified by
  `TestPersistNotice_HistoryFailureShownThenDismissed` (key both clears notice
  AND routes Home).
- NO_COLOR-safe: `PersistenceNotice` uses `RoleWarning` + `RoleTextFaint`
  styles, which under NO_COLOR resolve to attribute-only (underline/bold/faint
  via `attrOnlyStyle`) — no raw color, no layout shift. The "⚠ " glyph is a
  single cell, present in both modes (consistent, layout-neutral).
- model_view.go single-return refactor is behaviorally equivalent on the
  no-error path when `m.w>0 && m.h>0` (the only reachable state past the
  degraded gate, which guarantees w>=60,h>=20):
  - Home/Result/Settings/History: old `return tea.NewView(body)` (self-placed,
    no Place) == new `out = ...View()` then `return tea.NewView(out)`. Same.
  - Typing/default: old fell through to `lipgloss.Place(...)` == new explicit
    `lipgloss.Place(...)`. Same.
  - With `persistErr==""` the overlay block is skipped → byte-identical.
    `TestPersistNotice_NoFailureNoNotice` + the existing golden/screen tests
    pass, confirming no golden drift.

### (d) No regression / no breaking change — PASS
- `AppendHistory`/`SaveSettings` signatures untouched; callers now consume the
  previously-discarded error instead of `_`. No other consumers affected
  (only call sites are `model_history.go:41`, `model_settings.go:23`).
- `theme.Available()` still `[]string`, `Load` still `(string,bool) Theme` —
  additive only.
- `handleResultMsg`/`onSettingsChange`/`handleKey`/`View` business logic
  unchanged apart from the additive error capture / clear / overlay. ResultMsg
  is dispatched in `Update` before the key switch (`model.go:81-83`) and
  returns immediately, so the notice set there is NOT self-cleared by the same
  event — it survives to the next keypress as intended.

### (e) Patterns / conventions — PASS
- Theme files follow `default-theme.go` structure (single constructor,
  `name`+`colors` map literal) and kebab-case filenames matching repo
  convention (`solarized-dark-theme.go` etc.).
- `persistence-notice.go` mirrors the existing degraded-notice.go style
  (stateless render fn taking theme, caller does placement).
- All new files < 200 LOC (largest theme file 35 LOC; persistence-notice.go
  22 LOC; model_view.go now 72 LOC).
- No new lint/vet/gofmt findings (golangci-lint not installed in env; gofmt
  and `go vet ./...` clean).

### (f) Doc commit behavior-neutral — PASS
- 7b5ed44 added-line filter for `internal/ui/word_stream_renderer.go` shows
  ONLY comment/docstring lines changed — zero non-comment added lines. The
  reworded comment now accurately describes the actual hard per-cell wrap
  (the old comment claimed "word-aware scan-back", which the code never did —
  this is a doc-accuracy fix, code already behaved this way). Other files in
  the commit are .md only.

## Findings

### Critical — none

### High — none

### Medium — none

### Low

1. `internal/config/theme_available_sync_test.go` (sync test, one-directional).
   Catches `Available()`-ahead-of-`Normalize()` drift but not the reverse
   (a name in the `Normalize` switch absent from `Available()`). Reverse drift
   is benign (unreachable via UI cycling) so not blocking; consider a second
   loop asserting the switch accepts only names in `Available()` for full
   lockstep. Severity: low (defensive completeness only).

2. `internal/app/model_view.go:62-68` notice overlay replaces
   `lines[len(lines)-1]`. Safe for all real screens because past the degraded
   gate every screen self-places/Place's to exactly `m.h` rows whose last row
   is blank padding, so no content is clobbered and line count is preserved.
   This invariant is implicit (relies on every screen filling to h). It holds
   today and is asserted indirectly by the dismiss test, but a future screen
   that renders fewer than `h` lines or puts content on the last row would be
   silently overwritten. Consider a defensive comment or a height assertion.
   Severity: low (no current defect; latent coupling).

3. `PersistenceNotice` uses a multi-byte glyph "⚠"; on terminals lacking the
   glyph it may render as a replacement box but stays single-cell-ish and does
   not break layout (last row only). Consistent with existing UI glyph usage
   elsewhere in the repo. Severity: low (cosmetic, pre-existing pattern).

## Positive Observations

- Persistence tests use a real ENOTDIR write failure (dir under a regular
  file) instead of mocking the filesystem — matches the project's no-mock
  testing stance and genuinely exercises the error path.
- Sync test placed in external `config_test` package precisely to respect the
  core layering rule while still validating the duplicated set — correct
  resolution of the import-cycle constraint.
- model_view.go refactor comment explicitly states the byte-identical
  guarantee and the reasoning, aiding future maintainers.
- Settings screen tests rewritten generically (cycle over
  `m.rows[rowTheme].values`) so future theme additions need no test edits —
  good DRY/maintainability move; stale fixtures (`solarized-dark` ->
  `totally-unknown-theme`) correctly updated to keep the "unknown normalizes"
  intent valid now that solarized-dark is a real theme.
- Comment-only doc commit corrects a previously inaccurate wrap description
  (code was right, comment was wrong) — net documentation improvement, zero
  behavior change.

## Verification Performed

- `go build ./...` — clean
- `go vet ./...` — clean
- `gofmt -l internal/` — no output (clean)
- `go test ./...` — all packages ok
- Manual diff trace of model_view.go old vs new for every screen branch on
  the no-error path under the post-degraded-gate invariant (w>=60,h>=20).
- Confirmed Load NO_COLOR short-circuit ordering at theme.go:41 (before switch).
- Confirmed 16/16 role coverage per new theme file vs roles.go enum.
- Confirmed no extra consumers of AppendHistory/SaveSettings beyond the two
  modified call sites.

## Unresolved Questions

None.

---

Status: DONE
Score: 9.5/10
Critical issues: 0
(0.5 deducted only for the latent last-row-overlay coupling and
one-directional sync test — both Low, non-blocking, ship-safe.)
