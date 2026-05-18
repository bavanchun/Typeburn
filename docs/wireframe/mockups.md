# Terminal Mockups — Monkeytype TUI

> Monospace ASCII / box-drawing mockups, ~80-col content width inside an 80×24 terminal.
> Callouts `← role` map a region to a color role from `docs/design-guidelines.md` §2.
> Box edges shown for clarity; the **test screen has NO borders** at runtime (frame here is the terminal edge only, illustrative).

Legend of inline char-state markers used in the typing screen:
`░` faint=untyped · normal=correct(dim) · `R` = red+underline incorrect · `[X]` = block cursor · `‗` red underline on wrong space.

---

## 1. Home / Welcome

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                                                                                │
│                                                                                │
│        ███╗   ███╗ ██████╗ ███╗   ██╗██╗  ██╗██╗   ██╗                          │
│        ████╗ ████║██╔═══██╗████╗  ██║██║ ██╔╝╚██╗ ██╔╝   ← accent (#22C55E/42) │
│        ██╔████╔██║██║   ██║██╔██╗ ██║█████╔╝  ╚████╔╝                           │
│        ██║╚██╔╝██║██║   ██║██║╚██╗██║██╔═██╗   ╚██╔╝                            │
│        ██║ ╚═╝ ██║╚██████╔╝██║ ╚████║██║  ██╗   ██║                             │
│        ╚═╝     ╚═╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═╝   ╚═╝   t y p e   ← text-muted    │
│                                                                                │
│                                                                                │
│              ┌─ Time ─┐   Words      Quote          ← selected tab: accent+▎   │
│              ▎ Time   │                              unselected: text-muted    │
│                                                                                │
│                15    [30]    60    120               ← [30]=accent bold(chosen)│
│                                                        others = text-muted     │
│                                                                                │
│                                                                                │
│                  press enter to start                ← text-primary, bold      │
│                                                                                │
│                                                                                │
│                                                                                │
│  tab mode · ←→ length · enter start · 2 settings · 3 history · ctrl+c quit      │
└──────────────────────────────────────────────────────────────────────────────┘
   ↑ footer: keys=text-muted, words=text-faint, ' · '=text-faint
```

Notes:
- Logo = ASCII block art in `accent`; the trailing word `type` in `text-muted` (calm).
- Mode selector: active tab wrapped in a light `border` rounded segment + `▎` accent left bar; inactive tabs plain `text-muted`.
- Length options: chosen value `accent` `Bold` with `[ ]` brackets; siblings `text-muted`; whole row centered.
- Everything centered as one block via `lipgloss.Place`.

---

## 2. Typing Test (mid-test)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                                                                                │
│  87 wpm   0:23 / 0:30                          ← WPM=accent bold · timer=muted │
│                                                                                │
│                                                                                │
│   the quick brown fox jumps over the lazy dog while the sun ░░░░░░ ░░░ ░░░░░    │
│   ▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔▔ correct (dim grey, text-muted #8B949E)         │
│                                                                                │
│   slowly behind tall hilR[s] and the cold wind keeps ░░░░░░░░ ░░░ ░░░░░░░░░░    │
│         correct ──┘    │ └─ [s] = block cursor: cursor-bg #22C55E / cursor-fg  │
│                        └─ R = 'l' typed wrong: error #F85149 + red underline   │
│                                                                                │
│   pushing the dry leaves‗across the empty road into ░░░░ ░░░░░░░░░░ ░░░░░░░░    │
│                         └─ wrong char where space expected: error fg +         │
│                            error-bg #5C1A1A block + underline ‗                │
│                                                                                │
│   far away from ░░░░ ░░░░░░░░ ░░░░░░░░░░░░░░░░░░░░ ░░░░░░░░░ ░░░ ░░░░░░░░░░░░    │
│            └─ untyped upcoming = text-faint #484F58 (recedes)                   │
│                                                                                │
│                                                                                │
│                                                                                │
│                                                                                │
│  tab restart · ctrl+r new · esc menu                      ← footer text-faint  │
└──────────────────────────────────────────────────────────────────────────────┘
```

Char-state mix shown above (authoritative table §5.2):
- `the quick brown fox … sun` + `slowly behind tall hil` + `pushing the dry leaves` → **correct**, dim `text-muted`.
- `R` (the wrong `l` in `hil R s`) → **incorrect**: `error` fg + `Underline`.
- `[s]` → **current**: block, `cursor-bg`/`cursor-fg` — single brightest cell.
- `‗` after `leaves` → **incorrect-space**: `error` fg, `error-bg` block, `Underline`.
- All `░` and everything after the cursor → **untyped**, `text-faint`.
- Header ultra-minimal: `87` `accent` `Bold`, `wpm` `text-muted`, timer `text-muted`. No border anywhere. Word-stream hard-wraps at content width (≤80), centered block.

Time mode header = `WPM   elapsed / total`. Words mode header = `WPM   12 / 25` (words done/total). Quote mode = `WPM   42% ▰▰▰▱▱` progress bar in `accent-dim`/`accent`.

---

## 3. Result Summary

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                                                                                │
│         ╭──────────────────────────── result ─────────────────────────────╮   │
│         │                                                                  │   │
│         │   ██╗    ██╗██████╗ ███╗   ███╗                                  │   │
│         │   ██║    ██║██╔══██╗████╗ ████║      94          ← big WPM       │   │
│         │   ██║ █╗ ██║██████╔╝██╔████╔██║     w p m           accent bold  │   │
│         │   ██║███╗██║██╔═══╝ ██║╚██╔╝██║                                  │   │
│         │   ╚███╔███╔╝██║     ██║ ╚═╝ ██║   97% acc          ← acc=success │   │
│         │    ╚══╝╚══╝ ╚═╝     ╚═╝     ╚═╝   raw 108 wpm        if ≥97 else │   │
│         │                                   95% consistency    text-prim  │   │
│         │                                                                  │   │
│         │   wpm over time                                                  │   │
│         │   110┤                              ▁▂▄▆█▇▆        ← graph line  │   │
│         │    90┤              ▂▃▅▆▇▆▅▄▅▆▇▇▆▇▇█████████          accent     │   │
│         │    70┤      ▁▃▅▆▇▇▇█                                  axis=faint │   │
│         │    50┼──┬────┬────┬────┬────┬────┬────┬────┬──                    │   │
│         │       0    4    8   12   16   20   24   28  s                     │   │
│         │                                                                  │   │
│         │   correct 142  incorrect 4   extra 1   missed 0   ← labels muted │   │
│         │   30s · time 30 · english                            values prim │   │
│         ╰──────────────────────────────────────────────────────────────────╯   │
│                                          panel: surface bg, border rounded     │
│                                                                                │
│  tab restart · ctrl+r new · esc menu · 3 history          ← footer text-faint  │
└──────────────────────────────────────────────────────────────────────────────┘
```

Notes:
- Hero `94` in ASCII big-digits, `accent` `Bold`. `wpm` label `text-muted`.
- Accuracy `97% acc` colored `success` when ≥ target, `warning` if low, else `text-primary`. Raw + consistency `text-primary`.
- Sparkline/graph: bars `▁▂▃▄▅▆▇█` in `accent`; y-axis ticks + baseline in `text-faint` (`border` color). One rounded `border` panel only here; title `result` on the top border.
- Footer stats row: labels `text-muted`, numbers `Bold` `text-primary`; `incorrect` value `error` if > 0.
- New best → append ` ★ new best` in `success` next to WPM.

---

## 4. Settings

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                                                                                │
│                              S E T T I N G S            ← title accent bold    │
│                  ────────────────────────────────────   ← border separator     │
│                                                                                │
│   ▎ Theme                                  ‹ default ›   ← selected row:        │
│                                                            surface-alt bg,      │
│                                                            ▎=accent, value=     │
│                                                            accent bold          │
│     Error mode                             ‹ allow-continue ›                   │
│     Default mode                           ‹ time ›      ← unselected:          │
│     Default length                         ‹ 30 ›          label text-muted     │
│     Smooth cursor                          ‹ off ›         value text-muted     │
│     Blink cursor                           ‹ off ›                              │
│     Restart flash                          ‹ off ›                              │
│     Sound                                  ‹ off (stub) › ← text-faint, faint   │
│                                                                                │
│                  ────────────────────────────────────                          │
│   Smooth cursor: trail effect — limited in terminals     ← help line, faint     │
│                                                            (desc of sel. row)   │
│                                                                                │
│                                                                                │
│                                                                                │
│  ↑↓ move · ←→ change · esc save & back · ctrl+c quit      ← footer text-faint   │
└──────────────────────────────────────────────────────────────────────────────┘
```

Notes:
- Selected row: `▎` `accent` left bar + `surface-alt` bg across the row width; label `text-primary` `Bold`; value in `‹ › ` brackets `accent` `Bold`.
- Unselected rows: label `text-muted`, value `text-muted`.
- Stub row (`Sound`): whole row `text-faint` `Faint`, ` (stub)` suffix.
- Contextual help line under separator reflects the selected row; `text-faint`.
- Settings auto-persist; `esc` just returns.

---

## 5. History

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                                                                                │
│                               H I S T O R Y             ← title accent bold    │
│                                                                                │
│   trend  ▁▂▂▃▃▄▅▄▅▆▆▇▆▇▇█▇█  last 18 tests           ← sparkline accent       │
│                                                          label text-muted      │
│   ──────────────────────────────────────────────────────────────────────────  │
│    DATE              MODE        WPM    ACC    CONS                ← header     │
│   ──────────────────────────────────────────────────────────────  text-muted  │
│  ▎ 2026-05-18 14:02   time 30     94    97%    95%   ★          ← selected row │
│    2026-05-18 09:51   words 50    88    96%    91%               surface-alt   │
│    2026-05-17 22:13   time 60     91    98%    93%               ▎=accent      │
│    2026-05-17 20:40   quote       85    94%    88%                              │
│    2026-05-16 18:05   time 30     90    97%    92%   ← rows: WPM=text-primary  │
│    2026-05-16 12:30   words 25    79    93%    85%     bold, acc=success if    │
│    2026-05-15 23:11   time 15     96    99%    96%   ★ ≥95 else muted          │
│    2026-05-15 08:02   time 30     83    95%    89%     ★ = new-best=success    │
│    2026-05-14 19:44   words 100   87    96%    90%                              │
│   ──────────────────────────────────────────────────────────────              │
│   showing 1–9 of 142            ↑↓ scroll                ← meta text-faint      │
│                                                                                │
│  ↑↓ scroll · g/G top/bottom · esc back · ctrl+c quit      ← footer text-faint  │
└──────────────────────────────────────────────────────────────────────────────┘
```

Notes:
- Top trend sparkline `▁`-`█` in `accent`, label `text-muted`.
- Table header row `text-muted` UPPERCASE; single-line `border` rules above/below header and at list bottom.
- Selected row: `▎` `accent` + `surface-alt` bg. WPM `Bold` `text-primary`; ACC `success` if ≥95 else `text-muted`; `★` best badge `success`.
- Scrollable: window of rows, `g/G` jump; meta line shows range + count in `text-faint`.

---

## Responsive / Degraded (all screens)

`termW < 60` or `termH < 20` → replace entire screen with:

```
┌──────────────────────────────────────────┐
│                                            │
│            Terminal too small              │  ← warning #E3B341/179
│      Need at least 60×20 (now 54×18)       │  ← text-muted
│       Resize to continue · ctrl+c quit     │  ← text-faint
│                                            │
└──────────────────────────────────────────┘
```

Narrow (60–72 cols): footers drop action words, keep glyphs: `↹ · ⌃r · esc`. Word-stream re-wraps to `termW-4`. Logo: if `termW < 64`, swap ASCII art for plain `Bold` `accent` text `monkeytype`.
