# Release Recovery Research

## Verified Current State

- `main`, `origin/main`, and annotated v2.5.0 peel to
  `758dcb8c238b6f5061c102ece009545cca7348b1`.
- GitHub v2.5.0 is stable/latest with seven assets; release run 29149206457 passed.
- Proxy-only install fails deterministically because the module path lacks `/v2`.

## Recovery Contract

- Never mutate v2.5.0. Its source cannot be repaired by rerunning the workflow.
- Merge correction through protected main, freeze one SHA, and validate archive
  production with a disposable prerelease before stable tagging.
- A disposable archive release cannot prove stable proxy ingestion. Exhaustive
  local module/command checks are the pre-tag guard; clean exact and latest
  proxy-only installs are mandatory post-publish gates.
- A transient 404 within the ingestion bound permits retry. A semantic mismatch
  after publication requires v2.5.2, never a moved v2.5.1 tag.
- The old plan must close as superseded/recovered only after v2.5.1 succeeds.
