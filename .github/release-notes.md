## [2.4.0] - 2026-05-30

### Changed

- The in-app "update available" hint on the result screen now points at
  `typeburn update` (the self-updater) instead of the check-only
  `typeburn version --check-update`.
- `typeburn update` now prints `downloading` / `verifying` / `installing`
  progress lines during the swap instead of blocking silently, and surfaces the
  release-notes URL (`Release notes: <url>`) before installing — matching the
  output of `typeburn version --check-update`.

[2.4.0]: https://github.com/bavanchun/Typeburn/releases/tag/v2.4.0
