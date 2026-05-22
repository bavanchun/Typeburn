# Typeburn Defect Audit Report
**Date:** 2026-05-22  
**Scope:** Full codebase — all packages, v2.1.0

## Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH     | 0 |
| MEDIUM   | 2 |
| LOW      | 4 |

---

## Defects

### [MEDIUM] notui runLoop: O(n²) full metrics.Compute on every keystroke

- **File:** `internal/cli/notui/runner.go:50`
- **Bug:** `runLoop` calls `metrics.Compute(session.Engine.Log(), ...)` on every keystroke for live WPM display. This does an O(n) log copy + AFK trim + replay + bucketing + consistency pass per keystroke → O(n²) total. The TUI path uses the cheaper `liveWPM()` (one forward scan) throttled at 250ms. `--no-tui` has no throttle.
- **Repro:** `typeburn run --no-tui --mode code --text <10000-rune file>` — noticeable slowdown on long snippets.
- **Fix:**
  ```go
  live := liveWPMFromLog(session.Engine.Log(), nowMs-session.Engine.StartMs())
  ```
  Mirror the `ui.liveWPM` approach (count non-zero Typed entries, divide by 5 / elapsed-minutes).

---

### [MEDIUM] renderVersionCheckJSON: error written to stdout AND propagated as return value

- **File:** `internal/cli/cmd_version.go:104-106`
- **Bug:** When `checkErr != nil`, `renderVersionCheckJSON` encodes a JSON error to stdout then **returns `checkErr`**. Cobra prints the error to stderr. Machine consumers get a valid JSON object on stdout + a prose error on stderr + non-zero exit code — breaks parsers expecting clean stderr.
- **Repro:** `typeburn version --json --check-update` with GitHub API unreachable.
- **Fix:**
  ```go
  _ = enc.Encode(out)
  return nil  // error already embedded in JSON
  ```

---

### [LOW] stripANSI: only terminates on SGR 'm', not full CSI final-byte range

- **File:** `internal/ui/result_render_helpers.go:76-92`
- **Bug:** `stripANSI` exits `inEsc` only on `'m'` (SGR). Any non-SGR CSI sequence (e.g., `ESC[2J`) would leave `inEsc=true` indefinitely, eating all subsequent characters including border chars. Corrupts width measurement in `injectBorderTitle`.
- **Repro:** Any Lipgloss v2 upgrade that emits non-SGR CSI sequences (not current, but fragile).
- **Fix:**
  ```go
  case inEsc:
      if r >= '@' && r <= '~' { inEsc = false }
  ```

---

### [LOW] update/check.go: prerelease/draft result not cached → redundant network calls

- **File:** `internal/update/check.go:32-40`
- **Bug:** When latest GitHub release is a prerelease/draft, `Check` returns a synthetic `UpgradeAvailable: false` result but never calls `cacheSave`. Every TUI launch hits the live API again, consuming the 800ms timeout budget indefinitely.
- **Repro:** Enable update check; ensure latest release on GitHub is a prerelease.
- **Fix:**
  ```go
  r := &Result{..., UpgradeAvailable: false, CheckedAt: time.Now().UTC()}
  _ = cacheSave(r)
  return r, nil
  ```

---

### [LOW] notui/reader.go: discardEscape leaves CSI remnants when sequence arrives split across reads

- **File:** `internal/cli/notui/reader.go:62-72`
- **Bug:** `discardEscape` only drains already-buffered bytes. If an escape sequence (e.g., `ESC[A` arrow key) arrives split across two reads — `ESC` in first, `[A` in second — `discardEscape` exits immediately. The next `ReadByte` call reads `[` and emits it as a typed character, injecting a stray `[` into the typing engine.
- **Repro:** Press a function key / arrow key in `--no-tui` mode on a terminal with fragmented PTY reads.
- **Fix:** After reading `ESC`, block-read one more byte; if `[` or `O`, drain until letter or `~`.

---

### [LOW] package-level mutable globals are latent test-race targets

- **Files:** `internal/update/cache.go:27` (`var cacheFilePath`), `internal/cli/cmd_version.go:14` (`var checkFn`)
- **Bug:** Test helpers mutate these globals without synchronization. No active race now (no `t.Parallel()` in these packages) but will race immediately if parallelism is added.
- **Repro:** Add `t.Parallel()` to any `update` or `cli` test and run `make test-race`.
- **Fix:** Inject via function parameter or `sync/atomic` rather than mutable package globals.

---

## Clean Areas (verified correct)

- `internal/typing/engine.go` — backspace on empty, extra-chars, startMs sentinel
- `internal/typing/completion.go` — word boundary detection, edge cases
- `internal/metrics/afk_trim.go` — >7s threshold, all-backspace edge case
- `internal/metrics/compute.go` — division-by-zero guards present
- `internal/storage/history_store.go` — 200-record cap is exact (no off-by-one)
- `internal/storage/new_best.go` — first-record sentinel (-1.0) correct
- `internal/app/routing.go` — NavHistoryMsg → ScreenHistory correct
- `internal/cli/cmd_run.go` — ctx/cmd shadowing fix verified correct
- `internal/config/settings.go` — parseBool, mode/theme enum validation
- `internal/update/compare.go` — semver v-prefix, git-describe, malformed input
- `internal/update/client.go` — 64KB body cap, redirect-block, timeout, body close
- `internal/update/cache.go` — injection guard, atomic write with PID suffix
- `internal/codetext/codetext.go` — BOM, binary guard, CRLF, rune/line caps
- `internal/ui/screen_typing.go` — tick start, Time-mode endMs, backspace guard
- `internal/words/generator.go` — 600-word buffer, quote fallback chain

## Unresolved Questions

None — all flagged areas were verified.
