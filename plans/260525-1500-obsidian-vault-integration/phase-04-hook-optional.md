---
phase: 4
title: "Hook — Auto-sync on Write/Edit"
status: pending
priority: P3
effort: "30m"
dependencies: [2]
---

# Phase 4: Hook — Auto-sync on Write/Edit

## Overview

Optional PostToolUse hook that auto-triggers `/ck:obsidian-sync` (or a targeted single-file sync) whenever Claude Code writes or edits a file under `plans/` or `docs/`. Eliminates the need to manually invoke the sync skill after every plan/journal update.

## Requirements

- Functional: after any `Write` or `Edit` tool call on `plans/**/*.md` or `docs/**/*.md`, push that file to the vault
- Non-functional: fast (single-file push, not full resync); non-blocking (hook failure must not break the primary write)

## Architecture

Claude Code hooks live in `.claude/settings.json` under `hooks.PostToolUse`. The hook fires a shell command that calls a Python or shell script to push the changed file via MCP.

```json
"hooks": {
  "PostToolUse": [
    {
      "matcher": "Write|Edit|MultiEdit",
      "hooks": [
        {
          "type": "command",
          "command": ".claude/scripts/obsidian-sync-file.sh \"$TOOL_INPUT_FILE_PATH\""
        }
      ]
    }
  ]
}
```

`obsidian-sync-file.sh` logic:
1. Check if `$1` matches `plans/**/*.md` or `docs/**/*.md` (skip everything else)
2. If match: call the MCP obsidian put_content via Claude Code's MCP bridge — or write a small Python script using the obsidian REST API if the MCP server exposes HTTP
3. Exit 0 always (hook failure must not surface as an error)

**Alternative (simpler):** Instead of a shell script, configure the hook to run the `/ck:obsidian-sync` skill as a follow-up prompt. This is heavier (full resync) but zero new code.

## Related Code Files

- Modify: `.claude/settings.json` — add PostToolUse hook entry
- Create: `.claude/scripts/obsidian-sync-file.sh`

## Implementation Steps

1. Decide hook strategy (single-file push vs full resync):
   - Single-file: build `obsidian-sync-file.sh` that uses the Obsidian Local REST API plugin (`http://localhost:27123`)
   - Full resync: configure hook to invoke `/ck:obsidian-sync` skill (no new script, but slower)
2. Add hook entry to `.claude/settings.json`
3. Test: edit a plan file via Claude Code → verify vault copy updates within seconds
4. Verify hook failure (Obsidian not running) exits 0 and doesn't interrupt Claude Code

## Success Criteria

- [ ] Edit to `plans/*/plan.md` triggers vault update automatically
- [ ] Edit to `docs/journals/*.md` triggers vault update automatically
- [ ] Hook failure (vault unreachable) does not error or block Claude Code
- [ ] Non-plan/docs files are not synced

## Risk Assessment

| Risk | Mitigation |
|------|-----------|
| Obsidian Local REST API not installed | Fallback to full `/ck:obsidian-sync` on-demand; hook is explicitly optional (P3) |
| Hook adds latency to every Write | Single-file push is fast (<200ms); full resync hook should be background (`&`) |
| `.claude/settings.json` hook format changes | Test after any Claude Code update |
