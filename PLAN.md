# PLAN.md - ova_cg, the OVA character generator, in Go with Fyne

`ova_cg` is a port of `po_ova_cg` (`~/Repos/RPG/Tools/po_ova_cg`), an
Oberon-2 program built with poc, FLTK (pofltk) and libfyaml
(polibfyaml), to Go, with the Fyne GUI toolkit and a pure-Go YAML
library. It builds a character for OVA, the Open Versatile Anime
engine, in a window laid out like the official character sheet: the
user picks Abilities and Weaknesses by name and level, builds attacks
from Perks and Flaws, and the program works out each attack's and
Ability's Endurance cost and the sheet's ROLL, DX, Defense, Health,
Endurance and TV as the character changes. It holds no rulebook text:
every name shows its rulebook page, and **?** opens the rulebook's PDF
at that page in the user's PDF viewer. It saves and loads YAML in
ova_fmt's format, so po_ova_fmt and the Ada ova_fmt print what it makes.

**The port changes the language and the libraries, not the program.**
The rules, the data tables, the file format, the window's contents and
behaviour, and the tests' oracles are po_ova_cg's. Its `PLAN.md` is the
design document for all of that and is not repeated here: read its
"Sources and findings", "The rules the program computes", "Limiting the
character", "Data tables", "The window", "Files" and "The rulebook
viewer" before changing any rule, table, key or viewer behaviour. This
file records how each part maps to Go and Fyne, what the port decided
differently and why, and the build order.

## Sources

| What | Where |
|---|---|
| The Oberon program (the specification) | `~/Repos/RPG/Tools/po_ova_cg/` (`src/*.Mod`, `PLAN.md`, `AGENTS.md`, `README.md`) |
| Its tests (to port, check for check) | `~/Repos/RPG/Tools/po_ova_cg/test/` |
| The fixtures (copied into `testdata/`) | `~/Repos/RPG/Tools/po_ova_cg/test/data/` (themselves the Ada ova_fmt's `test-data/`) |
| The file oracle | po_ova_fmt, `~/Repos/RPG/Tools/po_ova_fmt/po_ova_fmt` |
| Fyne | `fyne.io/fyne/v2` v2.8.1 (module cache: `~/go/pkg/mod/fyne.io/fyne/v2@v2.8.1`) |
| YAML | `go.yaml.in/yaml/v3` v3.0.4 (the maintained fork of `gopkg.in/yaml.v3`) |
| The rulebook and sheet | `~/Reference/RPG/OVA/` (see AGENTS.md: names and pages only) |

## Layout

```
go.mod                      module github.com/tkurtbond/ova_cg
cmd/ova_cg/                 main: command line, file, display check, window
cmd/listrules/              PAGE<TAB>NAME of every item, for make check-pages
cmd/openpage/               test/viewer.sh's driver (po_ova_cg's OpenPage)
internal/rules/             OvaRules
internal/calc/              OvaCalc
internal/entities/          OvaEntities (and what of OvaText it needs)
internal/character/         OvaCharacter
internal/save/              OvaSave
internal/paths/             OvaPaths
internal/spawn/             OvaSpawn (no C part: os/exec and /proc)
internal/viewer/            OvaViewer
internal/prefs/             the preferences file (FlPreferences' part)
internal/ui/                OvaWidgets, OvaTraitPanel, OvaAttackDialog,
                            OvaLinkedDialog, OvaSettings, OvaBackground,
                            OvaWindow: everything that imports Fyne
testdata/                   the fixtures, copied, kept the same
test/                       files.sh, cli.sh, quit.sh, viewer.sh,
                            fake-viewer.sh, fake-gdbus.sh
tools/                      rulebook-headings.py (copied), with-sway.sh
GNUmakefile                 make, make test, make test-headless, ...
```

Only `internal/ui` and `cmd/ova_cg` import Fyne. `cmd/ova_cg` alone
imports `fyne.io/fyne/v2/app` (the GLFW driver); `internal/ui` is
written against the `fyne.App`/`fyne.Window` interfaces, so its tests
run on Fyne's test driver (`fyne.io/fyne/v2/test`) with no display at
all.

## Module by module

### rules (OvaRules)

A table of `*Item` (name, kind, page, flags, linked content, cost kind,
cost, min/max cost, max times) built in an `init`, in the book's order,
with the same 173 names, pages, flags and costs; the two aliases;
`Find` (ASCII case-insensitive), `Count`, `Get`, `Label`, `PageText`,
`LabelOf`, `PdfIndex`; `ToughBonus`, `ReserveBonus`,
`EnduranceUseCost`, `SpellCost`; the rule pages as constants. Oberon's
`SET` of flags becomes a `uint32` bit set (`Flags`), with a `Has`
method. `ASSERT`s on table bounds become panics: they are programmer
errors.

### calc (OvaCalc)

The same records, as Go structs: `Number` (value, half, plus,
overridden, the override's three), `Sheet`, `Trait`, `Mod`, `Spell`,
`Attack`, `Options`, `Warning`; the same `Compute` and every rule in
the same order, so the warnings come out in the same order.

- **Extension by embedding and a back pointer.** Oberon's
  `OvaCharacter.Trait` extends `OvaCalc.TraitDesc`, and the lists are
  linked through the base type. In Go, `character.Trait` embeds
  `calc.Trait`, calc's lists hold `*calc.Trait` (the embedded field's
  address), and each calc record has `Owner any`, a pointer back to the
  record that embeds it. A warning names a `*calc.Trait`, `*calc.Attack`
  and so on, as in Oberon; the window reaches the character's record
  through `Owner`. So calc still imports nothing of character, holds no
  strings, and is tested on its own.
- **Linked lists become slices** (`[]*Trait`, `[]*Mod`, ...), in the
  same order. The Oberon code's list surgery (Insert, Remove) becomes
  slice operations in `character`.
- `Parse` and `Text` keep the override box's grammar exactly
  (`TestCalc`'s Parse checks).

### entities (OvaEntities)

The loader, on `yaml.Node` trees from `go.yaml.in/yaml/v3`
(`yaml.NewDecoder`, one `Decode` per document), with the same records,
the same two kinds of error (recorded `LoadError`s with a path and the
`name` scalar's line and column; or a fatal message), the same sort
(insertion sort by name then detail, folded as the Ada `To_Lower`
folds: A-Z and the Latin-1 bytes C0-DE but D7), and the same trimming.

- **Scalars' grammar is polibfyaml's, not yaml.v3's**: an integer is
  the text of the scalar, quoted or not, as `[+-]` then decimal, `0x`,
  `0o` or `0b` digits with single `_`s between them, within 32 bits; a
  boolean `true`/`True`/`TRUE`/`false`/`False`/`FALSE`. Messages are
  polibfyaml's: `not a valid integer: "x"`, `integer out of range:
  "..."`, `not a valid boolean: "..."`.
- **Paths** are polibfyaml's `ByPath` syntax (`/1/abilities/2/name`),
  built as the walk descends, since yaml.v3 nodes don't know their
  parents.
- A file's YAML error is yaml.v3's message; a missing file is
  `PATH: error: <strerror>`, as libfyaml says it.
- Aliases (`*x`) are resolved to their anchors' nodes.

### character (OvaCharacter)

`Character` embeds `calc.Sheet`; `Trait`, `Mod`, `Attack`, `Spell`
embed theirs, with the names, details, notes and has-flags. `Build`,
`Load` (with `Adopt` and the notices), `ReadEntity`, `Recompute` (the
Endurance Use tie, then `calc.Compute`), and the edits the window makes
(`NewTrait`, `InsertTrait`, `RemoveTrait`, `ChangeTrait`, `NewPartner`,
`NewSpell`, `RemoveSpell`, `NewAttack`, `RemoveAttack`, `NewMod`,
`InsertMod`, `RemoveMod`, `CopyMods`). Ropes become `string`s; "the
empty rope is NIL, so absence needs a flag" stays: a field that may be
absent keeps its `Has` flag beside it.

`Load` returns `(c, notices, err)` instead of setting a module variable.

### save (OvaSave)

A `yaml.Node` document built in the fixtures' key order and written
with a `yaml.Encoder` (indent 2), to `NAME.tmp`, then renamed over
`NAME`. Text of more than one line is a literal block; other text is
asked for plain, and **yaml.v3's emitter quotes it whenever plain YAML
couldn't hold it** (`#1 fan`, a leading quote, `- `, `? `, `|`, `>`,
`%`, `@`, a backquote, blanks at either end; it is libyaml's
`analyze_scalar`), which is what polibfyaml's `CreateStyledScalar` was
added to do. Text is tagged `!!str`, so a string that reads as a number
or boolean (`age: "15"`) is quoted too. Numbers are plain scalars with
no tag; the sheet's texts `4/5`, `1/2`, `10+` are plain too. A list of
`overrides` is a flow sequence, as the fixtures write it. `TestFiles`'
awkward-text case and `test/files.sh` check this.

### paths (OvaPaths)

The same text-only functions: `Normalize`, `IsAbsolute`, `Absolute`,
`Dir`, `Relative`, `Rebase`. Not `path/filepath`: `filepath.Rel` and
`Clean` differ in edge cases (`/..`, a relative path's leading `..`)
that `TestPaths` fixes, and no symbolic link is ever followed.

### spawn (OvaSpawn)

No C part. `os/exec` with `SysProcAttr{Setsid: true}`, standard input,
output and error `nil` (which is `/dev/null`), so the viewer is in a
session of its own and a hangup of the terminal doesn't reach it; a
goroutine `Wait`s for it, so it is never a zombie. A failed start comes
back as its `errno` (`exec.ErrNotFound` and a missing path as
`ENOENT`). `Capture` runs a program and returns its standard output
and exit status (-1 for a signal). `Process` keeps the pid and the
start time from `/proc/PID/stat` (field 22), so `Terminate` ends it
only if that pid is still the same process. `OnPath` is `exec.LookPath`.
The argument vector is a `[]string` (no `MaxArgs` buffer, so no
`E2BIG` of ours).

What the double fork bought in C (the viewer re-parented to init at
once) Go gives when the program exits; while it runs, the reaping
goroutine keeps it from being a zombie. `TestSpawn` checks the same
facts: its own session, no terminal, standard files `/dev/null`, the
arguments exact, `ENOENT`, `Capture`'s output whole.

### viewer (OvaViewer)

The same: `Choose`/`FindRulebook`, `Default`, `Parse`, `Name`,
`Command` (with the template's `%f`, `%p`, `%l`, `%%`), `FileUrl`,
`DocPath`, gdbus's GVariant strings, okular over D-Bus, evince's own
reuse, mupdf's last ended first, xdg-open's message. The programs'
names stay package variables, so `TestViewer` runs on
`test/fake-viewer.sh` and `test/fake-gdbus.sh`.

### prefs (FlPreferences' part)

A small `key=value` file, `$XDG_CONFIG_HOME/ova_cg/ova_cg.prefs` (else
`~/.config/...`), holding `rulebook` and `viewer`, written (through a
`.tmp` and rename) only when one is chosen. Not Fyne's `Preferences`:
those live under Fyne's own storage root, keyed by the application ID,
and a test couldn't point them at `obj/` as easily. A test opens its
own file in a temporary directory.

### ui (the window and its dialogs)

| Oberon / FLTK | Go / Fyne |
|---|---|
| `Fl.DoubleWindow`, 1280 x 800, resizable | `fyne.Window`, `Resize(1280, 800)`; each tab's two halves a `container.Split` (a grid of two made each half as wide as the wider, and the window 1422 wide) |
| `FlMenus.MenuBar` | `fyne.MainMenu` (File, Character, Rulebook); shortcuts `desktop.CustomShortcut` with `KeyModifierShortcutDefault` |
| Character > Transformed form, a toggle | a `fyne.MenuItem` with `Checked` |
| `FlLayout.Tabs` | `container.AppTabs` ("Character", "Background") |
| `FlBrowsers.HoldBrowser` with a page column | `widget.List`, each row a name label and a right-hand page label |
| `FlMenus.Choice` of labels | `widget.Select` of `rules.Label`s |
| `FlValuators.Spinner` | `ui.Spinner`: a bounded integer with `-` and `+` buttons (Fyne has none) |
| `FlInputs.Input`, `MultilineInput` | `widget.Entry`, `widget.NewMultiLineEntry` |
| `FlText.TextEditor` | multi-line `widget.Entry`, wrapping by word |
| `NumberBox` (computed + override box) | a read-only label and an `overrideEntry` (a `widget.Entry` that takes its text on Enter and on losing focus) |
| `HelpButton` "?" | `widget.Button` "?" calling `OpenPage` |
| modal `Fl.DoubleWindow` dialogs | `dialog.NewCustomWithoutButtons` on the main window, with their own OK/Cancel/Close buttons (modal, in-window); Escape is ours (`Window.Escape`, "Findings") |
| `FlDialogs.Message`, `Choice`, `Input` | custom dialogs: one button, three buttons, an entry with OK and Cancel |
| `NativeFileChooser` | `dialog.NewFileOpen` / `NewFileSave`, filtered by extension |
| `FlImages` portrait, scaled | `canvas.Image`, `FillMode` contain, in a fixed 160 x 200 box (smaller than po_ova_cg's, so the Background tab fits 800 high) |
| `FlLayout.Grid` of fluff | `container.NewGridWithColumns(4)` |
| `FlLayout.Scroll` of attack rows | `container.NewVScroll` of a `VBox` of rows |
| red text (`SetTextColor(Fl.Red)`) | `widget.Label` with `Importance = widget.DangerImportance` |
| `Fl.AddTimeout(rebuild, 0)` after a row's Remove | not needed: Fyne lets a callback replace its own row (rebuilt directly) |

Absolute pixel positions become Fyne layouts (borders, grids, boxes):
the same widgets in the same places relative to each other, the sheet's
right half on the first tab and its left half on the second.

**Nothing in the window waits for the user**, as in po_ova_cg: a
message, the save-first question, the file choosers, a line of text and
the rulebook go through function fields of `ui.Window` (`Tell`, `Ask`,
`ChoosePath`, `ChoosePdf`, `AskText`, `ChooseImage`, `OpenPage`), each
callback-shaped (`Ask(question, func(answer))`), since Fyne's dialogs
are asynchronous anyway. The tests set synchronous fakes. OK, Cancel
and Close of each dialog are methods a test calls.

**Every change goes through the model**, then `character.Recompute`,
then the window shows it again: no widget holds the only copy of
anything. While the window fills its widgets from the character
(`filling`), their `OnChanged` callbacks change nothing.

## Decisions the port makes

- **The program is `ova_cg`**, its window "OVA Character Generator",
  its preferences `~/.config/ova_cg/`. It is a separate program from
  po_ova_cg; the two share the file format and nothing else.
- **Fyne main-menu shortcuts are tried before the focused widget**
  (`internal/driver/glfw/window.go`, `triggersShortcut`, v2.8.1), so,
  unlike FLTK, a text entry doesn't swallow Ctrl+E. The menu keeps
  po_ova_cg's keys (Ctrl+N, O, S, Shift+S, Q, E, T), and never binds
  Ctrl+A, C, V, X, Z or Y, which would take select-all, copy, paste,
  cut, undo and redo away from every entry. `test/quit.sh` checks
  Ctrl+E from Character Name on the real program.
- **The window tests need no display.** Fyne's test driver renders in
  memory: `go test ./internal/ui` runs anywhere, never on the desktop.
  Only the real program needs a display (`make test-headless`, Xvfb;
  the X11 GLFW driver needs GLX, which Xvfb's Mesa provides).
- **A display is X11 by default.** Fyne's GLFW driver is X11 unless
  built with `-tags wayland`; on a Wayland desktop it runs through
  XWayland's `DISPLAY`. The display check is po_ova_cg's (neither
  `DISPLAY` nor `WAYLAND_DISPLAY` set: say so, with the tmux hint, exit
  1), and a `wayland` build is a make variable (`TAGS=wayland`).
- **The YAML emitter's layout differs from libfyaml's** (indentation,
  `|-` for a block without a final line break). That is not part of
  the format: the oracle is po_ova_fmt's output for the saved copy
  against the fixture (`test/files.sh`), as in po_ova_cg, and a saved
  copy must load with no notices and save the same bytes again.
- **No valgrind targets.** Go is memory-safe and collected; the
  equivalent checks are `go vet` and `go test -race` (`make check`).

## Build order

Each phase ends with its tests passing (`make test`; `make
test-headless` for the program itself) and its entry here marked
`[done]`, with what was found.

0. **Skeleton.** [done] `go.mod`, `GNUmakefile`, `README.md`, `AGENTS.md`,
   this plan; `testdata/` copied from po_ova_cg's `test/data/`;
   `test/*.sh` and `tools/` copied and adapted.
1. **rules.** [done] The tables; `rules_test.go` ports `TestRules` (every
   Quick List entry found, of its kind, in order, at its cost; the
   labels; the small tables); `cmd/listrules` for `make check-pages`.
2. **calc.** [done] Every rule; `calc_test.go` ports `TestCalc`: the twelve
   samples, every number their files hold, the two rulings as
   overrides, and every hand-made case.
3. **entities, character, save, paths.** [done] `entities` loads every
   fixture as po_ova_fmt does (Broken's errors listed); `character_test`
   ports `TestModel`; `save_test` ports `TestFiles` (each fixture loaded
   with its expected notices, saved into `obj/files/`, loaded again with
   none, saved the same; the refused files; a failed save; awkward
   text); `paths_test` ports `TestPaths`; `test/files.sh` compares every
   saved copy with its fixture in all seven po_ova_fmt formats.
4. **spawn, viewer, prefs.** [done] `spawn_test` and `viewer_test` port
   `TestSpawn` and `TestViewer` on the fake viewer and fake gdbus;
   `cmd/openpage` and `test/viewer.sh` for `make test-viewers` (the
   real okular, evince and mupdf on a headless sway with a private
   bus); `prefs_test`.
5. **ui: the first tab.** [done] `Window` with the menu, the tabs, the names,
   the two trait panels, the selected Ability's END and its buttons,
   the totals and warnings, Combat Notes, the combat box with every
   number's override box, and the attack dialog; `window_test` ports
   `TestWindow` on Fyne's test driver.
6. **ui: the second tab.** [done] Summary, the four prose sections, the
   portrait (relative to the file, rebased on Save As), the fluff, the
   notes; `background_test` ports `TestBackground`.
7. **ui: linked content.** [done] The one button, the linked dialog's three
   shapes, Character > Transformed form; `linked_test` ports
   `TestLinked`.
8. **ui: limits and preferences.** [done] The Limits dialog, the red totals,
   the Rulebook menu, the preferences; `limits_test` ports
   `TestLimits`.
9. **The program.** [done] `cmd/ova_cg`: the command line (`-h`, `-r`/
   `--rulebook`, `-v`/`--viewer`, one file), the file loaded before the
   display is looked for, the display check; `test/cli.sh`;
   `test/quit.sh` on Xvfb; README's user guide.

`make test-viewers` and `make check-pages` were not run in the port's
session: they need the real viewers and the rulebook's PDF. Run them
once before trusting `internal/viewer` with okular, evince and mupdf.

## Testing

- `make test` (no display): `go test ./...` (every package, the window
  tests on Fyne's test driver), then `test/files.sh` (needs po_ova_fmt
  at `PO_OVA_FMT`) and `test/cli.sh`.
- `make test-headless`: the program itself on Xvfb: still running after
  `RUNTIME` seconds, and `test/quit.sh` (Ctrl+E, Escape, Ctrl+Q).
- `make test-viewers`: the real viewers (`test/viewer.sh`).
- `make check-pages`: rules' pages against the PDF's headings.
- `make check`: `go vet ./...` and `go test -race ./...`.
- The oracle for calculations is the sample characters; the oracle for
  files is po_ova_fmt.

## Findings

What the port found out about Go, Fyne v2.8.1 and yaml.v3 that
po_ova_cg's design didn't have to know.

- **Fyne's save chooser creates the file before it answers.**
  `dialog.NewFileSave` opens the chosen file for writing
  (`internal/repository/file.go`, `os.Create`) and hands over a
  `URIWriteCloser`; the window takes only its path, closes it, and saves
  through `NAME.tmp` and a rename. So a Save As whose save then fails
  leaves an empty file at the chosen name, which po_ova_cg's never did.
  Accepted: the failure is said, and the old file, if any, is the
  chooser's to have emptied (it asks before overwriting).
- **yaml.v3 quotes text that reads as a number or boolean** once it is
  tagged `!!str` (`age: "15"`); libfyaml wrote `15`. The two load the
  same, and po_ova_fmt prints them the same (`test/files.sh`).
- **Fyne's in-window dialogs have no Escape and no close button.** The
  window keeps a stack of the open dialogs' Escape actions
  (`Window.escapes`, pushed by `modal`, popped by the dialog's
  `OnClosed`); the canvas's typed-key handler runs the top one. A
  focused widget takes Escape before the canvas sees it, so `modal`
  lets go of the focus when it opens a dialog, and the one-line text
  dialog's entry passes Escape on itself (`escEntry`). Escape in an
  entry of the attack or linked dialogs does nothing until the focus
  leaves it. The file choosers are Fyne's and keep their own keys.
- **A Fyne dialog is not a window**, so `test/quit.sh` can't find
  Limits with xdotool as po_ova_cg's did. It sees Limits open by its
  letting go of Character Name's focus (an `x` typed then changes
  nothing; typed without Ctrl+E, the title gets its ` *`); Escape
  closing it is `TestEscape`.
- **Fyne's test window doesn't run the close intercept**, so the
  close button's save-first question is tested through `Window.Quit`,
  which the intercept calls, and on the real program only by Ctrl+Q.
- **The viewer stays the program's child while it runs** (no double
  fork in Go); the reaping goroutine keeps it from being a zombie, and
  it is re-parented when the program exits.
- **`widget.List.Select` of the line already selected does nothing**,
  so a test's select unselects first.
- **Fyne adds a Quit item to the first menu** unless its last item is
  labelled "Quit"; ours is, so it keeps its own action (save first).
- **Fyne writes its own settings and cache** under `~/.config/fyne/`
  and `~/.cache/fyne/` (by application ID), apart from our preferences;
  `make test-gui` and `quit.sh` give the program a scratch `HOME` so
  neither reaches the user's.

## Open questions

None yet.
