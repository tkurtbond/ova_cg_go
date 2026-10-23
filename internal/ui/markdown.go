package ui

// The window's last tab: the character's sheet in Markdown, to copy or
// save (ova_cg_po's OvaMarkdownTab).
//
// The sheet the markdown package makes, shown as text the user can
// select, with Copy, which puts all of it on the clipboard, and Save
// Markdown..., which writes it to a file. Inline, checked, shows the
// inline paragraph instead (markdown.Inline), as an adventure's cast
// lists the character. The window shows the character here again after
// each change (Show), so the text is always the character's.

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/tkurtbond/ova_cg_go/internal/character"
	"github.com/tkurtbond/ova_cg_go/internal/markdown"
)

// MarkdownView is the Markdown tab's content.
type MarkdownView struct {
	w          *Window
	Sheet      string        // the text shown
	Text       *widget.Label // selectable, monospaced
	Copy, Save *widget.Button
	Inline     *widget.Check // checked: the inline paragraph
	name       string        // the character's, for the file chooser's suggestion
	object     fyne.CanvasObject
}

func newMarkdownView(w *Window) *MarkdownView {
	m := &MarkdownView{w: w}
	m.Text = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})
	m.Text.Wrapping = fyne.TextWrapWord
	m.Text.Selectable = true
	m.Copy = widget.NewButton("Copy", m.copy)
	m.Save = widget.NewButton("Save Markdown...", m.save)
	// The character shown again, the other way.
	m.Inline = widget.NewCheck("Inline", func(bool) { m.Show(m.w.Character) })
	m.object = container.NewBorder(container.NewHBox(m.Copy, m.Save, m.Inline, layout.NewSpacer()), nil, nil, nil,
		container.NewVScroll(m.Text))
	return m
}

// Show shows c's sheet, or its inline paragraph if Inline is checked.
func (m *MarkdownView) Show(c *character.Character) {
	if m.Inline.Checked {
		m.Sheet = markdown.Inline(c)
	} else {
		m.Sheet = markdown.Sheet(c)
	}
	m.name = c.Name
	m.Text.SetText(m.Sheet)
}

// copy is Copy: the whole sheet on the clipboard.
func (m *MarkdownView) copy() { m.w.App.Clipboard().SetContent(m.Sheet) }

// save is Save Markdown...: the sheet in a file the user chooses, the
// character's name suggested: NAME.md, or NAME-inline.md for the inline
// paragraph.
func (m *MarkdownView) save() {
	suggested := ""
	if m.name != "" && m.Inline.Checked {
		suggested = m.name + "-inline.md"
	} else if m.name != "" {
		suggested = m.name + ".md"
	}
	sheet := m.Sheet
	m.w.ChooseMarkdownPath(suggested, func(p string) {
		if p == "" {
			return
		}
		if err := markdown.Save(sheet, p); err != nil {
			m.w.Tell("couldn't write " + p + ": " + err.Error())
		}
	})
}
