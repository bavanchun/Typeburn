## [2.4.1] - 2026-06-20

### Added

- **UI/UX animation system** — a stdlib-only terminal motion layer, always on
  and auto-adapting under `NO_COLOR`:
  - Animated typing caret: blink (530ms), freshly-typed cell fade, and a
    just-vacated trail.
  - Result screen reveal: WPM count-up (fixed-width slot, no jitter), sparkline
    draw-in, and staggered stat cards.
  - One-shot sparkle burst when a result beats the per-mode personal best.
  - A short Typing→Result transition (dim-curtain crossfade in color, a
    row-wipe under `NO_COLOR`).
  - Driven by a self-stopping ~33ms frame loop that schedules zero ticks when
    idle, independent of the existing 100ms timer. Motion never shifts layout
    (line count + rune width preserved) and every settled frame matches the
    prior static render; the typing hot path is kept cheap by a prefix-token
    cache.

[2.4.1]: https://github.com/bavanchun/Typeburn/releases/tag/v2.4.1
