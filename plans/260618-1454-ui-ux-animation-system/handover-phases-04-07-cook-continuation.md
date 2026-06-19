# Handover — UI/UX Animation System, phases 4–7

Historical continuation brief. This was written after PR #43, when phases 1–3
were merged and phases 4–7 were still pending. It is now **superseded**:
phases 4–7 shipped on `main` through PRs #44–#47. Keep this file as build
history and implementation context, not current work guidance.

## Status

| Phase | State | PR |
|---|---|---|
| 1 Anim core (`internal/anim`) | merged | #40 |
| 2 Frame driver | merged | #41 |
| 3 Caret animation | merged | #42 |
| 4 Result reveal | merged | #44 |
| 5 New-best celebration | merged | #45 |
| 6 Screen transitions | merged | #46 |
| 7 Hardening + docs | merged | #47 |

Final branch base was latest `main` after each squash merge.

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

## Deviations from the original plan (applied)

1. **Frame-tick command lives in `ui`, not `app`.** Phase 2's plan text put
   `frameTickCmd`/`frameInterval` in `app/anim_driver.go`, but Phase 3 needs a
   screen sub-model (package `ui`) to bootstrap the loop — impossible across
   packages. So it's **`ui.FrameTickCmd()` + `ui.FrameInterval` (33ms)** in
   `internal/ui/frame_tick.go`. Phases 4/5/6 call `ui.FrameTickCmd()` to
   bootstrap the loop (not an app-local helper). The app re-arms via
   `Model.maybeFrameCmd()` (in `app/anim_driver.go`).

2. **`transitionActive` was filled in during Phase 6.** `app.Model` now owns
   `transition *transitionState`, and `transitionActive` returns true while the
   root-owned Typing→Result transition is inside its duration window.

3. **`ResultModel` already has the frame plumbing.** From Phase 2 it has a
   `nowMs int64` field, a `FrameTickMsg` case storing `nowMs`, and
   `HasActiveAnim(nowMs) bool`. Phase 4 filled in the reveal state
   (`revealStartMs`, etc.) and the real active window.

4. **`app.Model.animNowMs`** is the shared clock, stamped in `handleFrameTick`
   (`app/anim_driver.go`). The Result reveal/celebration and transitions read it.
   The root forwards `FrameTickMsg` to the active screen's `Update` (typing/result)
   already; `handleResultMsg` (in `app/model_history.go`) sets
   `revealStartMs` and returns `ui.FrameTickCmd()` to start the loop.

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

## Resolved decision

The brainstorm said "byte-identical layout". Resolved in the plan as
**layout-identical** (line count + rune width preserved), NOT literal byte
identity — the celebration overlays glyphs onto blank margin cells, changing
those cells' rune *content* while preserving width. This is the shipped behavior
documented in `docs/system-architecture.md`: mid-animation frames are
layout-identical; settled frames are byte-identical to the static render.
