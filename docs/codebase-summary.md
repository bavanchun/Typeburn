# Codebase Summary

---

## Package Overview

### `internal/update` — GitHub Release Update Check + Self-Update

**Purpose:** Pure-stdlib (no bubbletea/lipgloss) package that fetches the latest GitHub release, compares semver, and caches the result (the update *check*), plus the self-update *pipeline* that downloads, verifies, and atomically installs a new binary. Opt-in feature; the check is never called on `--no-tui` paths.

**Key types:**
- `Release`: GitHub API payload (TagName, Draft, Prerelease, PublishedAt, HTMLURL)
- `Result`: {SchemaVersion, Current, Latest, UpgradeAvailable, ReleaseURL, CheckedAt}

**Entry points:**
- `Check(ctx, currentVer, force) (*Result, error)`: returns nil,nil for dev/unknown versions; uses cache unless force=true
- `FetchLatest(ctx, currentVer) (Release, error)`: raw HTTP call; ErrRateLimit/ErrUpstream sentinels
- `Compare(a, b string) int`: semver comparison (-1/0/1), tolerates leading v, strips pre-release suffix
- `IsPrerelease(tag string) bool`: detects -rc/-beta/-alpha/-pre and v0.0.0- prefix

**Cache:** `$XDG_STATE_HOME/typeburn/update-check.json` (default `~/.local/state/typeburn/`), 24 h TTL, 7 d max-age, schema-version + semver re-validation + URL-prefix check on load (injection guard).

**Test seams:** `getFetchURL()`/`setFetchURL()` and `getCacheFilePath()`/`setCacheFilePath()` — mutex-guarded accessors around the HTTP endpoint and temp-dir overrides used in tests.

**Self-update pipeline (`typeburn update`):**
- `Apply(ctx, currentVer, tag, execPath, goos, goarch, reportFn) (Outcome, error)`: the single entry point. Acquires an O_EXCL lock in the install dir, downloads + verifies the archive, extracts the binary, and atomically swaps it over `execPath`. `tag` MUST come from a live `Check(force=true)` Result, never the on-disk cache. `reportFn` (optional, pass nil to silence) is called with each `Stage` (downloading → verifying → installing) for progress reporting.
- `Preflight(execPath, env) Plan`: classifies the install (self-managed / Homebrew / `go install`) and probes the install dir's writability, so the CLI can refuse managed installs and fail fast on a read-only dir.
- Trust model: TLS + published SHA-256 checksums only — detects corruption/truncation, not a compromised host (binaries are unsigned; see SECURITY.md).
- `download.go`: redirect-restricted client (follows only GitHub-owned asset hosts / same host), size caps (50 MiB archive, 64 KiB checksums), O_EXCL temp writes; `assetName`/`assetURL` mirror the GoReleaser naming.
- `verify.go`: `parseChecksums` + streaming `verifySHA256` (case-insensitive).
- `archive.go`: `.tar.gz`/`.zip` extraction accepting only the exact top-level regular-file member (path-traversal + symlink hardened, decompression-capped).
- `selfpath.go`: `classifyInstall`, `goBinDir`, `canWrite`, `instructionFor`.
- `lock.go`: O_EXCL `.typeburn-update.lock` serialization.
- `progress.go`: `Stage` enum (downloading/verifying/installing) + `String()` for human labels; `report(fn, stage)` safely invokes optional progress callback.
- `replace_unix.go` / `replace_windows.go`: atomic same-dir rename; the Windows path moves the running exe aside with rollback + crash recovery (`restoreInterruptedUpdate`).

**Test seams:** `getFetchURL()`/`setFetchURL()` and `getCacheFilePath()`/`setCacheFilePath()` (check); `getDownloadBase()`/`setDownloadBase()` (self-update download).

**Files (check):** `result.go`, `compare.go`, `prerelease.go`, `client.go`, `cache.go`, `check.go` (+ `_test.go`).
**Files (self-update):** `download.go`, `verify.go`, `archive.go`, `selfpath.go`, `lock.go`, `progress.go`, `preflight.go`, `apply.go`, `replace_unix.go`, `replace_windows.go` (+ `_test.go`).

---

### `internal/version` — Build-Time Version Injection

**Purpose:** Expose the release version and commit metadata to the `--version` flag and error messages. Supports two injection paths: ldflags-stamped binaries (GoReleaser, `make release`) and fallback to module build info (bare `go install`).

**Key types:**
- `Info`: {Version, Commit, Date} — resolved version triple
- Package vars: Version, Commit, Date (linker targets via `-X`)

**Entry points:**
- `Resolve() Info`: precedence ldflags → debug.ReadBuildInfo() vcs settings → fallback "dev"
- `String() string`: renders one-line banner (e.g., "typeburn v1.0.0 (61a4afd, 2026-05-18T21:10:00Z, go1.26.2 darwin/arm64)")

**Behavior:**
- When released (GoReleaser): ldflags inject Version/Commit/Date; banner shows exact tag/SHA
- When installed via `go install`: ldflags empty; Resolve() pulls from go.mod version + git metadata in the binary
- Always succeeds; never panics

**Files:** version.go, version_test.go.

---

### Entrypoint — `main.go` & Flag Parsing

**Purpose:** Thin fang/cobra entrypoint. It builds `internal/cli.NewRoot()`,
executes it, and maps returned errors to process exit codes.

**Design:**
- `internal/cli.Decide(args)`: pure v1 compatibility helper for root aliases.
- Root-level unknown args fall through to the TUI; recognized subcommands parse strictly.
- `main()`: `fang.Execute(context.Background(), cli.NewRoot())`, then `os.Exit(cli.ExitCode(err))`.

**Rationale:** Avoids polluting the TUI with error banners or usage text; unknown input is gracefully treated as "user wants to type."

**Files:** `main.go`, `internal/cli/decide.go`, `internal/cli/decide_test.go`.

---

### `internal/cli` — Scriptable CLI Surface

**Purpose:** cobra/fang command surface: `run`, `history`, `version`,
`config`, `replay`, and `update`. Owns exit codes and command validation.

**Key behavior:**
- `run` launches the TUI directly into Typing via `ui.StartTestMsg`, or uses
  `internal/cli/notui` for raw terminal mode.
- `history` and `config` expose XDG persistence without opening the TUI.
- `replay` decodes `schema_version: 1` keystroke logs and calls `metrics.Compute`.
- `update` (`cmd_update.go`) wires `update.Check(force)` + `update.Preflight` +
  `update.Apply` into the self-update flow: `--check` is detect-only; managed
  installs refuse with `ExitManagedInstall`; non-tty refuses without `--yes`.
  Test seams: `setApplyFn`, `execPathFn`, `isInteractive`.
- `output` renders plain tables and deterministic indented JSON.

**Files:** `internal/cli/*.go`, `internal/cli/output/*.go`,
`internal/cli/notui/*.go`.

---

### `internal/runner` — Shared Session Construction

**Purpose:** Build typing sessions outside the UI layer. Keeps target generation
and `wordTarget` math shared by TUI and CLI.

**Entry points:**
- `NewSession(mode, length, quoteLen, seed, strict, punctuation, numbers) Session`
- `NewCodeSession(snippet, strict) Session`
- `RebuildEngine(target string, mode mode.Mode, length int, strict bool) *typing.Engine`

---

### `internal/mode` — Shared Mode Definitions

**Purpose:** Source-of-truth for typing mode identifiers and selectable length
policy. Pure package used by `typing`, `metrics`, `words`, `runner`, and
`config` without pulling in settings/keymap dependencies.

**Key types:**
- `Mode`: string enum {time, words, quote, code}

**Entry points:**
- `LengthsFor(mode)`: valid length options; quote/code return nil.

**Files:** mode.go, mode_test.go.

---

### `internal/app` — Root Elm Model & Routing

**Purpose:** Bubble Tea root model. Routes global messages (StartTestMsg, ResultMsg, AbortMsg, NavHistoryMsg, NavCodePasteMsg, CodePastedMsg) to sub-models. Manages screen enum + shared state (theme, keymap, terminal size, quit prompt overlay).

**Key types:**
- `Model`: root app state (screen enum, six sub-models, theme, keys, settings)
- `Screen`: enum (Home, Typing, Result, Settings, History, CodePaste)
- `quitPromptModel`: overlay shown when esc pressed on Home screen

**Entry points:**
- `NewFromDisk()`: loads settings from XDG, creates root model
- `(m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)`: routes StartTestMsg/ResultMsg/AbortMsg/NavHistoryMsg/WindowSizeMsg
- `(m Model) View() tea.View`: delegates to active screen's View; guards with degraded-mode check

**Animation driver:** `anim_driver.go` owns the self-stopping 33ms frame loop (`ui.FrameTickCmd`/`FrameInterval`, distinct from the 100ms timer tick). `handleFrameTick` stamps the shared clock `animNowMs`, forwards to the active screen, and re-arms only while `animActive` (active screen's `HasActiveAnim` OR a live transition) — an idle app schedules zero frame ticks. `transition.go` holds the root-owned Typing→Result transition (crossfade/wipe); expiry is derived in View and nil-ed out lazily in Update.

**Files:** model.go (root + Update/routing), model_constructor.go (New/Init), anim_driver.go (frame loop), transition.go (screen transition), model_code_paste.go, model_history.go (result persistence + transition start), model_settings.go, model_view.go (compose + degraded guard + transition blend), routing.go, quit_prompt.go, model_key_handler.go, model_time_helpers.go, smoke_test.go, model_test.go.

---

### `internal/ui` — Screen Sub-Models & Components

**Purpose:** Bubble Tea sub-models for Home, Typing, Result, Settings, History, and CodePaste screens. Emits domain messages (StartTestMsg, ResultMsg, AbortMsg, NavHistoryMsg, NavCodePasteMsg, CodePastedMsg). Implements rendering via Lip Gloss.

**Key types per screen:**
- `HomeModel`: mode/length picker, defaults display, best-result badge. Emits StartTestMsg.
- `TypingModel`: live typing test UI. Arms 100ms tick, delegates to typing.Engine, emits ResultMsg on completion.
- `ResultModel`: big-digit WPM display, sparkline chart, stat cards, footer navigation.
- `SettingsModel`: seven rows (Theme, Default mode, Default length, Blink
  cursor, Strict mode, Punctuation, Numbers), arrow navigation, auto-persist.
  `update_check` is the eighth persisted/CLI key and has no TUI row; the TUI
  Default mode row cycles Time/Words/Quote, although persisted/CLI values also
  accept Code.
- `HistoryModel`: scrollable table of all records; ★ marks eligible bucket
  leaders (Time/Words by mode+length, Quote by mode); Code/Strict never
  qualify. Supports vim-style navigation.
- `CodePasteModel`: paste-only Code-mode entry screen; normalizes bracketed paste via `codetext.Normalize`.

**Components (reusable):**
- `statCard`: displays WPM/Accuracy/Consistency with labels and big-digit rendering
- `wordStreamRenderer`: renders typed vs. untyped text with cursor block, handles rune-based positioning
- `historyTable`: scrollable table with column headers, best-result badges, mode filtering
- `sparkline`: mini-chart rendering per-second raw WPM data
- `timer`: displays elapsed time formatted as MM:SS
- `footer`: renders keybinds with terminal-width-aware collapse (full → short forms)

**Messages:** StartTestMsg, ResultMsg, AbortMsg, NavHistoryMsg, NavCodePasteMsg, CodePastedMsg, FrameTickMsg (in messages.go); FrameTickCmd/FrameInterval in frame_tick.go.

**Animation (always-on, NO_COLOR auto-adapting):**
- **Caret** (caret_anim.go, word_stream_anim.go, screen_typing_caret.go, screen_typing_input.go): blink (530ms, rides the 100ms tick), new-cell fade + just-vacated trail (150ms, ride the 33ms loop). A prefix-token cache re-Renders only the ≤3 animated cells per frame (≈70 allocs/op vs ≈3500 static — benchmarked).
- **Result reveal** (screen_result_reveal.go, screen_result_hero.go): WPM count-up in a fixed-width digit slot, sparkline draw-in (a `visible` count on the shared `Sparkline`), staggered stat cards.
- **New-best celebration** (celebration.go): deterministic one-shot sparkle burst on blank margin rows; new-best only; ASCII width-1 glyphs.
- All settle byte-identical to the static frame; under NO_COLOR each is layout-identical (line count + rune width preserved) via attribute-only variants.

**Files (by screen):** screen_home.go, screen_home_view.go, screen_home_test.go; screen_typing.go, screen_typing_view.go, screen_typing_actions.go, screen_typing_test.go; screen_result.go, screen_result_view.go, screen_result_test.go; screen_settings.go, screen_settings_view.go, screen_settings_test.go; screen_history.go, screen_history_view.go, screen_history_test.go; screen_code_paste.go, screen_code_paste_view.go, screen_code_paste_test.go. Plus: footer.go, timer.go, stat_card.go, sparkline.go, word_stream_renderer.go, ascii_big_digits.go, ascii_logo.go, degraded_notice.go, selectable_list.go, settings_rows.go, width_tier.go, history_table.go, result_render_helpers.go, typing_log_helpers.go, test_helpers_test.go, phase09_polish_test.go, ui.go.

---

### `internal/anim` — Pure Motion Math (UI-free)

**Purpose:** Stdlib-only animation math shared by the TUI motion system. UI-free (no `charm.land`/`lipgloss`/`bubbletea` imports) — joins typing/metrics/words as pure, table-tested logic. Every value is a pure function of `(startMs, nowMs, durMs)`, mirroring `metrics.Compute`'s post-hoc replay, so renders are deterministic.

**Key functions/types:**
- `EaseOutCubic` / `EaseOutQuad` / `EaseInOutQuad` / `Clamp01`: easing curves.
- `LerpColor(from, to color.Color, t)`: RGB interpolation over `image/color`; returns `nil` if either input is `nil` (the NO_COLOR branch signal). `LerpInt`/`LerpFloat`.
- `Tween{StartMs,DurMs,Ease}` with `Progress(nowMs)` / `Done(nowMs)`.
- `Clock`: aggregates tweens; `Active(nowMs)` powers the self-stopping frame loop.

**Files:** easing.go, color.go, tween.go, clock.go (+ table-driven tests).

---

### `internal/typing` — Pure Keystroke Engine

**Purpose:** Record and replay typed characters. Zero Bubble Tea / UI dependencies. All metric computation derives from keystroke log.

**Key types:**
- `Keystroke`: {TimeMs, Typed, Target, Correct} — one keystroke event (or backspace marker with Typed=0)
- `Engine`: {target []rune, typed []rune, log []Keystroke, startMs, mode, wordTarget, strict} — mutable state machine

**Entry points:**
- `New(target string, mode mode.Mode, wordTarget int)`: create engine for a test (default non-strict)
- `NewStrict(target string, mode mode.Mode, wordTarget int, strict bool)`: create engine with optional strict (stop-on-error letter) mode
- `Apply(rune, nowMs)`: record keystroke; auto-sets startMs on first call. When strict, blocks wrong forward keystrokes and doesn't advance cursor.
- `Backspace(nowMs)`: record deletion marker (Typed=0)
- `Typed()`: copy current typed buffer without replaying the log
- `ForwardKeystrokes()`: count forward entries for live metrics without copying the log
- `IsComplete() bool`: check if test goal met (Time/Words/Quote mode-dependent)
- `Replay() (final typed/target runes, duration, errors)`: reconstruct final state for metrics

**Files:** engine.go, completion.go, char_state.go, engine_test.go, char_state_test.go.

---

### `internal/metrics` — WPM, Accuracy, Consistency Formulas

**Purpose:** Pure computation of typing test metrics from keystroke log. Zero UI dependencies. Formulas verified against researcher-02-typing-metrics.md.

**Key types:**
- `Result`: {NetWPM, RawWPM, Accuracy, KeystrokeAccuracy, Consistency, CPS, TimeMs, CharCount, ErrorCount, ErrorHistory, WPMHistory, KeyMisses, ...} — final test metrics
- `KeyMiss`: {Key (folded rune), Label (display, e.g. "␣"), Misses, Attempts} — per-key fumble tally entry

**Entry points:**
- `Compute(log []typing.Keystroke, mode mode.Mode, endMs int64) Result`: compute all metrics post-hoc; populates `Result.KeyMisses` (nil on empty/zero-duration logs)
- `KeyHeatmap(log []typing.Keystroke) []KeyMiss`: per-key miss tally over the log — every wrong forward keystroke vs a real target (corrected fumbles included), case-folded, keys with ≥1 miss only, sorted misses desc → attempts desc → key asc. Surfaced on the Result screen and in CLI `key_misses` (JSON) / `most_missed_*` (table).
- `LiveWPM(log []typing.Keystroke, elapsedMs int64) float64`: log-based live WPM helper; returns 0 below 500 ms guard
- `LiveWPMFromCount(forward, elapsedMs) float64`: O(1) live WPM helper used by the Typing screen tick path
- `TrimAFK(log, mode, endMs) ([]typing.Keystroke, int64)`: remove trailing AFK seconds (Time mode only, >7s)
- `Consistency(wpmPerSecond []float64) float64`: 100 × tanh(1 − CV)

**Files:** compute.go, consistency.go, per_second.go, afk_trim.go, live_wpm.go, key_heatmap.go, compute_test.go, consistency_test.go, afk_trim_test.go, live_wpm_test.go, key_heatmap_test.go.

---

### `internal/words` — Generator & Quote Pack

**Purpose:** Generate test target text. Embedded word list (1000 words) + four quote buckets (short/medium/long/epic). Zero UI / Bubble Tea dependencies.

**Key types:**
- `QuoteLen`: enum {Short, Medium, Long, Epic} — buckets quotes by character range
- `quotePack`: {short, medium, long, epic []string} — embedded quotes for each bucket

**Entry points:**
- `ForMode(generator, mode, length, quoteLen, punctuation, numbers)`: return
  target string (Time: time buffer; Words: exact N words; Quote: random from
  bucket). Punctuation and numbers transform only Time/Words targets.
- `Word(n)`: return n-th word from wordlist (deterministic, seeded)
- `Quote(len QuoteLen)`: return random quote from bucket (seeded)
- `TimeBuffer()`: generate the oversized Words/Time buffer (~5 chars/word avg)

**Files:** generator.go, for_mode.go, quotes.go, generator_test.go, for_mode_test.go, quotes_test.go.

---

### `internal/config` — Settings, Keymap, XDG Paths

**Purpose:** User-facing configuration (theme, mode, length, cursor blink,
update check, strict mode, punctuation, numbers) +
centralized keybindings + XDG directory resolution. Settings and XDG helpers
stay storage-agnostic; keymap intentionally centralizes Bubble Tea key bindings.

**Key types:**
- `Mode`: alias of `mode.Mode` for persisted settings compatibility
- `Settings`: {Theme, DefaultMode, DefaultLength, BlinkCursor, UpdateCheck,
  StrictMode, Punctuation, Numbers} — loaded from disk or defaults
- `Keymap`: keybinds per screen (Home/Typing/Result/Settings/History) — maps Bubble Tea key codes to actions
- `QuoteLen`: enum for quote bucket selection

**Entry points:**
- `Defaults()`: baseline settings (theme="default", mode="time", length=30;
  BlinkCursor, UpdateCheck, StrictMode, Punctuation, and Numbers all false)
- `(s *Settings) Normalize()`: repair out-of-range values in place
- `DefaultKeymap()`: centralized keybindings
- `ConfigDir(), DataDir()`: resolve XDG paths (fallback to ~/.config, ~/.local/share)
- `LengthsFor(mode)`: valid length options for a mode

**Files:** settings.go, keymap.go, xdg_paths.go, settings_test.go.

---

### `internal/storage` — Atomic Persistence & New-Best Detection

**Purpose:** Load/save settings + history JSON. Atomic writes (temp+rename). XDG-compliant paths. Corrupt/missing files → safe defaults (never panic). Detect new personal bests.

**Key types:**
- `Record`: {WPM int, RawWPM, Accuracy, Consistency, Mode, Length, Time, Strict, ...} — one test result
- `HistoryStore`: interface-like functions (LoadHistory, AppendHistory, SaveSettings, LoadSettings)

**Entry points:**
- `LoadHistory()`: read history.json; return empty slice on any error (missing, corrupt, I/O)
- `AppendHistory(r Record)`: append + cap to 200 newest + atomic write
- `LoadSettings()`: read settings.json; return defaults on any error
- `SaveSettings(s Settings)`: atomic write to XDG_CONFIG
- `EffectiveWPM(r Record)`: precise NetWPM comparison with legacy WPM fallback
- `BestBucketKey(mode, length)`: shared leaderboard bucket key
- `BestWPMPerBucket(records)`: shared per-bucket bests for UI badges
- `EligibleForBest(r)`: excludes Code and Strict records from personal-best
  eligibility
- `IsNewBest(history, r)`: checks eligible records only; Time/Words use
  mode+length buckets and Quote uses one mode bucket

**Data flow:**
- Settings loaded at startup in `app.NewFromDisk()`
- History loaded on demand by HistoryModel + after each test completion
- Records appended via `AppendHistory()` immediately after test (from ResultMsg handler)
- New-best flag computed in root model's `handleResultMsg()`

**Files:** history_store.go, settings_store.go, new_best.go, atomic_write.go,
history_record.go, history_store_test.go, new_best_test.go, settings_store_test.go.

---

### `internal/theme` — Color Mapping via Roles

**Purpose:** Semantic color system. Decouple UI code from concrete hex colors.
Support grayscale `mono` and attribute-only `NO_COLOR` rendering without layout
changes.

**Key types:**
- `Role`: enum {RoleTextPrimary, RoleAccent, RoleError, RoleCursorBg,
  RoleCursorFg, ...} — semantic role (16 total)
- `Theme`: {name, colors map[Role]color.Color, noColor bool} — theme instance
- Default theme: dark + green accent + red error
- Mono theme: grayscale colors with a white accent

**Entry points:**
- `Load(name, noColor)`: return theme by name; respect NO_COLOR env
- `(t Theme) Style(r Role)`: return lipgloss.Style for role (color + attributes)
- `(t Theme) Color(r Role)`: return raw color.Color (nil under NO_COLOR)
- `Available()`: list selectable themes in display order: `default`, `mono`,
  `solarized-dark`, `solarized-light`, `dracula`, `nord`, `gruvbox-dark`,
  `gruvbox-light`

Any non-empty `NO_COLOR` value overrides the selected theme and renders by
attributes only; `mono` remains a color palette.

**Files:** theme.go, roles.go, default-theme.go, mono-theme.go,
solarized-dark-theme.go, solarized-light-theme.go, dracula-theme.go,
nord-theme.go, gruvbox-dark-theme.go, gruvbox-light-theme.go, theme_test.go.

---

## Installation & Distribution

### `install.sh` — Hardened POSIX Installer

**Purpose:** Standalone POSIX `/bin/sh` installer for systems without Go. Detects OS/arch, resolves the latest non-prerelease release tag, downloads the matching archive + checksums.txt, verifies SHA-256 integrity, and atomically installs the `typeburn` binary to ~/.local/bin (no sudo, no system package manager required).

**Design principles:**
- POSIX `/bin/sh` — works on any Unix-like system (Linux, macOS, BSD, etc.)
- Zero external dependencies — uses only `curl`, `tar`/`unzip`, `uname`, `sha256sum`
- Fail-safe verification — validates archive member before extraction; atomicity prevents partial/corrupt installs
- Test seams — environment variables (TYPEBURN_API, TYPEBURN_BASE_URL, TYPEBURN_LATEST_PATH, TYPEBURN_UNAME_S/M, VERSION, BIN_DIR) allow the offline test harness to mock GitHub API and filesystem

**Invocation:**
```bash
curl -fsSL https://raw.githubusercontent.com/bavanchun/Typeburn/main/install.sh | sh
```

**Files:** install.sh (root directory).

### `scripts/test-install-sh.sh` — Offline Installer Regression Harness

**Purpose:** Bash test harness that exercises 14 failure and success scenarios for install.sh *without internet access*. Mocks the GitHub API and release-asset downloads via localhost http.server.

**Design:**
- Subshell-local env per test case — full isolation, no test-to-test pollution
- Assertions on both exit status and filesystem state — a refused install must write nothing; a failed install must leave any prior binary byte-identical
- Integrated into CI as `installer` job in .github/workflows/ci.yml (runs shellcheck + harness + `goreleaser check`)

**Invocation:**
```bash
scripts/test-install-sh.sh
```

**Files:** scripts/test-install-sh.sh.

---

## File Naming Convention

**Actual convention used:** Mixed snake_case and kebab-case (Go standard + readability).

**Examples:**
- `screen_typing.go` (snake_case) — main struct + Update/View
- `screen_typing_view.go` (snake_case) — View rendering helper
- `screen_typing_actions.go` (snake_case) — action handlers (newTest, restartSame, etc.)
- `screen_typing_test.go` (snake_case) — unit tests
- `ascii-logo.go` (kebab-case) — ASCII art, non-essential utility
- `ascii_big_digits.go` (snake_case) — big-digit rendering
- `degraded-notice.go` (kebab-case) — degraded-mode notice, non-essential utility
- `char_state.go` (snake_case) — CharState type for typing state machine
- `char-state_test.go` (kebab-case in test) — matches implementation file name
- `default-theme.go` (kebab-case) — theme definition, semantic name
- `xdg-paths.go` (kebab-case) — XDG Base Directory Spec

**Pattern:** Core logic uses snake_case (engine, metrics, UI rendering). Utility/output modules use kebab-case (ascii art, theme names, XDG). All files <200 LOC (largest: ~190 LOC).

---

## Test Coverage

- **Unit tests:** metrics, typing, words, config, storage, theme (table-driven, no mocks, real data)
- **Integration tests:** smoke_test.go (full Home→Typing→Result flow)
- **UI tests:** teatest golden-file tests per screen (screen_home_test.go, screen_typing_test.go, etc.)
- **Race detection:** `go test ./... -race -count=1` — GREEN; no goroutine leaks
- **Format & vet:** `gofmt -l .` and `go vet ./...` — GREEN

---

## Release Engineering Files

**Build & Version Management:**
- `Makefile`: VERSION (git tag or "dev"), COMMIT (short SHA), DATE (UTC); LDFLAGS injection; targets: `build`, `test`, `test-race`, `version` (quick ldflags check), `snapshot` (dry-run), `release` (full publish)
- `.goreleaser.yaml` (v2, pinned v2.15.4): 6-platform matrix (linux/darwin/windows × amd64/arm64), trimpath + ldflags with v-prefixed version, archives (tar.gz for Unix, zip for Windows) include README/LICENSE/CHANGELOG, sha256 checksums, changelog filter excludes all git commits (uses `.github/release-notes.md` instead); determinism pins on `project_name`, `builds.binary`, `archives.name_template` (exact lowercase strings); no `before.hooks`; `release.prerelease: auto` safe-skips RC/beta tags; Homebrew cask commit to bavanchun/homebrew-tap-typeburn with `skip_upload: "auto"` and token isolation
- `.github/workflows/ci.yml`: ubuntu + macos matrix, build + vet + gofmt + race-test gates; new `installer` job runs shellcheck on install.sh + test-install-sh.sh harness + `goreleaser check`
- `.github/workflows/release.yml`: tag-triggered (`v*`), self-gating `test` job (contents:read) gates `publish` job (contents:write), SHA-pinned GoReleaser v2.15.4, concurrency-guarded, post-publish asset-count assertion (expects 7: 2 tar.gz linux + 2 tar.gz darwin + 2 zip windows + 1 checksums.txt)
- `.github/release-notes.md`: curated release notes handoff to GoReleaser (replaces auto-generated git log)
- `go.mod`: Go minimum version 1.25.0 (as of v1.5.0; bumped down from 1.26.2 to match bubbletea v2 + lipgloss v2 requirements)

**Supporting Documentation:**
- `CHANGELOG.md` (Keep a Changelog): semantic versioning, per-release sections with Added/Changed/Deprecated/Removed/Fixed
- `CONTRIBUTING.md`: build prerequisites (Go 1.25+ as of v1.5.0, GoReleaser v2.15.4 pinned), contribution guidelines, release process (tag-triggered CI)
- `SECURITY.md`: binary integrity model (SHA-256 verification), unsigned-binary disclosure policy
- `.github/ISSUE_TEMPLATE/*.md` & `.github/PULL_REQUEST_TEMPLATE.md`: standardized issue/PR submission
