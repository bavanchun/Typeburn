## [2.6.0] - 2026-07-31 — Result Screen Redesign

### Changed

- **Result screen redesigned (Monkeytype-style)**:
  - Hero shows two big numbers — ASCII big-digit WPM beside a prominent
    accuracy block; `raw` and `consistency` move to a secondary card row.
    The `★ new best` badge stays on WPM.
  - The bar sparkline is replaced by a dual-axis line graph: a braille
    sub-cell WPM line (left Y-axis) with red `x` per-second error markers
    (right Y-axis, consuming previously unused per-second error data);
    long runs downsample so the chart always fits the panel.
  - Character counts and test metadata merge into a two-column stats grid
    (test type / raw / characters | consistency / time) that stacks
    vertically on narrow terminals.
  - The most-missed-keys heatmap is unchanged. Reveal animation and
    `NO_COLOR`/mono layout invariants are preserved.

No CLI, config, storage, or release-archive contract changes. Existing
history and settings files are fully compatible.
