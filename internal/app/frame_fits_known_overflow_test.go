package app

// knownAppOverflow records composed root frames that do not yet fit their
// terminal. TestAppFrameFits asserts it in both directions: a listed case must
// still overflow with exactly these measurements, and an unlisted case must
// fit. Values are the measurement only; ownership belongs in the plan.
//
// Scope note: per-screen size debt lives in the ui package's knownOverflow.
// What is left here is the persistence toast, which is placed but never
// bounded and so spills past a narrow terminal.
var knownAppOverflow = map[string]appOverflow{
	"home@72x20":                  {Lines: 0, Width: 73},
	"home@72x24":                  {Lines: 0, Width: 73},
	"home@72x30":                  {Lines: 0, Width: 73},
	"home@72x50":                  {Lines: 0, Width: 73},
	"result/persist-notice@60x20": {Lines: 0, Width: 78},
	"result/persist-notice@60x24": {Lines: 0, Width: 78},
	"result/persist-notice@60x30": {Lines: 0, Width: 78},
	"result/persist-notice@60x50": {Lines: 0, Width: 78},
	"result/persist-notice@61x20": {Lines: 0, Width: 78},
	"result/persist-notice@61x24": {Lines: 0, Width: 78},
	"result/persist-notice@61x30": {Lines: 0, Width: 78},
	"result/persist-notice@61x50": {Lines: 0, Width: 78},
	"result/persist-notice@72x20": {Lines: 0, Width: 78},
	"result/persist-notice@72x24": {Lines: 0, Width: 78},
	"result/persist-notice@72x30": {Lines: 0, Width: 78},
	"result/persist-notice@72x50": {Lines: 0, Width: 78},
}
