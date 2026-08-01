## [2.8.0] - 2026-08-01 — Result Responsive Layout

### Changed

- **Result screen now uses the width it has.** The panel is capped and centred
  instead of stretching to fill the terminal, so a wide screen no longer shows a
  near-empty box with everything crammed into the top-left corner.
- The WPM chart fills its panel. It previously took its width from the data, so
  an eight-second test drew an eight-cell chart no matter how much room it had.
  Short runs now stretch across the full plot; long runs still downsample. No
  additional samples are synthesized; the connecting line between two measured
  seconds simply gets more pixels than it had.
- The chart's right-hand error axis is omitted when the run had no errors,
  instead of drawing a column of zeroes beside a clean result.
- `raw` and `consistency` are no longer printed twice. They stay in the hero as
  headline stats; the stats grid keeps `test type`, `characters`, and `time`,
  now as one aligned column.

### Fixed

- The Result panel under-counted its own border by two columns, which limited
  how much width any section could safely use.

No CLI, config, storage, or release-archive contract changes. Existing history
and settings files are fully compatible.
