---
phase: 3
title: "Boxed Update Renderer"
status: pending
priority: P1
effort: "5h"
dependencies: [2]
---

# Phase 3: Boxed Update Renderer

## Overview

New `internal/cli/updateui` package holding the inline Bubble Tea model that
renders the framed progress block, built on `bubbles/progress` and
`bubbles/spinner`. Self-contained and independently testable — this phase does
not touch `cmd_update.go`.

## Requirements

- Functional
  - Rounded-border box, title `typeburn update`, version transition header with
    archive size, four checklist rows (`checksums`, `downloading`, `verifying`,
    `installing`).
  - Rows show a settled `✓`, the live spinner, or a faint pending `·`.
  - The `downloading` row carries a spring-smoothed gradient bar with percent.
  - `Total == 0` renders an indeterminate bar, never a computed ratio.
  - A terminal success frame after `installing` completes.
- Non-functional
  - Row count and box width are constant for the whole run — no reflow.
  - Under `NO_COLOR`/`mono` the frame is layout-identical, attributes only.
  - View functions are pure over model state, so tests need no running program.
  - Every file under 200 LOC.

## Architecture

### Package placement

`internal/cli/updateui/`, following the existing `internal/cli/notui` and
`internal/cli/output` precedent for cli-local subpackages. It does **not** go in
`internal/ui` — that package is documented as the six screen sub-models of the
main TUI, and this is a CLI-command decoration with a different lifecycle.

Layering stays legal: `updateui` may import `bubbletea`, `lipgloss`, `bubbles`,
`internal/theme`, and `internal/update` (for `Progress`/`Stage`). The pure-logic
packages are untouched.

### Two NO_COLOR traps, both verified against `bubbles v2.1.1`

These were found by building a working prototype before this plan, not inferred:

1. **`progress.New()` seeds a default purple blend (`#7571F9`).** Omitting
   `WithColors` is *not* enough — a `NO_COLOR` run still emits SGR. The fill,
   the empty track, and the percentage are three separate knobs:

   ```go
   opts = append(opts,
       progress.WithFillCharacters('█', '░'),
       progress.WithColorFunc(func(_, _ float64) color.Color { return nil }),
   )
   bar := progress.New(opts...)
   bar.EmptyColor = nil
   bar.PercentageStyle = lipgloss.NewStyle()
   ```

2. **`lipgloss.Color` in v2 is a function, not a type.** `[]lipgloss.Color` does
   not compile; `progress.WithColors` takes `...color.Color`, so build
   `[]color.Color` from `theme.Color(Role)`.

### Colors come from `internal/theme` only

No hex literals in `updateui`. The gradient is
`RoleAccentDim → RoleAccent → RoleSuccess`; glyphs use `RoleSuccess` (settled),
`RoleAccent` (active), `RoleTextFaint` (pending); the border uses `RoleBorder`.
The theme is resolved by the caller in Phase 4 and passed in, so `updateui`
never reads env or disk itself.

### Layout, measured from the prototype

Box inner width `50`, bar width `20`. The archive size lives in the header, not
on the download row — with the byte counter inline the row overflowed 50 columns
and wrapped, breaking the frame:

```
╭── typeburn update ─────────────────────────────╮
│                                                │
│  v2.5.1   →   v2.6.0     4.3 MB                │
│                                                │
│  ✓  checksums               64 KB              │
│  ⠋  downloading  ████████░░░░░░░  52%          │
│  ·  verifying                                  │
│  ·  installing                                 │
│                                                │
╰────────────────────────────────────────────────╯
```

Lip Gloss v2 has no first-class bordered-title API, so the title is injected by
rewriting the top border line. The prototype's first attempt was off by one
column; the correct approach is to rebuild the line from the measured original
cell count (`╭` + N dashes + `╮`) rather than slicing at fixed offsets, and to
strip SGR before counting.

### Model shape

```go
type Model struct {
    from, to string
    theme    theme.Theme
    spin     spinner.Model
    bar      progress.Model
    cur      update.Progress
    done     bool
    err      error
}
```

`Init` batches `spin.Tick` and a 40 ms poll tick. The model does **not** own the
update goroutine — Phase 4 injects a snapshot function so this package stays
free of `Apply` and remains testable with hand-built state.

`View()` returns `tea.NewView(s)` with `AltScreen = false`: inline rendering, so
the block sits in normal scrollback exactly where `typeburn update` prints, and
prior output is not wiped.

Percent feeding the bar is `Done/Total` while downloading, held at its last
value afterwards. `Total == 0` switches the row to the spinner-only
indeterminate form.

## Related Code Files

- Create: `internal/cli/updateui/model.go` (model, `Init`, `Update`, `View`)
- Create: `internal/cli/updateui/view.go` (box, header, checklist rows, title injection)
- Create: `internal/cli/updateui/styles.go` (theme→bubbles wiring, NO_COLOR neutralization)
- Create: `internal/cli/updateui/model_test.go`, `internal/cli/updateui/view_test.go`

## Implementation Steps

1. `styles.go`: build the `progress.Model` and `spinner.Model` from a
   `theme.Theme`, applying both NO_COLOR neutralizations above. Detect NO_COLOR
   from the passed theme (`theme.Color(RoleBg) == nil`, the same predicate
   `internal/app/model_view.go:56` already uses) — not by re-reading the env.
2. `view.go`: header line, `stageRow` for each of the four stages, glyph and
   label selection driven by `cur.Stage` vs the row's stage, box render, title
   injection with SGR-safe cell counting.
3. `model.go`: `New(from, to string, th theme.Theme, snapshot func() update.Progress)`,
   the 40 ms poll tick, spinner and `progress.FrameMsg` forwarding, terminal
   success/failure frame.
4. Tests:
   - a frame at each stage matches an expected structure: exactly 10 lines,
     every line the same display width.
   - `NO_COLOR` frame contains no SGR color sequence (only attribute codes are
     permitted) and has identical line count and widths to the colored frame.
   - `Total == 0` renders the indeterminate form and never a `NaN`/`+Inf`
     percentage.
   - the title-injected top border has exactly the same cell width as the
     bottom border — the specific off-by-one the prototype hit.
   - glyph progression: for `cur.Stage == StageVerifying`, rows 1–2 are `✓`,
     row 3 is the spinner, row 4 is `·`.

## Success Criteria

- [ ] `go test ./internal/cli/updateui/ -race -count=1` green
- [ ] Colored and `NO_COLOR` frames have identical line count and per-line width
- [ ] `NO_COLOR` frame emits zero color SGR sequences
- [ ] Top and bottom border widths equal, asserted by test
- [ ] No hex color literal anywhere in the package
- [ ] Every file under 200 LOC; `make lint` green

## Risk Assessment

**Risk:** a future `bubbles` upgrade re-introduces default coloring through a
knob this plan does not clear, silently breaking `NO_COLOR`.
**Mitigation:** the NO_COLOR test asserts absence of color SGR on the rendered
frame — it fails on any new leak regardless of which knob caused it.

**Risk:** wide glyphs (`✓`, braille spinner) measure differently across
terminals and break the box.
**Mitigation:** width assertions use the same `lipgloss` width measurement the
renderer uses; the repo already depends on `uniseg`/`runewidth` transitively for
this. Stick to the `MiniDot` braille set already proven in the prototype.
