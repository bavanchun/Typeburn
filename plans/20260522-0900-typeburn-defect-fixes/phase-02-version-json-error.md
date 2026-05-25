---
phase: 2
title: "version-json-error"
status: pending
priority: P1
effort: 30m
dependencies: []
---

# Phase 2: version-json-error (MEDIUM-2)

## Overview

`renderVersionCheckJSON` (`internal/cli/cmd_version.go:75-127`) handles `typeburn version --json --check-update`. On a check error it does two conflicting things (`cmd_version.go:104-105`):

```go
_ = enc.Encode(out)   // writes {"version":..,"update_check":{"error":".."}} to stdout
return checkErr        // propagated to root.Execute()
```

`newVersionCmd` sets `SilenceErrors: true` (`cmd_version.go:21`), so cobra does **not** print the returned error itself. But `root.Execute()` returns it to the top-level entrypoint, which maps a non-nil error to a **non-zero exit code** (and the root may emit prose on stderr). Net effect for a machine consumer: the error is already fully described as JSON on stdout, yet the process **exits non-zero** and may duplicate the message on stderr — contradictory. The JSON contract says "the error is data, not a process failure."

**Fix:** the human path (`renderVersionCheckHuman`, `cmd_version.go:136-137`) already treats a check error as *informational* — prints to stderr and returns `nil` (exit 0). The JSON path must be symmetric: emit the error **inside** the JSON and return `nil`.

## Requirements

### Functional
- `version --json --check-update` on check error: write `{"version":{..},"update_check":{"error":"<msg>"}}` to **stdout** and exit **0**.
- No error prose on stderr for this path (the error is in the JSON).
- Success path and `result == nil` (dev-skip) path: unchanged (they already `return enc.Encode(out)` at lines 117, 126).
- Human path: unchanged.

### Non-functional
- Single-line change in `cmd_version.go`. No new symbols.
- Backwards compat: the JSON *shape* is unchanged — only the **exit code / stderr** behavior changes (check error no longer propagated). This is the intended contract fix.

## Architecture

### Current (broken) control flow — JSON error branch
```
checkErr != nil:
  enc.Encode({version, update_check:{error}})   // stdout, valid JSON
  return checkErr                                // → root maps to exit!=0 (+ possible stderr)
```

### Target
```
checkErr != nil:
  return enc.Encode({version, update_check:{error}})   // stdout JSON, exit 0
```

`enc.Encode` itself can fail (rare: closed pipe). Returning *its* error is correct — that is a genuine I/O failure, distinct from the *check* error which is now in-band data.

## Related Code Files

**Modify**
- `internal/cli/cmd_version.go` — `renderVersionCheckJSON`, error branch (lines 104-105).

**Modify (tests)**
- `internal/cli/cmd_version_test.go` — add `TestVersionCheckUpdate_JSONError`.

**Delete** — none.

## Implementation Steps

1. In `renderVersionCheckJSON`, change the `checkErr != nil` branch tail (lines 104-105):
   ```go
   // before
       _ = enc.Encode(out)
       return checkErr
   // after
       return enc.Encode(out)
   ```
   The `out` struct (version + `update_check:{error}`) is unchanged.

2. Add `TestVersionCheckUpdate_JSONError` to `cmd_version_test.go`, mirroring `TestVersionCheckUpdate_JSONWrapper` (verified pattern at test:122-148) and `_Error` (test:107-120):
   ```go
   func TestVersionCheckUpdate_JSONError(t *testing.T) {
       orig := checkFn
       checkFn = stubCheck(nil, errors.New("network unreachable"))
       defer func() { checkFn = orig }()

       var out, errOut bytes.Buffer
       // JSON mode: error is data → exit 0, no stderr prose.
       if err := versionRoot(t, &out, &errOut, "version", "--json", "--check-update"); err != nil {
           t.Fatalf("json check error path should exit 0, got: %v", err)
       }
       if errOut.Len() != 0 {
           t.Errorf("expected empty stderr, got:\n%s", errOut.String())
       }
       var got map[string]any
       if err := json.Unmarshal(out.Bytes(), &got); err != nil {
           t.Fatalf("invalid JSON: %v\noutput: %s", err, out.String())
       }
       uc, ok := got["update_check"].(map[string]any)
       if !ok {
           t.Fatalf("update_check missing/not object: %v", got["update_check"])
       }
       if uc["error"] != "network unreachable" {
           t.Errorf("expected error string in update_check.error, got: %v", uc["error"])
       }
   }
   ```
   Note: `versionRoot` wires `WithWriters(out, errOut)` (test:23-24). Asserting `root.Execute()` returns nil is the exit-0 proxy; `errOut.Len()==0` confirms no stderr.
   **Update LOW-4 dependency note:** after Phase 6 wraps `checkFn`, this test's `orig := checkFn` / `checkFn = ...` lines must migrate to `getCheckFn()` / `setCheckFn(...)`. Phase 6 owns that migration — write this test against the current var seam now; Phase 6 will rewrite it alongside the others.

3. `gofmt -w`, `go vet ./...`, `go test ./internal/cli/ -race -count=1`, then `make test-race`.

## Success Criteria

- [ ] JSON error branch returns `enc.Encode(out)` (not `checkErr`).
- [ ] `TestVersionCheckUpdate_JSONError` asserts exit 0, empty stderr, `update_check.error` present.
- [ ] Existing `TestVersionCheckUpdate_JSONWrapper` / `_Error` / `_UpToDate` still pass (no regression).
- [ ] `make test-race` GREEN, `go vet` clean, `gofmt -l .` empty.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Callers/CI relied on non-zero exit to detect check failure | Low | Medium | Documented contract fix: in `--json` mode the error is in-band data, matching the human path's exit-0 behavior. Note in changelog. |
| `enc.Encode` failure now silently swallowed? | None | — | Its error is returned (real I/O failure), unlike the check error which is now data. |
| Shape change breaks parsers | None | — | JSON body byte-shape unchanged; only exit code/stderr differ. |

### Rollback
`git revert` the phase commit; behavior returns to dual-emit. Isolated single-line change.

### File-ownership note
This phase and **Phase 6** both edit `internal/cli/cmd_version.go` (disjoint regions: this = `renderVersionCheckJSON` body; P6 = `checkFn` global + accessors). **Run Phase 2 before Phase 6; never in parallel.**
