---
phase: 4
title: "Opportunistic TUI Wiring + Result Footer"
status: pending
priority: P1
effort: "4h"
dependencies: [1, 2]
---

# Phase 4: Opportunistic TUI Wiring + Result Footer

## Overview

When `update_check=on` AND the user launches the TUI (`typeburn` bare, or
`typeburn run` without `--no-tui`), synchronously call
`update.Check(ctx, ver, force=false)` before `tea.NewProgram(...).Run()`.
On a cache hit (typical), this is sub-millisecond. On a cache miss, up to
1.5s. If a result hints at an upgrade, pass it via a new `app.New()`
parameter to the root model, which forwards it to `screen_result` for
footer rendering after the typing session.

**`--no-tui` path explicitly does NOT trigger** (locked decision; scout §11 Q1).

## Requirements

**Functional**
- TUI launch path: `cmd_run.go` runTUICommand — sync `update.Check()` when
  `settings.UpdateCheck == true` AND `version.Resolve().Version` is not a
  dev/pseudo-version.
- Bare `typeburn` (root-cmd fall-through to TUI): same gate applies.
  Confirm `cmd_run.go` is the actual entry — if not, find the right spot.
- Hint surfaces only on the Result screen footer (NOT Home, NOT during typing).
- Footer text (locked verbatim):
  `↑ v<latest> available — run "typeburn version --check-update"`
  Single line, muted style, below existing hints. No release URL/upgrade
  commands (those live in `version --check-update`; the TUI hint just points there).
- If `update.Check` returns error or nil: no footer change, no log spam.

**Non-functional**
- New code in `cmd_run.go` ≤ 10 LOC.
- New parameter on `app.New(...)` — confirm it doesn't break existing tests
  (scout §6 shows the constructor; will need test updates).
- `screen_result` LOC after change must stay < 200.
- Result footer style: reuse `theme.RoleTextMuted` (scout §5 recommendation).

## Architecture

Scout §4 confirms insertion point: `internal/cli/cmd_run.go` between settings
load (line 67) and `app.New(...)` call (line 84). The hint flows as a positional
param into `app.New(...)`, then via a new `ResultModel.WithUpdateHint(*update.Result)`
method when the root model transitions on `ResultMsg` (scout §6 lines 100-104).

**Why pass via constructor, not a `tea.Cmd` emitting `UpdateHintMsg`?**
- The check runs *before* `tea.NewProgram(...).Run()`, so a message-based
  flow would require firing the cmd in the model's `Init()`, which adds an
  async lifecycle to the TUI.
- Result is known at startup; passing by value through New() is simpler,
  predictable, and pure-Elm-compatible.
- If a future requirement needs mid-session update detection (re-check while
  user idles), promote to `UpdateHintMsg` then.

Scout §6 step 4 also suggests "Option-pattern builder for app.New" — we will
**not** introduce that for v2.1; just add a positional `*update.Result` param.
Justification: KISS; current `New(theme, settings, codeText, codeHint)`
signature is already positional; adding one more pointer param matches.

## Related Code Files

- **Modify:** `internal/cli/cmd_run.go` (lines 67-84): sync check + pass to app.New.
- **Modify:** `internal/app/model.go` (lines 27, 60-77, 100-104): field, ctor param, hint forwarding.
- **Modify:** `internal/ui/screen_result.go`: new field + `WithUpdateHint` method.
- **Modify:** `internal/ui/screen_result_view.go`: render hint when set.
- **Modify:** all existing call-sites of `app.New(...)`: add `nil` param.
- **Modify:** existing tests touching `app.New(...)` and `ResultModel`.

## Implementation Steps

1. **`cmd_run.go` insertion** (between lines 83-84):
   ```go
   var updateHint *update.Result
   if settings.UpdateCheck {
       ctx, cancel := context.WithTimeout(cmd.Context(), 1500*time.Millisecond)
       defer cancel()
       if r, err := update.Check(ctx, version.Resolve().Version, false); err == nil && r != nil && r.UpgradeAvailable {
           updateHint = r
       }
       // silent on any error — never block TUI launch
   }
   model := app.New(theme.Load(...), settings, codeText, "", updateHint)
   ```
2. **`app/model.go`:** Add `updateHint *update.Result` field; add param to
   `New(...)` signature; assign in body; in `handleResultMsg` (line 100-104),
   when constructing `ResultModel`, call `.WithUpdateHint(m.updateHint)`.
3. **`screen_result.go`:** Add `updateHint *update.Result` field; add method
   `WithUpdateHint(*update.Result) ResultModel` (returns updated copy, matches
   existing `WithBest` chain pattern — scout §5 exemplar line 53-56).
4. **`screen_result_view.go`:** After the panel (line 40), before footer hints,
   insert:
   ```go
   if m.updateHint != nil {
       hint := fmt.Sprintf("↑ %s available — run \"typeburn version --check-update\"", m.updateHint.Latest)
       lines = append(lines, m.th.Style(theme.RoleTextMuted).Render(hint))
   }
   ```
   Adapt to actual rendering shape from existing view code.
5. **Update all `app.New(...)` callers** — <!-- SUPERSEDED by red-team-finding-4 (cmd_replay.go does NOT call app.New) and red-team-finding-1 (runtime.go:81 is the missed caller). See Red Team Review Updates below. -->
   verified callers via `grep -rn "app\.New" --include="*.go"`:
   - `internal/cli/cmd_run.go:84` — pass new param.
   - `internal/cli/runtime.go:81` (`app.NewFromDisk`) — see finding 1 resolution.
   - All `*_test.go` callers — grep before edit.
6. **Tests:**
   - `cmd_run_test.go`: new `buildRunRequest` does not change; mock the
     update.Check call via test seam similar to Phase 3.
   - `screen_result_test.go`: construct `ResultModel.WithUpdateHint(&update.Result{...})`,
     assert View output contains the locked text.
   - `app/model_test.go`: confirm hint forwards through `handleResultMsg`.
7. **Manual smoke:**
   - `typeburn config set update_check on`
   - Pre-populate a stub cache file with a fake newer version (e.g., v9.0.0):
     `echo '{"current":"v2.0.0","latest":"v9.0.0","upgrade_available":true,"release_url":"https://example.com","checked_at":"<now>"}' > ~/.local/state/typeburn/update-check.json`
   - Launch `typeburn`, complete a quick test, verify the footer hint renders.
   - Clear cache, set `update_check off`, relaunch, verify NO hint and no
     outbound HTTP (use `tcpdump`/`lsof -i`/network log to confirm).
8. **CI gates** green: `make test-race`, `make lint`, `make size-check`,
   `notui-noexit-check`.

## Todo List

- [ ] cmd_run.go: conditional sync update.Check + hint capture
- [ ] app/model.go: field + ctor param + handleResultMsg wiring
- [ ] screen_result.go: field + WithUpdateHint method
- [ ] screen_result_view.go: render locked footer hint
- [ ] Update all app.New callers (cmd_run, cmd_replay, tests)
- [ ] Tests for cmd_run, screen_result, app model
- [ ] Manual smoke (stub cache + real launch)
- [ ] Confirm --no-tui path does NOT call update.Check (gated by runTUICommand only)
- [ ] All CI gates green

## Success Criteria

- [ ] With `update_check=off`: launching TUI does NOT call `api.github.com`
  (verifiable via packet capture or `lsof`).
- [ ] With `update_check=on` + stub cache showing newer version: Result
  screen footer shows the locked hint text.
- [ ] With `update_check=on` + dev build: NO check (silent skip).
- [ ] `--no-tui` runner never triggers update check regardless of config.
- [ ] All existing `app.New(...)` callers updated; build green.
- [ ] `make test-race -count=1` passes.

## Risk Assessment

| Risk | Mitigation |
|---|---|
| 1.5s startup latency on cold cache | Cache hit makes this near-zero after first call; opt-in default-OFF means most users never pay it |
| `app.New(...)` signature change ripples broadly | Compiler-checked; one positional add at end; all callers grep-pable |
| Hint renders on dev builds via stale cache | Phase 1 dev-skip happens BEFORE cache read; safe |
| `--no-tui` accidentally inherits the check | runTUICommand vs runNoTUICommand are separate (scout §11); explicit gate |
| Hint text changes break test goldens | No teatest goldens in repo (scout §10) — string-match tests only; just update assertions |
| Bare `typeburn` (root fall-through) path skipped | Verify which function handles bare entry — same gate must apply |

## Security Considerations

- Outbound traffic only when user opted in. No first-launch surprise.
- No PII in HTTP request or cache.
- Hint text contains no remote-controllable strings — version is parsed/validated
  by the comparator in Phase 1.

## Next Steps

- Phase 5: docs, CHANGELOG, manual install/upgrade README section refresh.

## Red Team Review Updates (2026-05-21)

<!-- red-team-finding-1 --> **C1 — bare `typeburn` path misses the check.** Verified at `internal/cli/runtime.go:80-82`:
```go
func runHomeTUI(ctx context.Context, codeText, codeHint string) error {
    return runModelTUI(ctx, app.NewFromDisk(codeText, codeHint))
}
```
Bare `typeburn` flows `root.go:62` → `runHomeTUI` → `app.NewFromDisk` — completely
separate from `cmd_run.go:84`'s `app.New`. As-written, Phase 4 only patches
`cmd_run.go`, so the most common launch path silently misses the opportunistic check.

**Resolution — shared helper at the runtime layer:**

1. Extract a helper `func resolveUpdateHint(ctx context.Context, settings config.Settings) *update.Result`
   into `internal/cli/runtime.go` (or a sibling file). Returns nil unless
   `settings.UpdateCheck && version.Resolve().Version` is non-dev. Wraps the
   `update.Check(ctx, ver, false)` call with an 800ms `context.WithTimeout`
   (per finding 9). Silent-degrade.
2. Modify `runHomeTUI` to load settings (or accept them as a param from caller),
   call `resolveUpdateHint`, then pass to `app.NewFromDisk`.
3. Extend `app.NewFromDisk(codeText, codeHint, updateHint)` and propagate
   `*update.Result` through to `Model`.
4. `cmd_run.go` (the `typeburn run` path) calls the same helper, passes the result
   to `app.New(...)` as the new 5th positional param.

**Net result:** ONE call site for the update-check decision (runtime.go helper);
TWO call paths that consume its result (bare `typeburn` and `typeburn run`).

<!-- red-team-finding-4 --> **H2 — `cmd_replay.go` does NOT call `app.New`.** Verified by `grep -rn "app\.New" --include="*.go"`:
only callers are `cmd_run.go:84` (app.New) and `runtime.go:81` (app.NewFromDisk).
`cmd_replay.go` is a headless metrics replayer — imports `metrics`/`typing`/`output`/`config`
only. **Strike the `cmd_replay.go` bullet from Implementation Step 5.** Replace with:
- `internal/cli/runtime.go` — `runHomeTUI` + `NewFromDisk` (per finding 1).
- `internal/cli/cmd_run.go` — `app.New` (Phase 4 main path).

<!-- red-team-finding-6 --> **H4 — TUI footer re-validation.** Even with cache load-time validation (Phase 1
finding 6), defense-in-depth: the screen_result_view footer SHOULD NOT use
`fmt.Sprintf` interpolation with raw `m.updateHint.Latest`. Instead, run the same
regex validation at render time and fall back to no-render on mismatch. Locked text becomes:
```go
if m.updateHint != nil {
    latest := m.updateHint.Latest
    if !validSemverRe.MatchString(latest) {
        // load-time guard should have caught this; belt-and-suspenders
        return ""
    }
    hint := fmt.Sprintf("↑ %s available — run \"typeburn version --check-update\"", latest)
    lines = append(lines, m.th.Style(theme.RoleTextMuted).Render(hint))
}
```

<!-- red-team-finding-9 --> **H7 — opportunistic timeout 800ms.** Per Phase 1 finding 9, the opportunistic
path uses a tighter 800ms outer cap (vs 1.5s for explicit). Update Implementation
Step 1 pseudo-code:
```go
ctx, cancel := context.WithTimeout(cmd.Context(), 800*time.Millisecond)
```
Trade-off: a cold-cache opportunistic check sees 800ms wall delay (acceptable for
TUI launch); explicit `--check-update` users still get the full 1.5s budget. Most
launches hit cache (sub-ms) once the first save persists (per Phase 1 finding 2).

### Updated Implementation Step 5 (post-finding-1+4)

Touch the following call sites — verified by `grep -rn "app\.New"`:
- `internal/cli/cmd_run.go:84` — pass new 5th positional `*update.Result`.
- `internal/cli/runtime.go:81` — refactor `NewFromDisk` signature OR plumb hint
  via the new shared `resolveUpdateHint` helper (preferred — see finding 1).
- `internal/app/model_settings.go` (wherever `NewFromDisk` is defined) — accept
  the new optional parameter.
- All `*_test.go` callers of `app.New` and `app.NewFromDisk` — grep before edit:
  `grep -rn "app\.New\(\|app\.NewFromDisk\(" --include="*_test.go"`.

### Updated Manual Smoke (Step 7)

Pre-populate cache, then verify BOTH launch paths:
- `typeburn` (bare) — should show hint after a typing session.
- `typeburn run` — should show same hint.
- `typeburn run --no-tui ...` — must NOT trigger any check (locked decision; verify
  with `lsof -i` or packet capture).
