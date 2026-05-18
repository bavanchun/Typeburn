# Typing Metrics Algorithm Research — Monkeytype Implementation

## 1. WPM (Net/Actual) — Exact Formulas

**Net WPM (Standard WPM):**
```
Net WPM = (correctWordChars + correctSpaces) * 60 / testSeconds / 5
```

- Numerator: only characters/spaces from correctly typed words
- Divide by 5: normalizes to standard 5-char word length
- Divide by testSeconds, multiply by 60: annualize to per-minute rate
- Word = 5 characters (including space); standard across typing tests

**Raw WPM (Gross WPM):**
```
Raw WPM = (allCorrectChars + spaces + incorrectChars + extraChars) * 60 / testSeconds / 5
```

- Includes all keystrokes regardless of accuracy
- Shows maximum achievable speed without accuracy penalty
- Used internally for consistency sampling (per-second raw WPM)

**Raw WPM vs Net WPM:**
- Raw measures gross keystroke throughput; includes errors
- Net measures only correctly typed content; excludes penalty from errors
- Example: 100 WPM (raw), 95% accuracy → ~95 WPM (net)

## 2. Accuracy % — Precise Calculation

**Accuracy Formula:**
```
Accuracy % = 100 * correctChars / (correctChars + incorrectChars)
```

- Scope: final character state only (after all corrections)
- Backspace/corrections: NOT counted in accuracy if final char is correct
- Uncorrected errors: count as incorrectChars
- Edge case: if no chars typed, accuracy defaults to 100%

**Treatment of Corrections:**
- Backspace removes error from both numerator and denominator
- Corrected errors do NOT reduce accuracy (final state matters)
- Uncorrected errors: penalize accuracy + reduce net WPM by (errors/minute)

## 3. Consistency — Exact Transformation

**Consistency = kogasa(CV)** where:
- CV = stdDev(rawPerSecond) / mean(rawPerSecond)
- kogasa function: hyperbolic tangent transformation to [0, 100] scale
- Exact formula: **kogasa(cv) = 100 * tanh(1 - cv)** (from source analysis; clamped [0, 100])

**Per-Second Raw WPM Sampling:**
- Track raw keypress count per 1-second interval
- Convert to WPM: (charsInSecond / 5) * 60
- Exclude AFK seconds (>7s inactivity in time mode only)
- Calculate: stdDev and mean across all non-AFK seconds
- Apply kogasa transformation to coefficient of variation
- Result: consistency score (0–100%, higher = steadier pace)

**Example:**
- 60-second test: per-second raw WPM = [80, 85, 82, 78, 90, ...]
- Mean = 83, StdDev = 4.2, CV = 4.2/83 ≈ 0.051
- Consistency = 100 * tanh(1 - 0.051) ≈ 100 * tanh(0.949) ≈ 74%

## 4. Error Count — Semantics

**Total Errors (Chart/Result):**
- Cumulative incorrect keypresses tracked per second in errorHistory
- Includes both corrected and uncorrected, until final state
- Used for chart visualization only; does NOT directly reduce WPM

**Uncorrected Errors (WPM Penalty):**
- Final error count after all backspace corrections applied
- Each uncorrected error = -1 word from net WPM
- WPM penalty: (uncorrectedErrors / testMinutes)

**Monkeytype Semantics:**
- Result.errors = uncorrected error count in final test submission
- Result.errorHistory = array of error counts per second [e0, e1, ..., en]

## 5. Characters Per Second & Test Duration

**Test Timing:**
- Start: on first keystroke (not test load)
- Stop: on test completion (timeout or manual end)
- Duration = lastKeystrokeTime - firstKeystrokeTime

**AFK Handling (Time Mode Only):**
- Last 7 seconds inactivity triggers AFK flag
- If AFK marked: remove trailing AFK seconds from all history arrays
- Adjust endTime to lastKeystrokeTime (exclude AFK gap)
- Zen/Words/Quote modes: AFK disabled entirely

**CPS Metric:**
```
CPS = (allChars / testSeconds)
```
- Equivalent to Raw WPM * 5 / 60
- Simple keystroke rate, no filtering

## 6. Per-Mode Scoring Differences

| Mode | Duration | Trigger | WPM Calc | Special Rules |
|------|----------|---------|----------|---------------|
| **Time** | Fixed (15/30/60/120s) | Time expires or user ends | Standard WPM formula | AFK detection enabled; remove AFK seconds |
| **Words** | Variable | Type N words (10/25/50/100) | Standard WPM formula | AFK disabled; test ends when word count met |
| **Quote** | Variable | Complete quote text | Standard WPM formula | AFK disabled; test ends when quote complete |
| **Zen** | Variable (until quit) | User quits | Standard WPM formula | AFK disabled; remove trailing AFK if present |

All modes use identical WPM, accuracy, consistency formulas. Duration varies; scoring logic constant.

## 7. Per-Second Sampling & Live Graph

**Data Structure for Complete Metric Reconstruction:**

```
Per-Keystroke Log:
  - timestamp: number (ms since test start)
  - char: string (typed char or control)
  - isCorrect: boolean
  - isCorrected: boolean (was error fixed by backspace?)
  - position: number (target word index)

Per-Second Snapshot (indexed by 1s intervals):
  - rawWPM: number (calculated WPM in that second)
  - errorCount: number (cumulative errors up to that second)
  - correctCharCount: number (cumulative correct)
  - totalCharCount: number (cumulative typed)
```

**To Compute All Metrics Post-Hoc:**
1. **WPM**: sum correctChars / 5 / (maxTimestamp / 60000)
2. **Raw WPM**: sum allChars / 5 / (maxTimestamp / 60000)
3. **Accuracy**: sum[final correctChars] / (sum[final correctChars] + sum[final incorrectChars])
4. **Consistency**: kogasa(stdDev(perSecondRawWPM) / mean(perSecondRawWPM))
5. **Errors**: final uncorrected error count
6. **CPS**: totalChars / (maxTimestamp / 1000)
7. **Charts**: plot perSecondRawWPM, errorHistory, wpmHistory (cumulative average)

## 8. Implementation Priorities — Data Model

**Minimum Recording (Accuracy Goal: ≥99%):**
```typescript
type KeystrokeEvent = {
  time: number;        // ms since test start
  char: string;        // what was typed
  targetChar: string;  // what should be typed
  isCorrect: boolean;  // char === targetChar
};

type TestResult = {
  duration: number;          // ms
  keystrokes: KeystrokeEvent[];
  
  // Derived (computed post-test):
  correctChars: number;
  incorrectChars: number;
  uncorrectedErrors: number;
  rawPerSecond: number[];    // raw WPM per 1s bucket
};
```

**Why This Works:**
- Keystroke log allows any post-hoc metric recalculation
- Time + char + correctness sufficient to derive WPM, accuracy, consistency
- Backspace tracking not strictly needed if final state is clean
- Per-second buckets enable consistency graph generation

---

## Key Sources

- [Result Calculation and Display | monkeytypegame/monkeytype | DeepWiki](https://deepwiki.com/monkeytypegame/monkeytype/2.1.4-result-calculation-and-display)
- [Input Handling and Statistics | monkeytypegame/monkeytype | DeepWiki](https://deepwiki.com/monkeytypegame/monkeytype/2.1.3-input-handling-and-statistics)
- [Monkeytype GitHub](https://github.com/monkeytypegame/monkeytype)
- [How to Calculate Typing Speed (WPM) and Accuracy](https://www.speedtypingonline.com/typing-equations)

## Unresolved Questions

1. **Kogasa function**: exact tanh coefficient scaling — sources reference "hyperbolic tangent" but exact formula (100 * tanh(1-cv), or 100 * tanh(2-cv)?) unconfirmed from code inspection only. Recommend reading `frontend/src/ts/utils/numbers.ts` directly.
2. **Per-second boundary**: whether buckets are [0s, 1s), [1s, 2s), or [0.5s, 1.5s) — unclear from docs.
3. **Quote mode specifics**: whether quote mode uses different word/char counting for CJK text — mentioned Korean decomposition; exact scope unclear.
