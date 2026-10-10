// Package viewer is the rulebook viewer: finding the PDF, and turning a
// viewer to a page.
//
// po_ova_cg's PLAN.md, "The rulebook viewer": reuse a viewer already
// open, start one only when none is. okular is turned over D-Bus (gdbus)
// when one shows the rulebook already, evince reuses its own window,
// mupdf can't be reused, so the one this program started last is ended
// (by its pid, never by name) before the next starts, and xdg-open can't
// turn to a page at all, so the caller is told which. A command template
// runs as given. Every program is started with package spawn, never a
// shell. Needs no display.
//
// The programs' names are variables, so the tests can put fakes in
// their place.
//
// The port of po_ova_cg's OvaViewer.
package viewer

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/tkurtbond/go_ova_cg/internal/rules"
	"github.com/tkurtbond/go_ova_cg/internal/spawn"
)

// Kind is a kind of viewer.
type Kind int

// The viewer kinds.
const (
	Okular Kind = iota
	Evince
	Mupdf
	XdgOpen
	Template
)

const (
	// defaultRulebook is the rulebook where it is looked for last,
	// under the home directory.
	defaultRulebook = "/Reference/RPG/OVA/OVA_The_Anime_Role-Playing_Game.pdf"

	// okular's interface, and its main window's object.
	okularIface = "org.kde.okular"
	okularShell = "/okularshell"
	// maxTabs is the most tabs of one okular looked through.
	maxTabs = 50
)

// Viewer is a viewer: its kind, and for a Template, the command, whose
// words' %f, %p and %l become the PDF, the page index and the label.
type Viewer struct {
	Kind     Kind
	Template string
}

// The programs run, by name on PATH.
var (
	OkularProgram  = "okular"
	EvinceProgram  = "evince"
	MupdfProgram   = "mupdf"
	XdgOpenProgram = "xdg-open"
	GdbusProgram   = "gdbus"
)

var (
	mu        sync.Mutex
	lastMupdf spawn.Process // the mupdf this program started last
)

// ---- The PDF ----

// absolute is path, made absolute against the working directory: okular
// is asked which tab shows it, and the viewers given it are run
// elsewhere.
func absolute(path string) string {
	if path != "" && path[0] != '/' {
		if cwd, err := os.Getwd(); err == nil {
			return cwd + "/" + path
		}
	}
	return path
}

// Exists says whether a file is at path.
func Exists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// Choose is the rulebook's PDF: the first of option (--rulebook), env
// (OVA_RULEBOOK), saved (the preferences) that isn't empty, else the
// default under home, made absolute; and whether a file is there.
func Choose(option, env, saved, home string) (path string, ok bool) {
	switch {
	case option != "":
		path = option
	case env != "":
		path = env
	case saved != "":
		path = saved
	default:
		path = home + defaultRulebook
	}
	path = absolute(path)
	return path, Exists(path)
}

// FindRulebook is Choose, with OVA_RULEBOOK and HOME from the
// environment.
func FindRulebook(option, saved string) (path string, ok bool) {
	return Choose(option, os.Getenv("OVA_RULEBOOK"), saved, os.Getenv("HOME"))
}

// ---- Viewers ----

// Default is the viewer used when the user hasn't chosen one: the first
// of okular, evince and mupdf on PATH, else xdg-open.
func Default() Viewer {
	switch {
	case spawn.OnPath(OkularProgram):
		return Viewer{Kind: Okular}
	case spawn.OnPath(EvinceProgram):
		return Viewer{Kind: Evince}
	case spawn.OnPath(MupdfProgram):
		return Viewer{Kind: Mupdf}
	}
	return Viewer{Kind: XdgOpen}
}

// Parse is the viewer s names: "okular", "evince", "mupdf", "xdg-open",
// or a command template, which must hold %f. false for anything else.
func Parse(s string) (Viewer, bool) {
	s = strings.Trim(s, " \t\n\r")
	switch s {
	case "okular":
		return Viewer{Kind: Okular}, true
	case "evince":
		return Viewer{Kind: Evince}, true
	case "mupdf":
		return Viewer{Kind: Mupdf}, true
	case "xdg-open":
		return Viewer{Kind: XdgOpen}, true
	}
	if strings.Contains(s, "%f") {
		return Viewer{Kind: Template, Template: s}, true
	}
	return Viewer{}, false
}

// Name is what Parse takes back: the viewer's name, or its template.
func (v Viewer) Name() string {
	switch v.Kind {
	case Okular:
		return "okular"
	case Evince:
		return "evince"
	case Mupdf:
		return "mupdf"
	case XdgOpen:
		return "xdg-open"
	}
	return v.Template
}

// addTemplate adds the words of template to a, each with %f, %p, %l and
// %% replaced by pdf, index, label and %, the replacement staying within
// its word however many blanks it holds.
func addTemplate(a []string, template, pdf string, index, label int) []string {
	var word strings.Builder
	inWord := false
	n := len(template)
	for i := 0; i <= n; i++ {
		ch := byte(' ')
		if i < n {
			ch = template[i]
		}
		if ch == ' ' || ch == '\t' {
			if inWord {
				a = append(a, word.String())
				word.Reset()
				inWord = false
			}
			continue
		}
		inWord = true
		var next byte
		if ch == '%' && i+1 < n {
			next = template[i+1]
		}
		switch next {
		case 'f':
			word.WriteString(pdf)
			i++
		case 'p':
			word.WriteString(strconv.Itoa(index))
			i++
		case 'l':
			word.WriteString(strconv.Itoa(label))
			i++
		case '%':
			word.WriteByte('%')
			i++
		default:
			word.WriteByte(ch)
		}
	}
	return a
}

// Command is the command that starts v on pdf at printed page page.
func (v Viewer) Command(pdf string, page int) []string {
	index := rules.PdfIndex(page)
	ix := strconv.Itoa(index)
	switch v.Kind {
	case Okular:
		return []string{OkularProgram, "--unique", "-p", ix, pdf}
	case Evince:
		return []string{EvinceProgram, "-i", ix, pdf}
	case Mupdf:
		return []string{MupdfProgram, pdf, ix}
	case XdgOpen:
		return []string{XdgOpenProgram, pdf}
	}
	return addTemplate(nil, v.Template, pdf, index, page)
}

// ---- okular, over D-Bus ----

// FileUrl is pdf as a file: URL for okular's openDocument, with only
// "%", "#", "?" and control characters percent-encoded: okular compares
// the URL with its tabs' as given, and its tab opened from a plain path
// has "'" not "%27", so a URL encoded more than this opens the PDF again
// in a new tab (po_ova_cg's PLAN.md, "Findings").
func FileUrl(pdf string) string {
	var b strings.Builder
	b.WriteString("file://")
	for i := 0; i < len(pdf); i++ {
		ch := pdf[i]
		if ch >= ' ' && ch != 0x7f && ch != '%' && ch != '#' && ch != '?' {
			b.WriteByte(ch)
		} else {
			fmt.Fprintf(&b, "%%%02X", ch)
		}
	}
	return b.String()
}

func hexDigit(ch byte) int {
	switch {
	case '0' <= ch && ch <= '9':
		return int(ch - '0')
	case 'A' <= ch && ch <= 'F':
		return int(ch-'A') + 10
	case 'a' <= ch && ch <= 'f':
		return int(ch-'a') + 10
	}
	return -1
}

// docPath is doc as a path: a file: URL's path, percent-decoded;
// anything else as it is.
func docPath(doc string) string {
	if !strings.HasPrefix(doc, "file://") {
		return doc
	}
	var b strings.Builder
	for i := 7; i < len(doc); {
		ch := doc[i]
		hi, lo := -1, -1
		if ch == '%' && i+2 < len(doc) {
			hi, lo = hexDigit(doc[i+1]), hexDigit(doc[i+2])
		}
		if hi >= 0 && lo >= 0 {
			b.WriteByte(byte(hi*16 + lo))
			i += 3
		} else {
			b.WriteByte(ch)
			i++
		}
	}
	return b.String()
}

// firstString is the first string in out from from, gdbus's print of a
// GVariant tuple, such as ('/a/b.pdf',) or ("/an O'Brien.pdf",): quoted
// by ' or ", with \ escapes; next, where to look for the one after. ok
// false if there is none.
func firstString(out string, from int) (s string, next int, ok bool) {
	i := from
	for i < len(out) && out[i] != '\'' && out[i] != '"' {
		i++
	}
	if i >= len(out) {
		return "", i, false
	}
	q := out[i]
	i++
	var b strings.Builder
	for i < len(out) && out[i] != q {
		ch := out[i]
		if ch == '\\' && i+1 < len(out) {
			i++
			ch = out[i]
			switch ch {
			case 'n':
				ch = '\n'
			case 't':
				ch = '\t'
			}
		}
		b.WriteByte(ch)
		i++
	}
	ok = i < len(out)
	return b.String(), i + 1, ok
}

// gdbusCall is a gdbus call on the session bus to dest's object obj, its
// method method (interface and name), with args.
func gdbusCall(dest, obj, method string, args ...string) []string {
	a := []string{GdbusProgram, "call", "--session", "--timeout", "5", "--dest", dest,
		"--object-path", obj, "--method", method}
	return append(a, args...)
}

// capture is the output of a, and whether it ran and succeeded.
func capture(a []string) (string, bool) {
	out, status, err := spawn.Capture(a)
	return out, err == nil && status == 0
}

// isOkular says whether name is an okular's on the bus: org.kde.okular
// (started --unique) or org.kde.okular-PID; not its Instance_ alias.
func isOkular(name string) bool {
	if name == okularIface {
		return true
	}
	pid, ok := strings.CutPrefix(name, "org.kde.okular-")
	if !ok || pid == "" {
		return false
	}
	for i := 0; i < len(pid); i++ {
		if pid[i] < '0' || pid[i] > '9' {
			return false
		}
	}
	return true
}

// tabObject is tab number tab's object in an okular: /okular,
// /okular2, ...
func tabObject(tab int) string {
	if tab > 1 {
		return "/okular" + strconv.Itoa(tab)
	}
	return "/okular"
}

// okularTurn looks through every okular on the session bus for a tab
// showing pdf; if one does, brings it forward and turns it to page
// index. Whether one did.
func okularTurn(pdf string, index int) bool {
	names, ok := capture(gdbusCall("org.freedesktop.DBus", "/org/freedesktop/DBus",
		"org.freedesktop.DBus.ListNames"))
	if !ok {
		return false
	}
	var dest, obj string
	found := false
	for pos, more := 0, true; more && !found; {
		var name string
		name, pos, more = firstString(names, pos)
		if !more || !isOkular(name) {
			continue
		}
		for tab := 1; tab <= maxTabs && !found; tab++ {
			obj = tabObject(tab)
			out, ok := capture(gdbusCall(name, obj, "org.kde.okular.currentDocument"))
			if ok {
				var doc string
				doc, _, ok = firstString(out, 0)
				if ok && docPath(doc) == pdf {
					found, dest = true, name
				}
			}
			if !ok {
				break
			}
		}
	}
	if !found {
		return false
	}
	// openDocument of a file open in a tab brings that tab forward (no
	// new tab); then the tab turns to the page.
	if _, ok := capture(gdbusCall(dest, okularShell, "org.kde.okular.openDocument", FileUrl(pdf))); !ok {
		return false
	}
	_, ok = capture(gdbusCall(dest, obj, "org.kde.okular.goToPage", strconv.Itoa(index)))
	return ok
}

// ---- Opening ----

// Open shows printed page page of pdf (an absolute path, as Choose
// gives) in v. An error saying why if the viewer couldn't be started;
// else msg is what the user has more to do (xdg-open can't turn to a
// page), "" if nothing.
func (v Viewer) Open(pdf string, page int) (msg string, err error) {
	index := rules.PdfIndex(page)
	if v.Kind == Okular && okularTurn(pdf, index) {
		return "", nil // turned over D-Bus; without gdbus, okularTurn is false
	}
	a := v.Command(pdf, page)
	if v.Kind == Mupdf {
		mu.Lock()
		lastMupdf.Terminate()
		lastMupdf, err = spawn.Detached(a)
		mu.Unlock()
	} else {
		_, err = spawn.Detached(a)
	}
	if err != nil {
		name := ""
		if len(a) > 0 {
			name = a[0]
		}
		return "", fmt.Errorf("can't start %s: %s", name, spawn.ErrorText(err))
	}
	if v.Kind == XdgOpen {
		return fmt.Sprintf("xdg-open can't turn to a page: turn to page %d (the PDF's page %d)", page, index), nil
	}
	return "", nil
}

// EndMupdf ends the mupdf Open started last, if it is still running.
func EndMupdf() {
	mu.Lock()
	lastMupdf.Terminate()
	mu.Unlock()
}

// LastMupdf is the mupdf Open started last, for the tests: its pid, 0
// for none or ended.
func LastMupdf() int {
	mu.Lock()
	defer mu.Unlock()
	if lastMupdf.Alive() {
		return lastMupdf.Pid
	}
	return 0
}
