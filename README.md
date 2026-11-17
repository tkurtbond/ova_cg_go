# ova_cg_go - an OVA character generator in Go, with Fyne

`ova_cg_go` builds a character for OVA, the Open Versatile Anime engine,
in a window laid out like the official character sheet: pick Abilities
and Weaknesses by name and level, build attacks from Perks and Flaws,
and it works out Endurance costs, ROLL, DX, Defense, Health, Endurance
and Threat Value as you go. It holds no rules text: each name shows its
rulebook page, and the program can open the rulebook's PDF at that page
in your PDF viewer. Characters are saved as YAML that
[ova_fmt_po](../../RPG/Tools/ova_fmt_po) and the Ada ova_fmt format.

It is a port of ova_cg_po (`~/Repos/RPG/Tools/ova_cg_po`, Oberon-2
with FLTK) to Go, with [Fyne](https://fyne.io) v2 for the window and
`go.yaml.in/yaml/v3` for YAML. The rules, the file format and the
window's contents are ova_cg_po's; [PLAN.md](PLAN.md) says what the
port changed and why.

**Status: complete** (the ten phases of PLAN.md). It covers what the
character sheet holds: the names, the Abilities and Weaknesses, the
attacks with their Perks and Flaws, the derived numbers with an
override beside each, the limits of p. 21, linked content (a
Transformation's form, a Magic's spells, a Companion or a Vehicle), and
the sheet's other half, the background, portrait, fluff and notes.

## Usage

```
ova_cg_go [options] [file.yaml]
```

`-h` lists the options: `-r`/`--rulebook PATH` gives the rulebook's PDF
(else `OVA_RULEBOOK`, else the one chosen with Rulebook > Choose
PDF..., else `~/Reference/RPG/OVA/`), and `-v`/`--viewer` the viewer
(`okular`, `evince`, `mupdf`, `xdg-open`, or a command with `%f` for
the PDF, `%p` its page and `%l` the printed page), else the one chosen
with Rulebook > Viewer.... Those two choices are kept in
`$XDG_CONFIG_HOME/ova_cg_go/ova_cg_go.prefs` (else under `~/.config`;
on Windows, under `%AppData%`). A file
that can't be read, or holds mistakes, is reported and the program
exits with status 1; a mistake on the command line exits with status 2.
The program needs a display (`DISPLAY` or `WAYLAND_DISPLAY`); without
one it says so and exits with status 1.

## Using the window

The window is the character sheet in four tabs, narrow enough to sit
beside the rulebook in your PDF viewer: **Character** and **Attacks**,
the sheet's right half, and **Background** and **Fluff**, its left;
and a fifth, **Markdown**, the sheet as text.
Every change goes into the character at once and every number is
recomputed as you work; the title shows the file's name, and ` *`
after it while there are unsaved changes.

### Menus

| Menu | Item | Key |
|---|---|---|
| File | New, Open..., Save, Save As..., Quit | Ctrl+N, Ctrl+O, Ctrl+S, Ctrl+Shift+S, Ctrl+Q |
| Character | Limits... | Ctrl+E |
| Character | Transformed form | Ctrl+T |
| Character | Add Character, Duplicate Character, Remove Character... | |
| Character | Move Up, Move Down | |
| Character | Next Character, Previous Character | Ctrl+PgDn, Ctrl+PgUp |
| Rulebook | Open, Choose PDF..., Viewer... | |

New, Open and Quit ask first whether to save unsaved changes (Save,
Don't save, Cancel), and so does the window's close button.

### Several characters in a file

A file can hold several characters, such as an NPC and their minions.
The window shows one at a time. The menu at the right of the tabs
lists them by number and name: choose one to show it. The title says
which is shown, of how many ("Acacia.yaml (1 of 3)").

- **Add Character** puts a new, empty character last, and
  **Duplicate Character** a copy of the one shown.
- **Remove Character...** asks first. A file keeps at least one
  character, so it is greyed out when there is only one.
- **Move Up** and **Move Down** change the order. ova_fmt sets every
  character after the first as a subentry, so put the main one first.
- **Next Character** and **Previous Character** step through them.

**Save** writes every character to the file, in order, and the
question about unsaved changes covers all of them.

The dialogs open inside the window. Escape cancels or closes the one
on top, unless a text field in it has the cursor.

### The Character tab

**NPC Type**, beside the Tagline, is none for a player's character, or
Heroic, Secondary, Extra or Boss for an NPC (p. 123): it sets where Health
and Endurance start (40, 20 or 10; a Boss has a character's 40) and the
TV's last step (4 less for a Secondary NPC, always 1 for an Extra), and
is saved with the character.

A Boss's Abilities offer the four Boss Abilities (p. 123) as well, below
a "Boss Abilities" row at the end of the list. Each adds 5 to the TV
(p. 124), and, since a Boss applies and removes them as needed, none
counts in the levels or points; its level stays 1. A sheet that stops
being a Boss keeps them, with a warning, until they are removed.

**Abilities and Weaknesses.** Each list shows a trait's level, name
and detail, its points when Scaled Cost is on, and the rulebook page
its name is on. To add one, choose
its name from the menu under the list (each name shows its page), set
its level and its detail, and press **Add**. Clicking a line puts it
back in those controls: change them and press **Change** to replace
it, or **Remove** to take it out. **?** opens the rulebook at the
chosen name's page.

With an Ability selected, **Perks and Flaws...** opens the attack
dialog for its Perks and Flaws, without the attack's own fields; its
END, when it has one, is beside the button. An Ability with linked
content turns the button under the Weaknesses into one for it:

- **Form...**, a Transformation's: the transformed form's Abilities
  and Weaknesses, two levels for each level of Transformation.
- **Spells...**, a Magic's: each spell's name, the Ability it mimics,
  its level, and its END. Witchcraft's spells are curses: each casts a
  Weakness, at -1 to -3, and costs by the Weakness's size (p. 58).
- **Companion...** or **Vehicle...**: the partner's own sheet: its
  name, whether it is a Fragment, whether it has combat stats, and its
  Abilities, Weaknesses, attacks and numbers.

These dialogs change the character as you work, and **Close** closes
them.

**Totals and warnings.** Under the lists are the Abilities' and
Weaknesses' levels and the Character Total (and, when Scaled Cost is
on, the Abilities' points and the Scaled Cost points, the Abilities'
less the Weaknesses'). Warnings say what the rules find wrong, with
the pages of the rules: a name the rulebook doesn't have (shown with
`p. ?`), a level out of range, a Perk taken too often, a Perk or Flaw
the rules don't allow with the others, a limit broken. A total a limit
finds too high turns red.

**Overrides.** Beside every computed number is an empty box. Type a
number there to override it, for a GM's ruling, a house rule, or a
case the rules here don't cover: the computed number stays shown
beside it, and anything computed from it uses the override. A number
is an integer, except that an attack's DX may be `1/2` and a spell's
END may end in `+` (`10+`); anything else is refused with a message.
Empty the box to go back to the computed number. A file whose numbers
differ from the rules' keeps them as overrides, and says which when it
opens.

**Character > Transformed form** shows and computes the sheet in the
Transformation's form (p. 63), and is saved with the character.

**Character > Limits...** turns on the rules of p. 21 and 57: Base
Zero (the Character Total within so many points of zero), starting
levels, a Power Ceiling, Scaled Cost, Knowledge and Ineptitude at half
cost, and Combined Effectiveness. Each number is active only while its
rule is on, and each rule has a **?**. The window follows each change;
**OK** keeps them, and **Cancel** or Escape puts them back. They are saved with the character.

### The Attacks tab

**Attacks.** Each attack is a row with its ROLL, DX and END. **Add
attack...** and **Edit** open the attack dialog: the attack's name,
affinity and description, whether it is made with Telekinesis or in
the transformed form, and its Perks and Flaws, chosen as the traits
are, with how many times (for one that can be taken more than once)
and a cost (for one whose cost you choose: Cancel, Requirement,
Unique). Its ROLL, DX and END follow each change. **OK** keeps it;
**Cancel** puts it back as it was, and takes out an attack just added.
**Remove** deletes the row's attack.

Below the attacks are DEFENSE, HEALTH, ENDURANCE and TV, with
Barrier's and Incorporeal Form's alternate DEFENSE and an Endurance
Reserve when the character has them.

### The Background tab

The Summary, and the Background, Appearance, Personality, Character
Hooks and Other Notes.

### The Fluff tab

The Character Portrait, an image chosen with **Choose...** (its path
is saved relative to the character's file, and **Remove** takes it
off); the sheet's fluff (Concept, Age, Birthday and the rest); and the
notes, each a header and text, added, changed and removed as the
traits are.

### The Markdown tab

The character's sheet in Markdown, the same text `ova_fmt_po -M`
makes of the saved file, kept up to date as you work. Select part of
it, or press **Copy** to put all of it on the clipboard, to paste
elsewhere; **Save Markdown...** writes it to a file (it suggests the
character's name, with `.md`). Saving the sheet doesn't save the
character.

Check **Inline** to see the character instead as one paragraph, as an
adventure's cast lists it, the same text `ova_fmt_po -i` makes: the
bold name and NPC type, the prose run together, the Abilities and
Weaknesses in italics as "Attack +2 (Blaster rifle)", then the attacks
and the combat numbers, filled to 70 columns. Copy and Save Markdown...
then take the paragraph (Save suggests `NAME-inline.md`).

### The rulebook

The program holds no rules text. Every name shows its page in the
rulebook, and **?** or **Rulebook > Open** opens the rulebook's PDF in
your PDF viewer, reusing a viewer that already shows it (okular or
evince). If the program can't find the PDF it asks for it once.
**Rulebook > Choose PDF...** and **Viewer...** choose the PDF and the
viewer, and they are kept for the next time.

## Building

You need:

- Go 1.26 or later.
- A C compiler and the X11 and OpenGL development headers, for Fyne's
  GLFW driver (cgo). On Fedora: `gcc libX11-devel libXcursor-devel
  libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel
  mesa-libGL-devel`. The first build compiles them: a minute or two.
- For a native Wayland build (`make TAGS=wayland`): the Wayland and
  xkbcommon development headers. Otherwise the program runs on Wayland
  through XWayland.
- For `make test`: ova_fmt_po, built, at `OVA_FMT_PO` (default
  `~/Repos/RPG/Tools/ova_fmt_po/ova_fmt_po`): what the generator saves
  must format as the file it loaded.
- For `make test-headless`: Xvfb (`xvfb-run`) and `xdotool`; for
  `make test-sway`, sway.
- For `make test-viewers`: sway (with Xwayland), okular, evince, mupdf,
  `gdbus`, `dbus-run-session`, grim, jq and python3.
- For `make check-pages`: mutool, python3 and the rulebook's PDF at
  `RULEBOOK`.

```sh
make                 # ./ova_cg_go
make test            # go test ./..., test/files.sh, test/cli.sh: no display needed
make check           # go vet and go test -race
make test-headless   # the program itself, on Xvfb
make test-sway       # the same, on a headless sway, built for Wayland
make test-viewers    # the real PDF viewers, on a headless sway of its own
make check-pages     # the rules' pages against the rulebook's headings
make install         # to $(PREFIX)/bin, default ~/local/bin
make uninstall
make clean
```

### Windows and macOS

`make windows` and `make macos` cross-build in podman containers, into
`obj/cross/`. macOS needs Apple's SDK, which you supply. See
[CROSS.md](CROSS.md).

## OVA and the rulebook

OVA, *OVA: The Anime Role-Playing Game*, is © 2015 Wise Turtle. This
project is not affiliated with or endorsed by Wise Turtle. It holds
none of the rulebook's descriptions: it knows the names of the rules'
Abilities, Weaknesses, Perks and Flaws, the numbers attached to them,
and the page each is described on, and it opens your own copy of the
rulebook at that page. You need the rulebook to use it.

The test fixtures in `testdata/` record the rulebook's sample
characters so that the program's numbers can be checked against them;
their backgrounds and descriptions are paraphrased, and only short
captions such as attack descriptions follow the book's wording.

## Documents

- [PLAN.md](PLAN.md): how the port maps ova_cg_po to Go and Fyne, what
  it decided differently, and what it found.
- [AGENTS.md](AGENTS.md): working notes on the build and the tests.
- [CROSS.md](CROSS.md): building for Windows and macOS.
- ova_cg_po's PLAN.md: the design, and the rules the program computes.
