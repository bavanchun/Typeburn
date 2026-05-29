package update

import "path/filepath"

// Plan tells the CLI whether and how a self-update may proceed for the running
// binary, keeping all install-path heuristics inside this package.
type Plan struct {
	Managed     bool   // installed by a package manager (Homebrew / go install)
	Instruction string // channel-correct upgrade command, set when Managed
	Writable    bool   // install dir is writable (meaningful only when !Managed)
	Dir         string // install directory (the atomic-replace target dir)
}

// Preflight classifies the binary at execPath and probes its directory's
// writability. env is the environment lookup (pass os.Getenv in production).
// A managed install short-circuits before the writability probe — there is
// nothing to write, the user must use their package manager.
func Preflight(execPath string, env func(string) string) Plan {
	if install := classifyInstall(execPath, env); install != InstallSelfManaged {
		return Plan{Managed: true, Instruction: instructionFor(install)}
	}
	dir := filepath.Dir(execPath)
	return Plan{Writable: canWrite(dir), Dir: dir}
}
