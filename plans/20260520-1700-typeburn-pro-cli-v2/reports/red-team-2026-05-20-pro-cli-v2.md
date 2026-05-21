# Red-Team Review — Typeburn Pro CLI v2

**Date:** 2026-05-20
**Plan:** `plans/20260520-1700-typeburn-pro-cli-v2/`
**Reviewer:** code-reviewer (adversarial)
**Verdict:** **APPROVE_WITH_REVISIONS** — sound design, but 4 HIGH issues need plan-text changes before code is written.

---

## Executive Summary — Top Findings

1. **HIGH (F2) — `typeburn --version` semantics under `DisableFlagParsing: true` is ambiguous.** The plan's RunE calls `decide(args)` on raw args including `--version`. That path is fine for `--version`/`--text` but cobra parsing of `typeburn version --bogus` (subcommand strict) vs `typeburn --bogus` (root, falls through to TUI) creates two different code paths with no test asserting both. Plan should explicitly call out: at root level, cobra never sees `--bogus`; it goes through decide → falls through → TUI. Currently implicit.
2. **HIGH (F5) — Phase 1 extraction scope misses `screen_typing_actions.go:14-18`** (`restartSame`) which duplicates the `wordTarget = length*1000 if Time` math. Plan only mentions `newTypingWithSeed` extraction. Post-extraction this becomes a known-duplication bug source: changing the math in `runner.NewSession` won't propagate to restart. Plan must list `restartSame` as a Phase 1 modification target.
3. **HIGH (F8) — Phase 5 raw-mode lifecycle has a race-window gap.** Plan installs signal handlers *inside* `runRaw` AFTER `MakeRaw`. If a signal fires between `term.MakeRaw` succeeding (line 56 of phase doc) and `signal.Notify` being called (~5 lines later), the default SIGINT handler kills the process, defers run, but the goroutine isn't yet spawned — actually OK. The real gap: `signal.Notify` is correctly *before* the goroutine, but BOTH researcher-02 and Phase 5 disagree on placement vs researcher-02 ("install before raw mode"). Plan inverts the order — should be reordered per researcher-02 to eliminate the window entirely.
4. **HIGH (F11) — fang is pre-1.0; no explicit pin strategy in `go.mod`.** Plan says "pin exact version via `go get …@<sha>`" but Phase 2 step 1 is literally `go get github.com/charmbracelet/fang@latest`. The pin-strategy comment is in the risk row, not the implementation step. Will land as `@latest` → next breaking release on fang's side breaks Typeburn install. Step 1 must be `@<concrete-sha-or-tag>`.
5. **MEDIUM (F6) — Phase 4 `config set` validation contradicts existing `Settings.Normalize()` semantics.** Plan says "invalid value exits 1 with a message listing valid options" but the plan-prescribed implementation pipes through `Normalize()` which silently coerces unknown themes to `"default"` (`internal/config/settings.go:55-67`). Plan needs a NEW validation pass that rejects BEFORE calling Normalize, otherwise `config set theme zzz` will succeed (coerced) instead of failing.

---

## Findings Table

| ID | Sev | Phase | Title | Evidence | Recommendation |
|----|-----|-------|-------|----------|----------------|
| F1 | MEDIUM | 2 | cobra "did you mean" suggestion may leak for `typeburn version` typos | Cobra has `cobra.Command.SuggestionsMinimumDistance` enabled by default | Set `root.DisableSuggestions = true` if fall-through is sacred |
| F2 | HIGH | 2 | `--version` vs `version --bogus` strictness asymmetry undocumented | phase-02 step 7 lists `version --bogus` but no test for `typeburn --bogus` | Add explicit test case + plan-text rule: "any root-level `--flag` decide() doesn't recognize → fall through to TUI" |
| F3 | LOW | 2 | `typeburn -h` falls through DisableFlagParsing | With `DisableFlagParsing: true`, cobra still injects `-h` BEFORE RunE? No — DisableFlagParsing means root sees `-h` as a raw arg. decide() ignores it → TUI launches, NOT help | Acceptance #1 (`typeburn -h` shows help) is at risk; verify whether fang/cobra intercepts `-h` before RunE under DisableFlagParsing. Likely need explicit help handling. |
| F4 | LOW | 2 | `typeburn run --text foo.txt` vs `typeburn --text foo.txt` code paths | Both must reach same `codetext.Load`. Plan says alias resolves to "run --text" but `decide()` returns textPath directly, not via cobra | Plan should make decide()'s `--text foo.txt` path call the SAME function as `cmd_run.go`'s `--text` handler. Currently two parallel impls implied. |
| F5 | HIGH | 1 | Extraction skips `screen_typing_actions.go:restartSame` `wordTarget` math | `internal/ui/screen_typing_actions.go:13-21` duplicates `if mode==Time { wordTarget = length*1000 }` | Add to Phase 1: `restartSame` must call `runner.NewSession(...).Engine` (re-derive engine via runner). |
| F6 | MEDIUM | 4 | `config set` invalid-value rejection contradicts `Settings.Normalize()` coercion | `internal/config/settings.go:55-67` Normalize silently coerces | Add a pre-Normalize validation `Settings.Validate() error` OR validate-against-allow-list in `cmd_config.go` before calling `SaveSettings`. |
| F7 | LOW | 3 | `--theme` override leak when user opens Settings screen mid-test | Plan says theme override "not persisted", but if user enters Settings + changes theme, persisted value will override the in-memory --theme | Document as known UX limitation OR disable Settings nav for `run --theme` invocations. |
| F8 | HIGH | 5 | Signal handler installed AFTER `term.MakeRaw` — disagrees with researcher-02 | phase-05 skeleton lines 56-65 → MakeRaw then Notify; researcher-02 §2 → "Install signal handlers BEFORE entering raw mode" | Reorder skeleton to Notify → MakeRaw → defer Restore. Tiny race window otherwise. |
| F9 | MEDIUM | 5 | `os.Exit` from inside notui/* bypasses `defer restore()` | Plan mentions grep guard but doesn't enforce via golangci-lint/CI | Add `make` target or CI grep that fails if `os.Exit` appears under `internal/cli/notui/`. Document. |
| F10 | LOW | 5 | `typeburn run --no-tui < /dev/tty` semantics undocumented | Plan asserts "stdin is not a TTY → exit 1" but doesn't cover the redirected-from-tty case | Either accept (it IS a TTY post-redirect) or document. Probably harmless. |
| F11 | HIGH | 2 | fang `@latest` in Phase 2 step 1; pin-strategy only in risk row | phase-02 step 1: `go get github.com/charmbracelet/fang@latest` | Replace with `@<tag>` (current is roughly v0.x). Document in CONTRIBUTING.md the upgrade gate. |
| F12 | MEDIUM | 6 | `schema_version: 1` lock with no migration policy | phase-06 risk row says "future versions add fields and validate" but no policy doc | Add a CONTRIBUTING.md section: "Adding a Keystroke field is non-breaking IFF JSON tag is added AND the field is omitempty-compatible. Anything else bumps schema_version." |
| F13 | MEDIUM | 6 | Adding JSON tags to `Keystroke` is back-compat for *external* callers, but `engine_test.go` and `metrics_test.go` may construct literals positionally | grep shows `Keystroke` is constructed only via `Apply`/`Backspace` paths in `internal/typing/engine.go`; tests in `internal/metrics/` use named fields | Verified false positive: no positional literals found. Plan's claim is correct. |
| F14 | LOW | 7 | macOS CI cost concern | `.github/workflows/ci.yml:16` ALREADY has `macos-latest` in matrix on every push/PR | Plan claims "extend: macOS-13 runner" but it's already there. Phase 7 should drop the "extend matrix" language. |
| F15 | MEDIUM | 7 | 8MB binary cap may not survive cobra (3.4MB) + fang + x/term + new code | researcher-01:248 cites cobra at +900KB stripped (over current); the 8MB cap = 5.5 + 2.5 tolerance | Tolerance is tight. Pre-Phase-7 spike to measure actual size with empty subcommands. If >7MB, raise cap to 10MB *deliberately* before merging. |
| F16 | LOW | All | Big-bang single PR for 7 phases = ~33h effort | Phase efforts: 4+6+4+5+8+3+3 = 33h | Acceptable per user direction; just flag that PR review of ~3-4kLOC across 7 phases will be heavy. |
| F17 | LOW | 2 | Dep-policy reversal user-confirmation trail | brainstorm-summary.md:39,112 explicitly says "user accepted" + "explicit policy break, user-approved" | Verified: not silent. Trail is documented. Non-issue per `.claude/rules/review-audit-self-decision.md` rule 3. |
| F18 | LOW | 5 | SIGKILL / parent-death / terminal disconnect = unrecoverable | Phase 5 covers SIGINT/SIGTERM/SIGHUP/SIGQUIT + panic + defer; SIGKILL is intrinsically uncatchable | Document explicitly in `cli-reference.md`: SIGKILL leaves terminal in raw mode; user remedy is `reset` or `stty sane`. This is OS-level; plan correctly does not attempt to handle. |
| F19 | LOW | 3 | `--text -` semantics: TUI reads stdin via `codetext.Load("-")` THEN enters Typing | If stdin is the keystroke source AND `--text -` consumed stdin, where do keystrokes come from? | TUI uses tea.NewProgram which reads from `/dev/tty` by default — fine. But document. |
| F20 | MEDIUM | 3 | `app.NewFromCLI` proposal at phase-03 §"Architecture" + Risk row contradicts itself | Architecture says "introduce `app.NewFromCLI`"; Risk row says "Implement as a thin wrapper that calls `app.New` then transitions via `StartTestMsg`" | Two different impls. Pick one. Recommend the wrapper-via-StartTestMsg approach (less surface). |

---

## Per-Finding Detail

### F2 (HIGH) — `--version` vs `version --bogus` asymmetry

**Evidence:** `phase-02-cobra-fang-skeleton-...md:75-83` lists tests for:
- `[]string{"anything-unknown"}` → TUI (RunE returns nil)
- `[]string{"version", "--json"}` → JSON
- `[]string{"version", "--bogus"}` → error

But NO test for:
- `[]string{"--bogus"}` (a root-level unknown flag — fall through? or error?)
- `[]string{"--text"}` (no value — should be parse error or fall through?)

**Impact:** Acceptance criterion #9 says "`typeburn run --bogus` exits 1" — subcommand strictness — but does NOT pin down behavior of root-level unknown flags. Per `decide()` purity (main.go:32-41), `fs.Parse` returns error → return `false, ""` → TUI launches. So `typeburn --bogus` should still launch TUI. Plan needs to make this explicit and add a regression test.

**Recommendation:** Add to phase-02 step 7: `[]string{"--bogus-root-flag"}` → RunE returns nil (TUI fall-through).

### F3 (LOW-MEDIUM) — `-h` handling under `DisableFlagParsing`

**Evidence:** Cobra normally auto-handles `-h`/`--help`. With `DisableFlagParsing: true` on root, the help flag is NOT parsed → falls into `args`. RunE → `decide(args)` → `fs.Parse` returns error (`-h` is undefined) → fall-through to TUI. **This breaks acceptance #1.**

But: fang may intercept `-h` BEFORE RunE via its own wrapping. Researcher-01 didn't verify this specific interaction.

**Recommendation:** Add explicit check in phase-02 step 7: assert `typeburn -h` exits 0 with help text. If broken under DisableFlagParsing, add `if len(args)>0 && (args[0]=="-h"||args[0]=="--help") { cmd.Help(); return nil }` shim in RunE.

### F5 (HIGH) — Phase 1 missing `restartSame` extraction

**Evidence:** `/Users/vchun/Codes/My-projects/Typeburn/internal/ui/screen_typing_actions.go:13-21`:
```go
func (m TypingModel) restartSame() TypingModel {
    wordTarget := m.length
    if m.mode == config.ModeTime {
        wordTarget = m.length * 1000
    }
    m.eng = typing.New(m.target, m.mode, wordTarget)
    ...
}
```

The `wordTarget` math is duplicated. Phase 1 plan only mentions `newTypingWithSeed`. If runner.NewSession changes the math later (e.g. unit overflow guard), restartSame silently diverges.

**Recommendation:** Add to Phase 1 Related Code Files: `internal/ui/screen_typing_actions.go` — `restartSame` must call a runner helper `RebuildEngine(target, mode, length) *typing.Engine` (or equivalent), eliminating the duplicate math.

### F6 (MEDIUM) — `config set` validation contradicts Normalize coercion

**Evidence:** `/Users/vchun/Codes/My-projects/Typeburn/internal/config/settings.go:53-67` — Normalize silently coerces unknown values. phase-04 §Architecture says: "After `Set`, call `s.Normalize()` then `storage.SaveSettings(s)`. If `Normalize` changes the value … warn on stderr but exit 0." but Requirements section says "invalid value exits 1 with a message listing valid options."

These two statements in the same phase doc are directly contradictory.

**Recommendation:** Pick: (a) Add explicit allow-list validation in `cmd_config.go:set` before Normalize → exit 1 on unknown — preferred. (b) Warn-and-coerce — keep current Normalize behavior. Resolve in plan text before implementation.

### F8 (HIGH) — Phase 5 signal handler ordering

**Evidence:**
- `phase-05-...md:56-65` skeleton: `term.MakeRaw` (line 56) → `signal.Notify` (line 63).
- `research/researcher-02-...md:84,381,445`: "Install signal handlers BEFORE entering raw mode."

**Impact:** Tiny race window between MakeRaw returning and Notify completing. If SIGINT arrives in that window, default Go handler runs (which calls `os.Exit(2)` for SIGINT? no — SIGINT default terminates the process WITHOUT running defers). Terminal stays in raw mode.

**Recommendation:** Reorder skeleton in plan text:
```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, ...)
defer signal.Stop(sigCh)

old, err := term.MakeRaw(fd)
if err != nil { return err }
defer restore()
```

### F11 (HIGH) — fang pin strategy

**Evidence:** `phase-02-...md:70` — `go get github.com/spf13/cobra@latest github.com/charmbracelet/fang@latest`. Risk row at line 102-103 says "Pin exact version via `go get …@<sha>` and document in CONTRIBUTING."

**Impact:** Step 1 as-written pulls `@latest` → reproducibility lost. Next `go mod tidy` could pull a breaking version of fang (pre-1.0).

**Recommendation:** Replace step 1 with explicit version pins. Plan should require the implementer to choose a SHA/tag before running the command. Add a CONTRIBUTING.md update line in Phase 7 about the upgrade gate (deliberate quarterly review).

### F14 (LOW) — macOS already in CI matrix

**Evidence:** `.github/workflows/ci.yml:16` — `os: [ubuntu-latest, macos-latest]` already present.

**Impact:** phase-07 §Implementation Steps step 4 says "extend `.github/workflows/ci.yml` to add macOS-13 runner in matrix" — but matrix already includes `macos-latest`. Either the plan wants a *specific* `macos-13` pin (which is more cost-stable than `-latest`) or this step is a no-op.

**Recommendation:** Clarify whether the plan needs `macos-13` (pinned) vs the existing `macos-latest`. If pin: justify in plan text. If no-op: drop the step.

### F15 (MEDIUM) — Binary size cap risk

**Evidence:** Current binary ~5.5MB (CLAUDE.md context). Plan asserts `size-check` limit at 8MB. researcher-01:244-248 estimates cobra alone +900KB stripped; fang adds Lip Gloss styling re-export but Lip Gloss already in the binary. x/term is small (~50KB).

**Math:** 5.5 + 0.9 (cobra) + ~0.3 (fang glue) + ~0.05 (x/term) + ~0.5 (new internal/cli + notui + output code) ≈ 7.25MB. Within 8MB but tight.

**Recommendation:** Run a measurement spike before Phase 7 cements the 8MB limit. If actual is >7.5MB, raise to 10MB *with rationale* committed alongside.

### F20 (MEDIUM) — `app.NewFromCLI` self-contradiction

**Evidence:** phase-03 lines 64-67 says "introduce `app.NewFromCLI(theme, settings, codeText, mode, length, ql) Model`" as a NEW constructor. Risk row at line 96-97 says "Implement as a thin wrapper that calls `app.New` then transitions via `StartTestMsg` programmatically."

These describe two different impls (a new constructor vs. a wrapper using existing constructor + injected message).

**Recommendation:** Pick the message-injection approach (smaller surface, leverages existing Elm flow). Update Architecture section to match Risk row.

---

## False Positives Found

- **F13** — engine.go:Keystroke JSON tag addition. Verified by grep: all callers in `internal/metrics/` and `internal/ui/` use named fields or pass `[]typing.Keystroke` slices opaquely. No positional struct literals. Plan's claim is correct.
- **F17** — Dep-policy reversal not silent. Verified at `brainstorm-summary.md:39,112,118` — explicit user-approval trail. Per review-audit rule 3, this is a properly surfaced reversal, not drift.
- **F18** — SIGKILL/disconnect = unrecoverable. Correctly out-of-scope; OS limitation, not plan defect.

---

## Verdict

**APPROVE_WITH_REVISIONS** — the plan is fundamentally sound and the research is thorough, but it must address F2, F5, F8, F11 before code is written, and clarify F6, F15, F20. The HIGH issues are all plan-text errors (contradictions, missing extraction targets, dep-pin oversight), not architectural flaws.

After revisions, this is a confident **APPROVE**.

---

## Open Questions

1. Does fang (with `DisableFlagParsing: true` on root) still intercept `-h`/`--help` before RunE? If not, manual shim needed (F3).
2. Is the `8MB` size cap a hard product requirement or a guideline? Affects F15.
3. For `config set theme zzz`: should the CLI behave differently from the TUI Settings screen (which uses Normalize silently)? Affects F6.
4. Does `macos-latest` vs `macos-13` matter for cost/stability? Affects F14.

---

**Status:** DONE
**Summary:** Plan is sound but has 4 HIGH plan-text defects (asymmetric flag handling, missing extraction target in Phase 1 `restartSame`, signal-handler ordering vs research, fang `@latest` instead of pinned). Verdict: APPROVE_WITH_REVISIONS. Two findings verified as false positives via grep (Keystroke JSON tags, dep-policy approval trail).
