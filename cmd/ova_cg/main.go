// Command ova_cg is an OVA character generator: a window that edits one
// character, kept in a YAML file.
//
//	ova_cg [options] [file.yaml]
//
// The port of po_ova_cg's OvaCg.Mod. PLAN.md, "Starting": the command
// line, then the file, then a display, then the window. A file that
// can't be opened is said on standard error, status 1, before a display
// is needed, as a mistake on the command line is (status 2). The program
// can't do anything without a window, so with no display it says so and
// exits with status 1, before Fyne is asked to open one.
package main

import (
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2/app"

	"github.com/tkurtbond/go_ova_cg/internal/character"
	"github.com/tkurtbond/go_ova_cg/internal/prefs"
	"github.com/tkurtbond/go_ova_cg/internal/ui"
	"github.com/tkurtbond/go_ova_cg/internal/viewer"
)

const (
	program       = "ova_cg"
	appID         = "io.github.tkurtbond.ova_cg"
	failureStatus = 1
	usageStatus   = 2
)

const usage = `ova_cg [options] [file.yaml]

  -h, --help
      Display this text.
  -r <arg>, --rulebook <arg>
      The rulebook's PDF (else OVA_RULEBOOK, else the one chosen before, else ~/Reference/RPG/OVA/).
  -v <arg>, --viewer <arg>
      okular, evince, mupdf, xdg-open, or a command: %f the PDF, %p its page, %l the printed page.
`

// options is what the command line gives.
type options struct {
	help        bool
	rulebook    string
	viewer      viewer.Viewer
	viewerGiven bool
	file        string // "" for none
}

func fail(status int, a, b string) {
	fmt.Fprintf(os.Stderr, "%s: %s%s\n", program, a, b)
	os.Exit(status)
}

// parse reads the command line; a mistake ends the program with status
// 2. Options take their argument as the next word, or after = in the
// long form; -- ends the options.
func parse(args []string) options {
	var o options
	onlyFiles := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if onlyFiles || arg == "-" || !strings.HasPrefix(arg, "-") {
			if o.file != "" {
				fail(usageStatus, "one file at a time: ", arg)
			}
			o.file = arg
			continue
		}
		if arg == "--" {
			onlyFiles = true
			continue
		}
		name, value, hasValue := arg, "", false
		if strings.HasPrefix(arg, "--") {
			name, value, hasValue = strings.Cut(arg, "=")
		} else if len(arg) > 2 {
			name, value, hasValue = arg[:2], arg[2:], true
		}
		// takeValue is the option's argument.
		takeValue := func() string {
			if hasValue {
				return value
			}
			if i+1 >= len(args) {
				fail(usageStatus, name, " takes an argument")
			}
			i++
			return args[i]
		}
		switch name {
		case "-h", "--help":
			if hasValue {
				fail(usageStatus, name, " takes no argument")
			}
			o.help = true
		case "-r", "--rulebook":
			o.rulebook = takeValue()
		case "-v", "--viewer":
			s := takeValue()
			v, ok := viewer.Parse(s)
			if !ok {
				fail(usageStatus, "--viewer takes okular, evince, mupdf, xdg-open, or a command with %f for the PDF: ", s)
			}
			o.viewer, o.viewerGiven = v, true
		default:
			fail(usageStatus, "unknown option: ", arg)
		}
	}
	return o
}

// isSet says whether the environment variable name is set and not empty.
func isSet(name string) bool { return os.Getenv(name) != "" }

// requireDisplay ends the program, saying why, with no display: and in
// tmux where that may have come from (PLAN.md, "Findings": a tmux session
// last attached from a terminal without a display drops DISPLAY and
// WAYLAND_DISPLAY from new panes).
func requireDisplay() {
	if isSet("DISPLAY") || isSet("WAYLAND_DISPLAY") {
		return
	}
	fmt.Fprintf(os.Stderr, "%s: no display: neither DISPLAY nor WAYLAND_DISPLAY is set\n", program)
	if isSet("TMUX") {
		fmt.Fprintf(os.Stderr, "%s: in tmux, the session may have been attached from a terminal without one;"+
			" reattach from the desktop, or set them from `tmux show-environment -g`\n", program)
	}
	os.Exit(failureStatus)
}

func main() {
	o := parse(os.Args[1:])
	if o.help {
		os.Stdout.WriteString(usage)
		return
	}
	var c *character.Character
	var notices []string
	if o.file != "" {
		var err error
		if c, notices, err = character.Load(o.file); err != nil {
			fail(failureStatus, err.Error(), "")
		}
	}
	requireDisplay()

	a := app.NewWithID(appID)
	w := ui.New(a)
	w.RulebookOption = o.rulebook
	if o.viewerGiven {
		w.Viewer, w.ViewerOption = o.viewer, true
	}
	w.UsePrefs(prefs.Open(prefs.DefaultPath()))
	w.Win.Show()
	w.Win.Canvas().Focus(w.NameIn) // where po_ova_cg's cursor starts
	if c != nil {
		w.Adopt(c, o.file, notices)
	}
	a.Run()
}
