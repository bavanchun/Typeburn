## [2.9.0] - 2026-08-02 — Audit Remediation

An audit found that the test suite proved components agreed with themselves
rather than with reality, that untrusted input reached allocation unbounded,
and that the typing viewport was wired for only one of two code paths. This
release closes all three.

### Changed

- **Consistency scores rise slightly for everyone.** The final, partial second
  of a run was scaled as though it were whole, so a perfectly even typist could
  not reach the formula's own maximum. An even 5 keys/second run over 3.2
  seconds scored **60.08** and now scores **76.16**, the true maximum. Stored
  history is not migrated — old records keep the numbers they were saved with.
- **Strict mode counts every wrong key,** including ones you corrected. Because
  strict mode blocks the cursor on a wrong key, the old final-state count was
  almost always zero. The same run that reported 2 errors now reports 5.
- **A run you abandoned mid-test is no longer saved.** Its speed describes the
  burst you typed before walking away, not a test you took, so storing it left
  a personal best nobody could beat by typing. The run is still shown, and the
  screen says why it was not saved.
- **The Result screen was rebuilt** around a three-zone hero band and a
  comparison rail that answers "was that any good?" from history already in
  memory: personal best, delta, average of the last ten, and rank within the
  same mode and length. The panel is eight rows shorter and its ink density
  more than doubled. The character breakdown is labelled
  (`220 correct · 8 wrong · 1 extra`) instead of relying on colour that
  disappears under `NO_COLOR`, and the chart's y-axis now fits the data
  instead of always starting at zero.
- The word stream shows three lines on a 24-row terminal and up to seven on a
  tall one, instead of rendering the whole test.

### Fixed

- **The big WPM digits were wrong.** The glyph table's `0` held a copy of the
  `2` artwork, so a score of 100 rendered as `122` and any zero was drawn as a
  two. Digits 3, 6 and 9 also had rows one cell wider than their neighbours.
- **Typing lost half its frame in the default mode.** A 30-second test emitted
  47 rows into a 24-row terminal; the excess was silently dropped, taking the
  footer with it, and past roughly 950 characters the caret itself left the
  screen.
- **Words are no longer split across line breaks.** Every wrap point broke
  whatever word straddled it, not just over-long ones.
- Wide characters (CJK, emoji) are measured in terminal cells rather than
  runes, so a pasted snippet no longer runs off the screen and drags the
  frame's centring with it.
- **Concurrent instances no longer lose history.** Two processes appending 60
  records each persisted 87–96 of 120; they now persist all 120. Writes take a
  bounded advisory lock, and if the lock cannot be taken the record is written
  anyway and the app says so.
- **A corrupt history file is no longer overwritten.** It is renamed to
  `history.json.corrupt-<timestamp>` and the app points you at it, instead of
  silently replacing 200 records with one.
- A stale temp file left by a killed process no longer blocks every future save.
- **`typeburn update` survives Ctrl-C.** Interrupting a plain-text update used
  to leak a lock file that refused every later update, and recovering meant
  deleting four files you had been told about one of. Recovery is now automatic
  in a single run.
- Snippets containing control characters are rejected instead of producing a
  test that cannot be finished or escaped — `Esc` and `Backspace` are bound to
  other actions, so those characters could never be typed.
- Oversized and endless input is refused before it is allocated: `--text` on a
  400 MiB file no longer allocates 1.2 GiB first, `--text -` on a pipe that
  never closes no longer hangs, and `--words` is capped at 10 000.
- The History sparkline, the Home footer, the Settings columns and the
  persistence notice are all measured against the terminal instead of using
  fixed widths, so nothing spills or loses centring on a narrow screen.
- `typeburn replay` validates timestamps before allocating, so a log with an
  implausible time span is rejected rather than exhausting memory.
