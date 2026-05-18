## [1.1.0] - 2026-05-19

### Added

- **Theme packs:** six new color themes selectable in Settings —
  `solarized-dark`, `solarized-light`, `dracula`, `nord`, `gruvbox-dark`,
  `gruvbox-light` (bringing the total to eight, with `default` and `mono`).
  `solarized-light` and `gruvbox-light` are the first light themes. `NO_COLOR`
  behavior is unchanged.
- **Persistence-failure notice:** if saving a result or settings to disk
  fails, a dismissible one-line notice now appears instead of the failure
  being silent. It never blocks input and clears on the next keypress.

### Changed

- Documentation corrected post-1.0: removed the stale "badges 404 until the
  first tag" note; the dependency-layering rule now describes the real
  invariant (`config`/`theme` intentionally bridge to the TUI); the word
  stream's wrap comment now matches the actual character-cell behavior.

[1.1.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.1.0
