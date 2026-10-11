// Package spawn runs other programs: the rulebook's viewer and its
// D-Bus queries.
//
// ova_cg_po's PLAN.md, "The rulebook viewer": a viewer is started
// detached, in a session of its own, so a hangup of the program's
// terminal doesn't end it, and a failed start comes back as an errno
// instead of a shell's message on the terminal. Capture runs a program
// and returns its standard output, for gdbus. A program is an argument
// vector, run without a shell, so nothing needs quoting.
//
// The port of ova_cg_po's OvaSpawn, with os/exec in place of its C part
// (PLAN.md, "spawn"). Needs no display.
package spawn

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// Process is a program Detached started: its pid, and when it started,
// which tells it from a later process given the same pid. Pid 0 for
// none.
type Process struct {
	Pid   int
	start int64
}

// errno is err as an errno: a missing program is ENOENT, one that can't
// be run EACCES; err itself if it is none of those.
func errno(err error) error {
	var e syscall.Errno
	switch {
	case errors.As(err, &e):
		return e
	case errors.Is(err, exec.ErrNotFound), errors.Is(err, fs.ErrNotExist):
		return syscall.ENOENT
	case errors.Is(err, fs.ErrPermission):
		return syscall.EACCES
	}
	return err
}

func command(args []string) (*exec.Cmd, error) {
	if len(args) == 0 {
		return nil, syscall.EINVAL
	}
	cmd := exec.Command(args[0], args[1:]...)
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil // found in a PATH entry for ".": run it, as execvp would
	}
	return cmd, nil
}

// Detached starts args' program detached: in its own session, with
// standard input, output and error on /dev/null, never left a zombie
// (a goroutine waits for it). An errno (syscall.Errno) for a failed
// start, when the Process is none.
func Detached(args []string) (Process, error) {
	cmd, err := command(args)
	if err != nil {
		return Process{}, err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return Process{}, errno(err)
	}
	p := Process{Pid: cmd.Process.Pid}
	// The program may already have ended; then start is -1, and
	// Terminate leaves the pid alone.
	p.start = startTime(p.Pid)
	go cmd.Wait()
	return p, nil
}

// Capture runs args' program and waits for it, with its standard output
// in out and its standard input and error on /dev/null. status is its
// exit status, or -1 if a signal ended it. An errno for a failed start,
// when out is "" and status -1.
func Capture(args []string) (out string, status int, err error) {
	cmd, err := command(args)
	if err != nil {
		return "", -1, err
	}
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Start(); err != nil {
		return "", -1, errno(err)
	}
	cmd.Wait()
	status = cmd.ProcessState.ExitCode() // -1 for a signal
	return b.String(), status, nil
}

// stat is field n (from 1) of pid's /proc/PID/stat; -1 if there is no
// such process.
func stat(pid, n int) int64 {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return -1
	}
	// The name, field 2, is in parentheses and may hold blanks: count
	// from the last ')'; field 3 follows it.
	s := string(b)
	i := strings.LastIndexByte(s, ')')
	if i < 0 {
		return -1
	}
	f := strings.Fields(s[i+1:])
	if n < 3 || len(f) < n-2 {
		return -1
	}
	v, err := strconv.ParseInt(f[n-3], 10, 64)
	if err != nil {
		return -1
	}
	return v
}

// startTime is when pid started, in clock ticks since boot (field 22),
// which tells a process from a later one given the same pid; -1 if
// there is no such process.
func startTime(pid int) int64 { return stat(pid, 22) }

// Alive says whether p is still running: the same process, not a later
// one given its pid.
func (p *Process) Alive() bool {
	return p.Pid > 0 && p.start >= 0 && startTime(p.Pid) == p.start
}

// Terminate ends p with SIGTERM, if it is still running, and makes it
// none. Only a process this program started is ended, never one found by
// name, which could be the user's own.
func (p *Process) Terminate() bool {
	ok := p.Alive() && syscall.Kill(p.Pid, syscall.SIGTERM) == nil
	*p = Process{}
	return ok
}

// Session is pid's session (field 6), -1 if there is no such process:
// for tests.
func Session(pid int) int { return int(stat(pid, 6)) }

// OnPath says whether name would be found and could be run: a name with
// a "/" as it is, any other in PATH.
func OnPath(name string) bool {
	if name == "" {
		return false
	}
	_, err := exec.LookPath(name)
	return err == nil || errors.Is(err, exec.ErrDot)
}

// ErrorText is err's text, as strerror gives it ("No such file or
// directory").
func ErrorText(err error) string {
	s := err.Error()
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
