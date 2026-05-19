## [1.4.0] - 2026-05-19

### Fixed

- **Theme and other Settings changes now apply immediately.** Changing the
  theme (or blink cursor, or default mode / length) on the Settings screen
  previously had no visible effect until you quit and relaunched — the new
  value was written to disk but never applied to the running session.
  Settings changes now take effect live across every screen, and the
  Settings row you just changed stays selected.

### Changed

- **The typing text uses more of a wide terminal and is vertically
  centered.** On wide terminals the typing line was capped at 80 columns
  and anchored to the top with the footer pinned to the very bottom,
  leaving the text small and lost in empty space. It now scales to about
  82% of the terminal width (never narrower than before, and it grows with
  the screen) and the whole block is centered, so the text fills the
  screen more comfortably. Narrow and mid-size terminals are unchanged.
  (On-screen character size itself is set by your terminal's font, not the
  app.)

[1.4.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.4.0
