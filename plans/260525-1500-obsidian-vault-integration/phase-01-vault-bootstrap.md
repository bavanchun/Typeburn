---
phase: 1
title: "Vault Bootstrap"
status: completed
priority: P2
effort: "30m"
dependencies: []
---

# Phase 1: Vault Bootstrap

## Overview

Create the Obsidian vault directory structure at `~/Obsidian/Typeburn/` and verify the MCP obsidian server can write to it. No Obsidian-specific plugin config yet — just the folder skeleton and a smoke-test write.

## Requirements

- Functional: vault directory exists with `plans/`, `docs/`, `MOC/` subdirs; MCP can write a test file
- Non-functional: idempotent — re-running must not destroy existing vault content

## Architecture

```
~/Obsidian/Typeburn/          ← Obsidian vault root (NOT in repo)
├── plans/                    ← synced from repo plans/*/plan.md + phase-*.md
├── docs/
│   └── journals/             ← synced from repo docs/journals/*.md
├── MOC/                      ← Obsidian-only (kanban.md, dashboard.md)
└── .obsidian/                ← created by Obsidian app; never synced to repo
```

The MCP obsidian server (`mcp__obsidian__*`) is already connected in Claude Code sessions. Vault path must match the path registered in the MCP server config.

## Related Code Files

- Create: none (vault is outside repo)
- Verify: MCP server config — confirm vault path matches `~/Obsidian/Typeburn/`

## Implementation Steps

1. Check the MCP obsidian server's registered vault path:
   - Use `mcp__obsidian__obsidian_list_files_in_vault` — if it succeeds, vault path is already configured
   - Note the actual vault root returned; use that path everywhere in Phase 2
2. Create subdirectories via MCP if they don't exist:
   - Write a sentinel file `plans/.keep`, `docs/.keep`, `MOC/.keep` via `mcp__obsidian__obsidian_put_content`
   - Writing a file implicitly creates parent dirs in Obsidian vaults
3. Smoke test: write `MOC/README.md` with content:
   ```markdown
   # Typeburn — Project Management Vault
   Source of truth: ~/Codes/My-projects/Typeburn (repo)
   Sync direction: repo → vault (one-way, on-demand)
   Plugins needed: Dataview, Kanban, Templater
   ```
4. Confirm file appears via `mcp__obsidian__obsidian_get_file_contents` on `MOC/README.md`

## Success Criteria

- [ ] `mcp__obsidian__obsidian_list_files_in_vault` returns without error
- [ ] `MOC/README.md` readable via MCP after write
- [ ] Vault path confirmed and documented for Phase 2

## Risk Assessment

| Risk | Mitigation |
|------|-----------|
| MCP vault path mismatch | Read actual path from MCP list response; don't hardcode `~/Obsidian/Typeburn/` |
| Obsidian app not open | MCP server may need Obsidian running — verify or document prerequisite |
