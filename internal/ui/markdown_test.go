package ui

// ova_cg_po's TestMarkdown: the tab shows the markdown package's sheet of
// the character shown, after every kind of change (a name typed, an
// Ability added, Background text typed, a file opened); Copy puts it on
// the clipboard; Save Markdown... writes it where the chooser says,
// suggesting the character's name, and says why when it can't. That the
// sheet is ova_fmt_po -M's is test/files.sh's.

import (
	"os"
	"strings"
	"testing"

	"github.com/tkurtbond/ova_cg_go/internal/markdown"
)

// markdownShown says whether the tab shows the sheet of the character
// shown, as the markdown package makes it.
func markdownShown(w *Window) bool {
	want := markdown.Sheet(w.Character)
	return want != "" && w.Markdown.Sheet == want && w.Markdown.Text.Text == want
}

func TestMarkdown(t *testing.T) {
	w, f := newWindow(t)
	f.answer = Discard
	const savePath = objDir + "files/markdown-tab.md"
	const badPath = objDir + "no/such/dir/markdown-tab.md"
	sheet := func() string { return w.Markdown.Sheet }

	w.Tabs.Select(w.MarkdownTab)
	check(t, w.Tabs.Selected() == w.MarkdownTab && w.MarkdownTab.Text == "Markdown", "the Markdown tab")
	check(t, markdownShown(w), "a new character's sheet")

	// ---- Kept current ----
	w.NameIn.SetText("Kiri")
	check(t, markdownShown(w) && strings.Contains(sheet(), "# Kiri"), "a name typed: the sheet's title")
	addAbility(w, "Attack", 3)
	check(t, markdownShown(w) && strings.Contains(sheet(), "Attack +3"), "an Ability added: listed")
	e := w.Background.Prose[ProseBackground]
	e.SetText("Kiri grew up by the sea.")
	check(t, markdownShown(w) && strings.Contains(sheet(), "Kiri grew up by the sea."), "Background typed: in the sheet")
	ok := w.Load(dataDir + "Ai.yaml")
	check(t, ok && !w.Modified && markdownShown(w) && strings.Contains(sheet(), "# Ai (Amnesiatic ESPer)"),
		"a file opened: its character's sheet")

	// ---- Copy ----
	w.Markdown.Copy.OnTapped()
	check(t, w.App.Clipboard().Content() == sheet(), "Copy: the whole sheet on the clipboard")

	// ---- Save Markdown... ----
	os.Remove(savePath)
	f.chosen = savePath
	w.Markdown.Save.OnTapped()
	check(t, f.markdownAsks == 1 && f.suggested == "Ai.md", "Save Markdown...: the character's name suggested")
	b, err := os.ReadFile(savePath)
	check(t, err == nil && string(b) == sheet() && f.tells == 0 && !w.Modified,
		"the sheet written where chosen, the character not marked changed")
	os.Remove(savePath)
	f.chosen = ""
	w.Markdown.Save.OnTapped()
	_, err = os.Stat(savePath)
	check(t, f.markdownAsks == 2 && err != nil && f.tells == 0, "cancelled: nothing written, nothing said")
	f.chosen = badPath
	w.Markdown.Save.OnTapped()
	check(t, f.tells == 1 && strings.HasPrefix(f.told, "couldn't write "+badPath), "a file that can't be written: said why")
	w.NewItem.Action()
	f.chosen = savePath
	w.Markdown.Save.OnTapped()
	b, err = os.ReadFile(savePath)
	check(t, f.suggested == "" && err == nil && string(b) == sheet(), "a character with no name: nothing suggested, the sheet saved")
}
