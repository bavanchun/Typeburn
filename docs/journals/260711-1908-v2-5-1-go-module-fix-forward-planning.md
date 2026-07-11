# v2.5.1 Go Module Fix-Forward Planning

## Context

The v2.5.0 GitHub release succeeded, but public Go installation failed because
its module directive did not match major version 2. The release is immutable,
so recovery requires v2.5.1.

## Decisions

- Migrate the root module to `github.com/bavanchun/Typeburn/v2`.
- Move the command to `cmd/typeburn` so Go install preserves lowercase `typeburn`.
- Update only module/import/install/linker paths; retain repository URLs.
- Publish through protected main and fix forward; never retag v2.5.0.
- Treat public proxy-only installation as a hard post-publish gate.

## Plan

See `plans/260711-1908-v2-5-1-go-module-v2-fix-forward/plan.md` for the five-phase implementation and release sequence.
