---
phase: 2
title: "Obsidian Sync Skill"
status: completed
priority: P2
effort: "1h"
dependencies: [1]
---

# Phase 2: Obsidian Sync Skill

## Overview

Build the `/ck:obsidian-sync` skill that reads all `plans/` and `docs/` markdown files from the repo and writes them to the Obsidian vault via MCP. Single command, idempotent, outputs a sync summary.

## Requirements

- Functional:
  - Sync `plans/*/plan.md` and `plans/*/phase-*.md` → vault `plans/<dir>/`
  - Sync `docs/journals/*.md` → vault `docs/journals/`
  - Sync `docs/*.md` (roadmap, codebase-summary, etc.) → vault `docs/`
  - Skip: `docs/wireframe/`, any non-`.md` files
  - Output: summary of files written/skipped
- Non-functional: idempotent; safe to run multiple times; no data loss in vault

## Architecture

The skill is a SKILL.md in `.claude/skills/obsidian-sync/` (in the **repo's** `.claude/skills/`, not `~/.claude/skills/`). It instructs Claude Code to use the MCP obsidian tools to iterate and push files.

```
.claude/skills/obsidian-sync/
└── SKILL.md        ← skill instructions for Claude Code
```

Sync logic (Claude Code executes this when skill is invoked):
1. `Glob("plans/**/*.md")` → for each: `mcp__obsidian__obsidian_put_content(path, content)`
2. `Glob("docs/**/*.md")` → filter out `wireframe/` → same put_content call
3. Print summary: `✓ synced N files`

Vault path mapping:
- Repo `plans/260522-0900-typeburn-defect-fixes/plan.md` → vault `plans/260522-0900-typeburn-defect-fixes/plan.md`
- Repo `docs/journals/20260522-defect-fixes.md` → vault `docs/journals/20260522-defect-fixes.md`
- Paths are relative in both systems; no transformation needed.

## Related Code Files

- Create: `.claude/skills/obsidian-sync/SKILL.md`

## Implementation Steps

1. Create `.claude/skills/obsidian-sync/SKILL.md` with:
   - Skill name, description, trigger (invoked via `/ck:obsidian-sync`)
   - Step-by-step sync instructions referencing the MCP tools
   - File inclusion/exclusion rules
   - Output format spec
2. Test by invoking `/ck:obsidian-sync` and verifying a sample plan appears in vault via `mcp__obsidian__obsidian_get_file_contents`

## SKILL.md Content Spec

```markdown
# Obsidian Sync

Sync repo plans/ and docs/ to the Obsidian vault (one-way, repo → vault).

## Steps

1. Glob all `plans/**/*.md` from repo root
2. For each file: Read content → mcp__obsidian__obsidian_put_content(path, content)
3. Glob all `docs/**/*.md`, exclude paths matching `docs/wireframe/`
4. For each file: Read content → mcp__obsidian__obsidian_put_content(path, content)
5. Print: `✓ synced N files to Obsidian vault`

## Exclusions
- `docs/wireframe/**`
- Any file > 500KB (skip with warning)

## Notes
- Vault path is relative — use the same relative path as in repo
- Idempotent: re-running overwrites with latest repo content (vault is never source of truth)
```

## Success Criteria

- [ ] `.claude/skills/obsidian-sync/SKILL.md` exists
- [ ] `/ck:obsidian-sync` invocation syncs at least one plan file to vault
- [ ] Vault file content matches repo file content after sync
- [ ] Re-running produces no errors

## Risk Assessment

| Risk | Mitigation |
|------|-----------|
| MCP put_content path format differs from repo paths | Test with one file first; adjust path prefix if needed |
| Large phase files hit MCP limits | Skill spec includes 500 KB skip guard |
