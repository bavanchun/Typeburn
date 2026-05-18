---
phase: 3
title: "Persistence Notice"
status: pending
priority: P1
effort: "3h"
dependencies: [1]
---

# Phase 3: Persistence Notice

## Overview
Surface a **non-blocking** notice when saving history/settings to disk fails,
instead of silently swallowing the error. One commit.

## Requirements
- Functional: on `AppendHistory` error (`model_history.go:40`) or
  `SaveSettings` error (`model_settings.go:24`), show a transient
  user-visible notice; do **not** crash, do **not** block input.
- Functional: notice auto-dismisses on next successful persist OR next
  screen navigation/keypress (no timers — KISS, deterministic).
- Non-functional: NO_COLOR-safe (uses theme Roles → attribute-only fallback
  automatically); layout unchanged when absent; new prod file < 200 LOC.

## Architecture
Elm flow. Add a transient field to `app.Model` (`model.go:26`), e.g.
`persistErr string` ("" = none). Capture the currently-discarded errors:
- `model_history.go`: `_, err := storage.AppendHistory(rec)` → on err set
  `m.persistErr = "Couldn't save result to disk"`.
- `model_settings.go`: `err := storage.SaveSettings(s)` → on err set
  `m.persistErr = "Couldn't save settings to disk"`.
New `ui.PersistenceNotice(msg string, th theme.Theme) string` mirrors the
`degraded-notice.go` pattern (RoleWarning headline + RoleTextFaint hint
"press any key to dismiss"). Root `View()` (`model_view.go:18`) renders it as
a single appended line (footer-adjacent) when `persistErr != ""`. Clear
`persistErr` in the key handler (`model_key_handler.go`) on any key and on
screen transition (`routing.go`/wherever screen changes) — set to "" when a
later persist succeeds too.

`SaveSettings`/`AppendHistory` signatures already return errors
(`history_store.go:57`, `settings_store.go:52`) — no storage change.

## Related Code Files
- Create: `internal/ui/persistence-notice.go`,
  `internal/ui/persistence-notice_test.go`
- Modify: `internal/app/model.go` (field), `internal/app/model_history.go`
  (capture err), `internal/app/model_settings.go` (capture err),
  `internal/app/model_view.go` (render line),
  `internal/app/model_key_handler.go` (clear on key) + app test file
- Delete: none

## Implementation Steps
1. `model.go`: add `persistErr string` to `Model` struct (documented).
2. `model_history.go`: change `_, _ = storage.AppendHistory(rec)` to
   `_, err := storage.AppendHistory(rec)` (first return stays discarded — do
   NOT introduce an unused `recs` var); `if err != nil { m.persistErr =
   "Couldn't save result to disk" }`. `handleResultMsg` is a value-receiver
   returning `Model` — the mutated `m` is already returned, so the notice
   propagates; just ensure the existing `return m` carries it. Drop the
   "ignore write errors" comment.
3. `model_settings.go`: `if err := storage.SaveSettings(s); err != nil {
   m.persistErr = "Couldn't save settings to disk" }`. Update comment.
   On success path elsewhere, leave persistErr untouched (cleared on key).
4. `internal/ui/persistence-notice.go`: `PersistenceNotice(msg, th)` →
   `RoleWarning` msg + `RoleTextFaint` "press any key to dismiss", joined.
5. `model_view.go`: when `m.persistErr != ""`, append the notice line to the
   composed view (below footer, never overlapping the typing area; must not
   shift layout when empty).
6. `model_key_handler.go`: at top of key handling, if `m.persistErr != ""`
   set it to "" (any key dismisses; the keystroke still processes normally).
   Confirm the handler's mutated/returned `Model` is the one propagated back
   through `Update` (value-receiver pattern) so the cleared state sticks.
7. Test (`persistence-notice_test.go`): notice contains the message text;
   empty msg → callers never invoke it. App-level test: inject a failing
   storage path (XDG dir pointed at an unwritable location via env/temp) so
   `AppendHistory`/`SaveSettings` error → assert `m.persistErr` set →
   simulate a key → assert cleared. Reuse existing storage test helpers for
   the unwritable-dir setup.
8. `make fmt && make lint && make test-race`. Commit:
   `feat(app): surface non-blocking notice on persistence failure`.

## Success Criteria
- [ ] Forced disk-write failure shows a visible notice; app does not crash.
- [ ] Notice clears on next keypress and on a subsequent successful save.
- [ ] Absent notice → byte-identical layout vs before (golden tests pass).
- [ ] NO_COLOR: notice still legible (attribute-only), layout unchanged.
- [ ] `gofmt`/`vet`/`test -race` green; new file < 200 LOC.
- [ ] One commit on the feature branch.

## Risk Assessment
- **Layout shift / golden test breakage:** notice must be additive only when
  present; verify `teatest` goldens unaffected in the no-error path.
- **Forcing a write failure in tests:** use a read-only temp dir or invalid
  XDG path; reuse `internal/storage` test helpers, do not mock the storage
  API (real failure path per development-rules).
- **Cross-screen leakage:** ensure clear-on-nav so the notice doesn't persist
  forever; covered by step 6 + test.
