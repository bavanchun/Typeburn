package update

// Stage identifies a phase of an Apply run, reported to an optional progress
// callback so a CLI front-end can show the user what is happening during the
// otherwise-silent download/verify/swap. Reporting is observational only — it
// never alters control flow or error handling.
type Stage int

const (
	// StageDownloading is reported just before the release archive is fetched.
	StageDownloading Stage = iota
	// StageVerifying is reported just before the SHA-256 integrity check.
	StageVerifying
	// StageInstalling is reported just before the binary is extracted and swapped.
	StageInstalling
)

// String returns the lowercase human label for a stage.
func (s Stage) String() string {
	switch s {
	case StageDownloading:
		return "downloading"
	case StageVerifying:
		return "verifying"
	case StageInstalling:
		return "installing"
	default:
		return "unknown"
	}
}

// report invokes fn(s) only when fn is non-nil, so every call site can pass a
// nil reporter to stay silent without a guard of its own.
func report(fn func(Stage), s Stage) {
	if fn != nil {
		fn(s)
	}
}
