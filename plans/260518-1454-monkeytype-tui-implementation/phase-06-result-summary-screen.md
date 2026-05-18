---
phase: 6
title: "Result summary screen"
status: pending
priority: P1
effort: ~5h
dependencies: [2, 4]
---

# Phase 6: Result summary screen

## Overview

Render the post-test result: big ASCII WPM digits, raw/accuracy/consistency, char stats, duration+mode line, and an ASCII WPM-over-time sparkline from per-second samples — all derived from the Phase 2 keystroke log. Replaces the placeholder Result route.

Refs: researcher-02 §7,8 (post-hoc derivation); design §5.3 (stat card), §5.6 (one panel), §8.4 (keys); mockups §3.

## Requirements

### Functional
- Consume `ResultMsg` (Phase 4) → `metrics.Result` already computed; render only.
- Hero: ASCII big-digit WPM (NetWPM rounded) `accent Bold`; `wpm` label `text-muted`.
- Secondary: `acc%` (`success` if ≥97, `warning` if low, else `text-primary`), `raw N wpm`, `consistency%`.
- Sparkline: per-second raw-WPM → `▁▂▃▄▅▆▇█` bars in `accent`; y-axis ticks + baseline `text-faint`.
- Char stats line: `correct / incorrect / extra / missed` (labels `text-muted`, values bold; incorrect `error` if >0).
- Meta line: `{duration}s · {mode} {length} · english`.
- One rounded `border` panel (`surface` bg), title `result` on top border.
- Keys (§8.4): `tab`/`enter` restart same; `ctrl+r` new; `esc`/`1` Home; `3` History.
- New-best badge ` ★ new best` (`success`) — wiring deferred to Phase 8 (leave a `IsBest bool` field, default false).

### Non-functional
- Pure rendering from `Result` (no recompute beyond formatting). Files <200 lines. Resize-safe.

## Architecture

Data flow: `app` receives `ResultMsg` → constructs `ResultModel{res, mode, length}` → routes ScreenResult. `tab/enter` → new TypingModel same params; `ctrl+r` → new (re-pick); `esc/1` → Home; `3` → History.

```go
// internal/ui
type ResultModel struct {
    res    metrics.Result
    mode   typing.Mode
    length int
    isBest bool      // set by Phase 8
    w, h   int
    th     theme.Theme
    km     config.Keymap
}
func NewResult(msg TypingResultMsg, th theme.Theme, km config.Keymap) ResultModel
func (m ResultModel) Update(msg tea.Msg) (ResultModel, tea.Cmd)
func (m ResultModel) View() string

// reusable
func BigDigits(n int, th theme.Theme) string         // ascii-big-digits.go
func Sparkline(vals []float64, width, height int, th theme.Theme) string // sparkline.go
func StatCard(label, value string, role theme.Role, th theme.Theme) string // stat-card.go
```

`Sparkline` and `StatCard` are reusable (History reuses Sparkline in Phase 8).

## Related Code Files

Create:
- `internal/ui/screen-result.go`
- `internal/ui/ascii-big-digits.go`
- `internal/ui/sparkline.go`
- `internal/ui/stat-card.go`
- `internal/ui/screen-result_test.go` (Update smoke)

Modify:
- `internal/app/model.go`, `internal/app/routing.go` (real ResultModel; ResultMsg → ScreenResult; key routing)

Delete: none.

## Implementation Steps

1. `ascii-big-digits.go`: 0–9 (+ optional 3-row glyphs) ASCII map; `BigDigits(n)` joins per-digit columns.
2. `sparkline.go`: scale `[]float64` → `▁..█`; render y-axis tick labels + baseline `text-faint`, bars `accent`, width-aware.
3. `stat-card.go`: §5.3 styling helper.
4. `screen-result.go`: `View` builds panel (rounded `border`, title `result` on top edge): hero big-digit WPM + acc/raw/consistency block, `wpm over time` sparkline, char-stats line, meta line. Accuracy color thresholds per mockup §3. `★ new best` appended if `isBest`. `Update` per §8.4.
5. Wire `app`: replace placeholder; `tab/enter`→same params TypingModel, `ctrl+r`→re-pick, `esc/1`→Home, `3`→History placeholder.
6. Build, run a full Time/Words/Quote test → verify numbers match expected (sanity vs metrics), sparkline shape, key nav. vet/gofmt.

## Success Criteria

- [ ] WPM/acc/raw/consistency match `metrics.Result` (formatting only, no recompute drift).
- [ ] ASCII big-digit WPM renders; accuracy color thresholds correct (≥97 success / low warning).
- [ ] Sparkline reflects per-second raw-WPM shape; axis in faint.
- [ ] Char stats + meta line correct; incorrect colored `error` when >0.
- [ ] One rounded panel, title on border; keys per §8.4 route correctly.
- [ ] `isBest` field exists (default false) ready for Phase 8.
- [ ] Build/vet/gofmt clean; smoke test passes.

## Risk Assessment

| Risk | L×I | Mitigation |
|---|---|---|
| Sparkline scaling distortion (all-equal / single sample) | M×M | Guard min==max → flat mid bar; handle len<2 |
| Big-digit glyphs overflow narrow panel | M×M | Width check; fall back to bold numeric if panel too narrow (Phase 9 polish) |
| Recomputing metrics in View causes drift | L×H | Render only from passed `Result`; no `metrics.Compute` in screen |
