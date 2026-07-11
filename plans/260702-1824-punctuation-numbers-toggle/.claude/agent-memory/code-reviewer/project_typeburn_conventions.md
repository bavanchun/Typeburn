---
name: project-typeburn-conventions
description: Typeburn repo conventions relevant to reviewing config/settings toggle features (punctuation/numbers, strict-mode precedent)
metadata:
  type: project
---

Typeburn wires new persisted boolean settings through a fixed call chain:
`config.Settings` field -> `words.Generator` transform -> `words.ForMode` ->
`runner.NewSession` -> `ui.NewTyping`/`newTypingWithSeed` -> `app/model.go`
StartTestMsg handler, plus `internal/cli/cmd_run_notui.go` for the non-TUI
path, plus `internal/ui/settings_rows.go` + `screen_settings.go` for the
Settings screen row, plus `internal/cli/cmd_config.go` for `config set`.
Code mode (`NewCodeSession`/`NewTypingCode`) and Quote mode are intentionally
excluded from generation-time transforms — verify this exclusion is tested,
not just claimed.

**Why:** StrictMode (PR #52-54) established this pattern; the
punctuation/numbers toggle (plan `260702-1824-punctuation-numbers-toggle`)
mirrors it. `ApplySettings` (live settings propagation to an in-flight
TypingModel) only updates `blink`/`theme`, not `strict`/`punctuation`/
`numbers` — this is pre-existing, not a regression, when reviewing similar
features.

**How to apply:** When reviewing a new settings-toggle feature in this repo,
check every link in this chain is threaded, check Quote/Code mode exclusion
is asserted by an actual test (e.g. `TestForMode_QuoteIgnoresPunctuationAndNumbers`
pattern), and check the plan's phase-01 file's TDD requirements list
line-by-line against the shipped implementation — plan validation logs in
this repo have caught real drift before (see plan.md's "Validation Log"
section), and phases can still silently drop a spec'd sub-requirement (e.g.
quote-wrapping in punctuation transform) even when all listed tests pass.
