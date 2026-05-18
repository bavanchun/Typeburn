## [1.0.1] - 2026-05-18

### Fixed

- **New-best precision:** the ★ personal-best badge compared rounded integer
  WPM, so a strictly faster run that rounded to the same integer (e.g. 75.4 vs
  75.0) did not earn the badge. New-best detection now compares the full-precision
  net WPM. History records written by v1.0.0 (which lack the new field) fall
  back to their stored rounded WPM, so existing personal bests are preserved.

### Removed

- **`missed` stat:** the result screen showed a `missed` counter that was always
  `0` (the metrics package never received the target text to compute it). The
  unusable field and its display were removed; no real metric is affected.
[1.0.1]: https://github.com/bavanchun/Typeburn/releases/tag/v1.0.1
