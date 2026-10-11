package prefs

// Each test has a file of its own, in a temporary directory: never the
// user's preferences.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrefs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ova_cg_go", "ova_cg_go.prefs")
	p := Open(path)
	if p.Get("rulebook") != "" {
		t.Error("no file: nothing kept")
	}
	odd := "/tmp/an O'Brien = 100% \\ book\n.pdf"
	if err := p.Set("rulebook", odd); err != nil {
		t.Fatalf("Set: written, its directory made: %v", err)
	}
	if err := p.Set("viewer", "  zathura -P %l %f "); err != nil {
		t.Fatal(err)
	}
	q := Open(path)
	if q.Get("rulebook") != odd {
		t.Errorf("a value with =, \\ and a line feed read back: %q", q.Get("rulebook"))
	}
	if q.Get("viewer") != "zathura -P %l %f" {
		t.Errorf("Get: trimmed: %q", q.Get("viewer"))
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Error("no .tmp left")
	}
	if err := os.WriteFile(path, []byte("junk\n=x\nviewer=evince\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := Open(path); r.Get("viewer") != "evince" || r.Get("") != "" || r.Get("rulebook") != "" {
		t.Error("lines that aren't name=value ignored")
	}
	bad := Open(filepath.Join(t.TempDir(), "file", "x.prefs"))
	os.WriteFile(filepath.Dir(bad.Path()), nil, 0o644) // its directory is a file
	if bad.Set("viewer", "mupdf") == nil || bad.Get("viewer") != "mupdf" {
		t.Error("a file that can't be written: an error, the value kept for this run")
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/x/config")
	if DefaultPath() != "/x/config/ova_cg_go/ova_cg_go.prefs" {
		t.Error("under XDG_CONFIG_HOME")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/u")
	if DefaultPath() != "/home/u/.config/ova_cg_go/ova_cg_go.prefs" {
		t.Error("else under ~/.config")
	}

	// Windows: %AppData%, from os.UserConfigDir, unless XDG_CONFIG_HOME
	// is set.
	defer func(g string, u func() (string, error)) { goos, userConfigDir = g, u }(goos, userConfigDir)
	goos = "windows"
	userConfigDir = func() (string, error) { return `C:\Users\u\AppData\Roaming`, nil }
	if DefaultPath() != filepath.Join(`C:\Users\u\AppData\Roaming`, "ova_cg_go", "ova_cg_go.prefs") {
		t.Error("on Windows, under %AppData%")
	}
	t.Setenv("XDG_CONFIG_HOME", "/x/config")
	if DefaultPath() != "/x/config/ova_cg_go/ova_cg_go.prefs" {
		t.Error("on Windows, under XDG_CONFIG_HOME if it is set")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	userConfigDir = func() (string, error) { return "", os.ErrNotExist }
	if DefaultPath() != "/home/u/.config/ova_cg_go/ova_cg_go.prefs" {
		t.Error("on Windows with no %AppData%, under ~/.config")
	}
}
