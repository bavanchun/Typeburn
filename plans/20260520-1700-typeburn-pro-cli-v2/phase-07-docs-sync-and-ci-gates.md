---
phase: 7
title: "docs sync and CI gates"
status: completed
priority: P1
effort: "3h"
dependencies: [2, 3, 4, 5, 6]
---

# Phase 7: Docs sync and CI gates

## Overview

Document the new surface, formalize the dep-policy revision, and add CI gates
that prevent regressions (size growth + cross-platform raw-mode integration).

## Requirements

- All user-facing docs reflect the new CLI surface
- CLAUDE.md "no new deps" rule formally revised with explicit allow-list
- CI runs the same `make test-race / lint / size-check` on linux + macOS
- Binary size growth bounded by an asserted threshold

## Architecture

```
.github/workflows/
  ci.yml                # extend: macOS-13 runner; add `make size-check`
Makefile                # add `size-check` target with platform-specific limit
docs/
  cli-reference.md      # NEW: full subcommand reference + exit codes
README.md               # update "Usage" section with new surface
CONTRIBUTING.md         # add dep-policy revision note + completions/man-page TODO
CHANGELOG.md            # v2.0.0 entry
CLAUDE.md               # revise "Conventions & Constraints" → permit charm + x/* deps
docs/codebase-summary.md  # add internal/cli + internal/cli/notui + internal/cli/output + internal/runner
docs/system-architecture.md  # update architecture diagram & dep layering
```

## Related Code Files

- Modify: `README.md`, `CONTRIBUTING.md`, `CHANGELOG.md`, `CLAUDE.md`, `Makefile`, `.github/workflows/ci.yml`, `docs/codebase-summary.md`, `docs/system-architecture.md`, `docs/project-roadmap.md`
- Create: `docs/cli-reference.md`
- Create: `internal/theme/names_sync_test.go` (asserts theme.Names() == Settings.Normalize accept-set == CLI theme allow-list)

## Implementation Steps

1. Write `docs/cli-reference.md`:
   - Subcommand table (run/history/version/config/replay)
   - Per-subcommand flag table + exit codes
   - Examples (the 10 acceptance criteria + more)
   - JSON shape reference for `--json` modes
2. Update `README.md` "Usage" section to point to `docs/cli-reference.md` and show 5 representative examples.
3. **Pre-step (F15 measurement spike):** before locking the size cap, build the binary once
   after Phase 6 lands and record actual size. Per researcher-01: cobra ≈ +900KB, fang glue ≈ +300KB,
   x/term ≈ +50KB, new CLI code ≈ +500KB → estimate ~7.25 MB on top of ~5.5 MB current.
   If actual ≤ 7.5 MB, lock cap at **8 MB** (8388608). If between 7.5–8 MB, raise cap to **10 MB**
   (10485760) WITH the actual measurement and rationale committed in `Makefile` comment.

4. Add `make size-check` target (cap value from step 3):

   ```make
   SIZE_LIMIT ?= 8388608   # see size-check rationale comment above
   size-check: build
   	@actual=$$(stat -f%z bin/typeburn 2>/dev/null || stat -c%s bin/typeburn) ; \
   	if [ $$actual -gt $(SIZE_LIMIT) ]; then \
   		echo "binary $$actual > $(SIZE_LIMIT)" >&2; exit 1; \
   	fi
   ```
5. Update `.github/workflows/ci.yml`:
   - macOS is ALREADY in the matrix as `macos-latest` (F14 verified at `ci.yml:16`). Pin to `macos-13` ONLY IF the implementer wants cost stability and accepts the documented rationale; otherwise leave as `macos-latest`. Pick one; do not silently change.
   - Add `make size-check` step after `make test-race` (runs on both OS).
   - Add `make notui-noexit-check` step (from Phase 5).
   - Confirm `release.yml` is byte-identical post-change per the CI-vs-release separation rule in CLAUDE.md.
6. Update `CLAUDE.md` "Conventions & Constraints" → replace "No new dependencies for core behavior without strong justification" with:

   > Allowed deps: stdlib, `charm.land/*`, `github.com/charmbracelet/*`, `github.com/spf13/cobra`, `golang.org/x/*`. Anything else requires explicit user OK on a per-dep basis.

7. Add `internal/theme/names_sync_test.go` asserting the 3-way consistency: `theme.Names()` == `Settings.Normalize` accept-set == CLI theme allow-list.
8. Append `CHANGELOG.md` **v2.0.0** entry (validate-7 — major bump). Summary: new top-level CLI surface; back-compat preserved (no breaking changes for v1.5 users).
9. Update `docs/codebase-summary.md` with new packages and dep policy. Update `docs/system-architecture.md` dep diagram.
10. Update `docs/project-roadmap.md`: mark "Pro CLI v2" complete; add a M4 "shell completions + man page in archives" follow-up.
11. F12 — add to `CONTRIBUTING.md` a "Keystroke schema versioning" subsection:
    > Adding a `typing.Keystroke` field with a JSON tag is non-breaking IFF the field is `omitempty` or has a sane zero value AND old logs decode safely. Anything else (renaming a field, changing semantics, removing a field) bumps `schema_version` in `replay` input and adds an explicit migration path in `cmd_replay.go`.
12. Document SIGKILL/parent-death limitation in `docs/cli-reference.md` (F18): "If Typeburn's `--no-tui` process is killed via SIGKILL or its parent terminal disconnects, the terminal may be left in raw mode. Run `reset` or `stty sane` to recover. This is an OS-level limitation; SIGKILL cannot be caught."

## Success Criteria

- [ ] `docs/cli-reference.md` covers every subcommand + flag + exit code + SIGKILL caveat
- [ ] README "Usage" section accurate vs implementation
- [ ] Size measurement spike recorded in Makefile comment; `SIZE_LIMIT` value justified
- [ ] `make size-check` passes locally and in CI
- [ ] `make notui-noexit-check` passes (grep guard from Phase 5)
- [ ] CI matrix retains ubuntu-latest + macOS (latest or pinned 13 with rationale)
- [ ] `release.yml` byte-identical post-change
- [ ] `CLAUDE.md` dep rule updated with explicit allow-list
- [ ] `names_sync_test.go` passes
- [ ] CHANGELOG v2.0.0 entry merged
- [ ] CONTRIBUTING.md has "Keystroke schema versioning" + "Dep upgrade gate" subsections
- [ ] No teatest goldens broken

## Risk Assessment

- **Risk:** macOS CI runner cost / queue time.
  **Mitigation:** Run macOS only on PR + main; not on every dev push. `if: github.event_name == 'pull_request'` guard.
- **Risk:** `stat` flag differs `-f%z` (BSD) vs `-c%s` (GNU).
  **Mitigation:** Pattern above tries BSD first then falls back.
- **Risk:** Size limit becomes a maintenance burden.
  **Mitigation:** Document the limit + rationale in CONTRIBUTING; raise deliberately with a commit message that explains why.
- **Risk:** `release.yml` accidentally gets the new macos runner & breaks release infra.
  **Mitigation:** Per CLAUDE.md, `release.yml` is sacred — touch ONLY `ci.yml`. Reviewer must confirm.
