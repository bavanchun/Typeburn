# Project Overview & Product Development Requirements (PDR)

**Typeburn:** Terminal typing test in Go · Bubble Tea v2 + Lip Gloss v2

---

## What It Is

Distraction-free, keyboard-driven typing test for terminal. Six screens (Home,
Typing, Result, Settings, History, CodePaste) + four test modes (Time, Words,
Quote, Code) with live WPM/accuracy/consistency metrics, persistent history,
and a scriptable v2 CLI.

**Target user:** Developers, terminal enthusiasts, speed-typers who prefer keyboard-first, no-browser tools.

---

## v1 Feature Scope (Shipped)

### Modes & Durations
- **Time:** 15, 30, 60, 120 seconds (timer counts down; test ends when time expires)
- **Words:** 10, 25, 50, 100 words (test ends when word count typed)
- **Quote:** Short/Medium/Long/Epic from embedded pack (test ends when quote complete)
- **Code:** User text via `--text`, in-app paste, or `run --mode code --text`
  (exact whitespace match)

### Test Metrics
- **Net WPM:** correct characters / 5 / seconds × 60
- **Raw WPM:** all keystrokes / 5 / seconds × 60
- **Accuracy:** 100 × correct / (correct + incorrect)
- **Consistency:** 100 × tanh(1 − CV); measures keystroke pace steadiness
- **Per-second chart:** raw WPM bucketed by 1-second intervals

### UI Features
- **Home:** Mode/length picker; status displays (default settings, best result badge)
- **Typing:** Live word-stream, cursor block, header WPM/elapsed, footer keybinds
- **Result:** Big-digit WPM, sparkline, accuracy/consistency cards, top stats
- **History:** Scrollable table of all tests; per-mode best-result marker (★)
- **Settings:** Theme, default mode, default length, cursor-blink toggle (4 settings only)

### Persistence
- **Settings:** XDG_CONFIG_HOME/typeburn/settings.json (atomic write, 0600)
- **History:** XDG_DATA_HOME/typeburn/history.json (cap 200 newest; atomic write)

### Themes
- `default`: Dark green accent (Monkeytype-style), red error, amber warning
- `mono`: Greyscale attributes only (bold, underline, faint)
- `NO_COLOR` env: Switches to attribute-only rendering (no color codes, same layout)

### Input & UX
- **Typing mode:** Printable chars + backspace; allow-continue default, with optional letter-strict (stop-on-error) mode
- **Keybindings:** Centralized, per design spec (Home/Typing/Result/Settings/History each have distinct binds)
- **Resize handling:** Graceful degradation notice if <60 cols or <20 rows; auto-resume when resized
- **Paste support:** Ctrl+V pastes entire clipboard into typing; each char logged
- **CLI:** `run`, `history`, `version`, `config`, and `replay` subcommands with
  JSON output where useful

---

## Explicitly Excluded from v1

- Vim keybindings / emacs mode
- Multiplayer / online sync
- Leaderboard
- Sound effects
- Smooth scrolling / animations beyond cursor
- Plugin system
- AI-generated prompt content
- Daily challenges / streaks
- Custom wordlists
- CJK (wide-rune) word spacing (ASCII wordlist only)

---

## Success Criteria

- ✅ All 10 phases green (build + tests + vet)
- ✅ go test ./... -race -count=1 PASS
- ✅ gofmt clean, no vet warnings
- ✅ Metrics match spec (WPM/accuracy/consistency formulas verified)
- ✅ Persistence atomic (no partial writes; corrupt files → defaults)
- ✅ Degraded mode single chokepoint (no partial paint below 60×20)
- ✅ All 5 screens + 3 modes tested via teatest + manual smoke
- ✅ NO_COLOR + mono theme render correctly

---

## Known Limitations (Backlog)

1. **M1 (code-review):** Timer tick may skip re-arm on restartSame in Time mode (header frozen until next keystroke). Recommended fast-follow fix.
2. **M2:** New-best detection uses rounded int WPM (loses sub-WPM precision). Occasional missed ★ badge. Recommended post-1.0 fix; history is local-only.
3. **m3:** History file not parent-dir-fsync'd (acceptable for local user data; document trade-off).
4. **m4:** MissedChars field advertised but always 0 (no target delivery method in v1). Remove or pass target in v2.
5. **m5:** CJK runes not width-aware (low risk: ASCII wordlist only; breaks if CJK quotes added).

---

## Stack

- **Language:** Go 1.25+
- **UI Framework:** Bubble Tea v2.0.6 (Elm-style state machine, Cursed Renderer)
- **Styling:** Lip Gloss v2.0.3 (ANSI color + attributes)
- **Storage:** atomic JSON (XDG-compliant, gofmt-checked)
- **Testing:** go test + teatest (golden-file rendering)
- **CI/CD:** GitHub Actions (ubuntu-latest, macos-latest)

---

## Development Status

**v2.0 CLI implementation complete in working tree.** CI gates must pass before
release.
