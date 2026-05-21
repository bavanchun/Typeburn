---
phase: 5
title: "Docs Sync and Release Prep"
status: pending
priority: P2
effort: "2h"
dependencies: [1, 2, 3, 4]
---

# Phase 5: Docs Sync and Release Prep

## Overview

Update every doc that mentions the CLI surface or upgrade flow; write
`CHANGELOG.md [Unreleased]` content for v2.1.0; pre-stage `.github/release-notes.md`
so the release pipeline never trips the "stale notes" trap that caught us
during the v2.0.0 dry-run.

## Requirements

**Functional**
- `README.md`: add `--check-update` flag mention in Install/Upgrade section;
  document the `update_check` config key + opt-in nature; explicitly state
  "no auto-update — manual install".
- `docs/cli-reference.md`: new flag, new config key, JSON schema for
  update-check output, exit-code policy.
- `docs/codebase-summary.md`: new `internal/update/` package paragraph in the
  per-package list; note the layering (UI-free, stdlib + storage + config).
- `docs/system-architecture.md`: update if the diagram or layering description
  needs the new package; otherwise leave alone.
- `docs/project-roadmap.md`: mark M3 / v2.1 item complete (if tracked there).
- `CHANGELOG.md`: add `[2.1.0]` section with `Added`/`Changed` bullets;
  move from `[Unreleased]` at tag time.
- `.github/release-notes.md`: pre-stage v2.1.0 body (one of the lessons from
  the v2.0.0 dry-run was a stale notes file blocking release).

**Non-functional**
- Sacrifice grammar for concision (per CLAUDE.md project rule).
- No code changes in this phase.

## Architecture

Doc precedence (CLAUDE.md): `CHANGELOG.md` is the authoritative source for
release notes — extracted verbatim to `.github/release-notes.md`. We pre-stage
the latter to match exactly, with the link reference at the bottom.

## Related Code Files

- **Modify:** `README.md`, `docs/cli-reference.md`, `docs/codebase-summary.md`,
  `docs/project-roadmap.md`, `CHANGELOG.md`.
- **Modify:** `.github/release-notes.md` (stage v2.1.0 content).
- **Maybe modify:** `docs/system-architecture.md` (if package map changes warrant).
- **Add:** `docs/journals/2026-MM-DD-update-check-v2.1.md` (post-merge, via /ck:journal).

## Implementation Steps

1. **CHANGELOG.md** — add `[Unreleased]` entries (will be promoted to `[2.1.0]`
   at tag time):
   ```markdown
   ## [Unreleased]

   ### Added

   - `typeburn version --check-update`: explicit, synchronous check against
     GitHub Releases for a newer version. Supports `--json` (wrapped output
     under `update_check` key). Honors a 1.5s hard timeout and silent-degrades
     on any network/parse error.
   - Opt-in opportunistic update check on TUI launch (`typeburn config set
     update_check on`). When enabled, a one-line "↑ vX.Y.Z available" hint
     renders on the Result screen footer after a typing session. 24h cache.
     `--no-tui` path is unaffected.
   - New config key `update_check` (bool, default `false`). Exposed via
     `typeburn config set/get/list update_check`.

   ### Changed

   - `internal/storage.atomicWrite` is now exported as `storage.AtomicWrite`
     for reuse by the new `internal/update` package. Behavior unchanged.

   ### Notes

   - No self-upgrade: the check notifies only. Install methods unchanged —
     `install.sh`, `brew upgrade`, and `go install ...@latest` are the three
     upgrade paths printed verbatim in `--check-update` output.
   - Zero new `go.mod` entries; stdlib-only implementation.
   ```
2. **`.github/release-notes.md`** — replace v2.0.0 content with the `[2.1.0]`
   section above (with `## [2.1.0] - YYYY-MM-DD` header + trailing
   `[2.1.0]: https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0` link).
3. **`README.md`** — under Install/Upgrade section, add a subsection:
   ```markdown
   ### Check for updates

   - `typeburn version --check-update` — explicit, on-demand. Adds `--json`
     for scripting.
   - Opt-in opportunistic notice on TUI launch:
     `typeburn config set update_check on`. The TUI prints a footer hint
     on the Result screen if a newer release exists. Disable any time with
     `typeburn config set update_check off`.

   The check is read-only: it reports a new version and prints upgrade
   commands; it never modifies the binary. Default is **off** to keep
   typeburn offline-first.
   ```
4. **`docs/cli-reference.md`** — add `--check-update` row to the `version`
   subcommand table; add `update_check` row to the config keys section;
   document the JSON wrapper schema:
   ```json
   {
     "version": {"version": "...", "commit": "...", "date": "..."},
     "update_check": {
       "current": "v2.0.0",
       "latest": "v2.1.0",
       "upgrade_available": true,
       "release_url": "https://...",
       "checked_at": "2026-..."
     }
   }
   ```
   Document exit codes: 0 for human; non-zero for `--json` with error.
5. **`docs/codebase-summary.md`** — under "Pure logic" list, add bullet:
   ```
   - `internal/update`: GitHub Releases API client + 24h JSON cache + stdlib
     semver comparator + orchestrator. UI-free. Stdlib + storage + config only.
   ```
6. **`docs/project-roadmap.md`** — locate the v2.1 milestone (or add one)
   and mark the update-check task complete.
7. **Pre-PR review:** grep for `update_check` and `--check-update` across
   docs to confirm every mention is consistent (snake_case for config key,
   kebab-case for flag).

## Todo List

- [ ] CHANGELOG.md [Unreleased] block
- [ ] .github/release-notes.md pre-staged for v2.1.0
- [ ] README.md Install/Upgrade "Check for updates" subsection
- [ ] docs/cli-reference.md flag + config + JSON shape
- [ ] docs/codebase-summary.md new package entry
- [ ] docs/project-roadmap.md v2.1 entry
- [ ] grep -nE 'update_check|--check-update' across all docs for consistency
- [ ] Optionally update docs/system-architecture.md if package map relevant

## Success Criteria

- [ ] Every mention of the feature in `./docs` matches the locked text in
  Phases 3-4 (footer hint text, 3 upgrade commands, JSON schema).
- [ ] `.github/release-notes.md` matches `CHANGELOG.md [2.1.0]` body verbatim
  before tagging.
- [ ] No `chore:` or `docs:` in commit messages for `.claude/` (per project
  CLAUDE.md). For `.github/release-notes.md` and `docs/`, use `docs:` or
  `feat(docs):` as fits.
- [ ] CI green.

## Risk Assessment

| Risk | Mitigation |
|---|---|
| Docs drift between `CHANGELOG.md` and `.github/release-notes.md` | Pre-stage both at the same time; grep verification step |
| Footer hint text in docs goes stale if Phase 4 changes wording | Lock text in Phase 4 plan; reference Phase 4 file in this phase |
| Roadmap doesn't track v2.1 | Add it if missing; do not silently skip |

## Security Considerations

None — docs only.

## Next Steps

- After PR merge to `main`: tag `v2.1.0` via the standard release runbook
  (dry-run `v0.0.0-rc.test` → verify 7 assets + correct notes body → delete
  → annotated v2.1.0 tag in a separate push).
- Run `/ck:journal` for session reflection + decision record.

## Red Team Review Updates (2026-05-21)

<!-- red-team-finding-8 --> **H6 — release-notes ordering hard precondition.** Phase 5 Step 2 instructs
replacing `.github/release-notes.md` content with `[2.1.0]`. CLAUDE.md release
engineering rules + plan.md Dependencies require this happens ONLY after v2.0.0
is published. Risk: if the feature branch updates `release-notes.md` before
v2.0.0 is tagged, and an unrelated v2.0.0 dry-run/tag fires from main with the
v2.1.0 content present, v2.0.0's GitHub Release publishes v2.1.0 notes — the
exact trap the v2.0.0 dry-run already caught once.

**Hard precondition added to Todo List:**

- [ ] **GATE — do not start Phase 5 Step 2 until ALL of:**
  - `v2.0.0` tag exists at `https://github.com/bavanchun/Typeburn/releases/tag/v2.0.0`.
  - The `v2.0.0` GitHub Release body matches `CHANGELOG.md [2.0.0]` (i.e., PR #18 landed).
  - Feature branch `feat/update-check-v2.1` was rebased onto `main` post-v2.0.0 tag.

If PR #18 stalls > 7 days, escalate: cut `feat/update-check-v2.1` from current `main`
HEAD anyway; defer Phase 5 docs Step 2 until v2.0.0 ships. Phase 5 CHANGELOG.md
edits must **rebase** (preserve prior `[2.0.0]` entries), not overwrite.

### Updated Step 2 wording

Replace "replace v2.0.0 content" with:
> "After the v2.0.0 tag is published, **append** the `[2.1.0]` content to
> `.github/release-notes.md` and remove the now-published `[2.0.0]` block —
> the file should always reflect the *next* release's body, not the prior one.
> Verify the rendered v2.0.0 GitHub Release page is unchanged (release.yml has
> already consumed its notes by tag time)."
