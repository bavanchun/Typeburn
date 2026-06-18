# Handover — UI/UX Animation System, phases 4–7

Continuation brief for whoever cooks the rest. Phases **1–3 are merged to `main`**;
**4–7 remain**. Read this before touching the phase files — it records deviations
from the original plan that the remaining phases depend on.

## Status

| Phase | State | PR |
|---|---|---|
| 1 Anim core (`internal/anim`) | merged | #40 |
| 2 Frame driver | merged | #41 |
| 3 Caret animation | merged | #42 |
| 4 Result reveal | TODO | — |
| 5 New-best celebration | TODO | — |
| 6 Screen transitions | TODO | — |
| 7 Hardening + docs | TODO | — |

Branch base for the next phase: latest `main` (already contains 1–3).

## Required workflow (per user instruction)

**Every phase = its own branch off `main` → ≥1 commit → `/vchun-git prc` → squash-merge → next phase branches off updated `main`.** One merged PR per phase (Pull Shark). Steps that actually work in this repo:

1. `git checkout main && git pull --ff-only origin main`
2. `git checkout -b feat/anim-<slug>`
3. Implement the phase; keep **every Go file < 200 LOC** (split by concern).
4. Gate locally — must all be clean:
   `go test ./... -race -count=1` · `go vet ./...` · `gofmt -l .` (empty) · `go build ./...`
5. `git add -A`; security-scan the diff; commit with conventional message **+ co-author footer**:
   `Co-authored-by: Hoang Ba Van <174591781+bavan02@users.noreply.github.com>`
6. `git push -u origin feat/anim-<slug>` → `gh pr create --base main ...`
7. **Auto-merge is DISABLED on this repo.** Wait for CI then merge manually:
   `gh pr checks <N> --watch --interval 15` → `gh pr merge <N> --squash --delete-branch`
8. `git checkout main && git pull --ff-only origin main`

`main` is protected (PR required, `ci.yml` must pass, linear history, squash-only). Never push to `main`. CI = Build & Test on ubuntu + macos (~1–2 min) + an install.sh/release-config job.

Note: `prc`'s `git add -A` stages **everything** in the tree — keep the working tree limited to the current phase's files (the pre-existing plan/`docs/mdocs` artifacts already merged in #40, so the tree is clean now).

## Deviations from the original plan (IMPORTANT — build on these)

1. **Frame-tick command lives in `ui`, not `app`.** Phase 2's plan text put
   `frameTickCmd`/`frameInterval` in `app/anim_driver.go`, but Phase 3 needs a
   screen sub-model (package `ui`) to bootstrap the loop — impossible across
   packages. So it's **`ui.FrameTickCmd()` + `ui.FrameInterval` (33ms)** in
   `internal/ui/frame_tick.go`. **Phases 4/5/6 must call `ui.FrameTickCmd()`** to
   bootstrap the loop (not an app-local helper). The app re-arms via
   `Model.maybeFrameCmd()` (in `app/anim_driver.go`).

2. **`transitionActive` is a stub.** `app/anim_driver.go` has
   `func (m Model) transitionActive(nowMs int64) bool { return false }` and
   `animActive` already ORs it in. **Phase 6 must**: add a `transition *transitionState`
   field to `app.Model` (in `model.go`) and replace the stub body with
   `m.transition != nil && nowMs < m.transition.startMs+m.transition.durMs`.

3. **`ResultModel` already has the frame plumbing.** From Phase 2 it has a
   `nowMs int64` field, a `FrameTickMsg` case storing `nowMs`, and
   `HasActiveAnim(nowMs) bool` returning `false`. **Phase 4 fills in** the real
   reveal state (`revealStartMs`, etc.) and the real `HasActiveAnim` window; do
   not re-add the field/case.

4. **`app.Model.animNowMs`** is the shared clock, stamped in `handleFrameTick`
   (`app/anim_driver.go`). The Result reveal/celebration and transitions read it.
   The root forwards `FrameTickMsg` to the active screen's `Update` (typing/result)
   already; `handleResultMsg` (in `app/model_history.go`) is where Phase 4 sets
   `revealStartMs = m.animNowMs` and returns `ui.FrameTickCmd()` to start the loop.

5. **Pre-start typing tick.** Root `StartTestMsg` now returns `m.typing.InitCmd()`
   (bootstraps the 100ms tick so the caret blinks pre-keystroke). Unrelated to 4–7
   but don't remove it.

## Reusable helpers already on `main`

- `internal/anim`: `EaseOutCubic/EaseOutQuad/EaseInOutQuad`, `Clamp01`,
  `LerpColor` (nil→nil for NO_COLOR), `LerpInt`/`LerpFloat`, `Tween{Progress,Done}`,
  `Clock{Add,Active,Prune}`. Pure, no UI deps. **Phase 4 count-up = `anim.LerpInt(0,final,EaseOutQuad(p))`.**
- `internal/ui/caret_anim.go`: pattern for NO_COLOR-aware styling —
  `th.Color(role)==nil` ⇒ attribute path. `lipgloss` `Foreground` accepts a
  `color.Color` directly (pass `anim.LerpColor(...)` straight in — see `fadeStyle`).
- `internal/ui/word_stream_anim.go`: `streamTokenCache{base,valid}` +
  `invalidate()` — the pointer-field-on-value-receiver cache pattern. Reuse the
  idea if a Phase 4/5 render needs per-frame caching (Result screen is NOT perf-
  critical, so probably unnecessary).
- Tests: `strip(s)` (ANSI strip) + `ansiRE` in `code_stream_renderer_test.go`;
  `newTestTyping(mode,len)` in `screen_typing_test.go`; `makeTestMetricsResult()`
  in `test_helpers_test.go`. Use `strip()` to assert the **layout-identical**
  invariant (same runes/line-count/width, attributes may differ).

## Phase-specific gotchas

- **P4 Result reveal**: reserve fixed digit width for the count-up (no jitter);
  every `ResultMsg` must unconditionally (re)set `revealStartMs` (ResultModel is
  reused on the root — stale window would suppress the animation). Settled frame
  must be byte-identical to today's static Result render (add a "settled==static"
  golden). Big digits via `internal/ui/ascii_big_digits.go`; sparkline via
  `internal/ui/sparkline.go`; cards via `internal/ui/stat_card.go`.
- **P5 Celebration**: depends on P4. Pass `isNewBest` into `ResultModel` on
  **every** `ResultMsg` (true or false — no residue ⇒ no spurious confetti).
  New-best is detected only in `handleResultMsg` (`app/model_history.go`,
  `storage.IsNewBest`). Glyphs must be **ASCII display-width 1** (`* + · .`);
  overlay only onto **blank full-width margin rows** (never splice styled lines).
  Deterministic jitter from `(index, revealStartMs)` — no `math/rand` global.
- **P6 Transitions**: root-owned (spans two screens); see deviation #2. Capture
  the already-`lipgloss.Place`d outgoing frame. Expiry is **derived in `View`**
  (`animNowMs < startMs+durMs`) — `View` is a value receiver and must NOT mutate;
  nil-out `transition` lazily in `Update`. Cancel on `WindowSizeMsg` + `AbortMsg`;
  skip when degraded (`<60×20`). Scope = **Typing→Result only**.
- **P7 Hardening**: benchmark the typing hot path (verify the P3 word-stream
  prefix cache engages: animated allocs/op ≈ static). **Code-mode has no token
  cache** (intentional P3 scope call — the code renderer recomputes the viewport;
  only ≤3 cells differ per frame). If P7's code-mode benchmark regresses, add a
  cache there. Write the NO_COLOR layout-invariant table test across all four
  moments. Update docs (`codebase-summary.md`, `system-architecture.md`,
  `project-roadmap.md`, `README.md`). No plan-artifact refs in code comments.

## Open decision still needing the user (raise in P5/P7)

The brainstorm said "byte-identical layout". Resolved in the plan as
**layout-identical** (line count + rune width preserved), NOT literal byte
identity — the celebration overlays glyphs onto blank margin cells, changing
those cells' rune *content* while preserving width. **Confirm this reading is
acceptable before shipping P5.** (Phases 1–3 are strictly layout-identical AND
settle byte-identical, so this only bites at P5.)
