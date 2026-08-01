package app

// knownAppOverflow records composed root frames that do not yet fit their
// terminal. TestAppFrameFits asserts it in both directions: a listed case must
// still overflow with exactly these measurements, and an unlisted case must
// fit. Values are the measurement only; ownership belongs in the plan.
//
// Scope note: per-screen size debt lives in the ui package's knownOverflow.
// The entries here are the ones only the root can show — the Result panel's
// height as the user meets it, the same height bleeding through a transition,
// and the persistence toast, which is placed but never bounded and so spills
// past a narrow terminal.
var knownAppOverflow = map[string]appOverflow{
	"home@72x20":                   {Lines: 0, Width: 73},
	"home@72x24":                   {Lines: 0, Width: 73},
	"home@72x30":                   {Lines: 0, Width: 73},
	"home@72x50":                   {Lines: 0, Width: 73},
	"result/persist-notice@120x20": {Lines: 29, Width: 0},
	"result/persist-notice@120x24": {Lines: 29, Width: 0},
	"result/persist-notice@200x20": {Lines: 29, Width: 0},
	"result/persist-notice@200x24": {Lines: 29, Width: 0},
	"result/persist-notice@60x20":  {Lines: 29, Width: 78},
	"result/persist-notice@60x24":  {Lines: 29, Width: 78},
	"result/persist-notice@60x30":  {Lines: 0, Width: 78},
	"result/persist-notice@60x50":  {Lines: 0, Width: 78},
	"result/persist-notice@61x20":  {Lines: 29, Width: 78},
	"result/persist-notice@61x24":  {Lines: 29, Width: 78},
	"result/persist-notice@61x30":  {Lines: 0, Width: 78},
	"result/persist-notice@61x50":  {Lines: 0, Width: 78},
	"result/persist-notice@72x20":  {Lines: 29, Width: 78},
	"result/persist-notice@72x24":  {Lines: 29, Width: 78},
	"result/persist-notice@72x30":  {Lines: 0, Width: 78},
	"result/persist-notice@72x50":  {Lines: 0, Width: 78},
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
