## [1.3.0] - 2026-05-19

### Added

- **In-app paste for Code mode:** you no longer need `typeburn --text <file>`
  to practice on your own snippet. On the Home screen, tab to **Code** and
  press enter on the empty row to open a paste screen; bracket-paste your
  snippet and it is validated and loaded in place — press enter again to
  start the test. `--text` is still supported and takes precedence (when a
  snippet is supplied that way, Code is enabled and enter starts immediately;
  the paste screen is not used that run).
- Pasted snippets go through the same normalization and caps as `--text`
  (CRLF→LF, one trailing newline trimmed, UTF-8 BOM stripped; empty,
  non-text, or oversized >10k runes / >500 lines input is rejected with a
  clear reason and you can paste again). `esc` returns to Home unchanged.
  Code runs still save to History but never count toward the ★ personal
  best. `NO_COLOR` behavior is unchanged.

[1.3.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.3.0
