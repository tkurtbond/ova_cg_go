# AGENTS.md

Working notes for an agent in this repo, which holds `ova_cg`, a
character generator for OVA (Open Versatile Anime engine), written in
Go with the Fyne GUI toolkit. It is a port of po_ova_cg
(`~/Repos/RPG/Tools/po_ova_cg`), an Oberon-2 program built with poc and
FLTK. PLAN.md is the port's design and build order; this file is
operational notes, not a design doc.

**po_ova_cg is the specification.** Its `PLAN.md` holds the rules, the
data tables, the file format, the window and the viewer findings; its
`src/*.Mod` the behaviour; its `test/*.Mod` the checks to port. Read the
Oberon module before writing its Go package, and port its test check
for check. Where this repo must differ (Go, Fyne, yaml.v3), PLAN.md
says how and why; anything PLAN.md doesn't mention stays as po_ova_cg
has it.

## Reference material

| What | Where |
|---|---|
| The Oberon program and its notes | `~/Repos/RPG/Tools/po_ova_cg/` (`PLAN.md`, `AGENTS.md`, `src/`, `test/`) |
| OVA rulebook (PDF, 170 pages) | `~/Reference/RPG/OVA/OVA_The_Anime_Role-Playing_Game.pdf` |
| Rulebook as text | `~/Reference/RPG/OVA/ova-rulebook.txt`, `ova-rulebook-not-layout.txt` |
| Character sheet | `~/Reference/RPG/OVA/OVA-CharacterRecord-Letter.pdf` |
| Descriptions as Markdown | `~/Reference/RPG/OVA/abilities.md`, `weaknesses.md` -- **never copy into this repo** (below) |
| The YAML format | the Ada ova_fmt: `~/Repos/RPG/Tools/ova_fmt/PLAN.md` and `test-data/*.yaml` |
| po_ova_fmt, the file oracle | `~/Repos/RPG/Tools/po_ova_fmt/` (binary `po_ova_fmt`) |
| Fyne v2.8.1 source | `~/go/pkg/mod/fyne.io/fyne/v2@v2.8.1` (its `test` package drives widgets with no display) |
| yaml.v3 source | `~/go/pkg/mod/go.yaml.in/yaml/v3@v3.0.4` |

## The rulebook's text stays out of the program

The program knows the *names* of Abilities, Weaknesses, Perks and
Flaws, the numbers the rules attach to them, and the rulebook page each
is described on. **It holds no descriptions**: none in the source, the
data tables, tooltips, tests or docs. To explain an item it opens the
rulebook at that page in the system's PDF viewer. Don't paste text from
`abilities.md`, `weaknesses.md` or the rulebook into this repo, even as
a comment.

**Every name the window shows has its page beside it** (`Attack (p.
50)`), made by `rules.Label` or `rules.PageText`, never formatted by
hand. **Page numbers are the book's printed ones**; the PDF's page
index is the printed page + 2 (`rules.PdfIndex`).

## Build / test

```sh
make                 # ./ova_cg (go build ./cmd/ova_cg)
make test            # go test ./... (window tests included: Fyne's test
                     #   driver, no display), test/files.sh (needs
                     #   po_ova_fmt: PO_OVA_FMT), test/cli.sh
make check           # go vet ./... and go test -race ./...
make check-pages     # rules' pages against the rulebook PDF (needs mutool)
make test-viewers    # the real okular, evince, mupdf (test/viewer.sh: its
                     #   own headless sway and private bus; ~30 s)
make test-headless   # the real program on Xvfb: runs RUNTIME seconds,
                     #   then test/quit.sh (Ctrl+E, Escape, Ctrl+Q)
make install         # to $(PREFIX)/bin, default ~/local/bin (DESTDIR for staging)
make clean           # rm -rf obj ova_cg
```

- The first build compiles Fyne's GLFW C code (cgo; needs gcc and the
  X11/GL development headers): a minute or two. Later builds are cached.
- `go test ./...` runs from each package's directory; tests find the
  fixtures as `../../testdata/` and write into `../../obj/` (made by
  the test if missing). `test/files.sh` reads `obj/files/`, which
  `internal/save`'s test writes: run `make test`, not the script alone.
- **Don't run `make install` with the default `PREFIX` unless asked.**
- **`testdata/` is po_ova_cg's `test/data/`, copied** (itself the Ada
  ova_fmt's `test-data/`). Never edit the copies: a fixture changes in
  the Ada ova_fmt first, then po_ova_fmt and po_ova_cg, then here.
- A new YAML key goes into the Ada ova_fmt first, then po_ova_fmt (each
  in its own repo, with the user's go-ahead), before ova_cg saves it
  (po_ova_cg's PLAN.md, "Changes to ova_fmt").

## Windows and the display

- **An agent never opens windows on the user's desktop** (the user's
  rule for po_ova_cg, 2026-10-06, which holds here): not the program,
  not a test, not a scratch program. The desktop is Wayland, and
  XWayland is the desktop too. The window tests use Fyne's test driver
  (`test.NewTempApp`), which opens nothing. The real program runs only
  under `make test-headless` (Xvfb, `WAYLAND_DISPLAY` unset) or, for a
  single run, `env -u WAYLAND_DISPLAY xvfb-run -a ./ova_cg ...`.
- Never import `fyne.io/fyne/v2/app` outside `cmd/ova_cg`: it is the
  GLFW driver, and a test that imports it would want a display.
- **Never start the user's PDF viewer either.** `internal/spawn` and
  `internal/viewer` tests put `test/fake-viewer.sh` and
  `test/fake-gdbus.sh` in the programs' places (`viewer.Okular`, ...).
  Only `make test-viewers` runs the real viewers, on its own sway,
  inside `dbus-run-session`, with a scratch `HOME`: the agent's
  environment has the user's real session bus, where okular or evince
  would answer, on the desktop.
- **End only the processes you started, by PID.** Never `pkill -x
  okular` (or evince, mupdf), and never `pkill -f` with a pattern your
  own command line contains.
- **Never write the user's preferences** (`~/.config/ova_cg/`): a test
  opens its own file under `t.TempDir()`, and `make test-headless` runs
  the program with `XDG_CONFIG_HOME` in `obj/`.
- **Nothing in the window waits for the user**: a message, the
  save-first question, the file choosers, a line of text and the
  rulebook go through `ui.Window`'s function fields (`Tell`, `Ask`,
  `ChoosePath`, `ChoosePdf`, `AskText`, `ChooseImage`, `OpenPage`),
  which a test replaces with synchronous fakes. Anything new that would
  wait gets one too.

## Go, Fyne and yaml.v3 here

- **The rules and the window are separate.** `rules`, `calc`,
  `entities`, `character`, `save`, `paths`, `spawn`, `viewer` and
  `prefs` import no Fyne. Every number on the window comes from
  `calc.Compute`; no widget callback does arithmetic of its own.
- **calc's records are extended by embedding**: `character.Trait`
  embeds `calc.Trait`, calc's lists hold the embedded field's address,
  and `calc.Trait.Owner` points back at the `character.Trait`. Set
  `Owner` whenever one is made (the `character.New*` functions do);
  calc holds no strings. A warning is a code and the record it is
  about; the window words it, with `rules.Label`.
- **Every computed number has an override box beside it**; calc takes
  the overrides as input, so a number computed from an overridden one
  uses the override. A new computed number gets its box, its
  `overrides:` key when saved, and a test.
- **Absence needs a flag**: an optional text keeps its `HasX bool`
  beside `X string`, as po_ova_cg's records do, since "" and absent
  save differently. `character.SetText` sets both.
- **Integers and booleans in a file are read with polibfyaml's grammar**
  (`entities.parseInt`, `parseBool`), not yaml.v3's resolver, so the
  generator reads exactly what po_ova_fmt reads (a quoted `"3"` is 3).
- **Save through `yaml.Node`s, never YAML text by hand.** Text nodes are
  `!!str` with plain style (or literal for several lines): the emitter
  quotes them where plain YAML can't hold them. Keys and numbers are
  plain `!!int`/`!!str`. Check a change with `TestAwkwardText` and
  `test/files.sh`.
- **Fyne's main-menu shortcuts are tried before the focused widget**,
  so a menu shortcut works from inside an entry, and also takes the key
  from it: never bind Ctrl+A, C, V, X, Z or Y (select-all, copy, paste,
  cut, undo, redo).
- **Fill widgets with `filling` set**: an `Entry`'s `OnChanged` runs
  when the program sets its text too, so a load mustn't count as a
  change. `ui.Window.filling` guards every such callback.
- **`widget.List` rows are reused**: a row's update function sets every
  field from the item each time; never keep per-row state in the row.
- **Widgets and callbacks run on Fyne's goroutine.** Work from another
  goroutine (none yet) goes through `fyne.Do`.
- **A portrait's path is relative to the YAML file**, so Save As
  rebases it with `paths.Rebase`; paths are text, never looked up on
  disk.
- **Programs are started with `spawn`, never a shell**: an argument
  vector, no quoting, `Setsid` so closing the terminal doesn't close
  the viewer, and a missing program reported as `ENOENT`.
- A programmer error panics; bad input, a cancelled dialog or a viewer
  that won't start comes back as an `error` or `false` with a message,
  shown in a dialog.

## Testing policy

- Run `make test` after any change, and `make test-headless` after a
  change to the window or `cmd/ova_cg`.
- **The sample characters are the oracle for the calculations.** Every
  derived number on a sample character's sheet must come out of `calc`
  from its Abilities, Weaknesses, Perks and Flaws alone
  (`internal/calc`'s sample tests, against `testdata/`). A sample the
  rules can't reproduce is a finding for po_ova_cg's PLAN.md, not a
  test to loosen; the book's two rulings are set as overrides, each
  commented.
- **A file the program saves must load in po_ova_fmt** and format the
  same as the file it was loaded from (`test/files.sh`), and load here
  again with no notices, saving the same bytes.
- Port each Oberon check with the same label where it makes sense, so a
  failure can be compared with po_ova_cg's.
- **Prove a new check can fail**: plant the bug it targets and watch it
  fail.

## Conventions

- `gofmt` everything. Comments explain why and only claim what has
  been verified; a comment about how Fyne, yaml.v3 or a viewer behaves
  is checked with a throwaway program in the scratchpad first (with
  Fyne's test driver, or under Xvfb).
- A rule in the code cites its rulebook page (`// p. 83`), never its
  text.
- Record decisions and findings in PLAN.md as they happen. Mark a
  finished step `[done]` and strike through an open question once it's
  decided.
