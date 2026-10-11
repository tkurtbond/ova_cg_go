package ui

// po_ova_cg's TestBackground: the second tab, driven as a user would,
// and what it saves. The text, the fluff, the notes and the portrait
// shown from a character, each change made as the user makes it, and
// checked in the model, in the window's title, and in what is saved.
// The portrait is an image this test writes, saved as a path relative
// to the YAML file, and rebased when Save As moves the file.

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/tkurtbond/go_ova_cg/internal/entities"
	"github.com/tkurtbond/go_ova_cg/internal/paths"
)

const (
	imagePath = objDir + "face.png"      // written here
	savePath  = objDir + "files/bg.yaml" // beside it, in obj/files
	badPath   = objDir + "no/such/dir/bg.yaml"
)

// bgShown says whether every field of the tab shows the character's
// text, trimmed as the tab trims it.
func bgShown(w *Window) bool {
	b, c := w.Background, w.Character
	ok := trim(b.SummaryIn.Text) == c.Summary &&
		trim(b.Prose[ProseBackground].Text) == c.Background &&
		trim(b.Prose[ProseAppearance].Text) == c.Appearance &&
		trim(b.Prose[ProsePersonality].Text) == c.Personality &&
		trim(b.Prose[ProseCharacterHooks].Text) == c.CharacterHooks &&
		trim(b.Prose[ProseOtherNotes].Text) == c.OtherNotes &&
		b.PortraitPath.Text == c.Portrait
	for i, e := range b.FluffIn {
		ok = ok && trim(e.Text) == c.Fluff[i]
	}
	return ok
}

// notesShown says whether the notes' list is the character's notes'
// headers.
func notesShown(w *Window) bool {
	n := w.Character.Notes
	if len(n) != w.Background.Notes.Count() {
		return false
	}
	for i, x := range n {
		if w.Background.Notes.Line(i) != x.Header {
			return false
		}
	}
	return true
}

// marked says whether the character is modified, and the title says so.
func marked(w *Window) bool { return w.Modified && strings.HasSuffix(w.Win.Title(), " *") }

func TestBackground(t *testing.T) {
	w, f := newWindow(t)
	f.answer = Discard
	b := w.Background
	has := func(s string) bool { return strings.Contains(saved(w), s) }
	says := func(s string) bool { return b.PortraitText.Text == s }

	// An image to choose: a gradient, 40 by 60, as a PNG.
	img := image.NewRGBA(image.Rect(0, 0, 40, 60))
	for y := range 60 {
		for x := range 40 {
			img.Set(x, y, color.RGBA{uint8(x * 6), uint8(y * 4), 128, 255})
		}
	}
	out, err := os.Create(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(out, img); err != nil || out.Close() != nil {
		t.Fatal("an image written to choose")
	}

	check(t, w.Tabs.Selected() == w.SheetTab, "the Character tab first")
	w.Tabs.Select(w.BackgroundTab)
	w.Win.Show()
	check(t, w.Tabs.Selected() == w.BackgroundTab, "the Background tab, shown")
	check(t, bgShown(w) && notesShown(w) && says("No portrait.") && !b.ImageShown() && b.RemovePortrait.Disabled(),
		"a new character: every field empty, no portrait")

	// ---- Shown from a file ----
	ok := load(w, "Sheet.yaml")
	c := w.Character
	check(t, ok && bgShown(w), "Sheet.yaml: the summary, the prose sections, the fluff and the portrait's path shown")
	check(t, c.HasBackground && c.HasAppearance && c.HasPersonality && c.HasOtherNotes && c.HasSummary &&
		c.HasFluffField[entities.Concept], "and it has each")
	check(t, b.PortraitPath.Text == "images/hana.png" && !b.ImageShown() && says("The image can't be read.") &&
		!b.RemovePortrait.Disabled(), "its portrait, not there: the path, and why there's no image")
	ok = load(w, "Jiro.yaml")
	check(t, ok && bgShown(w) && w.Character.HasCharacterHooks, "Jiro.yaml: his Character Hooks shown")
	ok = load(w, "Raine.yaml")
	check(t, ok && len(w.Character.Notes) > 0 && notesShown(w), "Raine.yaml: the notes' headers listed")
	check(t, b.Notes.Line(0) == "Nightmares", "the first, Nightmares")
	b.Notes.Select(0)
	c = w.Character
	check(t, trim(b.NoteHeader.Text) == c.Notes[0].Header && trim(b.NoteText.Text) == c.Notes[0].Text,
		"a note selected: its header and text to edit")

	// ---- Text typed ----
	load(w, "Sheet.yaml")
	c = w.Character
	check(t, !w.Modified, "loaded: nothing changed")
	e := b.Prose[ProseAppearance]
	e.SetText(e.Text + " She hums.")
	check(t, strings.Contains(c.Appearance, "coat that never stops moving. She hums.") && marked(w),
		"typing in Appearance: the character's, and the title marked")
	check(t, has("She hums."), "and saved")
	e.SetText("  " + e.Text)
	check(t, c.Appearance[0] == 'S', "blanks at the ends are trimmed, as the loader trims them")
	e.SetText("   ")
	check(t, !c.HasAppearance && c.Appearance == "" && !has("appearance:"), "emptied: not saved")
	b.Prose[ProseOtherNotes].SetText("Owes two favors now.")
	check(t, has("other-notes: Owes two favors now."), "Other Notes saved")
	b.Prose[ProseCharacterHooks].SetText("Needs a courier.")
	check(t, c.HasCharacterHooks && has("character-hooks: Needs a courier."), "Character Hooks saved")
	b.SummaryIn.SetText("A schoolgirl and a storm.")
	check(t, c.Summary == "A schoolgirl and a storm." && has("summary: A schoolgirl and a storm."),
		"the summary typed and saved")

	// ---- The fluff ----
	b.FluffIn[entities.Hair].SetText("Black")
	check(t, c.HasFluffField[entities.Hair] && has("hair: Black"), "Hair typed: saved in fluff:")
	b.FluffIn[entities.FavoriteFood].SetText("Melon bread")
	check(t, has("favorite-food: Melon bread"), "Favorite Food: its key, favorite-food")
	for _, e := range b.FluffIn {
		e.SetText(" ")
	}
	check(t, !c.HasFluff && !has("fluff:"), "every field emptied: no fluff: block")
	b.FluffIn[entities.Age].SetText("15")
	// Text that plain YAML would read as a number is quoted: libfyaml
	// wrote it plain.
	check(t, c.HasFluff && has("fluff:") && has(`age: "15"`), "one field again: the block again")
	check(t, bgShown(w), "and every field still shows the character's")

	// ---- The notes ----
	f.tells = 0
	b.NoteHeader.SetText("Rivals")
	b.NoteText.SetText("")
	b.AddNote.OnTapped()
	check(t, f.tells == 1 && len(c.Notes) == 0, "a note with no text: said so, not added")
	b.NoteText.SetText("The track team's captain.")
	b.AddNote.OnTapped()
	b.NoteHeader.SetText("Secrets")
	b.NoteText.SetText("Nobody knows about the Gale.")
	b.AddNote.OnTapped()
	check(t, len(c.Notes) == 2 && notesShown(w) && b.Notes.Selected() == 1, "two added, in order, the last selected")
	text := saved(w)
	check(t, strings.Contains(text, "header: Rivals") && strings.Contains(text, "The track team's captain.") &&
		strings.Contains(text, "header: Secrets"), "and saved")
	b.Notes.Select(0)
	b.NoteHeader.SetText("Rival")
	b.ChangeNote.OnTapped()
	check(t, c.Notes[0].Header == "Rival" && notesShown(w) && b.Notes.Selected() == 0,
		"Change: the header changed, in its place")
	b.RemoveNote.OnTapped()
	check(t, len(c.Notes) == 1 && c.Notes[0].Header == "Secrets" && notesShown(w), "Remove")
	b.RemoveNote.OnTapped()
	check(t, len(c.Notes) == 1, "Remove with nothing selected removes nothing")

	// ---- The portrait ----
	// Sheet.yaml is in testdata; the image in obj.
	f.chosenImage = paths.Absolute(imagePath, "")
	b.ChoosePortrait.OnTapped()
	check(t, c.Portrait == "../obj/face.png" && bgShown(w), "an image chosen: its path relative to the character's file")
	check(t, b.ImageShown() && says("") && !b.RemovePortrait.Disabled(), "and shown")
	check(t, has("portrait: ../obj/face.png"), "and saved so")
	f.tells, f.chosen = 0, badPath
	w.SaveAsItem.Action()
	check(t, f.tells == 1 && c.Portrait == "../obj/face.png" && w.Win.Title() == "OVA Character Generator - Sheet.yaml *",
		"a save that fails: said so, the path as it was")
	f.chosen = savePath
	w.SaveAsItem.Action()
	check(t, c.Portrait == "../face.png" && w.Win.Title() == "OVA Character Generator - bg.yaml" && b.ImageShown() &&
		bgShown(w), "Save As to obj/files: the path rebased, the image still shown")
	text = saved(w)
	ok = w.Load(savePath)
	c = w.Character
	check(t, ok && f.tells == 1 && c.Portrait == "../face.png" && b.ImageShown() && bgShown(w) && notesShown(w),
		"loaded again: the portrait found from the file's directory")
	check(t, saved(w) == text, "and everything the tab set, the same")
	f.chosenImage = ""
	b.ChoosePortrait.OnTapped()
	check(t, c.Portrait == "../face.png" && !w.Modified, "the chooser cancelled: nothing changes")
	b.RemovePortrait.OnTapped()
	check(t, !c.HasPortrait && !has("portrait:") && says("No portrait.") && !b.ImageShown() &&
		b.RemovePortrait.Disabled() && marked(w), "Remove: no portrait")

	// A character never saved holds the absolute path, until its first
	// save.
	w.NewItem.Action()
	c = w.Character
	check(t, bgShown(w) && notesShown(w) && !b.ImageShown(), "New: every field empty again")
	f.chosenImage = imagePath
	b.ChoosePortrait.OnTapped()
	check(t, c.Portrait == paths.Absolute(imagePath, "") && b.ImageShown(), "no file yet: the image's absolute path")
	w.NameIn.SetText("Someone")
	w.SaveItem.Action()
	check(t, c.Portrait == "../face.png" && !w.Modified && b.ImageShown(), "the first save: made relative")
}
