# Brainstorm Report: Punctuation & Numbers Toggle

Date: 2026-07-02 | Branch: main | Status: approved, TDD plan mode selected

## Problem Statement

User asked for a new feature or worthwhile upgrade/polish, open-ended. Typeburn v2.5.0
has 4 modes, strict mode, animations, self-update, heatmap. No product gap flagged in
roadmap backlog was itself a "capability" ask — user wanted something not yet covered.

## Requirements (locked)

- Expected output: 2 new persisted settings (`Punctuation`, `Numbers`, both bool,
  default false) toggled from Settings screen, applied to Words + Time mode word
  generation only.
- Acceptance:
  - Punctuation ON: ~15-20% words get trailing `,`/`.`/`;`; word after `.` capitalized;
    rare word quoted.
  - Numbers ON: ~10-15% word-slots replaced with random 1-4 digit numeric token.
  - Both OFF by default; legacy settings.json without fields safely zero-value.
  - Quote/Code modes unaffected; typing/metrics engine unaffected (chars typed as-is).
- Scope boundary (explicitly OUT): Quote mode, Code mode, Home-screen quick-toggle UX,
  any typing-engine/metrics change, any new keystroke-accuracy category.
- Constraints: Go 1.25 stdlib only, `internal/words`/`internal/config` stay UI-free,
  <200 LOC/file, same wiring shape as `StrictMode` (PR #52-54).
- Touchpoints: `internal/config/settings.go`, `internal/words/generator.go`,
  `internal/words/for_mode.go`, `internal/app/model.go`, `internal/runner/session.go`,
  `internal/cli/cmd_run_validate.go`, `internal/cli/cmd_config.go`,
  `internal/ui/screen_settings.go` + `_view.go`, README.md, docs/project-roadmap.md.

## Evaluated Approaches (feature direction)

| Option | Effort | Pros | Cons |
|---|---|---|---|
| Practice weak-keys mode | Large | Closes real gap using existing heatmap data | Biggest scope, new History→generator bridge |
| **Punctuation/numbers toggle (chosen)** | Small-Medium | Monkeytype parity, proven wiring pattern (StrictMode x2) | Doesn't address a totally new capability, "just" a generator feature |
| Lifetime stats dashboard | Small | Pure polish, zero typing-engine risk | Doesn't add typing-mechanic value |

User chose punctuation/numbers toggle: worthwhile upgrade, contained risk.

## Final Design

1. `config.Settings` +2 bool fields, `Defaults()` false, `Normalize()` no-op (matches
   `StrictMode` precedent for backward-compat).
2. `words.Generator` gets a post-generation transform (uses existing seeded rng, no
   new randomness source — keeps generator tests deterministic).
3. `words.ForMode` gains `punctuation, numbers bool` params; only applies to
   `ModeWords`/`ModeTime`.
4. Call sites thread settings through exactly like `settings.StrictMode` already does
   in `app/model.go`, `runner/session.go`, `cli/cmd_run_validate.go`.
5. Settings screen: 2 new toggle rows, same on/off cycle as `rowStrictMode`.
6. CLI: `punctuation`/`numbers` config get/set keys mirroring `strict_mode`.
7. Docs: README features list + roadmap backlog entry marked shipped.

## Risks

- Low. Additive-only, proven pattern reused twice already (theme packs, strict mode).
- Generator transform must stay seed-deterministic or existing `generator_test.go`
  fixed-seed assertions could need updates — TDD mode chosen specifically to catch
  this early.

## Next Steps

Hand off to `/ck:plan --tdd` with this report as context.

## Unresolved Questions

- None.
