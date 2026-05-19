## [1.2.0] - 2026-05-19

### Added

- **Code mode:** practice typing on your own text or code. Supply it with
  `typeburn --text <file>` or pipe via `typeburn --text -`. The snippet is
  rendered with full-literal layout — real line breaks, tabs shown as two
  columns, and you type every space and indentation exactly; the test
  completes on an exact full-text match. Long snippets scroll with a
  caret-following viewport. Code mode is always listed in the mode tabs;
  without `--text` it shows a hint and is disabled (in-app paste is planned).
  `NO_COLOR` behavior is unchanged.
- Code runs are saved to History but never count toward the ★ personal
  best (custom text is not comparable run-to-run). Oversized
  (>10k runes / >500 lines), empty, or non-text input is rejected with a
  clear reason instead of starting.

[1.2.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.2.0
