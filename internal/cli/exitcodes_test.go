package cli

import (
	"errors"
	"fmt"
	"testing"
)

// Exit codes are a script-facing contract: a wrapper's whole point is that the
// meaning survives it. Matching on the concrete type instead of unwrapping
// collapses every documented code to ExitUsage the moment anything adds
// context, and the failure is invisible until a script silently mis-branches.
func TestExitCode_SurvivesWrapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, ExitOK},
		{"abort", abortError("cancelled"), ExitAbort},
		{"managed install", managedInstallError("use brew"), ExitManagedInstall},
		{"wrapped abort", fmt.Errorf("update: %w", abortError("cancelled")), ExitAbort},
		{"wrapped managed install", fmt.Errorf("refused: %w", managedInstallError("use brew")), ExitManagedInstall},
		{"twice-wrapped io", fmt.Errorf("a: %w", fmt.Errorf("b: %w", ioError("disk"))), ExitIO},
		{"wrapped usage", fmt.Errorf("cmd: %w", usageError("bad flag")), ExitUsage},
		{"plain error", errors.New("boom"), ExitUsage},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != tc.want {
				t.Errorf("ExitCode(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}
