## [2.5.0] - 2026-07-11

### Added

- **Strict typing mode** — an optional letter-strict typing mode:
  - Block wrong keypresses: the cursor freezes and does not advance past an incorrect character; the user must type the correct character to proceed.
  - Log blocked error keystrokes: they are recorded in the keystroke log and correctly reduce the typing accuracy.
  - Keystroke-level accuracy metric: strict runs compute and save `KeystrokeAccuracy` (based on total non-backspace forward keystrokes) rather than final-state accuracy.
  - Excluded from bests: strict runs are saved in history but excluded from personal best (★) records.
  - Settings TUI toggle and CLI config key (`typeburn config set strict_mode on|off`) persisted in `settings.json` (backward-compatible).
- **Punctuation and numbers toggles** — optional persisted Settings controls that
  add commas, periods, capitalization, and numeric tokens to Words/Time target
  generation; Quote and Code targets remain unchanged.

### Fixed

- Code records now use the `code` label rather than being displayed as Quote
  records in History and Result metadata.
- Code and Strict records consistently remain ineligible for personal-best (★)
  markers across result detection and History display.
- Settings now represents a CLI-persisted Code default without a missing length
  value or a misleading Time selection.

[2.5.0]: https://github.com/bavanchun/Typeburn/releases/tag/v2.5.0
