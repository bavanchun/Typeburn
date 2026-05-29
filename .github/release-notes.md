## [2.2.0] - 2026-05-29

### Added

- Per-key error heatmap. The Result screen now shows a "most missed:" line
  listing the keys fumbled most during the test (top 8, case-folded, with
  corrected mistakes counted). The CLI surfaces the same data: `typeburn run`
  and `typeburn replay` with `--json` gain a `key_misses` array, and the
  metrics table gains `most_missed_*` rows. Computed post-hoc from the
  keystroke log — no persistence and no history schema change.

[2.2.0]: https://github.com/bavanchun/Typeburn/releases/tag/v2.2.0
