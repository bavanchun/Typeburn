---
phase: 4
title: "Docs sync + full verification"
status: pending
priority: P2
dependencies: [3]
---

# Phase 4: Docs sync + full verification

## Overview

Document the shipped feature and run the full CI-equivalent gate. No new
behavior — reconcile docs with what phases 1-3 actually built.

## Requirements
- Functional: docs describe strict mode accurately (semantic, default, toggle,
  accuracy treatment).
- Non-functional: full gate green; roadmap reflects reality.

## Related Code Files
- Modify: `README.md` (settings/usage — add strict mode)
- Modify: `docs/design-guidelines.md` (typing behavior / §8 keymap context if it
  documents allow-continue)
- Modify: `docs/codebase-summary.md` (typing + metrics + config notes)
- Modify: `docs/project-roadmap.md` (move strict mode to SHIPPED with version)
- Modify: `CLAUDE.md` (the metrics/typing description currently says
  "allow-continue + backspace only; no stop-on-error mode" — update to note the
  optional letter-strict mode + the `KeystrokeAccuracy` metric)
- Modify: `CHANGELOG.md` (`[Unreleased]` → new entry)

## Implementation Steps
1. Update each doc above to match the merged implementation. Be precise about:
   - default off; letter semantic; applies to all modes;
   - blocked errors counted via keystroke accuracy (final-state accuracy
     unchanged for non-strict);
   - toggle via Settings + `typeburn config strict_mode`.
2. If a release is cut, follow the existing release runbook (separate concern;
   not required to merge this phase).
3. Full gate: `make test-race && make lint` (and `make build`).
4. **Commit** (e.g. `feat(docs): document strict typing mode`). Note: this
   touches `CLAUDE.md` and `docs/` but NOT `.claude/` — `feat`/standard types are
   fine; the `chore`/`docs` ban applies only to `.claude/` file changes.
5. **On completion run `/vchun-git prc`** (branch `feat/strict-mode-p4-docs`).
   Then `ck plan check phase-04-docs-sync-full-verification`.

## Success Criteria
- [ ] All listed docs updated and internally consistent with the code.
- [ ] `make test-race`, `make lint`, `make build` all green.
- [ ] Roadmap lists strict mode as shipped; CHANGELOG has an entry.
- [ ] Phase committed; PR squash-merged via `/vchun-git prc`; CI green.

## Risk Assessment
- **Risk:** doc claims drift from actual behavior (esp. accuracy). **Mitigation:**
  write docs from the merged code, not from this plan's intent; verify the
  accuracy wording against `metrics/compute.go`.
- **Risk:** stale roadmap (as happened with vim). **Mitigation:** state the exact
  files/fields shipped, not aspirational scope.
