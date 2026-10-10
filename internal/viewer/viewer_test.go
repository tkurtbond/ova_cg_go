package viewer

// po_ova_cg's TestViewer, with fakes in the programs' places:
// test/fake-viewer.sh for every viewer, recording its arguments in
// obj/fake-viewer.log, and test/fake-gdbus.sh, answering from
// obj/gdbus.scenario and logging its calls in obj/gdbus.log. Every fake
// viewer started is ended by its pid, never by name. The real viewers
// are test/viewer.sh's, on a headless sway.

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/tkurtbond/ova_cg/internal/rules"
	"github.com/tkurtbond/ova_cg/internal/spawn"
)

const (
	fake      = "../../test/fake-viewer.sh"
	fakeGdbus = "../../test/fake-gdbus.sh"
	missing   = "ova-cg-no-such-program"
	viewerLog = "../../obj/fake-viewer.log"
	gdbusLog  = "../../obj/gdbus.log"
	scenario  = "../../obj/gdbus.scenario"
	// The PDF: blanks and a "'", which a shell would need quoted, and
	// "#", "%" and "?", which a URL needs encoded; as okular's
	// openDocument is given it, and as a URL encoded in full.
	pdf     = "/tmp/an O'Brien #1 100% ?.pdf"
	url     = "file:///tmp/an O'Brien %231 100%25 %3F.pdf"
	fullUrl = "file:///tmp/an%20O%27Brien%20%231%20100%25%20%3F.pdf"
	page    = 60 // printed; the PDF's index is 62
)

func fakes() {
	OkularProgram, EvinceProgram, MupdfProgram, XdgOpenProgram = fake, fake, fake, fake
	GdbusProgram = fakeGdbus
}

// reset removes the logs, before a case.
func reset() {
	os.Remove(viewerLog)
	os.Remove(gdbusLog)
}

func waitFor(cond func() bool, tries int) bool {
	for range tries {
		if cond() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return cond()
}

func contents(name string) string {
	b, _ := os.ReadFile(name)
	return string(b)
}

// viewerRecord is the fake viewer's record, once written (up to 5 s);
// "" if none was.
func viewerRecord() string {
	waitFor(func() bool { _, err := os.Stat(viewerLog); return err == nil }, 100)
	return contents(viewerLog)
}

// argsOf is the record's arguments, one "<A>" each, joined by blanks:
// what started the fake viewer.
func argsOf(record string) string {
	var r []string
	for _, l := range strings.Split(record, "\n") {
		if a, ok := strings.CutPrefix(l, "arg "); ok {
			r = append(r, a)
		}
	}
	return strings.Join(r, " ")
}

// pidOf is the record's pid, -1 if none.
func pidOf(record string) int {
	for _, l := range strings.Split(record, "\n") {
		if p, ok := strings.CutPrefix(l, "pid "); ok {
			if n, err := strconv.Atoi(p); err == nil {
				return n
			}
		}
	}
	return -1
}

// kill ends the fake viewer pid, by its pid.
func kill(pid int) {
	if pid > 0 {
		syscall.Kill(pid, syscall.SIGTERM)
	}
}

// gone waits up to 5 s for pid to end; whether it did.
func gone(pid int) bool { return waitFor(func() bool { return spawn.Session(pid) == -1 }, 100) }

// noViewer says whether no fake viewer starts within a second.
func noViewer() bool {
	time.Sleep(time.Second)
	_, err := os.Stat(viewerLog)
	return err != nil
}

func commandOf(a []string) string { return "<" + strings.Join(a, "> <") + ">" }

// tab adds a tab showing doc, object obj of okular name, to the
// scenario.
func tab(t *testing.T, name, obj, doc string) {
	f, err := os.OpenFile(scenario, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.WriteString(name + "\t" + obj + "\t" + doc + "\n")
}

// noTabs empties the scenario: no okular on the bus.
func noTabs(t *testing.T) {
	if err := os.WriteFile(scenario, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

// starts opens page of pdf in v, which must start the fake viewer with
// args; ends it after.
func starts(t *testing.T, v Viewer, args, label string) {
	t.Helper()
	reset()
	msg, err := v.Open(pdf, page)
	record := viewerRecord()
	if err != nil || msg != "" || argsOf(record) != args {
		t.Errorf("%s: %v %q, got %s", label, err, msg, argsOf(record))
	}
	kill(pidOf(record))
}

func lines(l ...string) string { return strings.Join(l, "\n") + "\n" }

func TestChoose(t *testing.T) {
	if path, ok := Choose("/no/such/a.pdf", "/env.pdf", "/saved.pdf", "/home/u"); ok || path != "/no/such/a.pdf" {
		t.Error("Choose: --rulebook first; false when it isn't there")
	}
	if path, ok := Choose("", "/env.pdf", "/saved.pdf", "/home/u"); ok || path != "/env.pdf" {
		t.Error("Choose: then OVA_RULEBOOK")
	}
	if path, ok := Choose("", "", "/saved.pdf", "/home/u"); ok || path != "/saved.pdf" {
		t.Error("Choose: then the saved path")
	}
	if path, ok := Choose("", "", "", "/home/u"); ok ||
		path != "/home/u/Reference/RPG/OVA/OVA_The_Anime_Role-Playing_Game.pdf" {
		t.Error("Choose: then the default under the home directory")
	}
	cwd, _ := os.Getwd()
	if path, ok := Choose("viewer.go", "", "", ""); !ok || path != cwd+"/viewer.go" {
		t.Errorf("Choose: a relative path is made absolute; true for a file there: %s", path)
	}
}

func TestParse(t *testing.T) {
	for s, k := range map[string]Kind{" okular ": Okular, "evince": Evince, "mupdf": Mupdf, "xdg-open": XdgOpen} {
		if v, ok := Parse(s); !ok || v.Kind != k || v.Name() != strings.TrimSpace(s) {
			t.Errorf("Parse: the viewers' names, and Name: %q", s)
		}
	}
	if v, ok := Parse("zathura -P %p %f"); !ok || v.Kind != Template || v.Name() != "zathura -P %p %f" {
		t.Error("Parse: a command template")
	}
	for _, s := range []string{"zathura", ""} {
		if _, ok := Parse(s); ok {
			t.Errorf("Parse: a name it doesn't know, a template without %%f, nothing: false: %q", s)
		}
	}
}

func TestDefault(t *testing.T) {
	defer fakes()
	fakes()
	OkularProgram = missing
	if Default().Kind != Evince {
		t.Error("Default: evince when okular isn't on PATH")
	}
	EvinceProgram, MupdfProgram = missing, missing
	if Default().Kind != XdgOpen {
		t.Error("Default: xdg-open when none is")
	}
	fakes()
	if Default().Kind != Okular {
		t.Error("Default: okular first")
	}
}

func TestCommand(t *testing.T) {
	v, ok := Parse("view  --page=%p %f%% -l %l x%q %")
	if got := commandOf(v.Command(pdf, page)); !ok ||
		got != "<view> <--page=62> </tmp/an O'Brien #1 100% ?.pdf%> <-l> <60> <x%q> <%>" {
		t.Errorf("Command: a template's words, each substitution within its word: %s", got)
	}
	if rules.PdfIndex(page) != 62 {
		t.Error("the PDF's index of printed page 60 is 62")
	}
	if FileUrl(pdf) != url {
		t.Errorf("FileUrl: only %%, # and ? encoded: %s", FileUrl(pdf))
	}
	if docPath(fullUrl) != pdf || docPath(pdf) != pdf {
		t.Error("docPath: a file: URL decoded, a path as it is")
	}
}

func TestOpen(t *testing.T) {
	if err := os.MkdirAll("../../obj", 0o755); err != nil {
		t.Fatal(err)
	}
	fakes()
	defer fakes()
	noTabs(t)

	// ---- Starting each ----
	starts(t, Viewer{Kind: Evince}, "<-i> <62> </tmp/an O'Brien #1 100% ?.pdf>", "evince: -i INDEX PDF")
	v, _ := Parse(fake + " -P %l %f")
	starts(t, v, "<-P> <60> </tmp/an O'Brien #1 100% ?.pdf>", "a template runs as given")

	reset()
	msg, err := Viewer{Kind: XdgOpen}.Open(pdf, page)
	record := viewerRecord()
	if err != nil || argsOf(record) != "</tmp/an O'Brien #1 100% ?.pdf>" ||
		msg != "xdg-open can't turn to a page: turn to page 60 (the PDF's page 62)" {
		t.Errorf("xdg-open: PDF, and a message giving the page: %q %v", msg, err)
	}
	kill(pidOf(record))

	EvinceProgram = missing
	if _, err := (Viewer{Kind: Evince}).Open(pdf, page); err == nil ||
		err.Error() != "can't start ova-cg-no-such-program: No such file or directory" {
		t.Errorf("a missing viewer: an error, saying why: %v", err)
	}
	EvinceProgram = fake

	// ---- mupdf: the last one replaced ----
	reset()
	msg, err = Viewer{Kind: Mupdf}.Open(pdf, page)
	record = viewerRecord()
	first := pidOf(record)
	if err != nil || msg != "" || argsOf(record) != "</tmp/an O'Brien #1 100% ?.pdf> <62>" || first <= 0 ||
		LastMupdf() != first {
		t.Errorf("mupdf: PDF INDEX: %v %s", err, argsOf(record))
	}
	reset()
	_, err = Viewer{Kind: Mupdf}.Open(pdf, page+1)
	record = viewerRecord()
	pid := pidOf(record)
	if err != nil || !gone(first) || pid <= 0 || pid == first || LastMupdf() != pid ||
		argsOf(record) != "</tmp/an O'Brien #1 100% ?.pdf> <63>" {
		t.Error("mupdf: the one started before is ended, the next started")
	}
	EndMupdf()
	if !gone(pid) || LastMupdf() != 0 {
		t.Error("EndMupdf: ends the last")
	}
}

func TestOkular(t *testing.T) {
	if err := os.MkdirAll("../../obj", 0o755); err != nil {
		t.Fatal(err)
	}
	fakes()
	defer fakes()
	v := Viewer{Kind: Okular}

	// Two okulars, the second showing the PDF in its second tab; an
	// Instance_ alias and a name not okular's, which mustn't be asked.
	noTabs(t)
	tab(t, "org.kde.okular.Instance_1", "/okular", pdf)
	tab(t, "org.kde.okularx", "/okular", pdf)
	tab(t, "org.kde.okular-100", "/okular", "/tmp/other.pdf")
	tab(t, "org.kde.okular-200", "/okular", "/tmp/x.pdf")
	tab(t, "org.kde.okular-200", "/okular2", pdf)
	tab(t, "org.kde.okular-200", "/okular3", "/tmp/y.pdf")
	reset()
	msg, err := v.Open(pdf, page)
	want := lines(
		"org.freedesktop.DBus /org/freedesktop/DBus org.freedesktop.DBus.ListNames",
		"org.kde.okular-100 /okular org.kde.okular.currentDocument",
		"org.kde.okular-100 /okular2 org.kde.okular.currentDocument",
		"org.kde.okular-200 /okular org.kde.okular.currentDocument",
		"org.kde.okular-200 /okular2 org.kde.okular.currentDocument",
		"org.kde.okular-200 /okularshell org.kde.okular.openDocument <file:///tmp/an O'Brien %231 100%25 %3F.pdf>",
		"org.kde.okular-200 /okular2 org.kde.okular.goToPage <62>")
	if got := contents(gdbusLog); err != nil || msg != "" || got != want {
		t.Errorf("okular: the tab showing the PDF found, brought forward and turned, over D-Bus:\n%s", got)
	}
	if !noViewer() {
		t.Error("okular: none started when one shows the PDF")
	}

	// The tab's document as a file: URL.
	noTabs(t)
	tab(t, "org.kde.okular", "/okular", fullUrl)
	reset()
	msg, err = v.Open(pdf, page)
	want = lines(
		"org.freedesktop.DBus /org/freedesktop/DBus org.freedesktop.DBus.ListNames",
		"org.kde.okular /okular org.kde.okular.currentDocument",
		"org.kde.okular /okularshell org.kde.okular.openDocument <file:///tmp/an O'Brien %231 100%25 %3F.pdf>",
		"org.kde.okular /okular org.kde.okular.goToPage <62>")
	if got := contents(gdbusLog); err != nil || msg != "" || got != want {
		t.Errorf("okular: a document given as a file: URL is found, in an okular started --unique:\n%s", got)
	}
	if !noViewer() {
		t.Error("okular: none started")
	}

	// An okular, but not showing the PDF: one is started.
	noTabs(t)
	tab(t, "org.kde.okular-100", "/okular", "/tmp/other.pdf")
	tab(t, "org.kde.okular-100", "/okular2", pdf+".bak")
	starts(t, v, "<--unique> <-p> <62> </tmp/an O'Brien #1 100% ?.pdf>", "okular: started when none shows the PDF")
	want = lines(
		"org.freedesktop.DBus /org/freedesktop/DBus org.freedesktop.DBus.ListNames",
		"org.kde.okular-100 /okular org.kde.okular.currentDocument",
		"org.kde.okular-100 /okular2 org.kde.okular.currentDocument",
		"org.kde.okular-100 /okular3 org.kde.okular.currentDocument")
	if got := contents(gdbusLog); got != want {
		t.Errorf("okular: every tab asked, up to the first missing:\n%s", got)
	}

	noTabs(t)
	tab(t, "org.kde.okular-100", "/okular", pdf)
	GdbusProgram = missing
	starts(t, v, "<--unique> <-p> <62> </tmp/an O'Brien #1 100% ?.pdf>", "okular: started --unique without gdbus")
}
