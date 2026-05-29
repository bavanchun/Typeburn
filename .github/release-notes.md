## [2.1.3] - 2026-05-29

### Fixed

- `typeburn run --no-tui` in Time mode now ends automatically at the time limit.
  Previously the raw runner only checked completion after a keystroke, so a Time
  test would hang until the next keypress (indefinitely in piped/non-interactive
  runs); a keystroke arriving after the limit could also inflate the final
  metrics. The runner now completes on the clock with zero trailing input.
- `typeburn version --check-update` now prints a release URL only when it points
  at the official repository, matching the validation already applied to cached
  results.

[2.1.3]: https://github.com/bavanchun/Typeburn/releases/tag/v2.1.3
