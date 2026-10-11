//go:build !windows

package spawn

import "syscall"

// detachedAttr puts a started program in a session of its own.
func detachedAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}

func terminate(pid int) error { return syscall.Kill(pid, syscall.SIGTERM) }
