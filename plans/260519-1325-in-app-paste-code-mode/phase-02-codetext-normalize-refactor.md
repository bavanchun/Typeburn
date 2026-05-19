---
phase: 2
title: "codetext Normalize Refactor"
status: pending
priority: P1
effort: "1.5h"
dependencies: [1]
---

# Phase 2: codetext Normalize Refactor (TDD)

## Overview
Expose the existing normalize/validate core as an exported
`codetext.Normalize(string) (string, error)` so an in-memory paste reuses the
identical rules/caps as `--text`. `Load(path)` behaviour stays byte-identical.
One commit.

## Requirements
- Functional: `Normalize(s string) (string, error)` applies the SAME pipeline
  as `Load`: strip leading UTF-8 BOM (bytes `EF BB BF`), CRLF→LF, trim
  exactly one trailing `\n`, `ErrEmpty` (empty/whitespace-only),
  `ErrBinary` (NUL or invalid UTF-8), `ErrTooLarge` (>10000 runes OR >500
  lines). `Load` and `Normalize` share one core (no rule divergence).
- Non-functional: package stays pure stdlib; `Load`'s existing tests pass
  UNCHANGED (regression lock); file <200 LOC.

## Architecture
Current: `Load(path)` → `loadReader(io.Reader)` (unexported core that does
BOM/binary/CRLF/trim/empty/caps on the read bytes). Refactor: keep
`loadReader` as the byte/reader core; add exported `Normalize(s string)`
that wraps the same core via `strings.NewReader(s)` (or factor the
post-read string pipeline into a shared unexported `normalize([]byte)` both
call). No behaviour change to `Load`.

## Related Code Files
- Modify: `internal/codetext/codetext.go` (add `Normalize`, share core),
  `internal/codetext/codetext_test.go` (add Normalize cases + a
  Load/Normalize parity test)
- Create/Delete: none

## Implementation Steps (tests-first)
1. **RED:** add tests: `Normalize` mirrors every existing `loadReader` case
   (CRLF, BOM, one-trailing-nl, interior blanks/tabs preserved, unicode,
   ErrEmpty/ErrBinary/ErrTooLarge, at-cap boundaries); plus a **parity
   test**: for a representative set of inputs, `Normalize(s)` ==
   `loadReader(strings.NewReader(s))` (same result & error). Run → red
   (undefined `Normalize`).
2. **GREEN:** add `Normalize`; refactor so it and `Load` share the exact
   post-read pipeline (extract `normalize([]byte)` if cleaner). Do NOT alter
   `Load`'s signature/behaviour.
3. Confirm the pre-existing `Load`/`loadReader` tests pass UNCHANGED.
4. `make fmt && make lint && make test-race`. Commit:
   `refactor(codetext): export Normalize sharing the Load core`.

## Success Criteria
- [ ] `Normalize` exported; identical rules/caps/sentinels to `Load`.
- [ ] Parity test green; all prior codetext tests green unchanged.
- [ ] Package still pure stdlib; gofmt/vet/`-race` green; file <200 LOC.
- [ ] One commit.

## Risk Assessment
- Subtle divergence if the string path skips a byte-level step (BOM is
  byte-level `EF BB BF`) — the parity test + reusing the same core prevents
  it; assert a BOM-prefixed string normalizes identically via both paths.
- Accidental `Load` behaviour change — pre-existing tests are the lock; do
  not edit them.
