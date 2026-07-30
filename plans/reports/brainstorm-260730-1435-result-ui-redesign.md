# Brainstorm Summary — Result-Screen UI Redesign

**Date:** 2026-07-30 · **Status:** ACCEPTED (4/4 decisions resolved) · **Feeds:** plan `260730-1435-result-ui-redesign`

## Outcome

Result screen matches the Monkeytype target image: big `wpm` + `acc` numbers on top, a real dual-axis graph (WPM-over-time line + Errors line with red X markers), and a 2-column stats grid — while preserving Typeburn's NO_COLOR/mono/layout-invariant guarantees and the per-key heatmap.

## Constraints

- Go 1.25 + Bubble Tea v2 / Lip Gloss v2; no new dependencies (stdlib + charm only).
- Theme = semantic `Role` only, never hex literals (NO_COLOR/mono first-class).
- Every file <200 LOC (split by concern; `sparkline.go` already 173 → new sibling file for the line graph).
- Reveal animation: every frame layout-identical, settled frame byte-identical to static render.
- `nocolor_layout_invariant_test.go` must stay green; min terminal 60×20 with graceful degrade.
- Protected `main` → PR-only workflow; conventional commits, no AI refs.
- Data already present: `metrics.Result` (NetWPM/RawWPM/Accuracy/Consistency/Correct/Incorrect/Extra/DurationMs) + `PerSecond.{RawWPM,Errors}` — `PerSecond.Errors` is computed but UNUSED today.

## Non-goals

- No backend / online sync. No persistence schema break. No new deps.
- Not touching Home/Typing/Settings/History screens.
- No `missed` char-stat, no session timestamp (decided below).

## Resolved Decisions

| # | Decision | Choice | Rationale |
|---|---|---|---|
| 1 | Graph fidelity | **Full dual-axis line** | WPM line (left Y) + Errors line w/ red X markers (right Y), X-axis = seconds. Most faithful to image; `PerSecond.Errors` ready. Biggest new component. |
| 2 | Char stats | **Keep 3-tuple** (correct/incorrect/extra) | `MissedChars` removed v1.0.1 (always 0). YAGNI. |
| 3 | Heatmap | **Keep** | Typeburn v2.2.0 differentiator; place below the new stats grid. |
| 4 | Session timestamp | **Omit** | No start clock-time in `metrics.Result`; show duration `15s` only. Avoids schema bump. |

## Result Layout (target, top→bottom)

1. **Hero** — two big-number blocks side-by-side: `wpm <N>` (left, ASCII big-digits) + `acc <N>%` (right). New-best `★` stays on WPM. (Drop acc from the old stat-card column; keep `raw` + `consistency` as smaller secondary cards or fold into stats grid.)
2. **Graph** — dual-axis line chart: WPM line (accent, left Y 0..maxWPM) + Errors line/markers (error role, right Y 0..maxErr), red `x` on error seconds, X-axis = seconds. Replaces current bar sparkline.
3. **Stats grid (2 cols)** — left: `test type` (mode length / english), `raw <N>`, `characters c/i/e`; right: `consistency <N>%`, `time <N>s`.
4. **Heatmap** — existing "most missed" line, unchanged.
5. Wrapped in the existing rounded-border `result` panel; footer hints unchanged.

## Key Risks / Watch

- **New line renderer** must be its own file (<200 LOC) and reveal-compatible (draw-in animate, byte-stable when settled).
- **Dual-axis scaling**: WPM and Errors have different magnitudes — independent min/max per axis.
- **teatest goldens** (`screen_result_*_test.go`) will break → intentional re-goldening, not test weakening.
- **Narrow terminals** (60–72 cols): dual Y-axis labels + line may crowd — degrade via `width_tier.go` (e.g. drop right Y labels, keep WPM line + markers).

## Acceptance Criteria

- `go test ./... -race -count=1` GREEN; `gofmt -l .` empty; `go vet ./...` clean.
- NO_COLOR/mono layout-invariant test passes; settled Result render byte-stable across frames.
- Golden files intentionally re-goldened (no test weakened).
- `docs/wireframe/mockups.md` §3 + `docs/project-roadmap.md` updated; new release entry.
- Visually matches target image (hero wpm+acc, dual-axis graph with error markers, 2-col stats).

## Unresolved Questions

None at brainstorm gate. (Implementation-phase details — e.g. exact line-drawing chars, reveal easing for the line — deferred to plan.)

## Handoff

→ Plan skill (`/ak:plan`) to phase the work: likely (1) dual-axis graph renderer (new file, TDD), (2) hero 2-big-number layout, (3) 2-col stats grid re-arrange, (4) reveal + NO_COLOR invariants + golden re-golden, (5) docs sync + verify. Then `/ak:cook`.
