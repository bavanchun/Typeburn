---
title: "Obsidian vault integration for project management"
description: "One-way sync of plans/ and docs/ into a separate Obsidian vault via MCP. Replaces Jira/Trello/Notion/Linear with local markdown-native project tracking."
status: in_progress
priority: P2
branch: "main"
tags: [obsidian, tooling, project-management, mcp]
blockedBy: []
blocks: []
created: "2026-05-25T08:00:59.965Z"
createdBy: "ck:plan"
source: skill
---

# Obsidian vault integration for project management

## Overview

Repo is already markdown-native with YAML frontmatter (`status`, `priority`, `effort`, `tags`, `blockedBy`, `blocks`) in all `plans/*/plan.md` files. This plan wires Obsidian as a visual project-management layer over that existing structure — no new data formats, no external services.

**Architecture:** Separate Obsidian vault at `~/Obsidian/Typeburn/`. One-way sync: repo → vault via MCP obsidian tools. Repo is the single source of truth. Vault is a read/view layer with Obsidian-specific MOC files (kanban board, Dataview dashboard).

**Scope:** `.claude/skills/obsidian-sync/` skill + MOC seed files. No repo code changes. No Go code.

## Phases

| Phase | Name | Effort | Status |
|-------|------|--------|--------|
| 1 | [Vault Bootstrap](./phase-01-vault-bootstrap.md) | 30m | Completed |
| 2 | [Obsidian Sync Skill](./phase-02-obsidian-sync-skill.md) | 1h | Completed |
| 3 | [MOC Files](./phase-03-moc-files.md) | 30m | Completed |
| 4 | [Hook — Auto-sync on Write/Edit](./phase-04-hook-optional.md) | 30m | Pending (P3, deferred) |

## Key Decisions (from brainstorm)

- **Vault = separate** (`~/Obsidian/Typeburn/`) — not repo root. Obsidian config never touches git.
- **Sync = one-way** (repo → vault). Bidirectional sync risks conflicts; deferred unless explicitly requested.
- **Trigger = on-demand** (`/ck:obsidian-sync`) first. Auto-hook added in Phase 4 if desired.
- **Kanban columns** map to existing `status:` values: `pending` | `in_progress` | `completed` | `blocked`.

## Dependencies

None — no cross-plan blockers.
