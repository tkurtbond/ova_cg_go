// Package prefs keeps what the user chose between runs: the rulebook's
// path and the viewer.
//
// PLAN.md, "prefs": the part of FLTK's Fl_Preferences ova_cg_po used. A
// small file of "name=value" lines, at
// $XDG_CONFIG_HOME/ova_cg_go/ova_cg_go.prefs (else ~/.config/..., or
// %AppData%\... on Windows), written
// whole, through NAME.tmp and a rename, each time a value is set. In a
// value, "\" is "\\" and a line feed "\n"; a line that isn't a
// "name=value" is ignored.
package prefs

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Prefs is a preferences file and what it holds.
type Prefs struct {
	path   string
	values map[string]string
}

// goos and userConfigDir are runtime.GOOS and os.UserConfigDir, which
// a test replaces to take Windows' way on Linux.
var (
	goos          = runtime.GOOS
	userConfigDir = os.UserConfigDir
)

// DefaultPath is the preferences' file: $XDG_CONFIG_HOME (else
// ~/.config; on Windows, which usually sets neither, os.UserConfigDir's
// %AppData%), then ova_cg_go/ova_cg_go.prefs.
func DefaultPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" && goos == "windows" {
		if d, err := userConfigDir(); err == nil {
			dir = d
		}
	}
	if dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "ova_cg_go", "ova_cg_go.prefs")
}

var (
	escaper   = strings.NewReplacer(`\`, `\\`, "\n", `\n`)
	unescaper = strings.NewReplacer(`\\`, `\`, `\n`, "\n")
)

// Open is the preferences in the file at path: none if there is no file
// there, or it can't be read.
func Open(path string) *Prefs {
	p := &Prefs{path: path, values: map[string]string{}}
	f, err := os.Open(path)
	if err != nil {
		return p
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(nil, 1<<20)
	for s.Scan() {
		if name, value, ok := strings.Cut(s.Text(), "="); ok && name != "" {
			p.values[name] = unescaper.Replace(value)
		}
	}
	return p
}

// Path is the file's path.
func (p *Prefs) Path() string { return p.path }

// Get is what the preferences hold under name, trimmed; "" for nothing.
func (p *Prefs) Get(name string) string { return strings.Trim(p.values[name], " \t\n\r") }

// Set keeps text under name, and writes the file at once (its directory
// made if need be). An error if it couldn't be written, the value still
// kept for this run.
func (p *Prefs) Set(name, text string) error {
	p.values[name] = text
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(p.values))
	for n := range p.values {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		b.WriteString(n + "=" + escaper.Replace(p.values[n]) + "\n")
	}
	tmp := p.path + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, p.path)
}
