//go:build !windows

package update

import (
	"errors"
	"os"
	"syscall"
)

// processAlive reports whether pid names a running process. Signal 0 runs the
// kernel's existence and permission checks without delivering anything; EPERM
// means the process exists but belongs to another user, which still counts as
// alive — an update lock held by root must not be reclaimed by a normal user.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}
