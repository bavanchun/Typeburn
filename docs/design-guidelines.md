# Terminal UI Design System — Monkeytype TUI (Go / Charm Stack)

> Target stack: **Go + Bubble Tea v2 + Lip Gloss v2**. Output = ANSI-styled terminal text. No HTML, no DOM, no fonts, no pixels. Every value below is implementation-ready: truecolor hex + ANSI256 fallback + Lip Gloss guidance.
>
> Feel: fast, elegant, minimalist, distraction-free, developer-focused, highly polished. References: Monkeytype, ttyper, lazygit, Neovim.

---

## 1. Design Principles (Terminal-Adapted)

| Principle | Terminal application |
|---|---|
| Distraction-free | No borders/chrome during the typing test. Chrome appears only on non-test screens. |
| Content-first | Centered single column, generous negative space, one focal element per screen. |
| Dim-the-done | Typed-correct text dims; current char brightens (Monkeytype's core legibility trick). |
| Never color-alone | Wrong chars get color **and** an underline. Survives NO_COLOR / colorblind / mono terminals. |
| Honor the env | Respect `NO_COLOR`, `$TERM`, `COLORTERM`. Degrade gracefully, never crash. |
| Speed of feel | No blocking animation. "Motion" = at most a 1-frame cursor or instant restart. |

---

## 2. Color Palette

### 2.1 Theme: `default` (Dark) — source of truth

Slate-dark base + green accent ("code dark + run green"), red error, amber warning. Tuned for dark terminals; values verified against a near-black bg (`#0E1117`).

| Role | Purpose | Truecolor Hex | ANSI256 | NO_COLOR / mono fallback |
|---|---|---|---|---|
| `bg` | App background (usually = terminal default; do **not** force unless themed) | `#0E1117` | `233` | terminal default |
| `surface` | Cards, settings rows, table zebra, result panel | `#161B22` | `235` | terminal default |
| `surface-alt` | Hover/selected row base, sparkline gutter | `#21262D` | `237` | reverse video |
| `text-primary` | Primary text, untyped-but-focused, headers body | `#E6EDF3` | `255` | default fg |
| `text-muted` | Correct typed chars, secondary labels | `#8B949E` | `245` | default fg |
| `text-faint` | Untyped upcoming text, hints, disabled | `#484F58` | `240` | faint attr |
| `accent` | Primary brand/action: logo, WPM number, selected, progress | `#22C55E` | `42` | bold |
| `accent-dim` | Accent at rest (unfocused tab, idle progress) | `#15803D` | `28` | normal |
| `correct` | Correctly typed char (alias of `text-muted` — dimmed, not green) | `#8B949E` | `245` | default fg |
| `error` | Incorrect char fg | `#F85149` | `203` | underline + default fg |
| `error-bg` | Incorrect-space marker bg | `#5C1A1A` | `52` | underline only |
| `warning` | Caution states, "stop-on-error" lock, low accuracy | `#E3B341` | `179` | default fg |
| `success` | Result positive deltas, new-best badge | `#3FB950` | `41` | bold |
| `cursor-bg` | Block-cursor background | `#22C55E` | `42` | reverse video |
| `cursor-fg` | Char rendered under the cursor | `#0E1117` | `233` | reverse video |
| `border` | Borders, separators, table rules | `#30363D` | `238` | ASCII `-` `|` |
| `border-focus` | Border of focused/active panel | `#22C55E` | `42` | bold border chars |

**Contrast intent** (terminal-adapted; no true WCAG since no fixed luminance, target ≈ readable on `#0E1117`):

- `text-primary` on `bg` ≈ 13:1 — primary reading, well above 7:1.
- `text-muted`(correct) on `bg` ≈ 5.0:1 — comfortably readable; intentionally dimmer so the **current** char pops by relative contrast.
- `text-faint` on `bg` ≈ 2.4:1 — *intentionally low*. This is upcoming/not-yet-relevant text; it must recede. Acceptable because it carries no must-read info at that moment and brightens to `text-primary`/`accent` as the cursor reaches it.
- `error` on `bg` ≈ 5.6:1 + mandatory underline — never relies on hue alone.
- `accent` on `bg` ≈ 8.2:1 — large/bold numerics, exceeds 4.5:1.
- `cursor-fg` on `cursor-bg` ≈ 8.2:1 — the char under the block stays legible.

> Rule: any text the user must read *right now* ≥ 4.5:1. Deliberately receded text (`text-faint`) may go lower **only** when it is not yet actionable and will brighten before it matters.

### 2.2 Theme packs (shipped v1.1.0)

`mono` — single-hue greyscale; accent = bold white, error = underline-only. For strict/mono terminals & purists.

Six community palettes mapped onto the 16 roles (one `internal/theme/<name>-theme.go` file each). Role order below: Bg · Surface · SurfaceAlt · TextPrimary · TextMuted · TextFaint · Accent · AccentDim · Error · ErrorBg · Warning · Success · CursorBg · CursorFg · Border · BorderFocus.

| Theme | Bg | Surface | SurfaceAlt | TextPrimary | TextMuted | TextFaint | Accent | AccentDim | Error | ErrorBg | Warning | Success | CursorBg | CursorFg | Border | BorderFocus |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `solarized-dark` | `#002b36` | `#073642` | `#094d5c` | `#93a1a1` | `#839496` | `#586e75` | `#268bd2` | `#1f6a9e` | `#dc322f` | `#4a1715` | `#b58900` | `#859900` | `#268bd2` | `#002b36` | `#586e75` | `#268bd2` |
| `solarized-light` | `#fdf6e3` | `#eee8d5` | `#ded8c5` | `#586e75` | `#657b83` | `#93a1a1` | `#268bd2` | `#3a7ca5` | `#dc322f` | `#f5d0cd` | `#b58900` | `#859900` | `#268bd2` | `#fdf6e3` | `#93a1a1` | `#268bd2` |
| `dracula` | `#282a36` | `#343746` | `#44475a` | `#f8f8f2` | `#c9c9d1` | `#6272a4` | `#bd93f9` | `#7d5bbe` | `#ff5555` | `#5c1a1a` | `#f1fa8c` | `#50fa7b` | `#bd93f9` | `#282a36` | `#6272a4` | `#bd93f9` |
| `nord` | `#2e3440` | `#3b4252` | `#434c5e` | `#eceff4` | `#d8dee9` | `#4c566a` | `#88c0d0` | `#5e81ac` | `#bf616a` | `#4a2326` | `#ebcb8b` | `#a3be8c` | `#88c0d0` | `#2e3440` | `#4c566a` | `#88c0d0` |
| `gruvbox-dark` | `#282828` | `#3c3836` | `#504945` | `#ebdbb2` | `#a89984` | `#665c54` | `#fabd2f` | `#b57614` | `#fb4934` | `#4a1f1c` | `#fe8019` | `#b8bb26` | `#fabd2f` | `#282828` | `#665c54` | `#fabd2f` |
| `gruvbox-light` | `#fbf1c7` | `#ebdbb2` | `#d5c4a1` | `#3c3836` | `#7c6f64` | `#bdae93` | `#b57614` | `#79740e` | `#9d0006` | `#f2d0cd` | `#af3a03` | `#79740e` | `#b57614` | `#fbf1c7` | `#bdae93` | `#b57614` |

Sources: Solarized (ethanschoonover.com/solarized), Dracula (draculatheme.com), Nord (nordtheme.com), Gruvbox (github.com/morhetz/gruvbox). `solarized-light` / `gruvbox-light` are the first light themes — their Bg/Surface/Text roles are intentionally inverted vs the dark default; `palette_luminance_test.go` guards against accidental inversion, and per-screen contrast was reviewed manually.

Theme contract: a theme = a `map[Role]lipgloss.Color`. Screens reference **roles only**, never literals. Adding a theme = one map. `theme.Available()` is the single display-order source; `config.Settings.Normalize` duplicates the accepted-name set (core layering forbids importing `theme`) and `theme_available_sync_test.go` keeps them in lockstep.

---

## 3. Typography & Text Treatment

Terminal = one monospace font, fixed cell. "Typography" here = **weight, attribute, color, casing, spacing**.

| Need | Treatment |
|---|---|
| App logo / screen title | ASCII art or `Bold` + `accent`. UPPERCASE for section titles. |
| Key numbers (WPM, ACC%) | `Bold` + `accent` (primary) or `success` (positive delta). Largest visual via ASCII big-digits where space allows. |
| Body / labels | Normal weight, `text-primary` or `text-muted`. |
| Hints, footer keybinds | `Faint` + `text-faint`; the **key glyph** itself `text-muted` (not faint) so it's scannable. |
| Correct typed chars | `text-muted` (dim grey) — NOT green. Mirrors Monkeytype: completed work recedes. |
| Current char | `cursor-bg`/`cursor-fg` block — the single brightest cell on screen. |
| Untyped upcoming | `text-faint`. |
| Errors | `error` fg + `Underline`. |
| Emphasis in prose | `Italic` sparingly (e.g., mode name in result line). Avoid in word-stream. |

**Dim-vs-color rule (core):** correct chars are *dimmed*, not *colored*. Green is reserved for the accent system (logo, WPM, selection), so the word-stream stays calm and the eye tracks the bright cursor. Color enters the stream only for **errors** (red).

Spacing/rhythm: one blank line between logical groups; two blank lines around the focal element (WPM number, word-stream block). Never crowd — whitespace is the primary "luxury" signal in a TUI.

---

## 4. Layout System

### 4.1 Content column

- **Centered** horizontally and vertically (Lip Gloss `Place(termW, termH, Center, Center, content)`).
- Responsive content width:
  - `termW ≥ 88` (wide) → `clamp(round(termW × 0.82), 80, termW-8)` — scales
    with the screen, never narrower than 80, capped to keep side breathing
    room so the centered block stays comfortable on very wide terminals.
  - `72 ≤ termW < 88` → content width `termW - 8`.
  - `60 ≤ termW < 72` → content width `termW - 4`, footer keybinds collapse to short forms (`tab restart` → `↹`).
  - `termW < 60` or `termH < 20` → **degraded mode** (see 4.3).
- Vertical rhythm: the typing screen emits a **compact** block (header → blank
  → focal block → blank → footer) and the root `Place(Center, Center)` centers
  it; the footer is not pinned to the last terminal row.

### 4.2 Header / footer placement

- **Header:** top row(s). Test screen: ultra-minimal (`WPM · timer/progress`), left-aligned, `text-muted`, no border. Other screens: title centered.
- **Footer:** always the last terminal row, full-width, `text-faint`, key-hint style (§5.4). Single line. Pinned via spacer flex.
- The test screen shows **only** header + word-stream + footer. No panels, no borders.

### 4.3 Safe minimum & degraded behavior

- **Safe minimum: 60 × 20** (cols × rows). Designed-for: 80 × 24+.
- `termW < 60` OR `termH < 20`: render a single centered notice:
  ```
  Terminal too small
  Need at least 60×20 (current 54×18)
  Resize to continue · ctrl+c quit
  ```
  in `warning`. Re-render live on resize (`tea.WindowSizeMsg`). Never crash, never partial-paint.
- Word-stream needs ≥ 3 visible lines; if `termH` between 20 and 24, shrink blank-line padding first, never the stream.

---

## 5. Component Specs

### 5.1 Block cursor

- Default: solid block over the **current** char. `Background(cursor-bg)` + `Foreground(cursor-fg)`. The char stays readable inside it.
- Blink: **off by default** (Monkeytype-style steady block). Optional `blink` setting → 530ms toggle between `cursor-bg` and `surface-alt` bg. Document as opt-in; steady is the polished default.
- "Smooth cursor": **not truly feasible** in a cell-grid TUI — there are no sub-cell positions, so a literal slide is impossible. Provide a *perceptual* approximation only: the setting, when on, draws a faint trailing underline on the just-left cell for the next render (1-frame ghost) to imply motion. Tradeoff: extra redraw + can look noisy on slow/SSH terminals. **Default OFF.** Be honest in UI copy: "Smooth cursor (trail effect — limited in terminals)".

### 5.2 Character states (typing stream) — authoritative table

| State | Fg | Bg | Attr | Notes |
|---|---|---|---|---|
| `untyped` | `text-faint` `#484F58`/240 | none | — | upcoming text, recedes |
| `correct` | `text-muted` `#8B949E`/245 | none | — | dimmed, **not** green |
| `incorrect` | `error` `#F85149`/203 | none | `Underline` | color **and** underline |
| `incorrect-space` | `error` | `error-bg` `#5C1A1A`/52 | `Underline` | wrong char where a space was expected — bg block makes the missed-space visible |
| `extra` | `error` | none | `Underline` `Faint` | chars typed past the word (allow-continue mode) |
| `current` | `cursor-fg` `#0E1117`/233 | `cursor-bg` `#22C55E`/42 | — | the one bright cell |
| `current-error` | `cursor-fg` | `error` `#F85149`/203 | — | cursor sits on a position already wrong (stop-on-error mode) |

### 5.3 Stat card (result / settings)

- Container: `surface` bg, `border` rounded border (`lipgloss.RoundedBorder()`), padding `1 2`.
- Label: `text-muted`, UPPERCASE, small.
- Value: `Bold` `text-primary`; hero values (`WPM`) `accent` + ASCII big-digits.
- Layout: cards in a centered row, `MarginRight(2)` between; wrap to next row if `termW` too narrow.

### 5.4 Key-hint footer

- Format: `key action · key action · key action`.
- Key glyph: `text-muted` (scannable). Action word: `text-faint`. Separator ` · `: `text-faint`.
- Keys lowercased, special keys spelled: `tab`, `enter`, `esc`, `ctrl+c`, `↑↓`.
- Narrow terminals: drop action words, keep glyphs only.

### 5.5 Selected list row (settings / history / mode selector)

| State | Fg | Bg | Attr | Marker |
|---|---|---|---|---|
| normal | `text-muted` | none | — | two leading spaces |
| selected | `text-primary` | `surface-alt` | `Bold` | `accent` `▎` left bar + leading space |
| selected value | `accent` | `surface-alt` | `Bold` | — |
| disabled / stub | `text-faint` | none | `Faint` | `(stub)` suffix |

Selection moves with `↑/↓` or `j/k`. The `▎` bar in `accent` is the focus signal (works even where bg colors are weak).

### 5.6 Border usage rules

- **Test screen: zero borders.** Distraction-free is non-negotiable.
- Home: no border around logo; mode selector may use a subtle `border` rounded box only if it aids grouping (prefer none — whitespace instead).
- Result: ONE rounded `border` panel allowed for the stat block + graph. Title sits on the top border.
- Settings/History: light single-line separators (`border` color) between header and rows; outer border optional, default none.
- Focused panel: switch its border color to `border-focus`. Use `RoundedBorder` everywhere borders appear (consistency). Thick borders reserved for nothing in v1.

---

## 6. Interaction States & Motion

| Element | Resting | Focus / Selected | Active |
|---|---|---|---|
| Mode tab (Time/Words/Quote) | `text-muted` | `accent` + `Bold` + `▎` | selected mode persists `accent` |
| Mode option (15/30/…) | `text-faint` | `accent` `Bold` | chosen option `accent`, others `text-muted` |
| Settings row | `text-muted` | `surface-alt` bg + `▎` accent | value cycles inline on `←/→`/`enter` |
| Cursor | block caret | — | animates per motion policy below |

**Motion policy:** the TUI uses subtle, non-blocking motion for polish without changing layout.
- **Restart:** instant clear + repaint. No flash by default.
- **Counter (WPM in header):** updates on the existing timer path, no tween (tweening live numbers reads as laggy).
- **Caret:** always-on blink plus brief new-cell fade/trail; under `NO_COLOR` this is attribute-only.
- **Result reveal:** WPM count-up, sparkline draw-in, and staggered stat cards; settled frame equals the static render.
- **Screen transitions:** Typing→Result gets a short transition. Color themes use a dim-curtain crossfade; `NO_COLOR`/mono uses a row wipe. Other navigation stays instant.
- **Code paste screen (`ScreenCodePaste`):** instruction + single status line (waiting, or the validation reason on a failed paste). Role-only styling; the line structure is identical in every state and under `NO_COLOR`. Pasted text is validated by the same `codetext.Normalize` core as `--text` (no rule divergence), so the rejection reasons match the CLI path.
- **Reduced motion:** no user-facing setting yet; motion always auto-adapts to `NO_COLOR`/mono by preserving line count and rune width.

---

## 7. Accessibility (Terminal)

- **Never color-alone:** every error state carries `Underline` (and bg block for missed space). A mono/`NO_COLOR` user still sees underlined wrong chars and a reverse-video cursor.
- **`NO_COLOR`:** if `os.Getenv("NO_COLOR") != ""` → drop all color; map to attributes only: cursor = reverse video, error = underline, accent/selected = bold, faint stays faint. Layout unchanged.
- **`$TERM` / `COLORTERM`:** truecolor only when `COLORTERM` ∈ {`truecolor`,`24bit`}; else ANSI256; else 16-color/mono. Lip Gloss profile auto-detects — trust `lipgloss.HasDarkBackground()` / renderer profile; provide explicit ANSI256 numbers (col 2) for the 256 path.
- **Contrast:** must-read text ≥ 4.5:1 on `bg` (see §2.1). Don't force `bg`; inherit the user's terminal bg so their own contrast tuning applies.
- **No reliance on blink** for meaning (blink off by default; some terminals strip it).
- **Resize-safe:** always re-place on `WindowSizeMsg`; degraded notice instead of garbled paint.
- Keyboard-only by definition (TUI). Every action has a documented key (§8). No mouse dependency.

---

## 8. Keybinding Map (Source of Truth)

Consistent scheme across the app. Lowercase; `ctrl+c` always quits anywhere.

### 8.1 Global (active on every screen)

| Key | Action |
|---|---|
| `ctrl+c` | Quit immediately (hard) |
| `esc` | Back / cancel (test→home, settings→home, history→home; on home = quit prompt) |
| `ctrl+r` | Restart current test / reset to fresh test |
| `1` | Go to Home |
| `2` | Go to Settings |
| `3` | Go to History |

### 8.2 Home / Welcome

| Key | Action |
|---|---|
| `tab` / `shift+tab` | Cycle mode (Time → Words → Quote) |
| `←` `→` / `h` `l` | Change mode option (15/30/60/120, etc.) |
| `enter` / `space` | Start test |
| `2` | Settings · `3` History |

### 8.3 Typing test

| Key | Action |
|---|---|
| (any printable) | Type that char |
| `backspace` | Delete last char (allowed within current word; or whole-word per error mode) |
| `tab` | Restart same test (Monkeytype convention) |
| `ctrl+r` | New test (re-pick words) |
| `esc` | Abort → Home |

> `tab` = restart, `ctrl+r` = fresh: documented convention, mirrors Monkeytype muscle memory (`tab`+`enter` = quick restart).

### 8.4 Result summary

| Key | Action |
|---|---|
| `tab` / `enter` | Restart (same mode/length) |
| `ctrl+r` | New test |
| `esc` / `1` | Back to Home |
| `3` | View History |

### 8.5 Settings

| Key | Action |
|---|---|
| `↑` `↓` / `j` `k` | Move selection |
| `←` `→` / `h` `l` / `enter` | Cycle / toggle value of selected row |
| `esc` / `1` | Save & back to Home (settings auto-persist) |

### 8.6 History

| Key | Action |
|---|---|
| `↑` `↓` / `j` `k` | Scroll rows |
| `g` / `G` | Jump top / bottom |
| `enter` | (v1: no-op; reserved for detail view) |
| `esc` / `1` | Back to Home |

---

## 9. Implementation Notes (Lip Gloss v2)

- One `Theme` struct: `map[Role]lipgloss.Color`. Build styles from roles at startup, rebuild on theme change.
- Build `lipgloss.Color` from hex strings; let the renderer downsample to ANSI256/16. Keep the ANSI256 table (§2.1) as a sanity reference and for manual mono mapping.
- `lipgloss.NewStyle().Foreground(theme[RoleErr]).Underline(true)` — compose, don't hardcode.
- Word-stream: build per-char styled strings, `lipgloss.JoinHorizontal`, hard-wrap at content width, `JoinVertical`. Cursor = the styled "current" cell inline.
- Center everything with one `lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, body)`.
- Footer pinned: `lipgloss.JoinVertical(Left, header, body, spacer, footer)` where `spacer` height = `h - used`.
- Detect `NO_COLOR` early; if set, construct a "no-color theme" (attributes only) and swap the map — screens unchanged.
- Re-layout on every `tea.WindowSizeMsg`; gate render behind the 60×20 check.

---

## 10. Unresolved Questions

1. **Backspace policy vs error mode** — In `stop-on-error`, is backspace still allowed to fix, or is the test locked until the correct char? Assumed: backspace allowed to correct; needs product confirmation.
2. **Quote mode source** — bundled static quote list vs fetched? Affects whether History stores quote IDs. Assumed bundled for v1.
3. **History persistence path & cap** — file location (`$XDG_DATA_HOME/typeburn/`?) and max retained rows before rotation. Assumed XDG + last 200; confirm.
4. **Consistency metric formula** — Monkeytype uses coefficient-of-variation of raw WPM samples. Confirm exact formula/sampling interval for parity.
5. **Sound stub** — listed as setting; is any audio in v1 scope or pure placeholder? Assumed placeholder (no audio backend).
6. **`esc` on Home** — quit immediately vs confirm prompt? Spec'd as quit-prompt; confirm desired friction.
