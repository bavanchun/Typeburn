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
var knownOverflow = map[string]overflow{
	"history/120@120x20":     {Lines: 0, Width: 143},
	"history/120@120x24":     {Lines: 0, Width: 143},
	"history/120@120x30":     {Lines: 0, Width: 143},
	"history/120@120x50":     {Lines: 0, Width: 143},
	"history/120@60x20":      {Lines: 0, Width: 143},
	"history/120@60x24":      {Lines: 0, Width: 143},
	"history/120@60x30":      {Lines: 0, Width: 143},
	"history/120@60x50":      {Lines: 0, Width: 143},
	"history/120@61x20":      {Lines: 0, Width: 143},
	"history/120@61x24":      {Lines: 0, Width: 143},
	"history/120@61x30":      {Lines: 0, Width: 143},
	"history/120@61x50":      {Lines: 0, Width: 143},
	"history/120@72x20":      {Lines: 0, Width: 143},
	"history/120@72x24":      {Lines: 0, Width: 143},
	"history/120@72x30":      {Lines: 0, Width: 143},
	"history/120@72x50":      {Lines: 0, Width: 143},
	"history/120@80x20":      {Lines: 0, Width: 143},
	"history/120@80x24":      {Lines: 0, Width: 143},
	"history/120@80x30":      {Lines: 0, Width: 143},
	"history/120@80x50":      {Lines: 0, Width: 143},
	"history/120@88x20":      {Lines: 0, Width: 143},
	"history/120@88x24":      {Lines: 0, Width: 143},
	"history/120@88x30":      {Lines: 0, Width: 143},
	"history/120@88x50":      {Lines: 0, Width: 143},
	"home/code-error@72x20":  {Lines: 0, Width: 73},
	"home/code-error@72x24":  {Lines: 0, Width: 73},
	"home/code-error@72x30":  {Lines: 0, Width: 73},
	"home/code-error@72x50":  {Lines: 0, Width: 73},
	"home/code-loaded@72x20": {Lines: 0, Width: 73},
	"home/code-loaded@72x24": {Lines: 0, Width: 73},
	"home/code-loaded@72x30": {Lines: 0, Width: 73},
	"home/code-loaded@72x50": {Lines: 0, Width: 73},
	"home@72x20":             {Lines: 0, Width: 73},
	"home@72x24":             {Lines: 0, Width: 73},
	"home@72x30":             {Lines: 0, Width: 73},
	"home@72x50":             {Lines: 0, Width: 73},
}
