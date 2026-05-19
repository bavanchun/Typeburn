# Brainstorm Summary — Settings Live-Apply Bug + Typing Width

Date: 2026-05-19 16:30 · Status: approved, ready for `/ck:plan --tdd`

## Problem Statement

Two user-reported issues in Typeburn (TUI):

1. **Theme change does nothing.** User opens Settings, changes Theme — screen does not visibly change. (Same gesture also fails for blink cursor, default mode/length, and the in-session persist-error toast.)
2. **Text feels small / lost** on the typing screen on wide terminals.

## Root Cause — Issue #2 (theme), confidence ~95%

`app.New()` (`internal/app/model.go:75`) takes the pointer-receiver method
value `m.onSettingsChange` and passes `&m.settings` into `NewSettings`, then
`return m` **copies the struct**. `tea.NewProgram(app.NewFromDisk(...))`
(`cmd/typeburn/main.go:74`) drives the **copy**; the callback closure + the
`*config.Settings` pointer still target the **orphaned local `m` from
`New()`**.

Effect: a settings change runs `onSettingsChange` against the orphan
(rebuilds its theme/sub-models, persists to disk) — the rendered model is
never touched → **no visible change in-session**, but the new theme **is
persisted**, so a restart shows it. Same defect breaks live blink-cursor,
default-mode/length apply, and the `persistErr` toast (all share the path).

`internal/app/smoke_test.go:189-196` already documents the pointer-aliasing
but rationalizes it ("In production … this works correctly") — that claim is
false; this is a real production defect, not a unit-test artifact.

## Diagnosis — Issue #1 (text size)

This is a terminal app: glyph size = terminal font; the app cannot set it.
Perceived smallness comes from layout: `width_tier.go:42` hard-caps wide
terminals at **80 columns**, and `screen_typing_view.go:66-77` top-anchors
the stream with a large bottom spacer pinning the footer → content sits in a
sea of whitespace on big screens.

## Evaluated Approaches

| Issue | Option | Verdict |
|-------|--------|---------|
| #2 | Message refactor (drop callback+pointer; Settings emits `SettingsChangedMsg`; root applies on live model) | **Chosen** — matches existing Elm pattern (`AbortMsg`/`StartTestMsg`), fixes all 4 live-apply paths at once |
| #2 | Minimal patch (pointer Model program / rewire) | Rejected — keeps off-pattern design, weaker tests |
| #1 | Bounded cap ~100/120 + vertical center | Considered |
| #1 | %-of-width + vertical center | **Chosen by user** (≈80–85% termW, no fixed cap) |
| #1 | Big ASCII "zen" glyph mode | Rejected (out of scope, large UX change) |

**Brutal-honesty note (accepted by user):** edge-leaning full-width typing
text reduces readability; user explicitly chose %-based width over a bounded
cap. Implement with only the existing degraded/narrow floors as sanity
guards (no new hard cap).

## Recommended Solution

### Fix #2 — message refactor
1. Add `SettingsChangedMsg{ Settings config.Settings }` to `internal/ui/messages.go`.
2. `SettingsModel`: drop `s *config.Settings` + `onChange`; hold settings by
   value; on cycle emit a `tea.Cmd` returning `SettingsChangedMsg`.
3. Root `Model.Update`: handle `SettingsChangedMsg` on the live `m` — set
   `m.settings`, persist, `theme.Load`, rebuild `home/typing/result/sett`.
   Remove the pointer-receiver `onSettingsChange` method.
4. Preserve `SettingsModel.sel` across rebuild (fixes the row-resets-to-0 wart).
5. Replace the misleading `smoke_test.go` comment; add a real regression
   test: drive root → change theme → assert `m.theme` changed **and**
   `View()` output differs.
6. Free with the same fix: live blink cursor, default mode/length, in-session
   `persistErr` toast.

### Fix #1 — %-based width + vertical centering
1. `width_tier.go` `ContentWidth` TierWide: `80` → `round(termW * ~0.82)`,
   floored by the existing narrow/degraded guards (no fixed upper cap).
2. `screen_typing_view.go`: vertically center the stream block instead of
   top-anchor + large bottom spacer (footer stays pinned via balanced padding).
3. No glyph-size change (documented expectation).

## Implementation Considerations & Risks

- **#2 blast radius:** `model.go`, `model_settings.go`, `screen_settings.go`,
  `messages.go`, `smoke_test.go`, `screen_settings_test.go`. All internal
  (TUI) — no public API/back-compat concern. Keep files <200 LOC.
- **#1 blast radius:** `width_tier.go`, `screen_typing_view.go` + their tests
  `screen_typing_test.go`, `phase09_polish_test.go` (regression-locked) —
  must be updated intentionally under TDD, never bypassed. No `.golden`
  files exist → lower risk.
- Sequential: Fix #2 then Fix #1 (independent; #2 higher severity first).

## Success Metrics / Validation

- #2: in a running session, changing Theme/Blink/Default-mode visibly
  applies immediately; new regression test asserts `m.theme` + `View()`
  change; `persistErr` toast shows on a forced save failure.
- #1: on an 88+ col terminal the typing stream uses ≈80–85% width and sits
  vertically centered; degraded/narrow tiers unchanged; updated tier/typing
  tests green; full `-race` suite green; no regression to `phase09` intent.

## Next Steps

- `/ck:plan --tdd` with this summary as input (refactors existing behavior +
  regression-locked tests → TDD mandatory).
- Protected-main PR flow per `typeburn-release-runbook` (this is a bug-fix +
  UX change → likely a patch/minor release decided at plan time).

## Unresolved Questions

- Exact width percentage (80 vs 85%) — finalize in plan; default 82%.
- Release version bump (patch vs minor) — decide during planning.
