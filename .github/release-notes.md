## [2.8.1] - 2026-08-01 — Centring Fix

### Fixed

- The Result panel is centred again. v2.8.0 added horizontal padding to the
  panel while the frame was already being centred, so the panel drifted right —
  at 200 columns it sat 79 columns from the left edge and 27 from the right.

A regression fix for v2.8.0 only. No CLI, config, storage, or release-archive
contract changes.
