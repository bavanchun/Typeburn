---
phase: 3
title: "MOC Files"
status: completed
priority: P2
effort: "30m"
dependencies: [1]
---

# Phase 3: MOC Files

## Overview

Seed two Obsidian-only Map of Content files into `MOC/` — a Kanban board and a Dataview dashboard. These are written to the vault directly (not mirrored from repo) and are never synced back.

## Requirements

- Functional:
  - `MOC/kanban.md` — Kanban plugin board with columns mapped to `status:` values
  - `MOC/dashboard.md` — Dataview queries for plans table + journal index
- Non-functional: files must be valid Obsidian Kanban / Dataview syntax; self-updating via plugin queries

## Architecture

### MOC/kanban.md

Uses the Obsidian **Kanban** plugin format. Cards are seeded manually from current plan statuses; after initial seed, the board is managed by dragging cards in Obsidian.

```markdown
---
kanban-plugin: basic
---

## Pending

- [ ] [[plans/20260521-0700-typeburn-update-check-cli/plan|update-check-cli]]
- [ ] [[plans/260518-2348-v1.1.0-theme-packs-and-hygiene/plan|v1.1.0-theme-packs]]
- [ ] [[plans/260519-0142-code-mode-custom-text/plan|code-mode-custom-text]]
- [ ] [[plans/260525-1500-obsidian-vault-integration/plan|obsidian-vault-integration]]

## In Progress

## Completed

- [ ] [[plans/20260522-0900-typeburn-defect-fixes/plan|defect-fixes-v2.1]]
- [ ] [[plans/20260520-1700-typeburn-pro-cli-v2/plan|pro-cli-v2]]
...

## Blocked
```

### MOC/dashboard.md

Uses **Dataview** plugin. Queries run live over vault files — always reflects current sync state.

````markdown
# Typeburn — Project Dashboard

## Active Plans

```dataview
TABLE status, priority, effort, tags
FROM "plans"
WHERE file.name = "plan" AND status != "completed"
SORT priority ASC
```

## All Plans

```dataview
TABLE status, priority, effort
FROM "plans"
WHERE file.name = "plan"
SORT file.mtime DESC
```

## Recent Journals

```dataview
LIST
FROM "docs/journals"
SORT file.name DESC
LIMIT 10
```
````

## Related Code Files

- Create (in vault, via MCP): `MOC/kanban.md`, `MOC/dashboard.md`

## Implementation Steps

1. Read current plan statuses from repo (`plans/*/plan.md` frontmatter `status:` field)
2. Build `MOC/kanban.md` content — seed cards into correct columns from step 1
3. Write `MOC/kanban.md` to vault via `mcp__obsidian__obsidian_put_content`
4. Write `MOC/dashboard.md` to vault via `mcp__obsidian__obsidian_put_content`
5. Verify both files readable via `mcp__obsidian__obsidian_get_file_contents`

## Success Criteria

- [ ] `MOC/kanban.md` exists in vault with correct Kanban plugin frontmatter
- [ ] `MOC/dashboard.md` exists in vault with valid Dataview query blocks
- [ ] All current pending plans appear in Kanban "Pending" column
- [ ] All completed plans appear in Kanban "Completed" column

## Risk Assessment

| Risk | Mitigation |
|------|-----------|
| Kanban plugin not installed | Document prerequisite; board degrades gracefully to plain markdown |
| Dataview query FROM path wrong | Test with `dataview` LIST query first; adjust path prefix if vault structure differs |
