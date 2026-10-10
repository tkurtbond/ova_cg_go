package spawn

// po_ova_cg's TestSpawn. The viewer is test/fake-viewer.sh, which
// records its arguments, parent, session, terminal and standard files
// in obj/fake-viewer.log, then runs until ended: every one started here
// is ended by its pid (Terminate), never by name. Its log is its own,
// obj/spawn-viewer.log, as package viewer's tests, which may run at the
// same time, use the fake's default.

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const (
	viewer = "../../test/fake-viewer.sh"
	log    = "../../obj/spawn-viewer.log"
	// An argument holding a blank and a "'", which a shell would need
	// quoted.
	odd = "/tmp/an O'Brien book.pdf"
)

// waitFor waits up to 5 s for cond.
func waitFor(cond func() bool) bool {
	for range 100 {
		if cond() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return cond()
}

// field is the number after word word (from 0) of s, as "ppid 1 sid 2"
// holds: field(s, 1) is 1, field(s, 3) 2; -1 if none.
func field(s string, word int) int {
	f := strings.Fields(s)
	if word >= len(f) {
		return -1
	}
	n, err := strconv.Atoi(f[word])
	if err != nil {
		return -1
	}
	return n
}

// zombies is how many of our children are zombies, by ps.
func zombies(t *testing.T) int {
	out, _, err := Capture([]string{"ps", "-o", "stat=", "--ppid", strconv.Itoa(os.Getpid())})
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(out, "Z")
}

func TestMissing(t *testing.T) {
	p, err := Detached([]string{"ova-cg-no-such-viewer", "x.pdf"})
	if err != syscall.ENOENT || p.Pid != 0 {
		t.Errorf("Detached: a missing program is ENOENT, and no process: %v %d", err, p.Pid)
	}
	if err != nil && ErrorText(err) != "No such file or directory" {
		t.Errorf("ErrorText: ENOENT's text: %q", ErrorText(err))
	}
	if p.Alive() || p.Terminate() {
		t.Error("a failed start is neither alive nor terminated")
	}
	if _, err := Detached([]string{"/etc/passwd"}); err == nil {
		t.Error("Detached: a file that isn't a program fails")
	}
	if _, err := Detached(nil); err != syscall.EINVAL {
		t.Errorf("Detached: no program at all is EINVAL: %v", err)
	}
}

func TestDetached(t *testing.T) {
	if err := os.MkdirAll("../../obj", 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_VIEWER_LOG", log) // the fake runs in this directory too
	os.Remove(log)
	p, err := Detached([]string{viewer, "-p", "62", odd, "", "$HOME *.pdf; `x` %f|&"})
	if err != nil || p.Pid <= 0 {
		t.Fatalf("Detached: the fake viewer starts: %v", err)
	}
	defer p.Terminate()
	if !waitFor(func() bool { _, err := os.Stat(log); return err == nil }) {
		t.Fatal("the fake viewer wrote its record")
	}
	b, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(b), "\n")
	want := []string{"argc 5", "arg <-p>", "arg <62>", "arg </tmp/an O'Brien book.pdf>", "arg <>",
		"arg <$HOME *.pdf; `x` %f|&>"}
	if len(lines) < 9 || strings.Join(lines[:6], "\n") != strings.Join(want, "\n") {
		t.Fatalf("the arguments arrive exactly: a blank, a ', an empty one, shell characters:\n%s", b)
	}

	// ---- Its own session, no terminal, no zombie ----
	if sid := field(lines[6], 3); sid <= 0 || sid == Session(os.Getpid()) || sid != Session(p.Pid) {
		t.Errorf("the viewer is in a session of its own: %s", lines[6])
	}
	// Unlike the C version's double fork, the viewer stays our child
	// while it runs; a goroutine waits for it (PLAN.md, "spawn").
	if field(lines[6], 1) != os.Getpid() {
		t.Errorf("the viewer is our child, waited for: %s", lines[6])
	}
	if lines[7] != "tty ?" {
		t.Errorf("the viewer has no terminal: %s", lines[7])
	}
	if lines[8] != "fds /dev/null /dev/null /dev/null" {
		t.Errorf("the viewer's standard input, output and error are /dev/null: %s", lines[8])
	}

	// ---- Alive and Terminate ----
	if !p.Alive() {
		t.Error("Alive: the viewer is running")
	}
	pid := p.Pid
	if !p.Terminate() || p.Pid != 0 {
		t.Error("Terminate: ends it, and the record is none")
	}
	if !waitFor(func() bool { return Session(pid) == -1 }) {
		t.Error("the terminated viewer is gone")
	}
	q, err := Detached([]string{"true"})
	if err != nil || !waitFor(func() bool { return !q.Alive() }) {
		t.Error("a program that ends by itself is no longer Alive")
	}
	if q.Terminate() {
		t.Error("Terminate: false for one already ended")
	}
	if !waitFor(func() bool { return zombies(t) == 0 }) {
		t.Error("no zombie is left behind")
	}
}

func TestCapture(t *testing.T) {
	out, status, err := Capture([]string{"sh", "-c", `printf "%s|" "$@"; echo; exit 3`, "sh", odd, ""})
	if err != nil || status != 3 || out != odd+"||\n" {
		t.Errorf("Capture: the output and the exit status: %q %d %v", out, status, err)
	}
	out, status, err = Capture([]string{"seq", "20000"})
	// seq 1 20000's bytes: 9 of 2, 90 of 3, 900 of 4, 9000 of 5, 10001 of 6.
	if err != nil || status != 0 || len(out) != 108894 || !strings.HasSuffix(out, "\n20000\n") {
		t.Errorf("Capture: output of many chunks, whole: %d %d %v", len(out), status, err)
	}
	out, status, err = Capture([]string{"true"})
	if err != nil || status != 0 || out != "" {
		t.Error("Capture: no output is empty")
	}
	out, status, err = Capture([]string{"sh", "-c", "echo x >&2; kill -TERM $$"})
	if err != nil || status != -1 || out != "" {
		t.Errorf("Capture: a signal is status -1; standard error isn't captured: %q %d %v", out, status, err)
	}
	out, status, err = Capture([]string{"ova-cg-no-such-program"})
	if err != syscall.ENOENT || status != -1 || out != "" {
		t.Errorf("Capture: a missing program is ENOENT: %v", err)
	}
	if zombies(t) != 0 {
		t.Error("Capture leaves no zombie")
	}
}

func TestOnPath(t *testing.T) {
	if !OnPath("sh") || !OnPath("/bin/sh") || !OnPath(viewer) {
		t.Error("OnPath: a program on PATH, by path, and relative")
	}
	if OnPath("ova-cg-no-such-program") || OnPath("/etc/passwd") || OnPath("") {
		t.Error("OnPath: missing, not executable, empty")
	}
}
