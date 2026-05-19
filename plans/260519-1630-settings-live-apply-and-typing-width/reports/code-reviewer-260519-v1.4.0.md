---
title: Code Review — v1.4.0 settings live-apply + typing width
date: 2026-05-19
branch: fix/v1.4.0-settings-live-apply-and-typing-width
commits: [25f655d, 5c0fa61]
reviewer: code-reviewer
verdict: APPROVE
---

# Code Review — v1.4.0 (2 commits)

## Scope
- Src: model.go, model_settings.go, messages.go, screen_settings.go, screen_typing_view.go, screen_typing_actions.go, width_tier.go
- Tests: smoke_test.go, screen_settings_test.go, phase09_polish_test.go, screen_typing_test.go, persistence_notice_test.go
- ~822 src+test LOC changed (excl. plan docs). Build OK, `go vet` clean, full `./internal/...` suite green.

## Verdict: APPROVE — no blocking issues. Zero high-confidence defects found.

## Acceptance Criteria — all met

(a) Live apply — VERIFIED
- `changedCmd()` captures `s := m.s` (value copy of `config.Settings`), emits `SettingsChangedMsg`. No pointer escape.
- root `Update` intercepts `ui.SettingsChangedMsg` before sub-model dispatch → `applySettings(sc.Settings)` (value receiver, returns Model). Returned copy is what Bubble Tea drives. Orphan-copy root cause eliminated.
- `applySettings` rebuilds theme + re-injects home/typing/result/sett, preserves row via `Sel()`/`WithSel()`. Returns `m` at the single exit; only caller (model.go:130) uses the return. Correct.
- Live apply proven by real tests: `TestSmoke_Settings_ThemeAppliesLive`/`BlinkAppliesLive` drain the cmd through `Update` and assert on root `theme.Name()` + `View()` delta — not persistence-only. False "works in production" comment removed, replaced w/ accurate note pointing at the drain-based test.

(b) Width contract — VERIFIED by independent recompute (88,89,97,98,99,100,120,160,200,400):
- `clamp(round(termW*0.82), 80, termW-8)` monotonic non-decreasing on [88,400], never < 80 (so never narrower than old hard-80).
- Floor-80 holds across the entire wide band up to termW=98 (raw 80.36 → cap termW-8=90 → 80); first value >80 at termW=99→81. Floor-at-wide-boundary (termW=88→80, cap also 80) exact. Test table `{88,80}{98,80}{100,82}{120,98}{160,131}{200,164}` matches recompute exactly; added monotonic+floor invariant loop is a strong contract lock.
- Vertical centering: full-height spacer removed from `screen_typing_view.go`; block now compact; root `model_view.go:52` wraps ScreenTyping in `lipgloss.Place(Center,Center)` — centering now effective. `TestTypingView_CompactNotFullHeight` (h=50 → <50 lines) guards regression.

(c) No unintended exported-contract change
- `NewSettings` signature change (4→3 args, value not ptr) + new `Sel()`/`WithSel()` + `SettingsChangedMsg`: all intended per plan. `ApplySettings`/`ContentWidth` signatures unchanged (value-only behavior change for TierWide, documented).
- grep confirms ZERO remaining `onSettingsChange`, `onChange`, or `NewSettings(&` 4-arg sites. All 5 `NewSettings(` callers updated to 3-arg value form.

(d) Message-pattern conformance — `SettingsChangedMsg{Settings config.Settings}` mirrors `CodePastedMsg{Text string}`; emitted via `func() tea.Msg{...}` cmd like `AbortMsg`/`StartTestMsg`. Width-tier conventions (Narrow/Mid/Degraded/`WidthTier`) untouched — confirmed by diff (only `TierWide` case body changed).

(e) Build/lint/LOC — `go build ./...` exit 0, `go vet ./internal/...` clean. All changed src files < 200 LOC (max model.go=196). No new files.

(f) Plan-artifact references — CLEAN in this branch. Scoped grep of added `.go` diff lines for `phase-N`/`F\d`/`§\d`/`Y\d`/`red-team`/`brainstorm`/`plan.md` returns only the `+++ b/...` diff header (not code). Pre-existing `§`/`Phase N` refs elsewhere point to `docs/design-guidelines.md` / mockups (stable external design docs) and predate this branch — out of scope, allowed. New comments encode the invariant (orphan-copy hazard, centering rationale, floor/cap reasoning) without origin labels. Compliant w/ hard rule.

## Regression Surface — clear
- `ContentWidth` has exactly ONE non-test caller (`screen_typing_view.go:22`); no other path affected. Narrow/Mid/Degraded unchanged.
- typing engine/metrics/history/code-paste/--text/NO_COLOR: no touch. `ApplySettings` body unchanged (comment-only). NO_COLOR settings test migrated to new ctor, still asserts `▎` marker — passes.
- `persistence_notice_test.go` change is the same ctor migration (1 line).

## Sanity Checks (requested)
- `math.Round(float64(termW)*0.82)` clamp: correct & monotonic — recomputed, see (b).
- `WithSel` clamp: safe — `if sel>=0 && sel<len(m.rows)` guards both bounds; out-of-range keeps default `sel=0`. `rows` always len 4 (buildRows fixed). No panic path.
- `applySettings` value receiver returns model at its sole `return m`; sole caller assigns the return (`return m.applySettings(...)`). No dropped-copy.

## Notes (non-blocking, no action required)
- INFO: `buildRows(&m.s)` in `NewSettings` aliases the local field by ptr — safe here because `m.s` is a value field of the returned struct copy and `applyRow` is the only writer (via `*SettingsModel`); each `cycleSelected` operates on the per-Update value copy, and root rebuilds `m.sett` fresh on every apply, so no shared-slice aliasing across renders. Behavior correct; called out only for future maintainers.
- INFO: `sm_drain` test helper caps at 8 cmd iterations — adequate for current single-message chain; fine.

## Unresolved Questions
None.
