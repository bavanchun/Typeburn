package update

import (
	"path/filepath"
	"testing"
)

func TestClassifyInstall(t *testing.T) {
	// env returns values from a fixed map; absent keys → "".
	envFrom := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}

	cases := []struct {
		name string
		exec string
		env  map[string]string
		want Install
	}{
		{"local bin self-managed", "/home/u/.local/bin/typeburn", nil, InstallSelfManaged},
		{"usr local self-managed", "/usr/local/bin/typeburn", nil, InstallSelfManaged},
		{"apple silicon brew", "/opt/homebrew/Cellar/typeburn/2.2.0/bin/typeburn", nil, InstallHomebrew},
		{"intel brew", "/usr/local/Cellar/typeburn/2.2.0/bin/typeburn", nil, InstallHomebrew},
		{"go install via GOBIN", "/home/u/devbin/Typeburn", map[string]string{"GOBIN": "/home/u/devbin"}, InstallGo},
		{"go install via GOPATH", "/home/u/go/bin/Typeburn", map[string]string{"GOPATH": "/home/u/go"}, InstallGo},
		{"go install via HOME default", "/home/u/go/bin/Typeburn", map[string]string{"HOME": "/home/u"}, InstallGo},
		{"GOBIN set but elsewhere", "/home/u/.local/bin/typeburn", map[string]string{"GOBIN": "/home/u/devbin"}, InstallSelfManaged},

		// lowercase binary name — go install ./cmd/typeburn produces "typeburn"
		{"go install lowercase via GOBIN", "/home/u/devbin/typeburn", map[string]string{"GOBIN": "/home/u/devbin"}, InstallGo},
		{"go install lowercase via GOPATH", "/home/u/go/bin/typeburn", map[string]string{"GOPATH": "/home/u/go"}, InstallGo},
		{"go install lowercase via HOME default", "/home/u/go/bin/typeburn", map[string]string{"HOME": "/home/u"}, InstallGo},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := classifyInstall(c.exec, envFrom(c.env)); got != c.want {
				t.Errorf("classifyInstall(%q) = %d, want %d", c.exec, got, c.want)
			}
		})
	}
}

func TestInstructionFor_V2Contract(t *testing.T) {
	const want = "go install github.com/bavanchun/Typeburn/v2/cmd/typeburn@latest"
	if got := instructionFor(InstallGo); got != want {
		t.Errorf("instructionFor(InstallGo) = %q, want %q", got, want)
	}
}

func TestInstructionFor(t *testing.T) {
	if instructionFor(InstallHomebrew) == "" || instructionFor(InstallGo) == "" {
		t.Error("managed installs must have an upgrade instruction")
	}
	if instructionFor(InstallSelfManaged) != "" {
		t.Error("self-managed install must have no instruction")
	}
}

func TestCanWrite(t *testing.T) {
	dir := t.TempDir()
	if !canWrite(dir) {
		t.Error("temp dir should be writable")
	}
	if canWrite(filepath.Join(dir, "does-not-exist")) {
		t.Error("nonexistent dir should not be writable")
	}
}
