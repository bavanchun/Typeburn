# Project Roadmap

---

## v1.0 — Complete (2026-05-18)

**Status:** SHIPPED. All 10 phases implemented, tested, shipped.
**Released:** Tagged `v1.0.0` (2026-05-18) — see [GitHub Releases](https://github.com/bavanchun/Typeburn/releases/tag/v1.0.0).

### Shipped Features

| Feature | Status | Notes |
|---------|--------|-------|
| **Modes** | ✅ | Time (15/30/60/120s), Words (10/25/50/100), Quote (short/medium/long/epic) |
| **Metrics** | ✅ | Net/Raw WPM, Accuracy, Consistency, CPS; per-second bucketing + chart |
| **UI Screens** | ✅ | Home (picker), Typing (live test), Result (big-digit display), Settings (4 settings), History (scrollable table) |
| **Persistence** | ✅ | XDG-compliant settings.json + history.json (cap 200 newest); atomic writes |
| **Themes** | ✅ | default (dark + green accent) + mono (greyscale attributes) + NO_COLOR support |
| **Keybindings** | ✅ | Per-screen central keymap (Home/Typing/Result/Settings/History); Vim-style navigation |
| **Error handling** | ✅ | Allow-continue + backspace only; no stop-on-error mode |
| **Resize handling** | ✅ | Graceful degraded notice (<60 cols or <20 rows); auto-resume |
| **Testing** | ✅ | go test -race -count=1 GREEN; teatest golden files; smoke tests |
| **CI/CD** | ✅ | GitHub Actions (ubuntu + macos); build → vet → gofmt → test |

### Shipped Phases

| Phase | Deliverable | Lines | Status |
|-------|-------------|-------|--------|
| 1 | Scaffold + theme + app skeleton | ~800 | ✅ Complete |
| 2 | Typing engine + metrics (TDD) | ~1200 | ✅ Complete |
| 3 | Words + embedded quotes | ~400 | ✅ Complete |
| 4 | Typing test screen | ~1100 | ✅ Complete |
| 5 | Home screen | ~900 | ✅ Complete |
| 6 | Result summary screen | ~1000 | ✅ Complete |
| 7 | Settings persistence | ~700 | ✅ Complete |
| 8 | History persistence | ~800 | ✅ Complete |
| 9 | Polish (resize, NO_COLOR) | ~600 | ✅ Complete |
| 10 | Tests, teatest, CI | ~300 | ✅ Complete |

**Total codebase:** ~7500 LOC (internal/ + main.go + tests)

---

## Backlog & Future Features

### High Priority (Post-1.0 Correctness Fixes)

#### M1: Timer Re-arm on restartSame — ✅ FIXED (commit d6369de)
- **Issue:** After Tab restart in Time mode, header WPM/elapsed froze until next keystroke
- **Root cause:** restartSame() returned bare model; now returns tickCmd() like newTest()
- **Fix location:** `internal/ui/screen_typing.go` (RestartSame branch)
- **Status:** Shipped in v1.0 (guard-verified safe: tick idles harmlessly while startMs==0)

#### M2: New-Best Precision (sub-WPM rounding) — ✅ FIXED (v1.0.1)
- **Issue:** New-best detection compared rounded int WPM; lost sub-WPM precision
- **Impact:** 75.4 and 75.0 both rounded to 75; faster run not flagged new-best (occasional missed ★)
- **Fix shipped:** Added `NetWPM float64` to `storage.Record`; `IsNewBest` and the
  history-table ★ now compare effective NetWPM, with legacy records (no `net_wpm`
  key → 0.0) falling back to their stored rounded WPM so prior bests are preserved
- **Status:** Shipped in v1.0.1 (2026-05-18)

### Medium Priority (Feature Enhancements)

#### Code Mode (Custom Text Input)
- **Description:** Paste arbitrary text for typing test instead of word/quote selection
- **Effort:** ~3 days
- **Dependencies:** new screen (Home → CodeInput) + input validation + word count estimation
- **Scope:** Would add one new mode type or variant

#### Vim Motion Keybindings
- **Description:** Alternative keybindings (hjkl, g/G for jump, etc.) alongside defaults
- **Effort:** ~1 day
- **Dependencies:** Settings addition (keybinding style selector)
- **Scope:** Centralized keymap already supports easy extension

#### Theme Packs — ✅ SHIPPED (v1.1.0)
- **Description:** Six community palettes mapped onto the 16 roles —
  `solarized-dark`, `solarized-light`, `dracula`, `nord`, `gruvbox-dark`,
  `gruvbox-light`. First light themes; `palette_luminance_test.go` guards
  against inverted palettes; `theme_available_sync_test.go` keeps the
  `config.Normalize` accepted-set in lockstep with `theme.Available()`.
- **Status:** Shipped in v1.1.0 (2026-05-19). Palettes documented in
  `design-guidelines.md` §2.2.

#### Additional Themes (future, if requested)
- One Dark Pro
- Catppuccin
- Tokyo Night

### Lower Priority (Nice-to-Have)

#### Sound Effects
- **Description:** Optional beep/chime on test completion, errors, or milestones
- **Dependencies:** cpal or beep library; might add 1-2 MB binary size
- **Scope:** Unlikely unless explicitly requested by user base

#### Smooth Scrolling / Animations
- **Description:** Gradual transitions between screens instead of instant
- **Effort:** ~2 days
- **Risk:** May impact responsiveness feel; Monkeytype focuses on snappy
- **Status:** Low priority (speed-of-feel = instant in current design)

#### Multiplayer / Online Sync
- **Description:** Race against other users, leaderboard integration
- **Effort:** 2-3 weeks (API client + state sync + backend integration)
- **Scope:** Requires backend service; out of scope for local TUI v1
- **Status:** Future major feature; scoped out of v1

#### Plugin System
- **Description:** User-defined themes, keybindings, or custom test generators
- **Effort:** 1-2 weeks (plugin loading, sandboxing, versioning)
- **Risk:** Security (arbitrary code execution) + maintenance burden
- **Status:** Deferred to v3+

#### Daily Challenges / Streaks
- **Description:** Themed daily test, streak counter, achievement badges
- **Effort:** ~3 days
- **Dependencies:** Backend for "today's challenge" seed, local streak tracking
- **Status:** Nice-to-have; requires additional product decision

#### Custom Wordlists
- **Description:** Load wordlist from file (.txt, one word per line)
- **Effort:** ~2 days
- **Dependencies:** File picker UI, validation
- **Status:** Medium priority if requested by users

---

## Known Limitations (v1)

### Code-Review Findings

| ID | Title | Severity | Impact | Recommended | Status |
|----|-------|----------|--------|-------------|--------|
| M1 | Timer tick re-arm on restartSame | MAJOR | Header frozen after Tab restart (Time mode) | Fixed | ✅ Shipped (d6369de) |
| M2 | New-best precision (rounded int WPM) | MAJOR | Occasional missed ★ badge | Fixed | ✅ Shipped (v1.0.1) |
| m3 | Parent dir not fsync'd on atomic write | MINOR | Power loss durability (acceptable for local data) | Document trade-off | Accepted |
| m4 | MissedChars field always 0 | MINOR | Field advertised but unusable (no target delivery) | Remove field | ✅ Removed (v1.0.1) |
| m5 | CJK runes not width-aware | MINOR | Text overflow in 60-col terminals with CJK quotes | Use lipgloss.Width if CJK added | Deferred |

### Design Constraints

| Constraint | Rationale | Impact |
|------------|-----------|--------|
| ASCII wordlist only | CJK width handling deferred; simple word length ~5 chars | No CJK quotes in v1 |
| 4 settings only | Minimize surface area; avoid scope creep | No sound, smooth scroll, etc. |
| Per-mode best only | Simpler history; full leaderboard deferred | ★ badge scoped to mode+length |
| 200-record history cap | Storage simplicity; no pagination | Oldest tests auto-rotated |
| No backend | Local-only; no sync | Multiplayer/leaderboard future |

---

## Breaking Changes & Deprecations

**None across v1.0 → v1.1.x.** All users can upgrade freely; the history/settings JSON schema stays backward-compatible.

**Already shipped (no user action required):**
- `NetWPM` added to the history Record in v1.0.1 — backward-compatible: records written by older versions lack the field and fall back to the stored rounded WPM.
- `MissedChars` removed in v1.0.1 — it was an always-zero stub; no real metric affected, no migration needed.

**Potential breaking changes for v2 (none planned):**
- Theme `Role` enum change if roles were renamed (unlikely; additive so far — v1.1.0 added 6 themes with zero role changes).

---

## Success Metrics & Next Steps

### v1 Launch Metrics
- ✅ All phases complete
- ✅ Tests green (-race)
- ✅ README + design-guidelines + system-architecture docs complete
- ✅ Code review SHIP verdict; M1 fixed in v1.0 (d6369de), M2 accepted as documented v1 decision

### Shipped Post-1.0
- ✅ **v1.0.1:** M2 new-best sub-WPM precision fixed; always-zero `missed` stat removed.
- ✅ **v1.1.0:** Six theme packs (Solarized D/L, Dracula, Nord, Gruvbox D/L);
  non-blocking persistence-failure notice; post-1.0 doc corrections.
- ✅ **v1.2.0:** Code mode — `--text <file>`/stdin, full-literal whitespace,
  new `internal/codetext` loader + isolated code renderer/viewport, saved to
  History but excluded from ★.
- ✅ **v1.3.0:** In-app paste for Code mode — `ScreenCodePaste` reached from
  the empty Code row, bracketed paste validated via the shared
  `codetext.Normalize` core; `--text` still supported and takes precedence.
- ✅ **v1.4.0:** Settings changes (theme, blink, default mode/length) apply
  live in-session — fixed an orphaned-callback bug that only persisted them
  to disk; typing content width scales to ~82% of wide terminals (floored at
  80) with the block vertically centered.

### Next (Optional)
1. **Gather user feedback** on missing features (Vim motions? more themes?)

### v2.0 Planning (Future)
- M4: Add target delivery mechanism; remove MissedChars 0-stub or make it meaningful
- m5: CJK width support (if quotes added)
- Code mode (custom text input)
- Backend sync / multiplayer (major feature)
- Plugin system (if ecosystem demand exists)

---

## Rollout & Support

### v1.0 Release
- **Artifact:** Binary in ./bin/typeburn
- **Distribution:** GitHub Releases (source + pre-built linux/darwin binaries)
- **Installation:** `go install Typeburn@latest` (via go.mod/go.sum versioning)
- **Minimum Go:** 1.26+
- **Minimum terminal:** 60 cols × 20 rows (ANSI 256-color or truecolor recommended)

### Support Policy
- **Bug reports:** GitHub Issues (typing logic, persistence, rendering)
- **Feature requests:** GitHub Discussions (prioritize by upvotes)
- **Documentation:** Wiki + inline code comments
- **Security:** Responsible disclosure to maintainer email

---

## Conclusion

**Typeburn v1.4.0 is the current release — feature-complete and production-ready.** The codebase is clean, tested, and well-documented. Post-1.0 work has been purely additive or corrective (v1.0.1 precision fix + stub removal; v1.1.0 six theme packs + persistence-failure notice + doc hygiene; v1.2.0 Code mode; v1.3.0 in-app paste for Code mode; v1.4.0 live Settings apply fix + wider centered typing layout) with zero breaking changes to existing users.

M1 (timer re-arm) and M2 (new-best precision) — the identified correctness bugs — were fixed in v1.0 (d6369de) and v1.0.1 respectively. v1.4.0 fixed a Settings live-apply bug (changes were persisted but not applied in-session) and improved the wide-terminal typing layout. Remaining backlog is additive or cosmetic.
