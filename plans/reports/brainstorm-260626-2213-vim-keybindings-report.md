---
title: "Brainstorm — Vim motion keybindings"
date: 2026-06-26
status: superseded
supersededReason: "Feature already shipped since v1 (always-on hjkl/g/G in keymap.go, wired to Home/Settings/History). Verified 2026-06-26 during --deep scout. No implementation needed; roadmap corrected."
flags: []
next: re-brainstorm another candidate
---

> **SUPERSEDED 2026-06-26.** Deep-scout found vim keybindings already implemented
> and wired (see `internal/config/keymap.go` + the 3 nav screens). No plan was
> created. Roadmap updated to mark this as already-shipped. Kept for audit trail.

# Brainstorm — Vim motion keybindings (UX-depth, single feature)

## Problem statement

Typeburn v2.4.1 targets the terminal/Monkeytype audience but navigation is
arrow/enter/tab only. Vim-style motion is the highest value-per-effort UX-depth
addition: the centralized per-screen keymap already supports extension, audience
fit is high, risk is near-zero. User goal this round: **one tight feature,
shipped fast, handed off carefully to another agent with a clean git workflow.**

## Requirements (locked)

1. **Expected output:** new `keybinding_style` setting (`default` | `vim`).
   When `vim`, `hjkl` + `g`/`G` navigation works **in parallel** with existing
   keys on Home, History, Settings. Toggle via Settings screen **and**
   `typeburn config`. Persisted in `settings.json`.
2. **Acceptance criteria:**
   - Arrows/enter/tab keep working in both styles.
   - `vim`: `j/k` down/up, `h/l` left/right, `g/G` top/bottom on the 3 screens.
   - Typing/Result/CodePaste unaffected — `j/k` still type / paste literally.
   - Setting persists + reloads; missing key in old `settings.json` → `default`.
   - `typeburn config` reads/writes the key; invalid value rejected.
   - `go test ./... -race`, `go vet`, `gofmt -l` all clean; teatest goldens green.
3. **Scope boundary (OUT):** vim on Typing/Result/CodePaste; count prefixes
   (`5j`); arbitrary remap; `:` command mode; replace-mode (parallel only).
4. **Non-negotiable constraints:** pure-logic packages stay UI-free; keymap in
   `internal/config`; protected `main` (PR + squash only, never direct push);
   conventional commits, no AI refs; every Go file < 200 LOC.
5. **Touchpoints (scout):** `internal/config/settings.go` (field + normalize),
   `internal/config` keymap binders, `internal/ui/screen_home*.go`,
   `screen_history*.go`, `screen_settings*.go`, `internal/cli/cmd_config.go`,
   teatest golden files.

## Approaches evaluated

| Candidate | Effort | Value | Verdict |
|-----------|--------|-------|---------|
| **Vim keybindings** | ~1d | High (audience fit, arch ready) | ✅ Chosen |
| More themes (Catppuccin/Tokyo Night) | hrs | Low (already 8) | Rejected — filler |
| Custom wordlists from file | ~2d | Medium | Rejected — Code mode covers most need; adds I/O+UI |

### Mechanism: parallel vs replace
- **Parallel (chosen):** arrows + vim both active under `vim`. Easy to learn, no
  muscle-memory break, backward-compat. KISS.
- Replace: purer vim but jarring for arrow users. Rejected.

### Toggle surface
- Settings screen + `typeburn config` (chosen): consistent with the existing 4
  settings and the Pro CLI config surface.

## Recommended solution

Add `KeybindingStyle` to `config.Settings` (default `"default"`). Extend the
per-screen keymaps so that, when style is `vim`, Home/History/Settings also bind
`h/j/k/l/g/G` to the same intents as the existing arrow/nav keys. Surface the
toggle in the Settings screen list and in `typeburn config get/set`. No changes
to pure-logic packages; Typing/Result/CodePaste keymaps untouched.

## Implementation considerations & risks

- **Risk — input bleed:** vim keys must NOT activate on Typing/CodePaste (would
  eat typed chars / paste). Mitigation: only the 3 nav screens consult the vim
  bindings; add explicit tests asserting `j/k` are literal in Typing.
- **Risk — golden churn:** Settings screen gains a row → teatest goldens shift.
  Mitigation: regenerate + review goldens deliberately as part of the phase.
- **Risk — backward compat:** old `settings.json` lacks the key. Mitigation:
  `Normalize` defaults missing/invalid → `default`; add a test with legacy JSON.
- **CLI parity:** `config set keybinding_style vim|default` validated against an
  accepted-set, mirroring how `theme` is validated.

## Success metrics / validation

- All acceptance criteria above verified by tests (unit + teatest).
- Manual smoke: toggle in Settings, navigate all 3 screens with vim keys, confirm
  Typing unaffected, restart app → setting persisted.

## Handoff & git-workflow notes (user priority)

Plan must be a careful handoff for a **different agent**:
- Self-contained phases with exact files, steps, acceptance, env context.
- Strict protected-main flow: branch `feat/vim-keybindings`, conventional
  commits, PR → CI green → squash-merge → branch auto-delete. Never push `main`.
- Phase ordering: config/setting + normalize (+tests) → keymap extension →
  per-screen wiring + goldens → CLI config surface → docs sync + verification.
- TDD recommended: existing teatest coverage should be locked before changing
  Settings render; tests-first per phase guards regressions.

## Next steps

`/ck:plan --tdd` with this report as input → phase plan for handoff.

## Open questions

None — requirements fully locked.
