package app

// knownAppOverflow records composed root frames that do not yet fit their
// terminal. TestAppFrameFits asserts it in both directions: a listed case must
// still overflow with exactly these measurements, and an unlisted case must
// fit. Values are the measurement only; ownership belongs in the plan.
//
// Scope note: per-screen size debt lives in the ui package's knownOverflow.
var knownAppOverflow = map[string]appOverflow{}
