# Validate Report — Typeburn Pro CLI v2

**Date:** 2026-05-20
**Mode:** /ck:plan --deep → validate gate
**Verdict:** APPROVE WITH MINOR FOLD-INS

## Critical questions interview — resolutions

| # | Question | Decision |
|---|----------|----------|
| 1 | `config set theme zzz` behavior | **Error before write**, exit 1, list valid options. NEW strict validation layer above `Settings.Normalize()`. |
| 2 | `run --mode code` without `--text` | **Error**: `--text required for code mode`. CLI refuses without snippet source. No fall-back to in-app paste. |
| 3 | `run --no-tui --mode code` | **Allow only with `--text <file>`**. Bracketed paste not supported in v2 `--no-tui`. |
| 4 | `--json` placement | **Per-subcommand**, as planned. Not a root persistent flag. |
| 5 | Mid-run `--theme nord` + Settings change | **Settings change wins**; CLI override discarded. Document in `docs/cli-reference.md`. |
| 6 | `history --json` empty case | **Print `[]` and exit 0** (valid JSON empty array). |
| 7 | Version bump | **v2.0.0** — major, even with back-compat preserved. New top-level CLI surface justifies the bump. |

## Plan fold-ins required

- **phase-03** Add explicit error: `--mode code` requires `--text`. Update flag validation table + `validateRunFlags()`.
- **phase-03** Add docs note: Settings screen change wins over `--theme` override. No code change needed (already the natural behavior since `--theme` only seeds initial theme).
- **phase-04** Tighten `config set`: validate value against per-key whitelist BEFORE calling `Settings.Normalize()`. New `validateConfigValue(key, value) error` function. Error message lists valid options.
- **phase-04** `cmd_history.go` empty-history path: emit `[]` for `--json`, "no history yet" line for table mode.
- **phase-05** Forbid `--no-tui --mode code` unless `--text` given. Validate at flag parse time.
- **phase-07** Update CHANGELOG entry to v2.0.0. Update CONTRIBUTING release runbook (already covers vN.0 path).

## Acceptance criteria additions

- AC11: `typeburn config set theme zzz` → exits 1, stderr lists valid options, settings.json unchanged
- AC12: `typeburn run --mode code` (no --text) → exits 1 with clear error
- AC13: `typeburn run --no-tui --mode code` (no --text) → exits 1 with clear error

## Open Questions

- Should `typeburn run --no-tui` print "v2.0 limitation: bracketed paste unsupported" on the first invocation? Probably YAGNI — document in `--help` instead.

---

**Status:** DONE
**Summary:** 7 user decisions locked; 6 small fold-ins required across phases 3/4/5/7. No structural changes to the plan.
