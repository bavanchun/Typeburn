---
phase: 2
title: "Codetext Loader"
status: pending
priority: P1
effort: "3h"
dependencies: [1]
---

# Phase 2: Codetext Loader (TDD)

## Overview
New pure `internal/codetext` package: read a file or stdin (`-`), normalize,
enforce a size cap, return `(string, error)`. Keeps `internal/words` /
`internal/typing` I/O-free (layering rule). One commit.

## Requirements
- Functional: `Load(path string) (string, error)` — `path=="-"` reads
  `os.Stdin`; else reads the file. Normalize: CRLF→LF; strip leading UTF-8
  BOM; trim exactly one trailing `\n`; reject empty/whitespace-only and
  non-UTF-8/binary (contains NUL or invalid runes) with a descriptive error;
  cap at **10000 runes AND 500 lines** — over cap → error (no truncation;
  caller shows a notice). Tabs/newlines are PRESERVED (full-literal).
- Non-functional: no dependency on `config`/`ui`/`typing`; stdlib only;
  file <200 LOC; deterministic + table-tested.

## Architecture
`package codetext`. `Load` injectable for tests: keep a small
`loadReader(io.Reader) (string,error)` core that `Load` calls (file/stdin
just pick the reader) so normalization is tested without touching the FS.
Error values: sentinel `var ErrEmpty, ErrTooLarge, ErrBinary error` (wrapped
with context) so callers can branch on cause.

## Related Code Files
- Create: `internal/codetext/codetext.go`, `internal/codetext/codetext_test.go`
- Modify/Delete: none

## Implementation Steps (tests-first)
1. **RED:** write `codetext_test.go` table covering: CRLF→LF; BOM stripped;
   one trailing `\n` trimmed (and only one — `"a\n\n"`→`"a\n"`); interior
   blank lines & tabs preserved; empty / whitespace-only → `ErrEmpty`;
   NUL byte / invalid UTF-8 → `ErrBinary`; >10000 runes → `ErrTooLarge`;
   >500 lines → `ErrTooLarge`; exactly-at-cap passes; stdin path via
   `loadReader(strings.NewReader(...))`; a real temp file via `Load(tmp)`.
   Run → compile-fail/red.
2. **GREEN:** implement `codetext.go` minimally to pass.
3. Refactor for clarity; keep <200 LOC.
4. `make fmt && make lint && make test-race`. Commit:
   `feat(codetext): file/stdin loader with literal-safe normalization`.

## Success Criteria
- [ ] All table cases green; FS-independent core tested via reader.
- [ ] words/typing import graph unchanged (no new I/O there) — `go list`
  shows `codetext` importing only stdlib.
- [ ] gofmt/vet/`-race` green; file <200 LOC.
- [ ] One commit.

## Risk Assessment
- Trailing-newline rule ambiguity: spec is "trim exactly one final `\n`";
  test `"a\n\n"`→`"a\n"` and `"a"`→`"a"` pin it.
- Binary detection false-positives: use "contains NUL or
  `utf8.Valid==false`"; document; test a UTF-8 snippet with unicode stays OK.
- Cap chosen 10k runes/500 lines — both enforced (whichever first).
