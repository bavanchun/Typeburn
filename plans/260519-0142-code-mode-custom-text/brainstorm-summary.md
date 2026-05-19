# Brainstorm Summary — Code Mode / Custom Text Input

Date: 2026-05-19 · Status: agreed, ready for `/ck:plan`

## Problem statement

Next sizeable roadmap feature: let users practice typing on their own text
(incl. code) instead of the embedded wordlist/quote pack. Add a `code` test
mode.

## Scout-grounded context

- Modes = `config.Mode` string enum (`time`/`words`/`quote`,
  `settings.go:10`), cycled on Home via `modeOrder` (`screen_home.go:24`),
  persisted in settings + `Record.Mode`. New mode touches the same
  multi-switch seam (LengthsFor / Normalize / home cycle / Record) — same
  DRY discipline used for theme packs.
- **Quote mode is Code mode's twin:** text via `words.ForMode`; engine
  completion = `runesEqual(typed,target)` exact full-text match
  (`completion.go:25`). Engine records keystrokes generically; metrics
  derive post-hoc. AFKTrim is Time-only.
- Renderer `word_stream_renderer.go` = single-stream hard char-cell wrap with
  golden tests across time/words/quote. Does NOT preserve `\n` or render
  tabs.
- No in-TUI text-input widget exists; `main.go` `decide()` flag plumbing
  exists & is tested (used by `--version`).
- History/new-best keys on `modeKey(Mode,Length)` (`new_best.go:63`).
- `internal/words` & `internal/typing` are pure (no I/O) per the layering
  rule (re-tightened in docs this cycle).

## Locked decisions

1. **Input:** `typeburn --text <file>` and `--text -` (stdin) this round.
   Architecture must NOT preclude a later in-TUI paste screen. No paste/edit
   UI now.
2. **Whitespace:** full literal preservation. Target contains literal `\n`
   and `\t` runes. User presses Enter to match `\n`, Tab to match `\t`, and
   types every space/indentation exactly. Tab visual width = **2 columns**
   (display-only; one Tab keypress matches one `\t`).
3. **Persistence:** Code runs ARE saved to history (visible in History) but
   are **excluded from ★ new-best** (custom text not comparable).
4. **Home behaviour:** Code mode is **always** in the Tab cycle; when no
   `--text` was supplied it is shown **disabled with a hint**
   (`pass --text <file> · in-app paste coming soon`) and Enter is a no-op.
   Code mode has **no length selector**.
5. **Out of scope:** syntax highlighting, saved-snippet library, language
   detection / per-language stats, in-TUI paste editing, AFKTrim for code.

## Architecture (agreed)

- **New mode:** add `ModeCode = "code"` to `config.Mode`; extend
  `LengthsFor` (Code → nil/none), `Normalize` (accept "code"), `modeOrder`,
  storage `Record.Mode` doc. Same documented-duplication + sync-test
  discipline as the theme work where a switch is duplicated.
- **Engine/completion:** `typing.New(target, ModeCode, …)` with target =
  literal snippet runes. Add `case ModeCode:` in `completion.go` aliasing the
  Quote exact-match rule. No word-count / no AFKTrim for code. Metrics
  unchanged (newlines/tabs counted as typed chars consistently).
- **Loader — new `internal/codetext` package (pure-core stays pure):**
  reads file or stdin (`-`), normalizes: CRLF→LF, strip UTF-8 BOM, trim a
  single trailing `\n` (so EOF needs no final Enter), reject/cap oversized
  (~10k runes / ~500 lines) and empty/whitespace-only/binary input with a
  graceful error (no crash; Home shows Code disabled + reason). Returns
  `(string, error)`. App wires it in; `words.ForMode` is bypassed for code.
- **Renderer — Approach A: new `internal/ui/code_stream_renderer.go`** +
  vertical viewport that scrolls to keep the caret line visible. Honors
  literal `\n` (real line breaks) and `\t` (2-col visual). Reuses the
  existing per-char `theme.Role` state styling. `word_stream_renderer.go`
  and its golden tests are **untouched** (regression isolation; the two
  rendering models are genuinely different — rejected generalizing into one).
- **Wiring:** `main.go decide()` gains `--text`; loaded text flows through
  `app.Model` → `StartTestMsg` (carry the code text/availability) →
  `NewTyping`. Home forks on text-supplied vs not.
- Rejected: collapse-to-one-stream (excluded by the full-literal choice);
  generalized single renderer (Approach B — high golden-test regression
  risk, false DRY).

## Expected output

`typeburn --text snippet.go` → Home shows Code selectable → start → snippet
rendered multi-line with literal indentation, 2-col tabs, real line breaks,
caret scrolling with a viewport; user types every char; Tab matches `\t`,
Enter matches `\n`; mistakes use existing char-state roles; completion on
exact full-text match; result screen shows; record persisted `Mode="code"`,
never ★; History lists it. No `--text` → Code in cycle but disabled + hint,
Enter no-op, other modes unaffected. `cat f | typeburn --text -` works.

## Acceptance criteria

- All of the above behaviours verifiable; time/words/quote unchanged
  (golden tests pass — word-stream renderer untouched).
- `codetext` normalization unit-tested (CRLF, BOM, trailing-nl trim, size
  cap, empty/binary → error).
- Completion/engine: ModeCode exact-match table-tested incl. `\n`/`\t`.
- Viewport scroll-follow-caret table-tested (caret near top/bottom/long
  file).
- Home disabled-Code state + hint + Enter-no-op tested; length selector
  absent for Code.
- `Record.Mode=="code"` never returned by `IsNewBest`; appears in history.
- gofmt/vet/`go test ./... -race -count=1` green; prod files <200 LOC;
  module stays pure (words/typing no new I/O imports).
- Ships via protected-main PR flow; semver **minor → v1.2.0**.

## Risks

- Viewport scroll math (new) — table-test caret at top/mid/bottom + file
  longer/shorter than screen; degraded (small terminal) gate still wins.
- Tab/Enter must reach the engine mid-test — verify
  `model_key_handler.go` delegates all keys to typing during a test (it
  does today); add a regression test.
- Multi-switch mode seam drift — mirror the theme sync-test discipline.
- Oversized/binary/empty input — must degrade gracefully, never panic;
  explicit tests.
- New renderer dup vs word-stream — accepted: different rendering models;
  isolation chosen deliberately over false-DRY.

## Out of scope (deferred)

In-TUI paste screen + editing, syntax highlighting, saved-snippet library,
language detection / per-language stats, elastic tab stops, AFKTrim for code.
Architecture leaves room for a later paste screen (text-source is an
injected string; Home already forks on availability).

## Open questions

None — all input/whitespace/persistence/home/renderer/loader decisions
locked with the user.
