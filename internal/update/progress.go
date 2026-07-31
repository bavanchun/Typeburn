package update

// Stage identifies a phase of an Apply run, reported to an optional progress
// callback so a CLI front-end can show the user what is happening during the
// otherwise-silent download/verify/swap. Reporting is observational only — it
// never alters control flow or error handling.
type Stage int

const (
	// StageChecksums is reported just before the release checksums.txt is
	// fetched. This work always happened; it simply was not reported before.
	StageChecksums Stage = iota
	// StageDownloading is reported just before the release archive is fetched,
	// then repeatedly as bytes arrive.
	StageDownloading
	// StageVerifying is reported just before the SHA-256 integrity check.
	StageVerifying
	// StageInstalling is reported just before the binary is extracted and
	// swapped. From this point the run is no longer safely cancellable: the
	// remaining work is the atomic rename that the updater's safety rests on.
	StageInstalling
)

// Stages are declared in run order, so `current > s` means "stage s is already
// finished" — front-ends rely on that ordering to render completed steps.

// String returns the lowercase human label for a stage.
func (s Stage) String() string {
	switch s {
	case StageChecksums:
		return "checksums"
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

// Progress reports how far an Apply run has advanced. Stage is always
// meaningful. Done and Total are only populated during StageDownloading, and
// Total is 0 when the server sent no Content-Length — render such a run as
// indeterminate rather than computing a ratio.
type Progress struct {
	Stage       Stage
	Done, Total int64
}

// report invokes fn(p) only when fn is non-nil, so every call site can pass a
// nil reporter to stay silent without a guard of its own.
func report(fn func(Progress), p Progress) {
	if fn != nil {
		fn(p)
	}
}

// reportStage is the shorthand for a stage transition carrying no byte counts.
func reportStage(fn func(Progress), s Stage) {
	report(fn, Progress{Stage: s})
}
