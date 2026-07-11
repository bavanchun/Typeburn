## [2.5.1] - 2026-07-11 — Corrective Release

### Fixed

- **Go module channel**: `go install github.com/bavanchun/Typeburn/v2/cmd/typeburn@latest`
  now works correctly. The v2.5.0 release used a root module path that the Go
  proxy could not resolve for install.
- **Command binary**: moved entrypoint to `cmd/typeburn/` so `go install`
  produces a lowercase `typeburn` binary matching all other install channels.
- **Documentation**: all install, build, and update guidance now uses the
  corrected `/v2/cmd/typeburn` path.

No features, dependency changes, or breaking changes. The v2.5.0 tag and release remain untouched.
