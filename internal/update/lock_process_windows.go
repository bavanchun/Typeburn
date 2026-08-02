//go:build windows

package update

import "os"

// processAlive reports whether pid names a running process. On Windows
// os.FindProcess opens a real process handle and fails when no such process
// exists, so a successful open is the liveness answer; the handle is released
// immediately. Unix's signal-0 probe has no Windows equivalent.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	_ = p.Release()
	return true
}
