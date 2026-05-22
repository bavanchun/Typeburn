# Six Audited Defects Fixed in One Session

**Date**: 2026-05-22 17:00
**Severity**: High
**Component**: CLI (notui runner, version check, result render, update cache), core metrics
**Status**: Resolved

Six independent audited defects fixed and committed to `feat/pro-cli-v2`:

- **Phase 1 (notui-perf)**: Extracted `metrics.LiveWPM()` into pure logic. Fixed O(n²) regression where every keystroke was replaying entire log instead of forward-only computation.
- **Phase 2 (version-json-error)**: Fixed `version --json --check-update` double-emitting on error. Was calling `enc.Encode()` then `return checkErr`, printing raw error after JSON object.
- **Phase 3 (stripansi-csi)**: Rewrote `stripANSI` boolean flag to 3-state machine (stNorm/stEsc/stCSI). Non-SGR CSI sequences (cursor, erase, scroll) never reached `m`-byte final; `inEsc` stayed true and ate visible runes.
- **Phase 4 (update-cache)**: `Check()` prerelease branch discarded computed `Result` instead of caching. Eliminated repeated network hits on every TUI launch during prerelease windows.
- **Phase 5 (notui-reader)**: Rewrote `discardEscape` to use blocking `ReadByte`. Split-read ESC at buffer boundary left stray `[` in input stream. Added `UnreadByte` guard for Ctrl-C/Ctrl-D.
- **Phase 6 (globals-race)**: Wrapped 3 package-level test-seam vars in `sync.Mutex` accessor pairs. Full `make test-race` now clean.

**Key decisions**: `LiveWPM` in pure `internal/metrics` (not UI) — both notui runner and TUI share same formula without cycles. `getCheckFn()` return type uses `context.Context` — required stdlib `"context"` import. Zero new external dependencies; all fixes stdlib-only.

**CI gates all green**: 15/15 packages pass race test, `go vet` clean, `gofmt` empty.
