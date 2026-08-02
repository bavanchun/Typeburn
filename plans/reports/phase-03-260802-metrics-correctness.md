# Phase 3 — Metrics And Typing Correctness

Status: **completed**. Gate green (`gofmt -l .` empty, `go vet ./...`,
`go test ./... -race -count=1`, `make lint`, `make build`).

Worktree: `/Users/vchun/Codes/My-projects/Typeburn/.claude/worktrees/agent-a31a73c5a499c69ab`
Branch: `fix/metrics-and-typing-correctness`
Commits: `f7ce7907` (typing), `2a2b8d2f` (metrics), `be8d3fb9` (app/storage)

## Per-defect before/after

All numbers measured by running the phase's exact reproduction inputs against
the tree before and after the change.

| Defect | Input | Before | After |
|---|---|---|---|
| AFK burst extrapolated | Time/30, 3 keys @ t=1000/1050/1100 | `Duration=100 Net=360.0 Raw=360.0` | `Duration=100 Net=0 Raw=0 AFKTrimmed=true` |
| AFK burst extrapolated | Time/30, 5 keys @ 1ms spacing | `Duration=4 Net=15000 Raw=15000` | `Duration=4 Net=0 Raw=0 AFKTrimmed=true` |
| Zero duration fabricates 100% | target `hello world`, one wrong `q` @1000, end 16000 | `Duration=0 Acc=100.0 KAcc=100.0` | `Duration=0 Acc=0.0 KAcc=0.0 Incorrect=1` |
| Strict replay desync | strict `abcdef`: `abc`, `z`×5 blocked, bksp×3, retype `abcdef` (engine buffer = `abcdef`) | `Correct=9 Incorrect=2 Acc=81.82 Net=63.53` | `Correct=6 Incorrect=0 Acc=100.00 Net=42.35` |
| Strict `Errors` (user decision: keystroke-level) | same run | `Errors=2` | `Errors=5` (every refused key) |
| Strict past-end guard | strict `ab`, type `ab` then `z`×10 | buffer `abzzzzzzzzzz`, `Extra=10` | buffer `ab`, `Extra=0`, 10 refused |
| `Progress()` Time | 30s engine | `0 / 30000` (ms as a word count) | `0 / 3` (words in the target) |
| `Progress()` Code | `x = 1` | `0 / 0` (zero denominator) | `0 / 5` runes |
| AFK persistence | abandoned Time/30 run | written to history, becomes bucket best | not written, not ranked, notice shown |
| Notice placement | Result frame at 80×24 (29 rows) | notice on row 28 — clipped, never visible | notice on row 23 — last row the terminal shows |

`KeystrokeAccuracy` for the strict desync run: 9 of 14 keys = 64.29 % (unchanged
— it always counted keystrokes; only the buffer-derived counts were wrong).

## Consistency worked example (CHANGELOG evidence)

Partial final second no longer sampled. Every number below is one typist typing
at an entirely even pace; the only difference between rows is where the clock
stopped.

| Run | Buckets | Before | After |
|---|---|---|---|
| even 5 c/s, 3.0 s (ends on a boundary) | 3 | 76.16 | 76.16 |
| even 5 c/s, **3.2 s** (the plan's example) | 4 | **60.08** | **76.16** |
| Words-25, even 5 c/s, 26.2 s | 27 | 70.85 | 76.16 |
| Words-25, even 5 c/s, 26.8 s | 27 | 76.16 | 76.16 |
| Words-10, even 6 c/s, 8.5 s | 9 | 67.60 | 73.80 |
| Time-30, even 5 c/s (ends on a boundary) | 30 | 76.16 | 76.16 |

76.16 is the formula's maximum (`100·tanh(1)`).

Direction of the change: **scores go up, never down**, and only for runs that
did not end on a second boundary. Time mode is largely unaffected because its
duration is a whole number of seconds. Words/Quote/Code runs end whenever the
last character lands, so most of them shift. The plan's "51.31" could not be
reproduced from an even 5 c/s 3.2 s log; the measured before-value is 60.08.
Stored history is untouched, so old records keep their old numbers.

## The `typedFromLog` duplicate

Collapsed, not fixed twice. The replay rule now lives once in
`internal/typing/replay.go`:

- `typing.ReplayBuffer(log) []Keystroke` — surviving forward keystrokes in
  buffer order, carrying target and correctness.
- `typing.TypedFromLog(log) []rune` — the rune view.

`metrics.replayFinalState` and `ui.typedFromLog` both delegate. `typedFromLog`
is kept as a one-line delegate rather than deleted so the three existing call
sites in files owned by Phase 2 stay untouched.

## `engine.go` split

197 LOC → four files, all well under the ceiling:

| File | LOC | Holds |
|---|---|---|
| `engine.go` | 156 | `Keystroke`, `Engine`, constructors, `Apply`, `Backspace`, `Log`, `Typed`, `ForwardKeystrokes` |
| `engine_states.go` | 42 | `States()` |
| `engine_progress.go` | 34 | `Progress()` |
| `replay.go` | 44 | `ReplayBuffer`, `TypedFromLog` |

Largest file touched anywhere in the phase: `internal/metrics/compute.go` at 181.
No file at or over 200. (`internal/metrics/compute_test.go` sits at exactly 200
and was not touched.)

Also renamed `Engine.wordTarget` → `Engine.limit` (and `runner.wordTarget` →
`engineLimit`). The field means a word count in one mode and a millisecond
deadline in another; reading it as the wrong one is the whole `Progress()` bug,
and the old name asserted the wrong one.

## Mutations run (each assertion proved falsifiable)

Twelve production-code mutations, each reverted after measurement. Every one was
caught:

| # | Mutation | Caught by |
|---|---|---|
| 1 | `ReplayBuffer` stops skipping refused keystrokes | `TestReplayFromLog_MatchesEngineBuffer` (3 subtests), `TestCompute_StrictAgreesWithTheEngineBuffer`, `TestCompute_StrictErrorsAreKeystrokeLevel`, `TestTypedFromLog_AgreesWithTheEngineUnderStrict` |
| 2 | strict guard restored to `pos < len(target)` | `TestStrict_RefusesKeysPastTheEndOfTheTarget` |
| 3 | engine stops setting `Blocked` | `TestReplayFromLog_MatchesEngineBuffer`, `TestStrict_Refuses…`, `TestCompute_StrictAgreesWithTheEngineBuffer` |
| 4 | zero duration returns `{Accuracy: 100, KeystrokeAccuracy: 100}` again | `TestCompute_ZeroDurationReportsWhatWasTyped` |
| 5 | rate floor lowered back to `durationMs > 0` | `TestCompute_NeverReportsAnImpossibleResult` (the ratchet), `TestCompute_AFKTrimIsReportedAndNoRateIsExtrapolated` |
| 6 | `TrimAFK` reports `false` when it trimmed | `TestAFKTrim` (2 subtests), `TestCompute_AFKTrim…`, **and all three `internal/app` seam tests** |
| 7 | `Errors` back to final-state only | `TestCompute_StrictErrorsAreKeystrokeLevel` |
| 8 | partial final second sampled again | `TestConsistency_EvenTypingScoresTheMaximum…` (3 subtests) |
| 9 | consistency stops sampling entirely | same test (all 4 subtests) — proves the fix did not just stop measuring |
| 10 | `Progress()` Time branch reverted, Code back in `default` | `TestProgress_IsACountInEveryMode` (2 subtests) |
| 11 | notice written to `lines[len(lines)-1]` unconditionally | `TestOverlayNotice_LandsOnARowTheTerminalShows/frame_taller_than_the_terminal`, `TestOverlayNotice_DoesNotOverwriteContent` |
| 12 | AFK gate removed from `decideOutcome` | all three `TestResult_*` seam tests |

Mutation 6 is the one that matters for R1: a metrics-package change is caught by
an `internal/app` test. That is the cross-package assertion that did not exist.

## Ratchet

`knownImplausible` is now `map[string]bool{}`. `assertPlausible` passes for every
case with zero allowlist entries, and the stale-entry sweep still runs.

## Deviations from the phase spec, and why

1. **`EligibleForBest` does not gain an AFK branch.** `storage.Record` has no
   AFK field and `internal/storage/history_record.go` is outside this phase's
   ownership. Since an AFK-trimmed run is never written, no stored record could
   ever carry the flag — the gate has to be at the writer. It is in
   `app.decideOutcome`, and `EligibleForBest`'s doc points at it. The three
   `internal/app` seam tests enforce it; a doc comment alone would not.
2. **The notice never lengthens a frame that already overflows.** The spec says
   "overlay at `min(len(lines)-1, m.h-1)`, require that row to be blank, else
   insert". Implemented as written, with one exception: when the frame is
   already taller than the terminal, the notice overlays instead of inserting.
   Inserting there pushes a row off the bottom regardless, and it would have
   changed `result/persist-notice@*` in Phase 1's `knownAppOverflow` from
   `Lines: 29` to `Lines: 30` — a file this phase must not touch, in a test that
   asserts the measurement exactly. With the exception, every `knownAppOverflow`
   measurement is byte-identical and the insert path becomes live for free once
   Phase 6 brings Result inside 24 rows.
3. **The h≥30 "footer destruction" in the phase brief does not occur.** Measured:
   at 80×30 and 80×50 the Result frame is exactly `h` rows and its last row is
   blank padding (the footer sits at `h-2`). Behaviour at those sizes is
   unchanged. The real defect was the h≤24 clipping, which is fixed.
4. **AFK rate suppression uses the existing 500 ms floor**, not a new constant.
   `LiveWPM` already refuses to project a rate from under 500 ms and documents
   why. Reusing it means the live header and the final result agree on when
   there is nothing to report, and it is a threshold the product already chose
   rather than one invented here. This is what makes the ratchet green; the
   user's "withhold, do not rescale" decision is honoured — nothing is scaled,
   the rate is simply not claimed.
5. **`internal/ui/mode_header.go` gains `ModeCode`** to the Quote branch. Code
   mode had no branch at all, so its progress area rendered empty; now that
   `Progress()` returns a real denominator for it, the bar works.
6. **`internal/cli/notui/runner.go`** needed no logic change — its call site is
   correct now that `Progress()` is. Comment added recording the unit.

## Draft CHANGELOG copy

```markdown
### Changed

- **Consistency is no longer dragged down by where the clock stopped.** The
  per-second breakdown scaled a partial final second as though it were a whole
  one, so a run ending part-way through a second reported that second at a
  fraction of the pace actually being held. Fed to a variance measure, the
  invented dip was indistinguishable from erratic typing. An entirely even
  typist over 3.2 s scored **60.08**; the same run now scores **76.16**, the
  formula's maximum. Scores go up, never down, and only for runs that did not
  end on a second boundary — Time mode is largely unaffected. The partial second
  is still drawn on the result graph; it is simply not treated as a sample of a
  per-second rate. Stored history is not migrated, so past results keep the
  numbers they were recorded with.

- **Strict mode counts every wrong key.** A strict run's cursor never passes an
  error, so almost nothing wrong survives to the end of the test and the error
  count was reporting a run full of mistakes as near-flawless. Errors are now
  counted per keystroke, including ones that were corrected afterwards. A strict
  run will report a higher error count than the same run did before. Non-strict
  runs are unchanged.

### Fixed

- **A test you walked away from is no longer saved.** When a timed test ended in
  a long idle stretch, the clock was trimmed back to the last key pressed and the
  speed was then projected from whatever burst remained — three keystrokes in a
  tenth of a second reported 360 wpm, and it was written to history as a personal
  best nothing could beat. Such runs are now shown but not recorded, and the
  result screen says why rather than leaving the run to vanish.

- **A test with no measurable duration no longer reports 100% accuracy.** A
  single wrong keystroke could score a flawless run and, being the first record
  in its bucket, become a permanent best.

- **Strict mode results match what you typed.** Keys the mode refused were being
  replayed as though they had been accepted, so a strict run that ended matching
  the target exactly was scored with wrong characters in it. Strict mode also now
  refuses keys typed past the end of the target.

- **Progress counts the right thing in every mode.** A timed test reported its
  progress against a millisecond deadline instead of a word count, and a Code
  test had no total at all.

- **The notification about a result that could not be saved is now visible.** It
  was written to a row that gets clipped whenever the screen is short — which is
  exactly when there is something to report.
```

## Unresolved questions

1. **Consistency for runs under one second is now `0`, not ~76.** No complete
   second means no sample, and `Consistency` already documents `0` as "no data".
   Unreachable in practice (Words-10 and Quote both take seconds; Time is ≥15 s),
   but it is a behaviour change and nobody signed off on it explicitly.
2. **`persistErr` is now also the AFK-notice channel.** The field name and its
   comment in `internal/app/model.go` (outside this phase's ownership) still say
   it is for write failures. It should be renamed to something like `notice`, and
   `ui.PersistenceNotice` with it. Whoever owns `model.go` next should do that.
3. **Phase 4 still owns width-bounding the notice.** `overlayNotice` centres with
   `lipgloss.PlaceHorizontal(w, …)` and does not truncate, so the AFK copy
   (`⚠ paused mid-test — not saved to history  ·  press any key to dismiss`,
   68 cells) will overflow a 60-column terminal exactly as the existing copy
   does. Not made worse; not fixed here.
4. **The AFK notice costs Result a row it does not have at h=24.** Phase 6's
   budget gate needs to account for it — flagged in the phase brief, restated
   here because the notice is now reachable on the ordinary result path, not just
   on a disk failure.
5. **Phase 2 must adapt `screen_typing_view.go:25`.** `Progress()`'s contract has
   changed and that call site was deliberately not touched. `ModeHeader` ignores
   `done/total` for Time mode, so nothing breaks today, but Words/Quote/Code now
   receive different-meaning values for Code (runes, previously `words / 0`).
6. **Plan status not updated.** The phase file's `status:` frontmatter is plan
   state; left for the orchestrator per the repo's plan-state rule.
