package ui

// The window's second tab: the sheet's left half, the fluff, and the
// notes (po_ova_cg's OvaBackground).
//
// po_ova_cg's PLAN.md, "The window", the second tab: the Summary;
// Background, Appearance, Personality and Other Notes, each a multi-line
// entry; the Character Portrait, an image file whose path is saved
// relative to the YAML file; the fluff fields (p. 20) in a grid; and the
// headed notes ova_fmt prints as callouts (notes:), so everything a file
// holds can be edited.
//
// The window owns the character and the file: it shows a character here
// (Show), and is told of each change. Every change goes into the
// character at once, as the user types; nothing here holds the only
// copy.

import (
	"bytes"
	"image"
	_ "image/gif"  // portraits
	_ "image/jpeg" // portraits
	_ "image/png"  // portraits
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	_ "golang.org/x/image/bmp" // portraits

	"github.com/tkurtbond/go_ova_cg/internal/character"
	"github.com/tkurtbond/go_ova_cg/internal/entities"
	"github.com/tkurtbond/go_ova_cg/internal/paths"
)

// The prose sections, Background.Prose's indices.
const (
	ProseBackground = iota
	ProseAppearance
	ProsePersonality
	ProseOtherNotes
	NProse
)

// The portrait's box.
const (
	PortraitW = 160
	PortraitH = 200
)

// fluffLabels are the fluff fields' labels, by entities' constants.
var fluffLabels = [entities.NFluff]string{
	entities.Concept: "Concept", entities.Birthday: "Birthday", entities.Age: "Age",
	entities.Gender: "Gender", entities.Height: "Height", entities.Weight: "Weight",
	entities.Hair: "Hair", entities.Eyes: "Eyes", entities.Association: "Association",
	entities.Hobbies: "Hobbies", entities.FavoriteFood: "Favorite Food", entities.Allies: "Allies",
	entities.Enemies: "Enemies", entities.LoveInterests: "Love Interests",
}

// Background is the window's second tab.
type Background struct {
	w       *Window
	object  fyne.CanvasObject
	fileDir string // the YAML file's directory, absolute; "": never saved
	filling bool   // the widgets being set from the character: no change to report

	SummaryIn                       *widget.Entry
	Prose                           [NProse]*widget.Entry
	Portrait                        *canvas.Image // the portrait shown; its Resource nil: none
	PortraitText                    *widget.Label // why there is none
	PortraitPath                    *widget.Label // the path as saved
	ChoosePortrait, RemovePortrait  *widget.Button
	FluffIn                         [entities.NFluff]*widget.Entry
	Notes                           *lines // the notes' headers, in order
	NoteHeader, NoteText            *widget.Entry
	AddNote, ChangeNote, RemoveNote *widget.Button
}

func (b *Background) character() *character.Character { return b.w.Character }

func (b *Background) changed() {
	if !b.filling {
		b.w.textChanged()
	}
}

// ---- The prose sections ----

func (b *Background) proseOf(which int) (has *bool, text *string) {
	c := b.character()
	switch which {
	case ProseBackground:
		return &c.HasBackground, &c.Background
	case ProseAppearance:
		return &c.HasAppearance, &c.Appearance
	case ProsePersonality:
		return &c.HasPersonality, &c.Personality
	}
	return &c.HasOtherNotes, &c.OtherNotes
}

// ---- The portrait ----

// readImage is the image at path, if it can be read: a PNG, JPEG, GIF,
// BMP or SVG.
func readImage(path string) (fyne.Resource, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	if strings.EqualFold(filepath.Ext(path), ".svg") {
		if !bytes.Contains(data, []byte("<svg")) {
			return nil, false
		}
	} else if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
		return nil, false
	}
	return fyne.NewStaticResource(filepath.Base(path), data), true
}

// showPortrait shows the character's portrait, or says why it can't.
func (b *Background) showPortrait() {
	c := b.character()
	b.PortraitPath.SetText(c.Portrait)
	b.Portrait.Resource = nil
	switch {
	case !c.HasPortrait:
		b.PortraitText.SetText("No portrait.")
	default:
		if r, ok := readImage(paths.Absolute(c.Portrait, b.fileDir)); ok {
			b.Portrait.Resource = r
			b.PortraitText.SetText("")
		} else {
			b.PortraitText.SetText("The image can't be read.")
		}
	}
	b.Portrait.Refresh()
	enable(c.HasPortrait, b.RemovePortrait)
}

// ImageShown says whether an image is shown, and PortraitSays what the
// box says instead: for tests.
func (b *Background) ImageShown() bool { return b.Portrait.Resource != nil }

func (b *Background) choosePortrait() {
	b.w.ChooseImage(func(p string) {
		if p == "" {
			return
		}
		// Relative to the file once there is one; until then absolute,
		// for the first save to make relative (paths.Rebase).
		if b.fileDir != "" {
			p = paths.Relative(p, b.fileDir)
		} else {
			p = paths.Absolute(p, "")
		}
		c := b.character()
		character.SetText(&c.HasPortrait, &c.Portrait, p)
		b.showPortrait()
		b.changed()
	})
}

func (b *Background) removePortrait() {
	c := b.character()
	character.SetText(&c.HasPortrait, &c.Portrait, "")
	b.showPortrait()
	b.changed()
}

// ---- The summary and the fluff ----

func (b *Background) fluffChanged(i int, s string) {
	if b.filling {
		return
	}
	c := b.character()
	character.SetText(&c.HasFluffField[i], &c.Fluff[i], trim(s))
	// The fluff: block is written while any field has text.
	c.HasFluff = false
	for _, has := range c.HasFluffField {
		c.HasFluff = c.HasFluff || has
	}
	b.changed()
}

// ---- The notes ----

// showNotes lists the notes' headers again, line sel selected (-1:
// none).
func (b *Background) showNotes(sel int) {
	c := b.character()
	var rows [][]string
	for _, n := range c.Notes {
		rows = append(rows, []string{n.Header})
	}
	b.Notes.Set(rows, sel)
	if sel >= 0 && sel < len(c.Notes) {
		b.NoteHeader.SetText(c.Notes[sel].Header)
		b.NoteText.SetText(c.Notes[sel].Text)
	} else {
		b.NoteHeader.SetText("")
		b.NoteText.SetText("")
	}
}

func (b *Background) noteSelected(i int) {
	if n := b.character().Notes; i >= 0 && i < len(n) {
		b.NoteHeader.SetText(n[i].Header)
		b.NoteText.SetText(n[i].Text)
	}
}

// noteRead is the header and text typed; ok false, said why, if either
// is blank: the loader requires both.
func (b *Background) noteRead() (header, text string, ok bool) {
	header, text = trim(b.NoteHeader.Text), trim(b.NoteText.Text)
	if header == "" || text == "" {
		b.w.tell("A note needs a header and some text.")
		return "", "", false
	}
	return header, text, true
}

// noteButton is what Add, Change and Remove do: Add puts the note last;
// Change and Remove need a line selected.
func (b *Background) noteButton(button *widget.Button) {
	c := b.character()
	line := b.Notes.Selected()
	switch {
	case button == b.AddNote:
		header, text, ok := b.noteRead()
		if !ok {
			return
		}
		c.Notes = append(c.Notes, &character.Note{Header: header, Text: text})
		line = len(c.Notes) - 1
	case line < 0:
		return
	case button == b.ChangeNote:
		header, text, ok := b.noteRead()
		if !ok {
			return
		}
		c.Notes[line].Header, c.Notes[line].Text = header, text
	default:
		c.Notes = append(c.Notes[:line:line], c.Notes[line+1:]...)
		line = -1
	}
	b.showNotes(line)
	b.changed()
}

// ---- Showing a character ----

// Show shows the window's character, whose file is in directory dir ("":
// none).
func (b *Background) Show(c *character.Character, dir string) {
	b.fileDir = dir
	b.filling = true
	b.SummaryIn.SetText(c.Summary)
	for i := range b.Prose {
		_, text := b.proseOf(i)
		b.Prose[i].SetText(*text)
	}
	for i, e := range b.FluffIn {
		e.SetText(c.Fluff[i])
	}
	b.showNotes(-1)
	b.showPortrait()
	b.filling = false
}

// SetDir says the file is now in directory dir: a portrait path is shown
// from it. The window has rebased the path already (paths.Rebase).
func (b *Background) SetDir(dir string) {
	b.fileDir = dir
	b.showPortrait()
}

// ---- Building ----

func newBackground(w *Window) *Background {
	b := &Background{w: w}
	b.SummaryIn = widget.NewEntry()
	b.SummaryIn.OnChanged = func(s string) {
		if !b.filling {
			c := b.character()
			character.SetText(&c.HasSummary, &c.Summary, trim(s))
			b.changed()
		}
	}
	titles := [NProse]string{"Background", "Appearance", "Personality", "Other Notes"}
	var prose []fyne.CanvasObject
	for i := range b.Prose {
		e := widget.NewMultiLineEntry()
		e.Wrapping = fyne.TextWrapWord
		// Each section's text trimmed as the loader trims it, so what is
		// saved loads the same; blank is absent.
		e.OnChanged = func(s string) {
			if !b.filling {
				has, text := b.proseOf(i)
				character.SetText(has, text, trim(s))
				b.changed()
			}
		}
		b.Prose[i] = e
		prose = append(prose, container.NewBorder(bold(titles[i]), nil, nil, nil, e))
	}
	b.Portrait = canvas.NewImageFromResource(nil)
	b.Portrait.FillMode = canvas.ImageFillContain
	b.PortraitText = widget.NewLabel("")
	b.PortraitText.Wrapping = fyne.TextWrapWord
	b.PortraitPath = widget.NewLabel("")
	b.PortraitPath.Truncation = fyne.TextTruncateEllipsis
	b.ChoosePortrait = widget.NewButton("Choose...", b.choosePortrait)
	b.RemovePortrait = widget.NewButton("Remove", b.removePortrait)
	var fluff []fyne.CanvasObject
	for i := range b.FluffIn {
		e := widget.NewEntry()
		e.OnChanged = func(s string) { b.fluffChanged(i, s) }
		b.FluffIn[i] = e
		l := widget.NewLabel(fluffLabels[i])
		l.Alignment = fyne.TextAlignTrailing
		fluff = append(fluff, l, e)
	}
	b.Notes = newLines()
	b.Notes.OnSelect = b.noteSelected
	b.NoteHeader = widget.NewEntry()
	b.NoteText = widget.NewMultiLineEntry()
	b.NoteText.Wrapping = fyne.TextWrapWord
	b.AddNote = widget.NewButton("Add", func() { b.noteButton(b.AddNote) })
	b.ChangeNote = widget.NewButton("Change", func() { b.noteButton(b.ChangeNote) })
	b.RemoveNote = widget.NewButton("Remove", func() { b.noteButton(b.RemoveNote) })

	left := container.NewBorder(
		container.New(layout.NewFormLayout(), widget.NewLabel("Summary"), b.SummaryIn), nil, nil, nil,
		container.NewGridWithRows(NProse, prose...))
	portrait := container.NewHBox(
		sized(container.NewStack(b.Portrait, b.PortraitText), PortraitW, PortraitH),
		container.NewVBox(widget.NewLabel("Image file, relative to the character's"),
			sized(b.PortraitPath, 380, 0), container.NewHBox(b.ChoosePortrait, b.RemovePortrait)))
	notes := container.NewBorder(nil, container.NewHBox(b.AddNote, b.ChangeNote, b.RemoveNote), nil, nil,
		container.NewGridWithColumns(2, b.Notes.List,
			container.NewBorder(container.New(layout.NewFormLayout(), widget.NewLabel("Header"), b.NoteHeader),
				nil, nil, nil, b.NoteText)))
	right := container.NewBorder(
		container.NewVBox(bold("Character Portrait"), portrait, container.NewGridWithColumns(4, fluff...),
			bold("Notes")),
		nil, nil, nil, notes)
	// A split, as the sheet's (window.go): a grid of two would be twice
	// the wider half.
	split := container.NewHSplit(left, right)
	split.SetOffset(0.5)
	b.object = split
	return b
}
