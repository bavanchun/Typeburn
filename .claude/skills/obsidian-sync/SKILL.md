# Obsidian Sync

Sync this repo's `plans/` and `docs/` markdown files to the Obsidian vault at `Projects/Typeburn/` (one-way: repo → vault).

## Vault prefix

All files sync under `Projects/Typeburn/` in the vault:
- Repo `plans/foo/plan.md` → vault `Projects/Typeburn/plans/foo/plan.md`
- Repo `docs/project-roadmap.md` → vault `Projects/Typeburn/docs/project-roadmap.md`
- Repo `docs/journals/foo.md` → vault `Projects/Typeburn/docs/journals/foo.md`

## Steps

`filepath` = relative path from repo root (e.g., `plans/foo/plan.md`). Apply all **Exclusions** rules on every file before pushing.

1. **Sync plans** — Glob `plans/**/*.md` from repo root
   - Skip files per the **Exclusions** section below
   - For each file: Read content → `mcp__obsidian__obsidian_put_content("Projects/Typeburn/" + filepath, content)`
   - Count files written

2. **Sync docs** — Glob `docs/**/*.md` from repo root
   - Skip files per the **Exclusions** section below (incl. `docs/wireframe` paths)
   - For each file: Read content → `mcp__obsidian__obsidian_put_content("Projects/Typeburn/" + filepath, content)`
   - Count files written

3. **Report**
   ```
   ✓ obsidian-sync: N plans files, M docs files → Projects/Typeburn/
   ```

## Exclusions

- `docs/wireframe/**` — binary/SVG assets, not useful in Obsidian
- Files > 500 KB — skip with warning: `⚠ skipped <path> (>500 KB)`

## Rules

- Repo is always source of truth. Overwrite vault content without prompting.
- Never write back from vault to repo.
- MOC files (`Projects/Typeburn/MOC/`) are vault-only — never touch them during sync.
- Idempotent: re-running is always safe.
