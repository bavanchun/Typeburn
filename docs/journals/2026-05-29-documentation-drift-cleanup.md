# Documentation Drift Cleanup v2.1.2

**Date**: 2026-05-29 15:00
**Severity**: Medium
**Component**: Documentation / Metadata
**Status**: Resolved

## What Happened

Performed comprehensive code audit on Typeburn @ v2.1.2 during brainstorm session. Ran real verification: `go test ./... -race` (all GREEN), `gofmt -l` (empty), `go vet` (clean). Found codebase mature + well-tested (coverage: runner/theme 100%, metrics 94%, typing 96%, codetext 92%, ui 87%; cli/notui lowest at 52.3%). Zero TODO/FIXME/HACK in code.

Audit identified 3 documentation-drift issues that would mislead future LLM sessions + plan maintenance. Triaged + fixed all three via PR #28 (chore/docs-drift-cleanup, squash-merged to main, commit 02dcf749).

## The Brutal Truth

The codebase itself is clean. The problem: documentation was out of sync with shipped reality. CLAUDE.md still claimed v1.0.1 feature (M2 new-best-precision) was a "tracked fast-follow" when it shipped 18 months ago. Plan frontmatter showed all 12 plans as `pending`/`in_progress` despite all being merged into releases. This drift doesn't hurt users, but it breaks the reliability of the docs for LLM-guided onboarding and future maintenance. An AI reading stale plan statuses might do redundant work or misunderstand the project's actual scope.

## Technical Details

**Fixed three drift issues:**

1. **CLAUDE.md:75 (M2 claim)** — "M2 new-best-precision is the tracked fast-follow" was written for v1.1.0 planning but feature shipped in v1.0.1. Removed stale claim, clarified as "shipped release history + deferred backlog".

2. **docs/project-roadmap.md (Go version floor)** — Said "Minimum Go: 1.26+" but go.mod explicitly requires 1.25.0 since v1.5.0 lowered the floor (2025-10-11). Corrected to "Go 1.25+".

3. **plans/ frontmatter (all 12 plans)** — Four plans had stale status:
   - `plans/2025-11-27-theme-packs-v1.1.0/plan.md`: `pending` → `completed` (shipped 2025-12-15)
   - `plans/2025-12-20-code-mode-v1.2.0/plan.md`: `pending` → `completed` (shipped 2026-01-22)
   - `plans/2026-01-10-update-check-cli-v2.1.0/plan.md`: `pending` → `completed` (shipped 2026-02-03)
   - `plans/2026-03-14-obsidian-vault-integration/plan.md`: `in_progress` → `completed` (merged PR #26, 2026-05-28)

All 12 plans now read `completed` and sync with Git commit history.

## What We Tried

Read every file in docs/ + plans/frontmatter. Cross-checked against git log, release tags, CHANGELOG, and merged PRs. Updated in-repo and verified via grep.

## Root Cause Analysis

Plan status hygiene wasn't part of the ship runbook. After each release, the commit+tag was pushed but plan.md frontmatter wasn't updated. CLAUDE.md carried forward outdated feature notes from early planning. Over time, three separate drift sources accumulated.

## Lessons Learned

For a mature, well-tested codebase with many shipped releases:
- **Plan metadata must be refreshed as part of the ship/merge workflow**, not post-hoc. Consider adding a pre-merge checklist item: "update all touched plan frontmatter to `completed`".
- **Documentation drift is a quality issue for LLM sessions.** An AI reading stale metadata will waste time re-researching closed decisions or re-implementing shipped features. Worth the small overhead to keep plan statuses and feature claims in sync with Git history.
- **Audit reveals false positives + opportunities.** No 🔴 critical issues found, but 3 🟡 drift problems + deferred 🟢 improvements (cli/notui coverage 52.3%, app/model.go = 204 LOC borderline, dormant feature backlog). Prioritize drift-hygiene before greenfield work.

## Next Steps

- **Drift-hygiene into ship runbook:** Add pre-merge checklist → "update plan.md frontmatter + CLAUDE.md feature claims if touched".
- **Deferred opportunities** (low priority, no blocker):
  - Improve cli/notui test coverage (currently 52.3%).
  - Split app/model.go if new screens push it past 200 LOC.
  - Audit dormant feature backlog (M2+ scope, some pre-planning complete but no recent velocity).

## Unresolved Questions

None. All identified drift is fixed. Future drifts prevented by runbook amendment.
