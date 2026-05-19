---
phase: 4
title: "Integration Verify"
status: pending
priority: P1
effort: "2h"
dependencies: [3]
---

# Phase 4: Integration Verify

## Overview

Prove both fixes together are correct and side-effect-free via the full
`-race` gate, a `tester` subagent run, and a mandatory `code-reviewer`
subagent; update `./docs` if warranted.

## Requirements
- Functional: Fix #2 + Fix #1 coexist; no regression to typing engine,
  metrics, history, code-paste, `--text`, NO_COLOR, goldens-equivalent
  assertions.
- Non-functional: coverage not below baseline (codetext ≥90%, ui ≥86%);
  all prod files <200 LOC; no plan-artifact refs anywhere.

## Architecture

Read-only verification gates (no implementation here). `tester` and
`code-reviewer` are spawned subagents; reports saved under this plan's
`reports/`. Reviewer is given the scout summary + acceptance criteria and
must check: (a) live-apply works on the rendered model, (b) no regression in
typing/metrics/history/code-paste touchpoints, (c) no public/CLI contract
change, (d) follows existing message + width-tier patterns, (e) no new
lint/type/build errors, (f) zero plan-artifact refs in code/test
names/comments.

## Related Code Files
- Create: `plans/260519-1630-settings-live-apply-and-typing-width/reports/tester-260519-v1.4.0.md`
- Create: `plans/260519-1630-settings-live-apply-and-typing-width/reports/code-reviewer-260519-v1.4.0.md`
- Modify (if needed): `docs/system-architecture.md`, `docs/design-guidelines.md`,
  `docs/project-roadmap.md`

## Implementation Steps
1. `gofmt -l .` clean, `go vet ./...` clean, `go test ./... -race -count=1`
   all pass (record N/N + package count), `make build` success.
2. Coverage: `go test -cover ./internal/codetext/... ./internal/ui/...` ≥
   baseline; note app/typing coverage for new branches.
3. Spawn `tester` subagent → verify acceptance criteria of Phases 2 & 3,
   regression set (Load, Typing PasteMsg, code-paste routing, history,
   `--text`, NO_COLOR, non-typing screen layout unchanged). Save report.
4. Spawn `code-reviewer` subagent with the (a)–(f) checklist. Save report.
5. If reviewer flags a side effect/regression → trigger
   HARD-GATE-NO-SIDE-EFFECTS: present cause + 2–4 options via
   `AskUserQuestion`; do not silently patch.
6. If reviewer flags a real convention/rule violation (e.g. plan-artifact
   refs) → fix before merge; do not rationalise as cosmetic.
7. Spawn `docs-manager` if architecture/design docs need the message-pattern
   + width-contract notes; otherwise state "Docs impact: none".

## Success Criteria
- [ ] gofmt/vet/`-race`/build all green; coverage ≥ baseline
- [ ] `tester` report: all acceptance criteria + regression set verified
- [ ] `code-reviewer` report: 0 critical/high; (a)–(f) all pass
- [ ] Any flagged side effect resolved via user decision (not silent patch)
- [ ] Docs updated or explicitly "Docs impact: none"

## Risk Assessment
- `code-reviewer` subagent transient "Not logged in" → retry; never skip the
  gate, report it as not-yet-completed if it cannot run.
- Interaction between the two fixes (settings rebuild reconstructs typing
  model — confirm width contract still applied after a live settings change).

## Next Steps
Phase 5 (release v1.4.0) only after both gates are clean.
