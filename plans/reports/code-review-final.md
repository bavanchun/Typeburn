# Final Code Review — monkeytype-tui

Date: 2026-05-18. Reviewer: code-reviewer (adversarial pass).
Scope: `internal/` + `main.go`. `go test ./... -race -count=1` GREEN, `go vet ./...` clean.

## Verdict: SHIP (with 2 MAJOR follow-ups recommended pre-1.0)

Codebase is solid. Metric formulas match researcher-02. State machine, persistence,
rune handling correct. No CRITICAL defects. Findings below are real but non-blocking.

## Counts
- CRITICAL: 0
- MAJOR: 2
- MINOR: 5

## MAJOR

### M1 — Live timer stops if a tick is dropped / not re-armed on every path
`internal/ui/screen_typing.go:130-155 handleKey`, `screen_typing_actions.go:13-21 restartSame`.
The 100ms tick self-reschedules only via `tickCmd()` returned from `handleTick`
(`screen_typing.go:126`) and on first keystroke (`applyText:174`). `restartSame()`
resets `startMs=0` and returns `(m, nil)` — NO tick re-armed. After a `tab` restart
the timer loop is dead until the user types the first char again (then `applyText`
re-arms). For ModeTime this is correct-ish (clock starts on first key), BUT the
header WPM/elapsed display is frozen between restart and first keystroke, and if the
user pastes (PasteMsg → `applyText`) the first-key branch re-arms, so OK there.
Real risk: `handleKey` backspace branch and any non-printable key return `(m,nil)`
mid-test; they do not re-arm. The tick is only kept alive by the *previous*
tickMsg's `tickCmd()`, so a single dropped/By-design-consumed tick is fine — Bubble
Tea guarantees the cmd runs. Verified: loop is self-sustaining once started. The
genuine gap: **after `restartSame` on a ModeTime test, time-completion can never
fire if the user never types** (expected) but also the live header never updates
even though design shows a running timer. Fix: `restartSame()` should return
`tickCmd()` like `newTest()` does (`screen_typing_actions.go:25` returns tick;
restartSame at line 20 returns bare `m`). One-line inconsistency.
Fix: `return m.restartSame(), tickCmd()` in `screen_typing.go:137`.

### M2 — New-best comparison uses rounded int WPM, loses sub-WPM precision
`internal/app/model_history.go:24` stores `WPM: int(math.Round(NetWPM))`;
`internal/storage/new_best.go IsNewBest` compares `Record.WPM int`.
Spec (phase-08:24,48) says new-best = "highest **NetWPM**". Two runs of 75.4 and
75.0 NetWPM both round to 75 → faster run is NOT flagged new-best (`r.WPM > best`
is strict). Also a 75.6 then 75.4 shows false "not best" inversion masked.
Impact: occasional missed/incorrect ★ badge; history WPM column loses precision
permanently (raw/acc/consistency stored as float64, only WPM truncated).
Fix: add `NetWPM float64` to `storage.Record`, compare that in `IsNewBest`;
keep int only for display. Backward-compat: old JSON lacks field → unmarshals 0,
acceptable (history is local, non-critical).

## MINOR

- **m3** `internal/storage/atomic_write.go`: file is fsync'd + renamed, but the
  **parent directory is not fsync'd** after rename. On power loss the rename may
  not be durable. For a local typing-stats file this is acceptable; document the
  trade-off or add a dir-fsync for completeness.
- **m4** `internal/metrics/compute.go:78` `MissedChars` hardcoded 0 with comment
  acknowledging target unavailable. Result struct advertises the field; callers
  may assume it's meaningful. Either remove the field or pass target to Compute.
- **m5** `internal/ui/word_stream_renderer.go:120-126`: `utf8.RuneLen` computed
  then discarded (`cellW := 1` always). Wide CJK runes (width 2) overflow narrow
  (60-col) terminals by 1 cell per wide rune. Comment says deferred to Phase 9;
  Phase 9 shipped without it. Latin/ASCII (the embedded wordlist) unaffected →
  low real risk. Use `lipgloss.Width`/`go-runewidth` if CJK quotes ever added.
- **m6** `internal/typing/completion.go:39-77 countCompletedWords`: the
  mid-sequence "word complete needs typed[i] (space) to exist" rule means an
  artificial target with MORE words than `wordTarget` would require typing into
  the next word to finish. Verified NOT reachable in production: generator
  returns exactly `wordTarget` words and the last-word branch (no trailing space
  needed) handles real cases correctly. Defensive note only.
- **m7** `internal/app/model.go:64-90`: message-type checks via sequential
  `if _,ok` before the `switch`. Functionally fine; `StartTestMsg` does not
  re-arm a tick (typing screen arms on first key — correct, verified).

## Verified Correct (adversarial checks that passed)
- NetWPM/RawWPM/Accuracy/CPS formulas == researcher-02 §1-§5. Divide-by-zero
  guarded (`durationMs<=0`, `correct+incorrect==0`, empty log → Accuracy=100).
- Accuracy uses FINAL replayed state; corrected errors do not penalise (replayFinalState pops on backspace). Verified.
- Consistency = 100*tanh(1-CV), population stddev, mean==0 & empty → 0, clamped
  [0,100]. Matches researcher-02 worked example (≈74 for the sample). NaN/Inf safe.
- Per-second bucketing half-open [Ns,(N+1)s), boundary keystroke→bucket N,
  backspaces excluded, AFK idle tail correctly excluded because bucket count is
  sized from max keystroke offset not endMs (probed: 4000ms duration, 5 buckets,
  no phantom AFK buckets). No off-by-one.
- AFK trim: ModeTime only, strictly >7s, Words/Quote never trimmed, empty log
  safe. Matches spec.
- State machine: ctrl+c hard-quits everywhere incl. quit-prompt (key_handler:17
  before prompt routing). esc-on-Home → quit-prompt, "no" default. Sub-model
  stored back on every delegate path (`m.typing, cmd = ...`). Engine is a
  pointer in value-struct → mutations survive Bubble Tea value-copy (verified
  via full words-test smoke completing to ScreenResult).
- Storage: atomic tmp+fsync+close+rename, 0600 file / 0700 dir, MkdirAll before
  write, corrupt/missing JSON → nil slice never panics, cap-200 keeps NEWEST
  (sort asc then `records[len-200:]`), IsNewBest scoped by mode+length bucket,
  first-ever → true (reasonable), pure/no-mutation. XDG→HOME fallback present.
- Degraded mode: single chokepoint in `app/model_view.go:20` BEFORE any screen
  delegation; `<60||<20` → centered notice only, no partial paint.
- Spec: exactly 4 settings rows (Theme/DefaultMode/DefaultLength/BlinkCursor) —
  no error-mode/sound/smooth/restart-flash leaked. Keymap matches design §8
  exactly. allow-continue+backspace only (no stop-on-error state). NO_COLOR/mono
  theme present.
- No goroutine/cmd leak: tea.Tick is one-shot, re-armed exactly once per tickMsg;
  newTest re-arms, abort/complete stop the loop (no further tickCmd). 0-length /
  instant-complete / paste paths return ScreenResult without panic (smoke-tested).

## Unresolved Questions
1. M2: confirm whether sub-WPM new-best precision matters for v1 or int is an
   accepted product decision (history is local-only, cosmetic ★).
2. m3: is power-loss durability of history.json in scope for v1? (currently
   file-fsync only, no parent-dir fsync).
3. m5: any roadmap intent to support CJK quote packs? If yes, m5 becomes MAJOR.

**Status:** DONE_WITH_CONCERNS
**Summary:** No CRITICAL/crash/data-loss/spec-violation. 2 MAJOR (timer re-arm
inconsistency on restartSame; rounded-int new-best precision), 5 MINOR. Ship-able.
**Concerns/Blockers:** M1/M2 are correctness-adjacent quality issues; recommend
fixing M1 (1-line) before ship, M2 can be fast-follow.
