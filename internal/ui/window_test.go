package ui

// po_ova_cg's TestWindow: the first tab, driven as a user would, and
// its files. Fyne's test driver draws in memory: no display is needed,
// and nothing reaches the desktop. Widgets are driven directly: a
// value set, then what the user's action does; a menu item's action
// run. What would wait for the user (a message, a question, a file
// chooser, the rulebook) is answered by the fakes here.

import (
	"fyne.io/fyne/v2"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/tkurtbond/ova_cg/internal/calc"
	"github.com/tkurtbond/ova_cg/internal/character"
	"github.com/tkurtbond/ova_cg/internal/rules"
	"github.com/tkurtbond/ova_cg/internal/save"
)

const (
	dataDir = "../../testdata/"
	objDir  = "../../obj/"
)

// fakes is what answers for the user.
type fakes struct {
	t           *testing.T
	tells       int    // how many messages were shown
	told        string // the last
	answer      Answer // what Ask answers,
	asked       int    // and how often it was asked
	chosen      string // what ChoosePath answers; "": cancelled
	chosenImage string // what ChooseImage answers
	page        int    // the last page OpenPage was asked for
}

func newWindow(t *testing.T) (*Window, *fakes) {
	t.Helper()
	if err := os.MkdirAll(objDir+"files", 0o755); err != nil {
		t.Fatal(err)
	}
	a := test.NewTempApp(t)
	w := New(a)
	f := &fakes{t: t, answer: Cancel}
	w.Tell = func(text string) { f.tells++; f.told = text; t.Logf("told: %s", text) }
	w.Ask = func(q string, answer func(Answer)) { f.asked++; answer(f.answer) }
	w.ChoosePath = func(save bool, then func(string)) { then(f.chosen) }
	w.ChooseImage = func(then func(string)) { then(f.chosenImage) }
	w.ChoosePdf = func(then func(string)) { then("") }
	w.AskText = func(prompt, text string, then func(string, bool)) { then("", false) }
	w.OpenPage = func(p int) { f.page = p }
	w.Prefs = nil
	return w, f
}

// allShown says whether every number box of the sheet and the combat box
// shows the model's number.
func allShown(w *Window) bool {
	c := w.Character
	for i, n := range []*calc.Number{&c.Defense, &c.Barrier, &c.Health, &c.Endurance, &c.Reserve, &c.TV,
		&c.AbilityLevels, &c.WeaknessLevels, &c.Total, &c.Points} {
		if !w.Numbers[i].Shows(n) {
			return false
		}
	}
	for _, r := range w.Rows {
		if !r.Roll.Shows(&r.Attack.Roll) || !r.DX.Shows(&r.Attack.DX) || !r.End.Shows(&r.Attack.End) {
			return false
		}
	}
	return true
}

// choiceLabelled says whether every name in p's Select is its item's
// label, with its page.
func choiceLabelled(p *Panel) bool {
	for i := range rules.Count(p.Kind) {
		if p.Choice.Options[i] != rules.Label(rules.Get(p.Kind, i)) {
			return false
		}
	}
	return len(p.Choice.Options) == rules.Count(p.Kind)
}

func traitNamed(w *Window, name string) *character.Trait {
	for _, t := range w.Character.Traits {
		if x := character.TraitOf(t); x.Name == name {
			return x
		}
	}
	return nil
}

func saved(w *Window) string { return save.ToText(w.Character) }

func load(w *Window, name string) bool { return w.Load(dataDir + name) }

func check(t *testing.T, ok bool, label string) {
	t.Helper()
	if !ok {
		t.Error(label)
	}
}

func TestWindow(t *testing.T) {
	w, f := newWindow(t)

	// ---- The frame ----
	file := w.Menu.Items[0]
	check(t, file.Label == "File" && file.Items[len(file.Items)-1] == w.QuitItem && w.QuitItem.Label == "Quit",
		"the menu bar has File > Quit")
	check(t, file.Items[0] == w.NewItem && file.Items[1] == w.OpenItem && file.Items[2] == w.SaveItem &&
		file.Items[3] == w.SaveAsItem && w.Menu.Items[2].Items[0] == w.RulebookItem,
		"and New, Open, Save, Save As, and Rulebook > Open")
	check(t, w.Tabs.Selected() == w.SheetTab, "the Character tab is shown first")
	check(t, choiceLabelled(w.Abilities.Panel) && choiceLabelled(w.Weaknesses.Panel),
		"every name in the Selects has its page: Attack (p. 50)")
	w.RulebookItem.Action()
	check(t, f.page == 1, "Rulebook > Open opens it at page 1")
	w.Win.Show()

	// ---- A character loaded ----
	check(t, load(w, "Ai.yaml") && f.tells == 0, "Ai.yaml loads, with nothing to say")
	check(t, w.NameIn.Text == "Ai", "the name is shown")
	check(t, w.Win.Title() == "OVA Character Generator - Ai.yaml", "the title is the file's name")
	check(t, w.Abilities.ListShown() && w.Weaknesses.ListShown(),
		"every Ability and Weakness is listed with its page")
	check(t, allShown(w), "every number shown is the model's")
	check(t, w.Numbers[Barrier].Visible() && !w.Numbers[Reserve].Visible() && !w.Numbers[Points].Visible(),
		"Barrier DEFENSE shown, for Barrier; the Reserve and the Scaled Cost points not")
	check(t, len(w.Rows) == 2, "a row in the combat box for each attack")
	r := w.Rows[0]
	check(t, r.Name.Text == "Telekinetic Push", "the first is Telekinetic Push")

	// ---- Overrides ----
	c := w.Character
	n := c.Health.Value
	check(t, !c.Health.Overridden && !strings.Contains(saved(w), "overrides"), "Ai has no overrides")
	w.Numbers[Health].Type("55")
	check(t, c.Health.Overridden && c.Health.Get() == 55 && c.Health.Value == n,
		"an override: HEALTH 55, the computed number kept")
	check(t, allShown(w), "and shown, beside the computed number")
	check(t, w.Modified, "a change marks the character modified")
	check(t, w.Win.Title() == "OVA Character Generator - Ai.yaml *", "and the title says so")
	check(t, strings.Contains(saved(w), "health: 55") && strings.Contains(saved(w), "overrides"),
		"saved: health 55, in overrides")
	w.Numbers[Health].Type("")
	check(t, !c.Health.Overridden && c.Health.Get() == n && allShown(w), "an empty box: the computed number again")
	check(t, !strings.Contains(saved(w), "overrides"), "and no override saved")
	w.Numbers[Health].Type("55")
	f.tells = 0
	w.Numbers[Health].Type("lots")
	check(t, f.tells == 1 && !c.Health.Overridden && allShown(w),
		"text that isn't a number: said so, and the box emptied")
	r.DX.Type("1/2")
	check(t, r.Attack.DX.Overridden && r.Attack.DX.IsHalf() && allShown(w), "a DX may be 1/2")
	f.tells = 0
	w.Numbers[Health].Type("1/2")
	check(t, f.tells == 1 && !c.Health.Overridden, "HEALTH may not")
	r.DX.Type("")
	m := w.Modified
	w.Modified = false
	w.Numbers[Health].Take()
	check(t, !w.Modified, "leaving a box unchanged changes nothing")
	w.Modified = m

	// An override counts in what is computed from it: Braun's TV comes
	// from his ROLL, overridden to -2.
	ok := load(w, "Braun.yaml")
	c, r = w.Character, w.Rows[0]
	check(t, ok && r.Attack.Roll.Overridden && r.Attack.Roll.Get() == -2 && c.TV.Get() == 0,
		"Braun: ROLL -2, overridden, and TV 0")
	r.Roll.Type("")
	check(t, !r.Attack.Roll.Overridden && c.TV.Get() != 0 && allShown(w),
		"ROLL's override cleared: TV computed from the rules' ROLL")
	r.Roll.Type("-2")
	check(t, c.TV.Get() == 0 && allShown(w), "and set again: TV 0 again")

	// A name the rulebook's lists don't have: "p. ?", and a warning that
	// says so.
	character.InsertTrait(&c.Traits, character.NewTrait(nil, rules.Ability, 1, "Magic, Testing", ""))
	w.Adopt(c, "", nil)
	i := w.Abilities.Find("Magic, Testing")
	check(t, w.Abilities.Line(i) == "+1 Magic, Testing\tp. ?" && w.Abilities.ListShown(),
		"an unknown name: listed with p. ?")
	i = w.Warnings.Find("Magic, Testing")
	check(t, w.Warnings.Line(i) == "Magic, Testing (p. ?) isn't in the rulebook's lists", "and a warning, with p. ?")

	// ---- Abilities and Weaknesses ----
	load(w, "Ai.yaml")
	c = w.Character
	before := c.AbilityLevels.Value
	w.Abilities.Pick("Flight")
	check(t, w.Abilities.Help.Page == 54, "choosing Flight turns ? to its page, 54")
	test.Tap(w.Abilities.Help.Button)
	check(t, f.page == 54, "and ? opens the rulebook there")
	w.Abilities.Spinner.SetValue(2)
	w.Abilities.Text.SetText("on a broom")
	test.Tap(w.Abilities.Add)
	check(t, w.Abilities.SelectedLine() == "+2 Flight (on a broom)\tp. 54", "Add: its line selected, with its page")
	check(t, c.AbilityLevels.Value == before+2 && allShown(w), "and counted: the Abilities' levels +2")
	check(t, w.Abilities.ListShown(), "the list in the model's order")
	w.Abilities.Spinner.SetValue(4)
	test.Tap(w.Abilities.Change)
	check(t, w.Abilities.SelectedLine() == "+4 Flight (on a broom)\tp. 54" && c.AbilityLevels.Value == before+4 &&
		w.Abilities.ListShown(), "Change: Flight +4")
	test.Tap(w.Abilities.Remove)
	check(t, c.AbilityLevels.Value == before && w.Abilities.ListShown() && w.Abilities.Selected() == -1 &&
		traitNamed(w, "Flight") == nil, "Remove")
	n = w.Abilities.Count()
	test.Tap(w.Abilities.Remove)
	check(t, w.Abilities.Count() == n, "Remove with nothing selected removes nothing")

	w.Weaknesses.Pick("Endurance Use")
	w.Weaknesses.Spinner.SetValue(-2)
	w.Weaknesses.Text.SetText("Telekinesis")
	test.Tap(w.Weaknesses.Add)
	check(t, w.Weaknesses.SelectedLine() == "-2 Endurance Use (Telekinesis)\tp. 70" && w.Weaknesses.ListShown(),
		"a Weakness added: Endurance Use -2 on Telekinesis")
	tk := traitNamed(w, "Telekinesis")
	w.Abilities.Select(w.Abilities.Find("Telekinesis"))
	check(t, w.AbilityEnd.Shown.Text == "10" && w.AbilityEnd.Active() && w.AbilityEnd.Shows(&tk.End),
		"Telekinesis selected: its END, 10, shown, with its box")
	check(t, w.Abilities.Help.Page == 62, "and ? at its page, 62")
	w.AbilityEnd.Type("15")
	check(t, tk.End.Overridden && tk.End.Get() == 15 && w.AbilityEnd.Shows(&tk.End), "its END overridden")
	w.AbilityEnd.Type("")

	// ---- An Ability's Perks and Flaws ----
	test.Tap(w.AbilityMods)
	d := w.Attack
	if d == nil {
		t.Fatal("Perks and Flaws... opens the dialog")
	}
	check(t, choiceLabelled(d.Perks.Panel) && choiceLabelled(d.Flaws.Panel),
		"every Perk and Flaw in its Select has its page")
	d.Perks.Pick("Area Effect")
	test.Tap(d.Perks.Add)
	n = tk.End.Get()
	check(t, len(tk.Mods) > 0 && n > 10 && d.End.Shows(&tk.End),
		"a Perk: the Ability's END goes up, shown in the dialog")
	test.Tap(d.Cancel)
	check(t, w.Attack == nil && len(tk.Mods) == 0 && tk.End.Get() == 10 && w.AbilityEnd.Shows(&tk.End),
		"Cancel: the Perk taken out again")
	test.Tap(w.AbilityMods)
	d = w.Attack
	d.Perks.Pick("Area Effect")
	test.Tap(d.Perks.Add)
	test.Tap(d.OK)
	check(t, w.Attack == nil && tk.End.Get() == n && w.AbilityEnd.Shows(&tk.End) && allShown(w),
		"OK: kept, and shown in the window")

	// ---- Attacks ----
	n = len(w.Rows)
	test.Tap(w.AddAttack)
	d = w.Attack
	if d == nil {
		t.Fatal("Add attack... opens the dialog")
	}
	d.NameIn.SetText("Wind Lance")
	d.Perks.Pick("Accurate")
	d.Perks.Spinner.SetValue(2)
	test.Tap(d.Perks.Add)
	check(t, d.Perks.SelectedLine() == "Accurate x2\t+10\tp. 83", "a Perk listed: its count, cost and page")
	d.Flaws.Pick("Cancel")
	check(t, d.Flaws.Text.Text == "-20", "Cancel's cost offered: its least, -20")
	d.Flaws.Text.SetText("-15")
	test.Tap(d.Flaws.Add)
	check(t, d.Flaws.SelectedLine() == "Cancel\t-15\tp. 87", "a Flaw listed, with the cost chosen")
	a := c.Attacks[len(c.Attacks)-1] // its row comes with OK
	check(t, d.Roll.Shows(&a.Roll) && d.End.Shows(&a.End), "the dialog's numbers are the attack's, as it changes")
	f.tells = 0
	d.Flaws.Text.SetText("a lot")
	test.Tap(d.Flaws.Change)
	check(t, f.tells == 1, "a cost that isn't a number: said so")
	test.Tap(d.OK)
	check(t, len(w.Rows) == n+1, "OK: a row for the new attack")
	r = w.Rows[len(w.Rows)-1]
	check(t, r.Name.Text == "Wind Lance" && allShown(w), "its name and numbers")
	text := saved(w)
	check(t, strings.Contains(text, "Wind Lance") && strings.Contains(text, "counts-as: 2") &&
		strings.Contains(text, "cost: -15"), "saved with its Perks and Flaws")
	test.Tap(r.Edit)
	d = w.Attack
	d.Perks.Select(0)
	test.Tap(d.Perks.Remove)
	check(t, d.Perks.Count() == 0 && !strings.Contains(saved(w), "counts-as: 2"), "Edit: a Perk removed")
	test.Tap(d.Cancel)
	check(t, saved(w) == text && len(w.Rows) == n+1 && allShown(w), "Cancel: the attack as it was")
	test.Tap(w.AddAttack)
	test.Tap(w.Attack.Cancel)
	check(t, len(w.Rows) == n+1 && saved(w) == text, "Cancel on a new attack: no attack added")
	test.Tap(w.AddAttack)
	w.Attack.NameIn.SetText(" ")
	f.tells = 0
	test.Tap(w.Attack.OK)
	check(t, f.tells == 1 && w.Attack != nil, "OK with no name: said so, and the dialog stays")
	test.Tap(w.Attack.Cancel)
	test.Tap(w.Rows[len(w.Rows)-1].Remove)
	check(t, len(w.Rows) == n && !strings.Contains(saved(w), "Wind Lance") && allShown(w),
		"Remove: the row and the attack gone")

	// ---- Names and notes ----
	w.PlayerIn.SetText("Sam")
	w.CombatNotes.SetText("Down to 10 HEALTH.")
	check(t, strings.Contains(saved(w), "player: Sam") && strings.Contains(saved(w), "Down to 10 HEALTH."),
		"the player and Combat Notes saved")

	// ---- Files ----
	f.chosen = objDir + "window.yaml"
	w.SaveAsItem.Action()
	check(t, !w.Modified && w.Win.Title() == "OVA Character Generator - window.yaml",
		"Save As: saved, the title its name")
	c = w.Character
	f.tells = 0
	ok = w.Load(f.chosen)
	check(t, ok && f.tells == 0 && w.Character != c && save.ToText(c) == saved(w),
		"loaded again: the same, with nothing to say")
	f.asked = 0
	w.NewItem.Action()
	check(t, f.asked == 0 && w.Character.Name == "" && w.Abilities.Count() == 0 && len(w.Rows) == 0 && allShown(w),
		"New: nothing unsaved, nothing asked, an empty sheet")
	w.NameIn.SetText("Someone")
	check(t, w.Character.Name == "Someone" && w.Modified, "a name typed")
	f.answer = Cancel
	w.NewItem.Action()
	check(t, f.asked == 1 && w.Character.Name == "Someone", "New with changes: asked; Cancel keeps them")
	w.QuitItem.Action()
	check(t, f.asked == 2 && !w.Closed, "Quit with changes: asked; Cancel stays")
	f.answer = Discard
	w.QuitItem.Action()
	check(t, f.asked == 3 && w.Closed, "Don't save: the window goes, which ends the program")
}

// The window's close button runs Quit, the close intercept, which Fyne's
// test window never calls: test/quit.sh checks it on the real program.

// TestShortcuts: the menu's keys, and none of an entry's.
func TestShortcuts(t *testing.T) {
	w, _ := newWindow(t)
	reserved := map[string]bool{"A": true, "C": true, "V": true, "X": true, "Z": true, "Y": true}
	seen := map[string]bool{}
	for _, m := range w.Menu.Items {
		for _, it := range m.Items {
			if it.Shortcut == nil {
				continue
			}
			name := it.Shortcut.ShortcutName()
			check(t, !seen[name], "no two items share a key: "+name)
			seen[name] = true
			for k := range reserved {
				check(t, !strings.HasSuffix(name, "+"+k) || strings.Contains(name, "Shift"),
					"no item takes an entry's key: "+name)
			}
		}
	}
	check(t, w.LimitsItem.Shortcut != nil && w.TransformedItem.Shortcut != nil, "Limits and Transformed have keys")
}

// TestEscape: Escape cancels the top dialog, and only that one; with no
// dialog it does nothing.
func TestEscape(t *testing.T) {
	w, _ := newWindow(t)
	w.Win.Show()
	esc := func() { w.Win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEscape}) }
	esc()
	check(t, !w.Closed && w.Settings == nil, "no dialog: nothing")

	w.Win.Canvas().Focus(w.NameIn)
	w.LimitsItem.Action()
	d := w.Settings
	check(t, d != nil && w.Win.Canvas().Focused() == nil, "Limits opened, the focus let go")
	d.BaseZero.SetChecked(true)
	check(t, w.Character.Options.BaseZero, "Base Zero on")

	var answer Answer = -1
	w.Ask = w.dialogAsk
	w.Ask("Save?", func(a Answer) { answer = a })
	esc()
	check(t, answer == Cancel && w.Settings == d, "Escape: the question above Limits cancelled, Limits open")
	esc()
	check(t, w.Settings == nil && !w.Character.Options.BaseZero && !w.Character.HasOptions,
		"Escape again: Limits cancelled, Base Zero off again")
	answer = -1
	esc()
	check(t, answer == -1 && len(w.escapes) == 0, "and then nothing")
}

// TestChoosers: the real file choosers open (the other tests put fakes
// in their place). Fyne's FileDialog.Resize before Show dereferences a
// dialog not yet built (v2.8.1, dialog/file.go: MinSize).
func TestChoosers(t *testing.T) {
	w, _ := newWindow(t)
	w.Win.Show()
	for _, open := range []func(){
		func() { w.dialogPath(true, func(string) {}) },
		func() { w.dialogPath(false, func(string) {}) },
		func() { w.dialogPdf(func(string) {}) },
		func() { w.dialogImage(func(string) {}) },
	} {
		open()
	}
}
