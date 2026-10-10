package ui

// What would wait for the user, done with Fyne's dialogs: the window's
// Tell, Ask, ChoosePath, ChoosePdf, AskText and ChooseImage unless a test
// sets its own. Each dialog is in the window and modal, and answers
// through its callback.

import (
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// escape is what Escape does in an open dialog.
type escape struct{ do func() }

// modal shows d, Escape doing esc while it is the top dialog. Escape
// reaches the window only with nothing focused (a focused widget takes
// the key), so the focus is let go: Fyne's in-window dialogs have no
// Escape of their own (PLAN.md, "Findings").
func (w *Window) modal(d dialog.Dialog, esc func()) {
	e := &escape{esc}
	w.escapes = append(w.escapes, e)
	d.SetOnClosed(func() {
		for i, x := range w.escapes {
			if x == e {
				w.escapes = append(w.escapes[:i], w.escapes[i+1:]...)
				break
			}
		}
	})
	d.Show()
	w.Win.Canvas().Unfocus()
}

// Escape is the Escape key: the top dialog cancelled or closed, if one
// is open.
func (w *Window) Escape() {
	if n := len(w.escapes); n > 0 {
		w.escapes[n-1].do()
	}
}

// escEntry is an entry that passes Escape on to its dialog.
type escEntry struct {
	widget.Entry
	esc func()
}

func newEscEntry() *escEntry {
	e := &escEntry{}
	e.ExtendBaseWidget(e)
	return e
}

func (e *escEntry) TypedKey(k *fyne.KeyEvent) {
	if k.Name == fyne.KeyEscape && e.esc != nil {
		e.esc()
		return
	}
	e.Entry.TypedKey(k)
}

// chooserSize is a file chooser's size.
var chooserSize = fyne.NewSize(900, 600)

func (w *Window) dialogTell(text string) {
	d := dialog.NewCustom(Title, "OK", widget.NewLabel(text), w.Win)
	w.modal(d, d.Hide)
}

func (w *Window) dialogAsk(question string, answer func(Answer)) {
	var d *dialog.CustomDialog
	button := func(label string, a Answer) *widget.Button {
		return widget.NewButton(label, func() { d.Hide(); answer(a) })
	}
	save := button("Save", SaveFirst)
	save.Importance = widget.HighImportance
	d = dialog.NewCustomWithoutButtons(Title, container.NewVBox(widget.NewLabel(question),
		container.NewHBox(layout.NewSpacer(), save, button("Don't save", Discard), button("Cancel", Cancel))), w.Win)
	w.modal(d, func() { d.Hide(); answer(Cancel) })
}

// startIn starts d in the file's directory, if there is one.
func (w *Window) startIn(d *dialog.FileDialog, file string) {
	if file == "" {
		return
	}
	if l, err := storage.ListerForURI(storage.NewFileURI(filepath.Dir(file))); err == nil {
		d.SetLocation(l)
	}
}

// dialogPath asks for a file to open, or to save to. Fyne's save chooser
// creates (or empties) the file chosen before it answers (PLAN.md,
// "Findings"); the save writes it whole at once.
func (w *Window) dialogPath(save bool, then func(string)) {
	var d *dialog.FileDialog
	if save {
		d = dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
			if wc == nil || err != nil {
				then("")
				return
			}
			p := wc.URI().Path()
			wc.Close()
			then(p)
		}, w.Win)
		d.SetTitleText("Save the character as")
		if w.Path != "" {
			d.SetFileName(filepath.Base(w.Path))
		} else if w.Character.Name != "" {
			d.SetFileName(w.Character.Name + ".yaml")
		}
	} else {
		d = w.openChooser(then)
		d.SetTitleText("Open a character")
	}
	d.SetFilter(storage.NewExtensionFileFilter([]string{".yaml", ".yml"}))
	w.startIn(d, w.Path)
	d.Resize(chooserSize)
	d.Show()
}

// openChooser is a chooser of a file to read; then(path), "" for none.
func (w *Window) openChooser(then func(string)) *dialog.FileDialog {
	return dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if rc == nil || err != nil {
			then("")
			return
		}
		p := rc.URI().Path()
		rc.Close()
		then(p)
	}, w.Win)
}

func (w *Window) dialogPdf(then func(string)) {
	d := w.openChooser(then)
	d.SetTitleText("The rulebook's PDF")
	d.SetFilter(storage.NewExtensionFileFilter([]string{".pdf", ".PDF"}))
	w.startIn(d, w.SavedRulebook)
	d.Resize(chooserSize)
	d.Show()
}

func (w *Window) dialogImage(then func(string)) {
	d := w.openChooser(then)
	d.SetTitleText("Choose the character's portrait")
	d.SetFilter(storage.NewExtensionFileFilter([]string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg",
		".PNG", ".JPG", ".JPEG", ".GIF", ".BMP", ".SVG"}))
	w.startIn(d, w.Path)
	d.Resize(chooserSize)
	d.Show()
}

func (w *Window) dialogText(prompt, text string, then func(string, bool)) {
	e := newEscEntry()
	e.SetText(text)
	var d *dialog.CustomDialog
	ok := widget.NewButton("OK", func() { d.Hide(); then(e.Text, true) })
	ok.Importance = widget.HighImportance
	e.OnSubmitted = func(string) { ok.OnTapped() }
	cancel := widget.NewButton("Cancel", func() { d.Hide(); then("", false) })
	d = dialog.NewCustomWithoutButtons(Title, container.NewVBox(widget.NewLabel(prompt), e,
		container.NewHBox(layout.NewSpacer(), ok, cancel)), w.Win)
	e.esc = cancel.OnTapped
	d.Resize(fyne.NewSize(700, d.MinSize().Height))
	w.modal(d, cancel.OnTapped)
	w.Win.Canvas().Focus(e)
}
