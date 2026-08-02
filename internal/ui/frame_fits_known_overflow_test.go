package ui

// knownOverflow lists screen×size combinations that do not yet fit their
// terminal. main is protected and CI must stay green, so the debt is encoded
// here rather than left as a failing test.
//
// TestFrameFits asserts this map in BOTH directions: a listed case must still
// overflow with exactly these measurements, and an unlisted case must fit. A
// change that fixes one dimension while breaking the other therefore fails
// instead of staying "correctly listed".
//
// Values record the measurement and nothing else. Which change owns which
// entry belongs in the plan, not in the code.
//
// Measured, not transcribed: run TestFrameFits after a layout change and it
// prints a fresh literal to paste here. The goal is for this map to reach
// empty and be deleted.
var knownOverflow = map[string]overflow{}
