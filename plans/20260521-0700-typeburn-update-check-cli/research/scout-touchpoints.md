# Scout Report — v2.1.0 update-check touchpoints

Per-phase code location guide for implementing the synchronous update-check feature. Each section identifies exact file:line ranges and brief impact summaries.

---

## 1. cmd_version.go

**File:** `internal/cli/cmd_version.go`

- **Flag registration:** Line 24 — `cmd.Flags().BoolVar(&asJSON, "json", false, ...)` pattern establishes flag attachment.
- **Insertion point for `--check-update`:** Add adjacent BoolVar at line 24+ (same block as `--json`).
- **Human output path:** Lines 29–31 — `fmt.Fprintln(cmd.OutOrStdout(), version.String())` is the text renderer.
- **JSON output path:** Lines 33–51 — JSON struct with `json:` tags and `json.NewEncoder().SetIndent()` + `Encode()`.
- **Change summary:** Add a new `checkUpdate` bool var, wire it into `runVersion(cmd, asJSON, checkUpdate)` signature, and conditionally invoke `update.Check()` before returning if true. No new JSON fields needed yet (--check-update is a side effect trigger, not output augmentation).

---

## 2. cmd_config.go

**File:** `internal/cli/cmd_config.go`

- **Config key pattern:** Lines 86–102 define `configRows()` and `configGet()` using a slice-of-slice-of-strings ("table") model; lines 104–136 handle set via a switch/case statement on the key string.
- **Key registry:** Fully explicit — a switch with each case being a valid config key (lines 105–134). No reflection; no map-based registry.
- **Normalized keys:** One row per key; order matches the struct fields in `config.Settings` (line 86: theme, default_mode, default_length, blink_cursor).
- **Minimum to add `update_check` bool key:**
  1. Add row to `configRows()`: `{"update_check", strconv.FormatBool(s.UpdateCheck)}`
  2. Add case to `configSet()` switch: `case "update_check":` with `parseBool(value)` validation + assign to `s.UpdateCheck`.
  3. Add case to `configGet()` slice iteration (no code change — it is loop-based).
- **Change summary:** Declarative; three add-points (rows, set switch, and config.Settings struct field). No infrastructure change needed.

---

## 3. internal/config/settings.go

**File:** `internal/config/settings.go`

- **Settings struct:** Lines 35–40 — Four fields: Theme (string), DefaultMode (Mode), DefaultLength (int), BlinkCursor (bool), all with `json:"..."` tags.
- **Defaults function:** Lines 44–51 — Hardcoded return of default Settings with BlinkCursor = false.
- **Normalize() function:** Lines 55–75 — Repairs unknown enum values in place. Does not repair bool fields (they are safe as-is).
- **Insertion points for new `UpdateCheck bool` field:**
  1. Add to Settings struct (line 35+): `UpdateCheck bool `json:"update_check"`` (note underscore in JSON tag).
  2. Add to Defaults() return (line 44+): `UpdateCheck: true,` (propose true to encourage safe early checking).
  3. No Normalize() changes needed (bool has no invalid values).
- **Change summary:** Add 1 line to struct, 1 line to Defaults(). LoadSettings/SaveSettings in storage/ handle JSON marshaling automatically; no changes needed there.

---

## 4. internal/cli/cmd_run.go (synchronous check insertion)

**File:** `internal/cli/cmd_run.go`

- **Pre-TUI setup location:** Lines 63–95 in `runTUICommand()`. This is called after `buildRunRequest()` validation.
  - Line 67: Settings are loaded and optionally overridden.
  - Line 84: Model is constructed with `app.New(theme.Load(...), settings, codeText, "")`.
  - Line 85: StartTestMsg is built and fed to the model.
  - **Insertion point:** **Between lines 83 and 84** (after all CLI flags are resolved, before the model is constructed).
- **Config access:** `settings := e.loadSettings()` at line 67 — the loaded config is available throughout. Check `settings.UpdateCheck` to decide whether to run the synchronous check.
- **Exact pseudocode insertion:**
  ```
  // Line 83: codeText is finalized
  if settings.UpdateCheck {
    result, _ := update.Check(ctx) // result is optional; ignore error for now
    // Hint is passed to New() or deferred to result-screen rendering
  }
  model := app.New(...)
  ```
- **Change summary:** 3–4 lines of conditional code; no new dependencies beyond the to-be-created `internal/update` package. Settings is already bound and available.

---

## 5. internal/ui/screen_result_view.go (footer hint slot)

**File:** `internal/ui/screen_result_view.go`

- **Current footer rendering:** Line 27 — `footer := RenderFooter(resultHints(), m.w, m.th)`.
- **resultHints() function:** Lines 14–21 — Returns a slice of Hint structs (Key, Action pairs). Currently 4 hints: tab, ctrl+r, esc, 3.
- **Footer layout:** Lines 39–42 in View() — footer is placed at the bottom; vertical padding keeps it pinned.
- **Optional upgrade line insertion:** After the panel (line 40) and before the vertical spacer logic (line 34+), a new optional "upgrade available" line can be inserted if a hint is set.
- **Natural placement:** Between the panel bottom border and the footer hints. Recommend a subtle style using `theme.RoleWarning` or `theme.RoleTextMuted` to avoid alarm.
- **Exemplar from existing code:** Line 27 in screen_history_view.go shows how to render a label with muted style; apply the same pattern.
- **Change summary:** ResultModel gains an optional `updateHint *update.Result` field; View() checks it and renders a 1-line notice if non-nil (e.g., "newer version available: v1.1.0"). No change to existing hints or layout logic.

---

## 6. internal/app/model.go (option pattern for update hint)

**File:** `internal/app/model.go`

- **Current constructor:** Lines 60–77 — `New(th theme.Theme, settings config.Settings, codeText, codeHint string) Model` with no functional options. Sets all fields by value in the returned Model struct.
- **No existing option pattern:** Unlike the cli/runtime.go `Option` pattern, app/model.go uses positional params only.
- **Insertion points for optional update hint:**
  1. Add field to Model struct (line 27+): `updateHint *update.Result` (pointer because it is optional).
  2. Add parameter to `New()` signature: `updateHint *update.Result` (simple approach) OR introduce an Option-pattern builder (breaking change but cleaner for future extensibility).
  3. Update `New()` body (line 66+): `m.updateHint = updateHint` assignment.
  4. Wire it to ResultModel (lines 100–104 in handleResultMsg): When ResultMsg arrives, instantiate ResultModel with `NewResult(...).WithUpdateHint(m.updateHint)` if the hint is set.
- **Hint flow to ResultModel:** ResultModel.Update/View already use method chaining (e.g., `WithBest()` line 53–56). Add a parallel `WithUpdateHint()` method:
  ```go
  func (m ResultModel) WithUpdateHint(hint *update.Result) ResultModel {
    m.updateHint = hint
    return m
  }
  ```
- **Change summary:** Add 1 field to Model, add 1 param to New(), wire one assignment in New(), add 1 method to ResultModel, and call it in handleResultMsg(). No refactor of existing logic; all additive.

---

## 7. internal/ui/messages.go (new UpdateHintMsg pattern)

**File:** `internal/ui/messages.go`

- **Existing messages (exemplars):**
  - `ResultMsg` (lines 18–24): Carries metrics.Result + Mode + Length + QuoteLen + CodeText.
  - `CodePastedMsg` (lines 38–40): Carries Text (the pasted string).
  - `SettingsChangedMsg` (lines 49–51): Carries Settings.
- **Pattern:** Simple struct with public fields (no methods), used as a signal + data carrier. No receiver methods on the message itself; the root model or sub-model handles it in Update().
- **Proposed UpdateHintMsg:**
  ```go
  // UpdateHintMsg is emitted by a synchronous update.Check() call (wrapped in a Cmd)
  // when a newer version is available. The root model receives it and passes it to
  // the next screen or caches it for the result screen. (Deferred pending design clarity.)
  type UpdateHintMsg struct {
    Result *update.Result  // nil = no update available
  }
  ```
- **Alternative (simpler, deferred):** For v2.1.0 Phase 1, skip the message entirely and pass the hint directly via app.New() parameter as shown in section 6. Only promote to a message if the hint needs to survive a screen transition (e.g., Home → Typing → Result).
- **Change summary:** 1 new message struct. No existing messages need changes. The message is optional if the hint is passed via New() direct param.

---

## 8. internal/cli/output/ package (update result renderer)

**File:** `internal/cli/output/json.go` and `internal/cli/output/table.go`

- **Existing renderers:**
  - `RenderJSON(w io.Writer, v any) error` (lines 9–12): Uses `json.NewEncoder()` + `SetIndent()` + `Encode(v)`.
  - `RenderTable(w io.Writer, headers []string, rows [][]string) error` (lines 10–20): Tab-aligned rows with tabwriter.
- **Shape:** Both are simple `(io.Writer, data) error` signatures. No format-selection dispatch layer; callers choose which to invoke based on a `--json` flag.
- **For update results:**
  - **Human output:** A 1-line summary, e.g., "version 1.1.0 available; run `typeburn version --check-update` to see details." Use an existing call like in cmd_version.go line 30: `fmt.Fprintln(cmd.OutOrStdout(), ...)`.
  - **JSON output:** Reuse RenderJSON with an UpdateResult struct: `{ "available": true, "current": "1.0.0", "latest": "1.1.0", ... }`. The struct marshals automatically.
- **Insertion point:** No changes to output/ needed for Phase 1. The human output is handled in cmd_version.go's runVersion(); JSON is a simple RenderJSON call with a struct. Add an UpdateResult type in internal/update/ and call RenderJSON from cmd_version.go.
- **Change summary:** None needed to output/. The two renderers are sufficient as-is. New code calls them with the appropriate data types.

---

## 9. internal/storage/ package (helper functions)

**File:** `internal/storage/atomic_write.go`, `internal/storage/settings_store.go`

- **Atomic write:** Lines 14–45 in atomic_write.go — `atomicWrite(path string, data []byte) error` handles write-to-temp, fsync, rename. Private function; used by SaveSettings.
- **LoadSettings:** Lines 25–46 in settings_store.go — Reads file, unmarshals JSON, calls Normalize(), returns config.Settings. Graceful fallback to Defaults() on any error.
- **SaveSettings:** Lines 52–69 in settings_store.go — Marshals Settings to JSON, ensures parent dir exists (mode 0700), calls atomicWrite(). No error on missing parent; MkdirAll succeeds silently.
- **SettingsPath:** Lines 13–19 in settings_store.go — Returns `$XDG_CONFIG_HOME/typeburn/settings.json` (via config.ConfigDir(), which calls config/xdg-paths.go).
- **For update check persistence (future):** If the feature later needs to cache the last-check timestamp, use the same atomic pattern. For v2.1.0 Phase 1, no new storage helpers are needed — the UpdateCheck bool field is part of config.Settings and auto-persists.
- **Change summary:** No new functions or changes needed. Existing LoadSettings/SaveSettings handle the new UpdateCheck field automatically via JSON struct tags.

---

## 10. Test conventions

**File:** Various test files in `internal/cli/`, `internal/ui/`, `internal/app/`

- **CLI tests (cmd_run_test.go exemplar):**
  - Lines 14–25: Construct a minimal `*cobra.Command` stub with flags preset.
  - Lines 49: Call the command's validation function directly: `buildRunRequest(cmd, flags, settings)`.
  - No full Execute() harness; unit tests validate logic in isolation.
  - **Pattern for update-check CLI:** Test `buildRunRequest()` with `--check-update` flag; mock `update.Check()` via dependency injection (pass a mock env.checkUpdate func).
  
- **UI tests (screen_result_test.go exemplar):**
  - Lines 5–37: Construct models directly: `NewResult(msg, theme, keymap)`.
  - Call Update() and View() methods on the model; assert output strings or returned commands.
  - No teatest golden files in the repo; all UI tests use string matching or snapshot assertions (e.g., line 34: `if st.CodeText != snippet`).
  - **Pattern for result-screen hint:** Test `ResultModel.WithUpdateHint()` with sample update.Result; assert the hint appears in View() output.

- **HTTP testing (none currently in repo):**
  - The codebase has no net/http imports in test files. If update.Check() does HTTP (via net/http or similar), wrap it behind an interface in the env struct (like loadCode, loadSettings) for dependency injection.
  - Mock by providing a stub function in tests, or use httptest.Server for E2E if needed.

- **CI gate:** `.github/workflows/ci.yml` line 54 — `make notui-noexit-check` enforces that `internal/cli/notui/` never calls `os.Exit`. If `internal/update/` is added, it should not import `cli/notui` (no circular dependency risk). The guard will pass.

- **Change summary:**
  - CLI: Existing pattern (flag stubs, direct function calls) is sufficient.
  - UI: Existing pattern (direct model construction) is sufficient.
  - HTTP: Wrap update.Check() behind an env dependency if testing is needed.
  - CI: No changes; notui-noexit-check already passes.

---

## 11. CI gates

**File:** `.github/workflows/ci.yml`

- **Line 54:** `make notui-noexit-check` — Runs the Makefile target defined at line 29 of Makefile:
  ```makefile
  notui-noexit-check:
      @if grep -R "os\\.Exit" internal/cli/notui >/dev/null 2>&1; then ...
  ```
  This ensures `internal/cli/notui/` code never calls `os.Exit` (exit codes are returned as errors instead, allowing the root command to decide).

- **Impact on new internal/update/ package:** None. The update package does not live in cli/notui/; it is a sibling to cli/, config/, ui/, etc. The grep is scoped to `internal/cli/notui/` only.

- **Full CI pipeline:** Lines 27–34 — Build, Vet, Format check, Test (race detector), notui-noexit-check, Binary size check. All pass with the new update package as long as:
  - No `import "os"` (exit) in update.go.
  - No infinite HTTP retry loops (test with reasonable timeouts).
  - No unexported internal types (keep it compartmentalized).

- **Change summary:** No CI changes needed. New package passes all existing gates.

---

## Summary of insertion points by phase

| **Phase** | **File** | **Line Range** | **Change Type** |
|-----------|----------|----------------|-----------------|
| 1 (Config) | internal/config/settings.go | 35–40, 44–51 | Add UpdateCheck bool field + update Defaults() |
| 1 (Config) | internal/cli/cmd_config.go | 86–102, 104–136 | Add update_check key to rows, set switch |
| 2 (Version flag) | internal/cli/cmd_version.go | 24 | Add --check-update BoolVar |
| 3 (Pre-TUI check) | internal/cli/cmd_run.go | 83–84 | Insert conditional update.Check() call |
| 4 (App model) | internal/app/model.go | 27, 60–77 | Add updateHint field, wire to New() |
| 5 (Result hint) | internal/ui/screen_result.go | 18–29 | Add optional updateHint field to ResultModel |
| 5 (Result view) | internal/ui/screen_result_view.go | 26–48 | Render optional hint in footer region |
| 5 (Result hint setup) | internal/app/model.go | 100–104 | Wire hint to ResultModel.WithUpdateHint() in handleResultMsg |

---

## Unresolved questions & design notes

1. **Update check on `--no-tui` path:** cmd_run_notui.go is separate from runTUICommand. Should --no-tui also trigger the check? Deferred to design review.

2. **Error handling for update.Check():** Should a network failure during the check block the test start, or should it be best-effort? Recommend best-effort (silently continue); propagate no error from cmd_run.

3. **Update hint persistence across Home → Result:** Currently proposed to pass via app.New() directly. If the hint needs to survive a Home-screen browse (user stays on Home, dismisses the hint, then starts a test), promote UpdateHintMsg to a proper message in the event loop.

4. **Version comparison semver:** The researchers found `github.com/Masterminds/semver/v3` is a solid choice for version parsing. Verify it is not already imported; if not, add to go.mod as part of Phase 3 (HTTP integration).

5. **Config flag semantics:** Is `update_check: true` the default (checks by default, user can opt out), or `false` (no checks unless explicitly enabled)? Recommend `true` with a prominent notice on first launch or in help text.

6. **Test coverage:** Existing test patterns (CLI stubs, UI direct construction) are sufficient. No new testing infrastructure needed. Plan for at least 1 unit test per new message type and 1 integration test (mock HTTP result + end-to-end render).

