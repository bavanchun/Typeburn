package app

// knownAppOverflow records composed root frames that do not yet fit their
// terminal. TestAppFrameFits asserts it in both directions: a listed case must
// still overflow with exactly these measurements, and an unlisted case must
// fit. Values are the measurement only; ownership belongs in the plan.
//
// Scope note: per-screen size debt lives in the ui package's knownOverflow.
// What is left here is the one thing only the root can show — the Result
// panel's height as the user meets it, on its own, with the persistence toast
// present, and bleeding through a transition. Every entry is a height; nothing
// spills sideways any more.
var knownAppOverflow = map[string]appOverflow{
	"result/persist-notice@120x20": {Lines: 29, Width: 0},
	"result/persist-notice@120x24": {Lines: 29, Width: 0},
	"result/persist-notice@200x20": {Lines: 29, Width: 0},
	"result/persist-notice@200x24": {Lines: 29, Width: 0},
	"result/persist-notice@60x20":  {Lines: 29, Width: 0},
	"result/persist-notice@60x24":  {Lines: 29, Width: 0},
	"result/persist-notice@61x20":  {Lines: 29, Width: 0},
	"result/persist-notice@61x24":  {Lines: 29, Width: 0},
	"result/persist-notice@72x20":  {Lines: 29, Width: 0},
	"result/persist-notice@72x24":  {Lines: 29, Width: 0},
	"result/persist-notice@80x20":  {Lines: 29, Width: 0},
	"result/persist-notice@80x24":  {Lines: 29, Width: 0},
	"result/persist-notice@88x20":  {Lines: 29, Width: 0},
	"result/persist-notice@88x24":  {Lines: 29, Width: 0},
	"result@120x20":                {Lines: 29, Width: 0},
	"result@120x24":                {Lines: 29, Width: 0},
	"result@200x20":                {Lines: 29, Width: 0},
	"result@200x24":                {Lines: 29, Width: 0},
	"result@60x20":                 {Lines: 29, Width: 0},
	"result@60x24":                 {Lines: 29, Width: 0},
	"result@61x20":                 {Lines: 29, Width: 0},
	"result@61x24":                 {Lines: 29, Width: 0},
	"result@72x20":                 {Lines: 29, Width: 0},
	"result@72x24":                 {Lines: 29, Width: 0},
	"result@80x20":                 {Lines: 29, Width: 0},
	"result@80x24":                 {Lines: 29, Width: 0},
	"result@88x20":                 {Lines: 29, Width: 0},
	"result@88x24":                 {Lines: 29, Width: 0},
	"transition/early@120x20":      {Lines: 29, Width: 0},
	"transition/early@120x24":      {Lines: 29, Width: 0},
	"transition/early@200x20":      {Lines: 29, Width: 0},
	"transition/early@200x24":      {Lines: 29, Width: 0},
	"transition/early@60x20":       {Lines: 29, Width: 0},
	"transition/early@60x24":       {Lines: 29, Width: 0},
	"transition/early@61x20":       {Lines: 29, Width: 0},
	"transition/early@61x24":       {Lines: 29, Width: 0},
	"transition/early@72x20":       {Lines: 29, Width: 0},
	"transition/early@72x24":       {Lines: 29, Width: 0},
	"transition/early@80x20":       {Lines: 29, Width: 0},
	"transition/early@80x24":       {Lines: 29, Width: 0},
	"transition/early@88x20":       {Lines: 29, Width: 0},
	"transition/early@88x24":       {Lines: 29, Width: 0},
}
