package update

import (
	"os"
	"path/filepath"
	"strings"
)

// Install classifies how the running binary was installed.
type Install int

const (
	InstallSelfManaged Install = iota // install.sh, manual, scoop/choco (writable)
	InstallHomebrew                   // under a Homebrew prefix / Cellar
	InstallGo                         // under GOBIN or GOPATH/bin
)

// classifyInstall classifies the binary at execPath. Symlinks are resolved
// first (a Homebrew `bin` symlink points into the Cellar). env is the
// environment lookup (pass os.Getenv in production; inject in tests) used to
// locate the Go install directory.
func classifyInstall(execPath string, env func(string) string) Install {
	resolved := execPath
	if r, err := filepath.EvalSymlinks(execPath); err == nil && r != "" {
		resolved = r
	}
	slashed := strings.ToLower(filepath.ToSlash(resolved))
	if strings.Contains(slashed, "/cellar/") || strings.Contains(slashed, "/homebrew/") {
		return InstallHomebrew
	}
	if gobin := goBinDir(env); gobin != "" && sameDir(resolved, gobin) {
		return InstallGo
	}
	return InstallSelfManaged
}

// goBinDir resolves the directory `go install` writes to: GOBIN if set, else
// GOPATH/bin, else $HOME/go/bin (the Go default when GOPATH is unset).
func goBinDir(env func(string) string) string {
	if gobin := env("GOBIN"); gobin != "" {
		return gobin
	}
	gopath := env("GOPATH")
	if gopath == "" {
		if home := env("HOME"); home != "" {
			gopath = filepath.Join(home, "go")
		}
	}
	if gopath == "" {
		return ""
	}
	// GOPATH may be a list; the first entry is where binaries land.
	first := strings.SplitN(gopath, string(os.PathListSeparator), 2)[0]
	return filepath.Join(first, "bin")
}

// sameDir reports whether file lives directly in dir. Compared case-insensitively
// so a capital-`Typeburn` (go install) under a lowercased path still matches on
// case-insensitive filesystems (macOS APFS, Windows).
func sameDir(file, dir string) bool {
	fd := strings.ToLower(filepath.Clean(filepath.Dir(file)))
	td := strings.ToLower(filepath.Clean(dir))
	return fd == td
}

// instructionFor returns the channel-correct upgrade command for a managed
// install, or "" for a self-managed one.
func instructionFor(i Install) string {
	switch i {
	case InstallHomebrew:
		return "brew upgrade typeburn"
	case InstallGo:
		return "go install github.com/bavanchun/Typeburn/v2/cmd/typeburn@latest"
	default:
		return ""
	}
}

// canWrite reports whether dir is writable, by creating and removing a probe
// file. Used as a pre-flight check before the expensive download pipeline so a
// root-owned install dir fails fast.
func canWrite(dir string) bool {
	f, err := os.CreateTemp(dir, ".typeburn-probe-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}
