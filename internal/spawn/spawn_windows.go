package spawn

import (
	"os"
	"syscall"
)

// Windows has no sessions or SIGTERM: a started program gets a process
// group of its own, so a Ctrl+C in the program's console doesn't reach
// it, and is ended with TerminateProcess.

func detachedAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func terminate(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
