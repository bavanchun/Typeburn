# Codebase Audit — Typeburn v2.8.1

**Status:** PAUSED mid-audit (user request). 2 of 5 scopes complete.
**Date:** 2026-08-01
**Baseline:** `go test ./... -race -count=1` GREEN, 12.4s, all 18 packages. Every defect below passes CI.
**Tree state:** clean. All reviewer probe files moved to scratchpad `killed-agent-probes/`.

---

## Scope status

| # | Scope | Status |
|---|---|---|
| 1 | Pure logic (typing, metrics, words, codetext, storage, version) | DONE |
| 2 | CLI + self-updater (internal/cli, internal/update) | DONE |
| 3 | UI layer (internal/ui, internal/app, theme, config) | **KILLED mid-work** |
| 4 | Repo hygiene / CI / release / docs | **KILLED mid-work** |
| 5 | Product UX + Result redesign | **KILLED mid-work** |

Partial signal from killed agents (unverified, worth chasing on resume):
- Scope 3 last words: "Both the Typing and Result screens overflow a standard 80x24 terminal." — NOT verified.
- Scope 5 was mid-way through building Result layout mockups at exact widths.
- Scope 4 was re-running metrics against a pristine `git archive` export of HEAD.

---

## CRITICAL — verified by me personally, not just by the subagent

### A0. The hero WPM number displays every zero as a "2"

`internal/ui/ascii_big_digits.go:12-93` → reaches the user at `internal/ui/screen_result_hero.go:28`

`digitGlyphs[0]` is **byte-identical to `digitGlyphs[2]`** — the zero slot contains the "2" artwork (ANSI Shadow "2": the diagonal sweep). Two independent confirmations: byte-equality check, and visual comparison against the real ANSI Shadow "0" (which has a `██╔═████╗` second row this glyph lacks).

Verified by rendering `BigDigits` through overlay (repo untouched):

```
BigDigits(0)   -> renders "2"
BigDigits(60)  -> renders "62"
BigDigits(90)  -> renders "92"
BigDigits(100) -> renders "122"
```

This is the single most prominent element on the Result screen — the giant number the whole screen is built around — and it shows the **wrong number**. Any score containing a 0 is misreported to the user. A 60 WPM run reads as 62.

**Second defect in the same table:** digits 3, 6, 9 have ragged row widths — row 4 is 9 runes where the other five rows are 8. So the hero block is not rectangular:

```
BigDigits(60) row widths (runes): 24, 24, 24, 25, 25, 25
BigDigits(90) row widths (runes): 24, 24, 24, 25, 24, 24
```

The comment at `:9-11` claims "Each glyph is 6 characters wide (including trailing space) for uniform column joining" — wrong on both counts: glyphs are 8 runes (4 for "1"), and they are not uniform.

**Why nothing caught it:** the 8 golden snapshot files recorded the defective output as the baseline. Golden tests confirm "unchanged", never "correct". Worth noting the v2.8.0 PR screenshot showed a 74 WPM run — no zero, so it looked fine.

**Fix:** replace `digitGlyphs[0]` with the real ANSI Shadow zero; pad rows of 3/6/9 to a uniform 8. Add a test asserting all glyphs are rectangular and that no two digit glyphs are equal. Regenerate goldens and READ the diff.

### A1. AFK trim divides by the burst window → absurd WPM written to history as a permanent personal best

`internal/metrics/afk_trim.go:41-43` + `internal/metrics/compute.go:49-54`

`TrimAFK` moves `endMs` back to the last forward keystroke. Nothing floors the resulting duration. `netWPM = chars/5/minutes` is then computed over the *burst*, not the test.

Reproduced independently (overlay test, repo untouched):

```
30s Time test, 3 keys at t=1000/1050/1100 then user walks away:
  DurationMs=100  NetWPM=360.0  RawWPM=360.0  Consistency=76.2
```

Scales without bound — 5 keys at 1ms spacing → `DurationMs=4, NetWPM=3000, RawWPM=15000`.

**Full persistence chain verified by me:**
- `internal/app/model_history.go:49` — `AppendHistory(rec)` called unconditionally. No minimum-keystroke gate, no minimum-duration gate.
- `internal/storage/new_best.go:6` — `EligibleForBest` excludes only `code` mode and `strict`. A Time-mode AFK burst **is** eligible.
- Result: 360 WPM lands in `history.json` permanently and suppresses every genuine future best in the `time/30` bucket.

**Fix needs a product decision (do NOT let an agent pick silently):**
- (a) floor the trim — if `lastKeyMs - startMs` below a minimum, return untrimmed `endMs`
- (b) Monkeytype behaviour — mark AFK-trimmed runs ineligible for best (`EligibleForBest` is the existing seam)
- (c) gate persistence on a minimum forward-keystroke count

### A2. Single wrong keystroke reports 100% accuracy and becomes a "best"

`internal/metrics/compute.go:52-57` — after trim, `startMs == endMs`, so `durationMs <= 0` returns `Result{Accuracy: 100, KeystrokeAccuracy: 100}`.

Reproduced independently:
```
target "hello world", Time mode, ONE wrong key 'q' at t=1000, end t=16000:
  DurationMs=0  Accuracy=100.0  KeystrokeAcc=100.0
```
A 0% run recorded as 100% accurate. Via `new_best.go:86-88` (`best := -1.0`) it becomes the first-ever best for the bucket.

---

## HIGH — pure logic (subagent-verified, I did not re-verify these)

### A3. Strict mode desyncs `replayFinalState` → inflated CorrectChars/NetWPM past target length
`internal/metrics/compute.go:142-166` vs `internal/typing/engine.go:72-81`.
Strict-blocked keystroke is appended to log with `Typed != 0` but NOT applied to `e.typed`. `replayFinalState` only skips `Typed == 0`, so it pushes blocked keystrokes into its reconstructed buffer. Backspaces then pop slots the engine never had. Buffers drift permanently.

Repro: target `"abcdef"` strict, type `abc`, `z`×5 (blocked at pos 3), backspace×3, retype `abcdef`. Engine final text exactly `"abcdef"`. Metrics say `targetRunes=6 CorrectChars=9 IncorrectChars=2 NetWPM=67.50 Accuracy=81.82`. Non-strict same pattern stays correct at `CorrectChars=6`.

Fix: mark blocked keystrokes (e.g. `Blocked bool` on `typing.Keystroke`), skip in `replayFinalState`, still count in `totalTyped`/`correctForward`/`KeyHeatmap`.

### A4. `Result.Accuracy` / `Result.Errors` contradict documented contract in strict mode
Same root cause as A3. `model_history.go:25-27` swaps in `KeystrokeAccuracy` for strict records which masks the Accuracy half — but `Errors`/`IncorrectChars` still wrong and `Errors` remapped nowhere.

### A5. `codetext.Load` reads entire input into memory before any cap; never terminates on endless stream
`internal/codetext/codetext.go:64-69` — `io.ReadAll` to EOF, 10 000-rune cap only checked at `:92`.
- 400 MiB file correctly rejected — after allocating **1274 MiB**.
- `codetext.Load("/dev/zero")` still reading after 3s, never returns. Same for `--text -` on a pipe that never closes.

Fix: `io.ReadAll(io.LimitReader(r, maxRunes*utf8.UTFMax + 1))`, reject when result hits cap.

### A6. `metrics.Compute` allocates per-second buckets from unvalidated timestamp span
`internal/metrics/per_second.go:31-43` — `numBuckets = maxOffsetMs/1000 + 1`, no validation. `internal/cli/cmd_replay.go:44` feeds file-supplied JSON; `loadReplayInput` validates only `schema_version`.
- 2-keystroke log at `TimeMs` 0 and `1<<32` → 4.29M buckets, 197 MiB.
- At `1<<45` the test binary was **OOM-killed by the kernel**.
- Realistic non-adversarial trigger: any log mixing `0` with epoch-ms timestamps → ~1.7e12 ms span → 63 GiB.

### A7. `AppendHistory` unsynchronised read-modify-write; two instances silently lose records
`internal/storage/history_store.go:57-90` (no lock) + `internal/storage/atomic_write.go:15` (`tmp := path + ".tmp"` — fixed name shared by every process).

Two real OS processes, one `XDG_DATA_HOME`, 60 appends each:
```
total persisted=114 (expected 120); per-instance map[1:57 2:57]
```
In-process variant (8 goroutines × 20): **2 of 160** survive.

Fix: `os.CreateTemp(dir, "history-*.json")` for unique temp name + advisory lock across load→append→write.

### A8. One corrupt byte destroys 200 records, no backup, no error
`history_store.go:43-46` returns `nil` on unmarshal failure; `:57-59` appends to that `nil` and overwrites.
```
200 valid records, truncate 3 bytes -> after one append: 1 record on disk; backups found: []
```
Silent, irreversible. Fix: quarantine to `history.json.corrupt-<ts>` before next write; surface via existing `m.persistErr` seam.

---

## HIGH — CLI / self-updater

### B1. Plain `typeburn update` path has NO signal handling; Ctrl-C leaks the lock

`internal/cli/cmd_update_run.go:51-52` + `cmd/typeburn/main.go:14`

**Verified by me:** `main.go:14` calls `fang.Execute(context.Background(), root, fang.WithoutVersion())` — no signal opts, so `fang.go:167` never installs `signal.NotifyContext`. `cmd.Context()` is a bare `context.Background()`. The plain branch calls `Apply` **synchronously**. SIGINT/SIGTERM hit Go's default disposition → process dies instantly mid-`Apply`, zero defers run.

Animated path IS hardened (bubbletea installs handlers → `InterruptMsg` → `!settled` → `stopApply` drains). Plain path is not — that covers pipes, redirects, CI, non-tty, and any terminal narrower than 56 columns.

Repo already knows the pattern: `internal/cli/notui/lifecycle.go:27` does `signal.Notify(SIGINT, SIGTERM, SIGHUP, SIGQUIT)`. Update command is the inconsistent one.

Subagent repro (subprocess + hanging httptest server):
```
interrupt:  lock file leaked, every future update refuses to start
terminated: lock file leaked, every future update refuses to start
```

Fix: wrap plain branch in `signal.NotifyContext`, route through existing `stopApply`/`reportApplyResult`.

### B2. A killed update requires THREE manual file deletions; only one error says so

`internal/update/lock.go:19`, `download.go:87`, `archive.go:42`

Four artifacts each independently and permanently block all future updates via `O_EXCL`, none cleaned up by a later run:

| stale file | error | actionable? |
|---|---|---|
| `.typeburn-update.lock` | `another update is already in progress (remove ... if stale)` | yes |
| `checksums.txt` | `create download file: ... file exists` | path only, no guidance |
| `typeburn_<v>_<os>_<arch>.tar.gz` | `create download file: ... file exists` | path only |
| `typeburn.new` | `create extracted binary: ... file exists` | path only |

Repro — remove only the file each error names:
```
attempt 1: another update is already in progress (remove .../.typeburn-update.lock if stale)
attempt 2: create download file: open .../checksums.txt: file exists
attempt 3: create download file: open .../typeburn_9.9.9_linux_amd64.tar.gz: file exists
attempt 4: SUCCESS
```
Deferred stale-lock risk is **worse than documented**: three files, one hint. After deleting the lock as instructed the user hits a raw `file exists` that reads like a bug. Non-technical user concludes self-update is broken.

Fix (best): PID + start time in lock file; on `IsExist`, if PID dead or lock stale, reclaim AND sweep the three siblings. Closes this and the deferred SIGKILL case together.

---

## MEDIUM

- **A9. Consistency systematically depressed** — `per_second.go:42,71` scale the final partial second as a full second. A *perfectly even* 5 c/s typist over 3.2s scores `Consistency=51.31` (formula max is 76.16). **Verified by me.** Worst on short tests (`words 10`). Fix: drop or prorate final bucket when it covers <1000ms.
- **A10. Control chars / ANSI survive `codetext` normalization** → uncompletable Code targets. `codetext.go:76-98` rejects only invalid UTF-8 and NUL. Accepted with `err=nil`: bare CR, `0x08`, vertical tab, DEL, ANSI CSI `\x1b[31m`, ANSI OSC `\x1b]0;pwned\a`. `\x1b` is bound to Back and `\x08` to Backspace, so those runes can never be produced by `screen_typing_input.go:33` → unwinnable test with no way to finish. Raw OSC/CSI also reaches the renderer. `result_render_helpers.go:130` proves the project already knows ESC needs stripping.
- **A11. Strict mode does not block runes past end of target** — `engine.go:72` guard is `pos < len(e.target)`, so past the end everything is accepted as Extra. Strict target `"ab"`, type `ab` then `z`×10 → `ExtraChars=10, RawWPM=120`.
- **A12. `Engine.Progress()` returns ms time limit as a word total** in Time mode — `engine.go:182-196`, `runner/session.go:53-57` stores `length*1000`. 30s engine → `Progress() = (0, 30000)`. Live in `screen_typing_view.go:25` and `notui/runner.go:117`. `ModeCode` returns `(words, 0)` — zero denominator.
- **B3. `restoreInterruptedUpdate` is dead code**; its doc comment asserts recovery that never happens. `replace_windows.go:39` / `replace_unix.go:38` — no caller anywhere. `go vet` cannot catch it.
- **B4. Attacker-controlled `tag_name` flows unvalidated into a filesystem path** — `download.go:53-56` → `:127,139`. NOT reachable today (the `IsPrerelease` + `Compare` gate incidentally blocks every tag containing a separator — subagent probed all variants). Problem: the protection is entirely accidental. `cache.go:23` already has `validSemverRe` and calls it an "injection guard", but it is applied only on the cache-read path, never to the live `FetchLatest` result. One-line fix: apply it in `check.go` after `FetchLatest`.
- **A13. Stale `history.json.tmp` permanently blocks all saves** — fixed by the A7 `os.CreateTemp` change.

## LOW
- `ExitCode` uses bare type assertion not `errors.As` (`internal/cli/exitcodes.go:52`) — any `%w` wrap collapses documented exit codes 3/5 to 1.
- `printReleaseNotes` writes unsanitised URL incl. ANSI to terminal (`cmd_update.go:149-153`); prefix-only validation at `check.go:51-53`.
- Unwanted archive members decompressed without cap (`archive.go:83-85`) — 1 MB archive → 1 GiB decompressed. DoS-only, post-integrity, requires attacker to already own the release.
- `classifyInstall` substring match (`selfpath.go:28`) — `~/homebrew/bin/` misclassified as managed.
- `config set` silently mutates a second key via `Normalize()` (`cmd_config.go:77`) with no output.
- `words.Words(n)` unbounded (`generator.go:56-65`) — `--words 2000000000` is an OOM.
- `sort.Slice` unstable in `history_store.go:62-64`; records sharing a timestamp reorder every append.
- `atomic_write.go:39` never fsyncs the parent dir after rename — package docstring claims crash-safety.
- Dead code `_ = wordStart` in `completion.go:46,53,65,67`. False comments at `completion.go:39-40`, `generator.go:111-114`.
- `words/for_mode.go:27` — `default:` swallows `ModeCode`, returns random words. Trap for next caller.

---

## Threat model note (self-updater)

Binary protects only its own executable image + local settings/history. No creds, no PII, no network service. BUT it rewrites its own executable from a network download → successful compromise = persistent arbitrary code execution as the user.

- **Network attacker without TLS break** — fully mitigated. HTTPS only, cleartext redirects refused, host allowlist checked on EVERY hop, SHA-256 verified against `checksums.txt` before extraction. All confirmed empirically.
- **Compromised release / TLS-MITM** — not mitigated, and correctly documented as such (`README.md:143-147`, `SECURITY.md`). Deliberate disclosed v1 scope. B4 lives entirely inside this attacker's envelope, so it buys them nothing — flagged as a maintainability landmine, not a breach.
- **Local attacker with write access to install dir** — out of scope by construction. The two TOCTOUs are only reachable by them. Do NOT "fix".

**The findings that matter to a real user are the boring ones: B1 and B2.** Nobody will MITM a typing tester. Plenty of people will press Ctrl-C during an update.

---

## What is genuinely good (verified, not padding)

- Pure-logic layering holds. Zero bubbletea/lipgloss/charm.land imports across all six packages, grep-verified. `anim` routes colour through `image/color` deliberately.
- `codetext` `Load`/`Normalize` genuinely share one `normalize` core — CLAUDE.md's claim is accurate.
- Updater ordering correct: verify strictly before extract, extract strictly before replace. Checksum mismatch leaves temp dir completely empty, never touches target binary.
- Size cap measured against bytes actually read, not `Content-Length`. Chunked 200 MB body with no Content-Length rejected at 52428801.
- No zip-slip. `extractBinary` computes dest independently of archive entry name. File modes come from code (0o755), never the archive — no setuid smuggling. `replaceBinary` uses `Perm()`.
- Redirects checked every hop; loop bound fires at exactly 10.
- AFK threshold boundary exactly as documented — 7000ms does not trim, 7001ms does. No off-by-one.
- The prior review's four updater fixes are real and correct, verified case by case.
- `internal/version.Resolve` precedence correct, cannot panic; the `(devel)` filter is a detail most implementations miss.
- Coverage: `internal/update` 83.3%, `internal/cli` 79.1%, `updateui` 80.4%, `notui` 78.3%, `output` 77.8%.

---

## SCOPE 3 — UI layer (COMPLETE)

### D1. CRITICAL — Typing screen renders the whole word buffer; no viewport outside Code mode. VERIFIED BY ME

`internal/ui/screen_typing_view.go:52-54`, `internal/ui/word_stream_renderer.go:93-137`

Bubble Tea v2 in altscreen draws into a `w×h` cell buffer, so **everything past row `h` is silently dropped** — no scroll, content simply gone.

My own overlay measurement through the real `app.Model.View()`:

| mode | 80×24 | 120×24 | 80×30 |
|---|---|---|---|
| **time30** | 47 lines → **23 clipped** | 36 → **12 clipped** | 46 → **16 clipped** |
| **time15** | 47 lines → **23 clipped** | 36 → **12 clipped** | 47 → **17 clipped** |
| words10 | 24 → fits exactly | 24 → fits exactly | 30 → fits exactly |

`config.Defaults()` is `DefaultMode: ModeTime, DefaultLength: 30` (`settings.go:43-44`). **The default mode on the default terminal clips 23 of 47 rows** — including the entire keybinding footer, permanently.

Root cause: **Code mode has a viewport, the word stream does not.** `screen_typing_view.go:46-51` clamps Code to `m.h - fixedOverhead` and calls `joinViewport`; the `else` branch at :53 calls `renderWordStreamAnim` with **no height argument at all**. Words/Quote fit only because their buffers are naturally short — it is luck, not adaptation.

Not merely cosmetic — **the caret leaves the screen**. Agent measurement:
```
term 80x24 typed=2000 caretRow=27 visibleRows=20  <- CARET OFF SCREEN
term 60x20 typed=1000 caretRow=17 visibleRows=16  <- CARET OFF SCREEN
```
At 60×20 you type blind after ~950 characters — a 60s run at 95 wpm, or any 120s run.

Fix: pass `m.h - fixedOverhead` into `renderWordStreamAnim` and route rows through the existing, already-tested `joinViewport(rows, caretRow, height)`. **The scroll-follow helper is already written; it just isn't wired to the non-Code branch.**

### D2. CRITICAL — Result screen height is content-fixed at 29 rows regardless of terminal
`internal/ui/screen_result_view.go:30-56`. `spacer` is clamped to ≥1 but the panel is never bounded by `m.h`. At 80×24: 5 rows clipped — losing the most-missed heatmap, **the panel's bottom border** (box reads as broken, not scrolled), and the whole footer. At the documented 60×20 minimum: 9 rows clipped. The screen offers no way to discover `tab`/`esc`/`3`.

### D3. HIGH — History trend sparkline unbounded: one cell per record, cap 200
`screen_history_view.go:38-45` + `sparklineInline:103` emits `len(vals)` runes with no width argument. At 120 records on an 80-col terminal the line is **143 cells**; the `last 120 tests` label is cut entirely, and since `lipgloss.Place` sizes from the widest line, **the whole History screen loses horizontal centring**. Triggers for any user with ≳60 saved tests. `bucketSamples` already exists in `result_graph_axes.go`.

### D4. HIGH — Width accounting uses rune count where display cells are meant; CJK/emoji overflow 2×
`code_stream_renderer.go:124` (`cellW := 1`) and `word_stream_renderer.go:112`. Code mode accepts arbitrary text via `--text` and paste, so directly user-reachable.
```
120 CJK runes, term 80x24 -> line cells=141 runes=72
60 emoji,      term 80x24 -> line cells=120 runes=60
```
Wrapper breaks at 72 runes = 144 cells on an 80-col terminal; 61 columns of text the user must type are invisible. `word_stream_renderer.go` carries a `// CJK double-width is not handled — deferred` note; `code_stream_renderer.go` carries none and is exactly where arbitrary text arrives. Fix: `uniseg.StringWidth` (already an indirect dep) or `ansi.StringWidth` in both wrappers.

### D5. MEDIUM batch
- **Hardcoded 62-col rules in History** (`history_table.go:26`, `screen_history_view.go:69`) overflow the 60-col minimum by 2 and kill centring.
- **Home footer overflows by exactly 1 column at width 72** — `footer.go:25` switches to full labels at `termW >= 72` but never measures; the 6-hint footer is 73 cells. Sweep 60→90 flags only w=72.
- **Persistence toast invisible exactly when it matters** — `model_view.go:62-69` writes into `lines[len(lines)-1]`; at h=20/24 that row is clipped, so "Couldn't save result to disk" never appears and the result is silently lost. At h≥30 it lands on the *footer* and destroys the keybindings.
- **Timer tick loop leaks one `tea.Tick` chain per restart** — `screen_typing.go:158-164`, no `armed` guard, even though `frameLoopArmed` exists two fields away. 3× tab → 4 live loops → `ResultMsg` emitted 4×. Persistence does not double (extras dropped on ScreenResult, verified), so perf/battery not corruption.
- **New-best celebration never fires on a standard terminal** — `celebration.go:42-45` needs an all-blank row; the fixed 29-line Result frame has none until h≥34. Whole feature dead at the most common size.
- **History meta line reports the cursor, not the window** — `history_table.go:81-89` `to := sel + 1` → `showing 1–1 of 120` with 14 rows on screen.
- **Screen transition not cancelled by nav keys** — `model_key_handler.go:52-72` clears `quitPrompt` but never `m.transition`, so for ≤250ms the root blends a stale *Typing* snapshot over History/Settings/Home.

### D6. Known issues — confirmed with severity judgement
1. **Error `x` marker (HIGH in UX terms, worse than filed).** `result_graph.go:97`: when one second holds the run's max, `errors/maxErr == 1` pins the marker to row 0. Agent's render shows the `x` floating at 64 wpm on a run that never dipped there — a reader takes it as "WPM spiked at second 33". Actively misleading, on the screen's centrepiece.
2. **`errAxisLabel` (MEDIUM, worse than filed).** Integer truncation at `result_graph_axes.go:65` makes the axis **non-monotonic** for every small max: `maxErr=1 → 1/0/0`, `maxErr=3 → 3/1/0`.
3. **Hero emptiness — quantified.** At 120 cols `innerW=88`; rows 3-11 and 22-26 (**14 of the panel's 26 content rows**) use under a third of it. ~60 dead columns × 14 rows. **Constraint for the redesign: the graph is the only element that wants 88 columns**, so shrinking `resultMaxContentW` trades one defect for another.

### D7. Degraded-notice copy is a false promise
It advertises "Need at least 60×20", but at exactly 60×20 Result loses 9 of 29 rows and History's rules overflow by 2 columns. Either raise the gate to what the screens can actually render (~72×30 given current Result height) or fix D2/D5 so the promise holds.

### D8. Genuinely good (scope 3) — held up under deliberate attack
- **Theme parity is exact.** 0 breaks across default/mono/NO_COLOR/dracula/gruvbox-light × 9 screens × widths 60/80/120, comparing stripped-ANSI byte-for-byte. Every Role mapped in every theme. The layout-identical invariant is real, not aspirational.
- **The Result reveal animation is airtight.** Every frame at 17ms steps across 7 configurations (1s/2s/30s/120s, zero-error, new-best, widths 60-200) matched the settled frame's line count *and* per-line width; fully-revealed frame byte-identical to static. Correct-by-construction via `revealLine` preserving `lipgloss.Width` and `BigDigitsFixed`.
- `applyCelebration` is defensively and correctly written (only rewrites all-blank rows, rebuilds to exact rune width) — dead at h≤31, but the mechanism is right.
- No hex literals in `internal/ui`/`internal/app` (grep clean), no reachable negative `strings.Repeat`, no sub-model doing disk writes, and `handleResultMsg` does surface `AppendHistory` failure rather than swallowing it.

### D9. The process gap that let all of this ship
`go test ./...`, `go vet`, `gofmt -l` are all green with **every defect above present**. There is not one assertion in the codebase of the form "rendered frame fits inside w × h". `result_layout_test.go:140,161` and `screen_result_test.go:440` check panel width against `lay.PanelW`/`lay.InnerW` — **the layout system agreeing with itself, never against the terminal.**

One table-driven test over {every screen} × {40,60,61,72,80,88,120,200} × {15,19,20,24,30,50} asserting `len(lines) <= h && maxWidth <= w` would have caught D1, D2, D3, D5-rules and D5-footer at authoring time. **Single highest-value change in this audit.**

---

## SCOPE 4 — Repo hygiene / CI / release / docs (COMPLETE)

### C1. One third of `ci.yml` is not a required check — VERIFIED BY ME
```
gh api repos/bavanchun/Typeburn/branches/main/protection --jq '.required_status_checks.contexts'
-> ["Build & Test (ubuntu-latest)","Build & Test (macos-latest)"]
```
The `installer` job — `name: install.sh & release config` (`ci.yml:47`) — is **absent from required checks**. CLAUDE.md states "`ci.yml` must pass" as a hard gate; it does not. A PR breaking `install.sh`, failing `shellcheck`, or breaking `.goreleaser.yaml` schema is mergeable to main with that job red.

Consequence: `install.sh` is what `README.md:31` tells users to pipe into a shell, and `goreleaser check` is the ONLY pre-tag validation of release config — `release.yml` never runs it. Broken config → tag publishes partially or not at all, and tags are immutable.

Fix: add `install.sh & release config` to `required_status_checks.contexts`. One API call.

### C2. Reachable known vulnerability on the render path; no scanner in CI — VERIFIED BY ME
```
govulncheck ./...
Vulnerability #1: GO-2026-5970  (infinite loop on invalid input, golang.org/x/text)
  Found in: golang.org/x/text@v0.24.0   Fixed in: golang.org/x/text@v0.39.0
  #1: internal/ui/ascii_big_digits.go:124:27: ui.BigDigits calls lipgloss.Style.Render
                                              -> norm.Form.Properties
```
Not a theoretical indirect — it is on the render path of every frame. `go.mod:38` pins v0.24.0; v0.40.0 available. No `govulncheck` step in `ci.yml`/`release.yml`, no `.github/dependabot.yml`. Nothing in this repo would ever tell the maintainer.

(Five further stdlib findings are against the local go1.26.2 toolchain; CI builds 1.25.x, whose own set is likewise unmeasured. Toolchain currency is unmanaged.)

### C3. `.gitignore:25` silently swallows new files under `cmd/typeburn/` — VERIFIED BY ME
```
$ sed -n 25p .gitignore
typeburn
$ git check-ignore -v cmd/typeburn/newfile.go
.gitignore:25:typeburn   cmd/typeburn/newfile.go
```
Bare pattern, no leading slash → matches any path component named `typeburn` at any depth. Tracked files unaffected (tracked beats ignore), which is why it went unnoticed. **Any NEW file under `cmd/typeburn/` is silently skipped by `git add .`** — no warning. Builds locally, vanishes from the pushed tree.

Fix: `/typeburn`.

### C4. `release.yml`'s gate is strictly weaker than `ci.yml`'s (HIGH)
`release.yml:41-48` runs build/vet/gofmt/race on **ubuntu only**. Skips `size-check`, `notui-noexit-check`, `shellcheck`, install.sh harness, `goreleaser check`; drops the macOS leg. The file's own header argues it exists *because* a tagged commit otherwise has zero CI — then reproduces ~40% of it.

### C5. `make notui-noexit-check` is a string grep; defeated in one line (MEDIUM)
`Makefile:57-61` = `grep -R "os\.Exit" internal/cli/notui`. Agent defeated it in a pristine copy with `import osx "os"; osx.Exit(1)` → guard exits 0. Also bypassed by `syscall.Exit`, `runtime.Goexit`, or an exiting helper elsewhere. Promoted to a named CI step and named in CLAUDE.md, so it reads as a real invariant. Guards the literal string, not the behaviour.

### C6. Windows binaries ship with zero Windows CI (MEDIUM)
`.goreleaser.yaml:32-38` builds windows×{amd64,arm64}, both published on v2.8.1. `ci.yml:16` matrix is `[ubuntu, macos]`. Never compiled or tested on Windows: `x/windows`, `x/termios`, notui raw mode, `%LOCALAPPDATA%` path resolution, and — highest risk — `internal/update`'s in-place binary swap, since **Windows forbids replacing a running executable by rename**. Break surfaces only after the tag exists, and tags are fix-forward only.

### C7. No gate that `.github/release-notes.md` matches the tag (MEDIUM)
`release.yml:103,111` passes `--release-notes` unconditionally. Forget to update → GoReleaser silently publishes the PREVIOUS release's notes under the new tag, permanently.

### C8. Documentation drift — the two "source of truth" docs contradict each other
Ranked:
1. **`CLAUDE.md:37` — two wrong API facts in the always-loaded doc.** Claims `metrics.Compute(log, startMs, durationMs)`; actual is `Compute(log, mode.Mode, endMs)` (`compute.go:43`) — wrong arity AND semantics. Claims `AFKTrim`; actual symbol is `TrimAFK` (`afk_trim.go:21`), `grep AFKTrim` → zero hits. `docs/codebase-summary.md:278,282` has both correct — so the wrong one is the one always in context.
2. **`docs/system-architecture.md:131-132` — layering invariant stated as fact, false today.** `internal/theme` "No UI/Bubble Tea" vs `default-theme.go:6` importing lipgloss. `internal/storage` "no UI deps" is false transitively: storage → config → bubbletea (`go list -deps ./internal/storage | grep -c charm` = 9). Same error in `CLAUDE.md:31` and `codebase-summary.md:9` (`internal/update` "pure-stdlib" — also imports config→bubbletea). **Nothing enforces it; there is no import-purity test anywhere.**
3. **`docs/project-overview-pdr.md` is three releases stale** — says v2.5.1 is current stable (actual v2.8.1), Bubble Tea v2.0.6 (actual v2.0.7), describes the pre-v2.6.0 Result screen, omits `update` from the CLI surface.
4. **A quote bucket "Epic" is documented in 5 places and does not exist.** Actual enum is `{QuoteShort, QuoteMedium, QuoteLong}` (`quotes.go:17-21`). `codebase-summary.md:294` invents an API member.
5. **`CLAUDE.md`'s "Homebrew is a documented prose TODO, intentionally not a dead YAML block" is flatly wrong now.** `.goreleaser.yaml:95-116` is a live `homebrew_casks:` block, the tap repo exists, `release.yml:88-97` injects the token. **A future agent reading CLAUDE.md would be told to delete working release infrastructure.**
6. `codebase-summary.md:451` "All files <200 LOC (largest ~190)" — false (`screen_home.go` = 208). `:229` `internal/ui` file list names 45 files against an actual 68, including three that do not exist, and contradicts its own kebab-case naming 200 lines later.
7. `README.md:237` understates `ci.yml` by five steps. `README.md:219-224` data-paths table omits `$XDG_STATE_HOME`.

### C9. Why a green gate + 79-93% coverage caught none of the confirmed defects
Not a coverage shortfall — three structural blind spots:

- **`-race` is structurally incapable of finding the `AppendHistory` defect.** It is a read-modify-write across separate OS *processes*; the race detector instruments shared memory in one process. `history_store_test.go` has 20 test funcs and **zero** `go func`/`WaitGroup`/`t.Parallel`. `grep "Flock\|LOCK_EX" internal/` → zero; there is no lock to test. Yet `codebase-summary.md:460` advertises "Race detection GREEN" as the concurrency assurance.
- **The AFK/360-WPM defect is a cross-package property; every test is a within-package example.** `TrimAFK` has exactly one test, in isolation. The defect only appears in `TrimAFK → Compute → Record.NetWPM → IsNewBest → AppendHistory`, spanning four packages. No test crosses that seam. Nearest miss is a *relative* invariant (`NetWPM >= RawWPM`). **No assertion anywhere that WPM is bounded by anything physically achievable.** 93.8% coverage means those lines ran; nothing asserted what they produced.
- **The update signal defect lives in the one file at 0.0% coverage.** Six `TestStopApply_*` tests cover the cancellation *mechanism* well — but `stopApply` is only reachable from the Bubble Tea branch. `grep "signal\." internal/cli/` → zero. The wiring is in `cmd/typeburn/main.go:14`, in the only package at **0.0%**.

Coverage table: mode/runner/theme 100%, anim/typing 97.8%, metrics 93.8%, words 92.9%, codetext 91.7%, ui 89.8%, update 83.3%, app 83.0%, updateui 80.4%, config 80.0%, cli 79.1%, storage 79.0%, notui 78.3%, output 77.8%, version 66.7%, **cmd/typeburn 0.0%**.

### C10. Size + deps
`bin/typeburn` = 9,002,530 B vs `SIZE_LIMIT` 10,485,760 → **85.9% of cap, 1.41 MiB headroom**. `Makefile:44-45` records a stale 5,302,642 B baseline; actual is +70% over it. One more `internal/update`-sized feature breaches the cap.

Outdated directs: bubbletea v2.0.7→v2.0.8, lipgloss v2.0.4→v2.0.5, x/term, x/sync, x/sys, x/text. No dependabot → all manual.

Allowlist unenforced and already exceeded: 9 module families outside CLAUDE.md's list are compiled into the shipped binary (all legitimate charm/cobra transitives, but the rule has no indirect carve-out and no CI check).

### C11. Hygiene
- Six `package-lock.json` files (~305 KB) committed under `.claude/skills/`, with **no tracked `package.json` anywhere**. Agent-tooling residue; the three largest tracked files by bytes.
- `.claude/hooks/` is not tracked, and `.claude/settings.local.json` is excluded by the user's *global* gitignore. CLAUDE.md presents the commit-blocking hook as one of two hard enforcement mechanisms — it is machine-local and does not travel. GitHub branch protection is the only portable gate (and `enforce_admins`, `required_linear_history`, `allow_force_pushes:false` are correctly set, so this is a docs-accuracy issue, not an exposure).
- `required_approving_review_count: 0` — PRs self-mergeable. Fine for solo, recorded as a decision not an accident.
- 31 plan dirs, all with matching journal/roadmap entries, no orphans. `docs/mdocs/20260618/01-20260618.md` is the only unexplained artifact.

### C12. Genuinely good (scope 4)
- **`scripts/test-install-sh.sh` is the strongest asset in the repo.** 14 offline scenarios testing what actually hurts: checksum mismatch writes nothing, truncated download aborts, a symlink archive member pointing at `/etc/passwd` is rejected, failed install leaves the prior binary byte-identical, a poisoned `releases/latest` returning a prerelease is refused. 14 passed / 0 failed; shellcheck clean. Adversarial, not happy-path.
- `release.yml`'s job graph is correct: no workflow-level permissions, `contents: write` isolated to `goreleaser` behind `needs: [test, verify-main]`, tap token confined to one step behind a non-mutating permission probe. `verify-main` rejecting tags unreachable from `origin/main` is genuinely thoughtful.
- `.goreleaser.yaml`'s comments explain *why*, not *what* — which is what stops a reviewer "simplifying" the changelog trap. Instructive contrast: the file with the most reasoning is the most current; the file with the most summary is three releases stale.

---

## SCOPE 5 — Product UX + Result redesign (COMPLETE)

### E1. Words are split mid-word on every line break — VERIFIED BY ME IN SOURCE
`internal/ui/word_stream_renderer.go:116-118` comments it outright: *"There is no scan-back to the last space, so a word wider than the line is split between runes here."* But the wrap condition `lineWidth+cellW > width && lineWidth > 0` breaks at the exact cell boundary, so **every** wrap point splits whatever word straddles it:
```
row 2: ... night book shi
row 3: p they boat love ...
```
For a *typing* app this is severe — the eye tracks whole words, and a split word destroys the read-ahead that typing speed depends on. Fix: one-token lookahead in `wrapTokens`, ~15 LOC. The token list already knows word boundaries.

### E2. Result emptiness — measured, not asserted
Inside the 88-col inner area at 120 cols, excluding the graph: **259 of 1144 cells inked = 22.6% fill**. Including graph, 48.5%. The unused cells form **one contiguous rectangle from col ~30 to col 88 across 13 rows** — the eye reads a solid rectangle of nothing as an unfinished region, not as whitespace.

Named failures:
1. **The gutter void** (above).
2. **The chart is 60% empty vertically too.** Y-axis scaled `0…max` while data lives in 55–85, so 3 of 5 plot rows carry no curve. The one element that fills the width horizontally is empty vertically. Fit-to-data scaling (±10% pad) fixes it for ~10 LOC.
3. **Hero aspect ratio contradicts its container** — a 25×7 block (3.6:1) stranded in an 88×7 band (12.6:1) with no counterweight.
4. **Optical centre is ~11 cols left of geometric centre.** Ink centroid ≈ col 33 of 88; border axis = col 44. **v2.8.0 centred the box and left the ink where it was** — precisely why capping felt tidier but not fixed.
5. **Hierarchy collapsed from 3 tiers to 2.** `acc 96%` is typographically indistinguishable from `time 30s`. Monkeytype renders wpm and acc at the same size; here accuracy has the weight of an extra-char count.
6. Two labelling conventions 5 columns apart — `wpm` below its digits, `acc` above its value.
7. **Nothing on screen says whether 87 wpm is good.** History is already loaded at `model_history.go:46` at the exact moment the ResultModel is built, and used only for a boolean `isBest`. PB, rank, last-10 average are in hand and thrown away. **Part of the emptiness is a content deficit — better whitespace management just centres the void.**

### E3. Recommendation: Option A2 — three-zone hero band + comparison rail + full-width chart
`[big wpm] [big acc] [right-flush comparison rail]`, chart full width beneath, closing meta line with ink at both edges.

Wins because it is the only option solving all four problems at once: **fits 80×24 with the footer intact** (22 panel rows + spacer + footer = exactly 24, a 5-row saving), puts ink at both edges so optical centre meets geometric centre, gives accuracy real typographic weight, and fills the gutter with *information the user wants* — is this better than my best? — from history already loaded.

**Deliberately does NOT widen past 88 at 200 cols.** The stretched variant (inner 116) puts `personal best` and `91 wpm` 50 columns apart and the pairing dies. The 88 cap is correct and should stay; the emptiness is *inside* the panel, the outer margin is legitimate max-measure whitespace.

- **Option B (per-tier split dashboard) loses:** its own mockups show it re-creates the dead zone inside the main column at every width (24×6 dead cells at 120, 62×6 at 200), costs a second layout to maintain forever, and fabricates chart resolution at 200 cols (3.4 cells per sample of interpolated line). The earlier rejection was right.
- **Option C (shrink container to content) loses, but is the cheap fallback.** ~60 LOC, 2 files, fill ratio to ~55%, optical centre correct. But it fills nothing — *"the complaint is 'there's nothing here'; C's answer is 'correct, here is a smaller nothing'."* Also throws away the wide terminal and leaves 87 wpm without meaning.

**A2 cost — Medium.** Hero rewrite ~110 LOC; new `result_comparison_rail.go` ~70 LOC; delete `renderStatsGrid` (−44); `layoutFor` +25. Data is nearly free — `BestWPMPerBucket`/`BestBucketKey`/`EffectiveWPM` already exist; only avg-last-10 + rank need a ~25 LOC pure helper.
**Prerequisite: the glyph table must be rectangular first** (A0), or the right rail shifts a column on digits 3/6/9.

**What A2 gives up:** wide terminals gain no extra info over 120; the stats block loses its aligned-table form; Result gains a content dependency on history (mitigate by passing a pre-computed struct, not `[]storage.Record`); and the rail is empty for the first ~3 runs, needing a dedicated first-run copy variant.

### E4. Colour-only encodings — one real information loss
**`220/8/1` (`screen_result_view.go:134-140`) is the only genuine one.** The middle number's meaning is carried *entirely* by `RoleError`; under `mono` that is `#FFFFFF` vs primary `#F2F2F2`, and under NO_COLOR it is gone. Fix: label it — `220 correct · 8 wrong · 1 extra`. Layout-identical either way.

The other two (`accColorRole`, History `accRole`) are decoration — the number carries its own meaning. Leave them.

Correctly handled already: the typing stream's incorrect state carries `.Underline`, so error identity survives NO_COLOR; `★` and `▎` are glyphs. **Any new delta indicator must be `▼ 4 wpm` / `▲ 6 wpm`, never a bare tinted number.**

### E5. Other screens
- **Home is the strongest screen in the app** — correct hierarchy, bold CTA, fully enumerated footer. Defects: all vertical slack dumped below content (`screen_home_view.go:31`) so it hangs off the top; tab row and length row centred on different axes (col 22 vs 28) so they read as unrelated; two selection idioms (`▎` and `[30]`) on adjacent rows.
- **Settings** — good, and the contextual helper line describing the selected row is a genuinely nice pattern. Punctuation/Numbers/Strict are in the wrong place (see E6).
- **History** — columns *are* correctly aligned. `showing 1–1 of 7` is wrong (cursor index as range end). Three rules for a 7-row table. Numeric columns left-aligned so the ones digit staggers. No mode filter, so `time 15` and `quote long` compete on a non-comparable WPM column.
- **Code Paste** — the only screen with **no footer hint row**, breaking the learned scan target. Answers none of the user's actual questions (size limit, tab handling). `typeburn --text` exists and is invisible from inside the TUI.
- **Quit prompt** — correct, minimal, defaults to `no`. No changes.

### E6. Flow / IA
Core loop measured: **launch → typing = 1 keystroke; result → retry = 1 keystroke.** That genuinely beats Monkeytype (tab+enter, mouse for config) and beats `tt`/`toipe`, which have no restart affordance at all. **This is Typeburn's real competitive advantage and must be protected in any redesign.**

Falls short of "Monkeytype in a terminal" mainly in one place: **per-test modifiers are buried two screens deep.** Punctuation/Numbers are *test modifiers*, not preferences — Monkeytype puts them in the config bar above the words. Here they sit in Settings next to `Blink cursor`, invisible from Home; a user will never discover them. Fix: a modifier row on Home under the length row (`p punctuation · n numbers · s strict`), which also fills Home's dead row.

Beats peers on: 1-key start/restart, per-second dual-axis WPM chart in a TTY, 8 themes with a tested luminance floor plus real mono/NO_COLOR, `--text` code mode, and a footer hint row on every screen.

**First run needs no onboarding** — Home already tells the user what to press. The weak moments are downstream: first Result has no reference point, first History is empty with no guidance, and `wpm`/`raw`/`consistency` are never defined anywhere in the UI (one faint line under the chart would do it).

---

## FINAL TRIAGE — all 5 scopes

### Tier 0 — wrong information shown to the user (fix first)
| | Finding | Cost |
|---|---|---|
| A0 | Hero renders every 0 as a 2 (`60`→`62`, `100`→`122`) | S |
| A1 | AFK burst → 360 WPM persisted as permanent personal best | S + product decision |
| A2 | Single wrong keystroke → 100% accuracy, becomes first best | S |
| D6.1 | Error `x` marker pinned to chart top, reads as a WPM spike | S |

### Tier 1 — the product is broken at its default size
| | Finding | Cost |
|---|---|---|
| D1 | Time mode clips 23 of 47 rows at 80×24; footer never visible; caret leaves screen | M |
| D2 | Result fixed at 29 rows; loses footer + bottom border at 80×24 | S–M |
| E1 | Words split mid-word at every line break | S |
| **NEW TEST** | **One table test: every screen × widths × heights asserts `lines <= h && width <= w`** — would have caught D1, D2, D3, D5-rules, D5-footer | S |

### Tier 2 — data integrity
A7 (concurrent `AppendHistory` loses records) · A8 (one corrupt byte destroys 200 records, no backup) · A3/A4 (strict-mode replay desync) · A5/A6 (unbounded input: `/dev/zero` never returns; replay JSON OOM-kills)

### Tier 3 — supply chain / process
C1 (installer job not a required check) · C2 (`GO-2026-5970` on the render path, no scanner) · C3 (`.gitignore` swallows new files under `cmd/typeburn/`) · B1 (Ctrl-C during update leaks lock) · B2 (recovery needs 3 manual deletions, 1 hint)

### Tier 4 — the UI/UX work the user actually asked about
E3 Option A2 (Medium) · chart y-axis fit-to-data (S) · E4 label `220/8/1` (S) · split vertical slack on 4 screens (S) · first-run empty states (S) · Home modifier row (M) · History de-chrome + filter (S/M)

### Tier 5 — docs
C8: `CLAUDE.md:37` wrong `Compute` signature + wrong `TrimAFK` name (always-loaded file); Homebrew claim would tell an agent to delete live release infra; layering invariant stated as fact but false and unenforced; `project-overview-pdr.md` 3 releases stale; phantom "Epic" quote bucket in 5 places.

### The one structural theme
Every scope independently found the same thing: **tests assert that a component agrees with itself, never that the output is correct.** Golden files baked in the wrong glyph; layout tests compare `PanelW` to `lay.PanelW`; `-race` cannot see cross-process writes; 93.8% coverage on metrics with no plausibility bound. CI is green with all of the above present.

---

## Resume plan

1. Re-dispatch scopes 3 (UI), 4 (hygiene/CI/docs), 5 (UX design). Prompts were identical in structure; the killed agents' probe files are preserved in scratchpad `killed-agent-probes/{app,ui,theme}/`.
2. **Instruct re-dispatched agents to use `go test -overlay` instead of writing probe files into the repo** — scope 1's agent did this correctly and left the tree clean; scope 3's did not.
3. Chase scope 3's unverified parting claim: "Both the Typing and Result screens overflow a standard 80x24 terminal."
4. Then: triage into fix batches, get user decisions on the open questions below, plan, implement.

---

## Unresolved questions (need user decision)

1. **AFK trim policy** — rescale to the active window, or invalidate the run? Decides A1's fix. Product decision, not a code one.
2. **Strict-mode `Accuracy`/`Errors`** — keystroke-level or final-state? `model_history.go:25-27` currently answers this for `Accuracy` only, leaving `Errors` unmapped.
3. **Words/Quote consistency window** — full test duration (incl. intentional pauses) or active seconds only? Currently the latter while WPM uses the former.
4. **Is `typeburn replay` meant to accept third-party logs**, or only logs this binary produced? Sets required validation strictness for A6.
5. **Upper bound for `--words`?** Any number is arbitrary; user picks.
6. **Is B1 (plain-path signal gap) in scope for a patch release**, or grouped with the deferred stale-lock work? They share a fix.
7. **`restoreInterruptedUpdate`** — wire into `main()`, or delete? Decides whether B3 is a bug fix or a cleanup.
