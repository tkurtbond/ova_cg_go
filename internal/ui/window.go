// Package ui is the program's window and its dialogs, in Fyne.
//
// ova_cg_po's PLAN.md, "The window". One window, narrow enough to sit
// beside the rulebook's viewer, a menu bar, and four tabs, each a
// column: the first two the sheet's right half, "Character" (the names,
// the NPC type, the Abilities and Weaknesses, the totals and warnings,
// the Combat Notes) and "Attacks" (the combat box and the sheet's
// numbers), with a box beside every computed number for its override;
// the other two the sheet's left half, "Background" and "Fluff".
//
// A file holds one character or several (ova_cg_po's PLAN.md, "Several
// characters in a file"): Characters, in the file's order, of which one,
// Character, is shown. The Select at the right of the tab bar, and the
// Character menu, switch between them, add, copy, remove and reorder
// them. Every change marks the file modified, and Save writes them all.
//
// The character is the Window's: every change a widget makes goes into
// it, then character.Recompute, then the window shows it again, so no
// widget holds the only copy of anything.
//
// What would wait for the user -- a message, a question, a file chooser,
// a line of text, the rulebook's viewer -- goes through a function field
// (Tell, Ask, Confirm, ChoosePath, ChoosePdf, AskText, ChooseImage,
// OpenPage),
// each taking a callback for its answer, so a test can answer it.
//
// The rulebook's path and the viewer the user chooses are kept in the
// preferences (UsePrefs), between runs.
//
// The port of ova_cg_po's OvaWindow, OvaWidgets, OvaTraitPanel,
// OvaAttackDialog, OvaLinkedDialog, OvaSettings and OvaBackground.
package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tkurtbond/ova_cg_go/internal/calc"
	"github.com/tkurtbond/ova_cg_go/internal/character"
	"github.com/tkurtbond/ova_cg_go/internal/entities"
	"github.com/tkurtbond/ova_cg_go/internal/paths"
	"github.com/tkurtbond/ova_cg_go/internal/prefs"
	"github.com/tkurtbond/ova_cg_go/internal/rules"
	"github.com/tkurtbond/ova_cg_go/internal/save"
	"github.com/tkurtbond/ova_cg_go/internal/viewer"
)

// Title is the window's title, before the file's name.
const Title = "OVA Character Generator"

// The window's size: a column, to sit beside the PDF viewer; and the
// characters' Select's width.
const (
	Width      = 640
	Height     = 800
	CharacterW = 230
)

// The sheet's numbers, Window.Numbers' indices.
const (
	Defense = iota
	Barrier
	Health
	Endurance
	Reserve
	TV
	AbilityLevels
	WeaknessLevels
	Total
	Points
	Incorporeal
	AbilityPoints
	NSheetNumbers
)

// Answer is Ask's answer.
type Answer int

// Ask's answers.
const (
	SaveFirst Answer = iota
	Discard
	Cancel
)

// AttackRow is one attack's row in the combat box.
type AttackRow struct {
	Attack       *character.Attack
	Name         *widget.Label
	Roll, DX     *NumberBox
	End          *NumberBox
	Edit, Remove *widget.Button
}

// Window is the main window, and the character it shows.
type Window struct {
	App fyne.App
	Win fyne.Window

	Menu                                                 *fyne.MainMenu
	NewItem, OpenItem, SaveItem, SaveAsItem, QuitItem    *fyne.MenuItem
	LimitsItem, TransformedItem                          *fyne.MenuItem
	AddCharacterItem, DuplicateItem, RemoveCharacterItem *fyne.MenuItem
	MoveUpItem, MoveDownItem, NextItem, PreviousItem     *fyne.MenuItem
	// The file's characters, by number and name; Current chosen.
	CharacterIn                                   *widget.Select
	RulebookItem, ChoosePdfItem, ViewerItem       *fyne.MenuItem
	Tabs                                          *container.AppTabs
	SheetTab, AttacksTab, BackgroundTab, FluffTab *container.TabItem
	NameIn, PlayerIn, TaglineIn                   *widget.Entry
	// The NPC type: item 0 none, then entities.Heroic, Secondary, Extra,
	// each at its constant + 1.
	NpcTypeIn             *widget.Select
	Abilities, Weaknesses *TraitPanel
	AbilityEnd            *NumberBox     // the chosen Ability's END
	AbilityMods           *widget.Button // its Perks and Flaws
	AbilityLinked         *widget.Button // its linked content: "Spells...", "Form...", ...
	Numbers               [NSheetNumbers]*NumberBox
	Warnings              *lines
	CombatNotes           *widget.Entry
	Rows                  []*AttackRow // the combat box's, in order
	rowBox                *fyne.Container
	AddAttack             *widget.Button
	Background            *Background
	Attack                *AttackDialog   // nil when closed
	Linked                *LinkedDialog   // nil when closed
	Settings              *SettingsDialog // nil when closed

	// The file's characters, in order, and the one shown,
	// Characters[Current]; the file they came from or were saved to ("":
	// none), and whether any has changed since.
	Characters []*character.Character
	Current    int
	Character  *character.Character
	Path       string
	Modified   bool

	// What would wait for the user; a test sets its own. Tell shows a
	// message; Ask asks whether to save first; ChoosePath asks for a
	// file to open, or to save to if save ("": cancelled); ChoosePdf for
	// the rulebook's PDF; AskText for a line of text, showing text as it
	// is (ok false: cancelled); ChooseImage for a portrait; OpenPage
	// opens the rulebook at a printed page.
	Tell    func(text string)
	Ask     func(question string, answer func(Answer))
	Confirm func(question, yes string, then func(ok bool)) // yes names the button that does it

	ChoosePath  func(save bool, then func(path string))
	ChoosePdf   func(then func(path string))
	AskText     func(prompt, text string, then func(text string, ok bool))
	ChooseImage func(then func(path string))
	OpenPage    func(page int)

	// The rulebook: --rulebook's path, or the one chosen this run ("":
	// none), and the one saved in the preferences.
	RulebookOption string
	SavedRulebook  string
	// The viewer, and whether --viewer chose it, so the preferences'
	// doesn't replace it.
	Viewer       viewer.Viewer
	ViewerOption bool
	// Where the rulebook and viewer are kept (nil: nowhere).
	Prefs *prefs.Prefs

	filling     bool      // the widgets being set from the character: no change to report
	wasModified bool      // Modified, when Limits... opened
	Closed      bool      // Quit closed the window
	escapes     []*escape // the open dialogs' Escape, the top one last
}

// ---- What would wait for the user ----

// openPage is OpenPage as it is now: the window's, which a test may have
// replaced.
func (w *Window) openPage(page int) { w.OpenPage(page) }

func (w *Window) tell(text string) { w.Tell(text) }

// ---- The rulebook ----

// keep keeps text under name in the preferences, written at once.
func (w *Window) keep(name, text string) {
	if w.Prefs != nil {
		if err := w.Prefs.Set(name, text); err != nil {
			w.tell("The preferences couldn't be written: " + err.Error())
		}
	}
}

// UsePrefs takes the rulebook's path and the viewer from p (nil: none),
// where the ones the user chooses are kept; --viewer's wins over the one
// kept, and --rulebook's and OVA_RULEBOOK's over the path kept
// (viewer.FindRulebook). A viewer kept that isn't one is ignored.
func (w *Window) UsePrefs(p *prefs.Prefs) {
	w.Prefs, w.SavedRulebook = p, ""
	if p != nil {
		w.SavedRulebook = p.Get("rulebook")
		if v, ok := viewer.Parse(p.Get("viewer")); ok && !w.ViewerOption {
			w.Viewer = v
		}
	}
}

// chooseRulebook asks for the rulebook's PDF, uses it from now on, and
// keeps it; then(ok).
func (w *Window) chooseRulebook(then func(ok bool)) {
	w.ChoosePdf(func(p string) {
		if p == "" {
			then(false)
			return
		}
		p = paths.Absolute(p, "")
		w.RulebookOption, w.SavedRulebook = p, p
		w.keep("rulebook", p)
		then(true)
	})
}

// openRulebook opens the rulebook at printed page page. If it isn't
// found, asks for it once; if not given, says why.
func (w *Window) openRulebook(page int) {
	open := func(pdf string) {
		msg, err := w.Viewer.Open(pdf, page)
		if err != nil {
			w.tell(err.Error())
		} else if msg != "" {
			w.tell(msg)
		}
	}
	if pdf, ok := viewer.FindRulebook(w.RulebookOption, w.SavedRulebook); ok {
		open(pdf)
		return
	}
	w.chooseRulebook(func(chosen bool) {
		pdf, ok := viewer.FindRulebook(w.RulebookOption, w.SavedRulebook)
		if !chosen || !ok {
			w.tell("The rulebook isn't at " + pdf +
				": choose it with Rulebook > Choose PDF..., or give it with --rulebook or OVA_RULEBOOK.")
			return
		}
		open(pdf)
	})
}

// chooseViewer is Rulebook > Viewer...: a viewer's name or a command,
// kept.
func (w *Window) chooseViewer() {
	w.AskText("Viewer: okular, evince, mupdf, xdg-open, or a command (%f the PDF, %p its page, %l the printed page)",
		w.Viewer.Name(), func(text string, ok bool) {
			if !ok {
				return
			}
			v, ok := viewer.Parse(text)
			if !ok {
				w.tell("Not a viewer: " + text +
					". Give okular, evince, mupdf, xdg-open, or a command with %f for the PDF.")
				return
			}
			w.Viewer = v
			w.keep("viewer", v.Name())
		})
}

// ---- Title and changes ----

// fileName is the file's name, its path without its directory.
func (w *Window) fileName() string { return w.Path[strings.LastIndexByte(w.Path, '/')+1:] }

func (w *Window) showTitle() {
	t := Title
	if w.Path != "" {
		t += " - " + w.fileName()
	}
	if len(w.Characters) > 1 {
		t += fmt.Sprintf(" (%d of %d)", w.Current+1, len(w.Characters))
	}
	if w.Modified {
		t += " *"
	}
	w.Win.SetTitle(t)
}

// warningText is w's text, as the Warnings list shows it.
func warningText(wn *calc.Warning) string {
	var r string
	if wn.Attack != nil {
		r = character.AttackOf(wn.Attack).Name + ": "
	}
	if wn.Trait != nil {
		t := character.TraitOf(wn.Trait)
		r += rules.NameLabel(t.Item, t.Name)
	}
	if wn.Mod != nil {
		m := character.ModOf(wn.Mod)
		if wn.Trait != nil {
			r += ": "
		}
		r += rules.NameLabel(m.Item, m.Name)
	}
	switch wn.Code {
	case calc.UnknownItem:
		r += " isn't in the rulebook's lists"
	case calc.LevelRange:
		if wn.Trait.Kind == rules.Ability {
			r += ": an Ability is +1 to +5 (p. 18)"
		} else {
			r += ": a Weakness is -1 to -3 (p. 18)"
		}
	case calc.TooOften:
		r += " is taken more often than it may be (p. 83)"
	case calc.CostRange:
		r += "'s cost is outside its range, or not in steps of 5 (p. 87)"
	case calc.NoDamageIneffective:
		r += "No Damage with Ineffective (p. 88)"
	case calc.RedirectableAlone:
		r += "Redirectable without Ranged (p. 86)"
	case calc.SpellRange:
		r += ": a spell's level isn't 1 to 5 (or -1 to -3, a Weakness), or its Magic's isn't 1 to 5: no cost (p. 58)"
	case calc.TableRange:
		r += "Tough, Vigorous or Endurance Reserve past +5, counted as +5"
	case calc.BaseZeroBroken:
		r += "The Character Total is beyond Base Zero's range (p. 21)"
	case calc.StartingBroken:
		r += "The Abilities are beyond the starting levels and the Weaknesses (p. 21)"
	case calc.CeilingBroken:
		r += "The Abilities are beyond the Power Ceiling (p. 21)"
	case calc.OverFive:
		if wn.Attack == nil {
			r += "DEFENSE: "
		}
		r += "Abilities give the roll more than +5 (p. 21)"
	}
	return r
}

func (w *Window) showWarnings() {
	var rows [][]string
	for _, wn := range w.Character.Warnings {
		rows = append(rows, []string{warningText(wn)})
	}
	w.Warnings.Set(rows, -1)
}

// pointNumbers points the sheet's number boxes at the character's
// numbers.
func (w *Window) pointNumbers() {
	c := w.Character
	for i, n := range []*calc.Number{&c.Defense, &c.Barrier, &c.Health, &c.Endurance, &c.Reserve, &c.TV,
		&c.AbilityLevels, &c.WeaknessLevels, &c.Total, &c.Points, &c.Incorporeal, &c.AbilityPoints} {
		w.Numbers[i].Target = n
	}
}

// showNumbers shows every number box again: Barrier's, Incorporeal
// Form's and the Reserve's only when the character has them, Scaled
// Cost's points (the Abilities' and the net) only under it. A total a
// limit finds too high is red (p. 21): Base Zero's the Character Total,
// or the net points under Scaled Cost; a Power Ceiling's or the starting
// levels' the Abilities' levels, or their points under Scaled Cost.
func (w *Window) showNumbers() {
	c := w.Character
	for _, b := range w.Numbers {
		b.Display()
	}
	baseZero := c.Warned(calc.BaseZeroBroken)
	spent := c.Warned(calc.CeilingBroken) || c.Warned(calc.StartingBroken)
	scaled := c.Options.ScaledCost
	w.Numbers[Total].SetRed(baseZero && !scaled)
	w.Numbers[Points].SetRed(baseZero && scaled)
	w.Numbers[AbilityLevels].SetRed(spent && !scaled)
	w.Numbers[AbilityPoints].SetRed(spent && scaled)
	visible(c.HasBarrier, w.Numbers[Barrier].Object())
	visible(c.HasIncorporeal, w.Numbers[Incorporeal].Object())
	visible(c.HasReserve, w.Numbers[Reserve].Object())
	visible(scaled, w.Numbers[AbilityPoints].Object(), w.Numbers[Points].Object())
	for _, r := range w.Rows {
		r.Roll.Display()
		r.DX.Display()
		r.End.Display()
	}
	w.AbilityEnd.Display()
	w.showWarnings()
}

// changed follows any change to the character: computed again and
// shown.
func (w *Window) changed() {
	character.Recompute(w.Character)
	w.Modified = true
	w.showTitle()
	w.showNumbers()
}

// textChanged follows a change of text alone: nothing to compute.
func (w *Window) textChanged() {
	w.Modified = true
	w.showTitle()
}

// numberChanged follows a number box's text taken; refused text is said
// so.
func (w *Window) numberChanged(b *NumberBox, ok bool) {
	if !ok {
		w.tell(notANumber(b))
	}
	w.changed()
}

func (w *Window) sheetBox(which int, label string) fyne.CanvasObject {
	b := NewNumberBox(label, true)
	b.Changed = w.numberChanged
	w.Numbers[which] = b
	return b.Object()
}

// ---- Abilities and Weaknesses ----

// showAbility shows the selected Ability's END, Perks and Flaws, and
// linked content: the box and buttons work only for one with them.
func (w *Window) showAbility(t *character.Trait) {
	linked := rules.NoLink
	if t == nil {
		w.AbilityEnd.Target = nil
		w.AbilityEnd.Disable()
		w.AbilityMods.Disable()
	} else {
		w.AbilityEnd.Target = &t.End
		enable(t.HasEnd, w.AbilityEnd)
		w.AbilityMods.Enable()
		if t.Item != nil {
			linked = t.Item.Linked
		}
	}
	w.AbilityLinked.SetText(linkedLabel(linked))
	enable(linked != rules.NoLink, w.AbilityLinked)
	w.AbilityEnd.Display()
}

// linkedLabel is the linked content's button's label.
func linkedLabel(l rules.Link) string {
	switch l {
	case rules.Spells:
		return "Spells..."
	case rules.Form:
		return "Form..."
	case rules.Partner:
		return "Companion..."
	case rules.Machine:
		return "Vehicle..."
	}
	return "Linked..."
}

func (w *Window) editMods() {
	if t := w.Abilities.SelectedTrait(); t != nil && w.Attack == nil {
		w.EditMods(w.Character, t, func(changed bool) {
			if changed {
				w.changed()
			}
			w.Abilities.Refresh(w.Abilities.Selected())
		})
	}
}

func (w *Window) editLinked() {
	t := w.Abilities.SelectedTrait()
	if t != nil && t.Item != nil && t.Item.Linked != rules.NoLink && w.Linked == nil {
		// What the linked dialog changed is in the character already.
		w.EditLinked(w.Character, t, func() { w.Abilities.Refresh(w.Abilities.Selected()) })
	}
}

// ---- The combat box ----

func (w *Window) attackDone(changed bool) {
	if changed {
		w.changed()
	}
	w.showRows()
}

func (w *Window) addAttack() {
	if w.Attack != nil {
		return
	}
	a := character.NewAttack("New attack")
	w.Character.Attacks = append(w.Character.Attacks, &a.Attack)
	character.Recompute(w.Character)
	w.EditAttack(w.Character, a, true, w.attackDone)
}

func (w *Window) attackBox(n *calc.Number) *NumberBox {
	b := NewNumberBox("", false)
	b.Target = n
	b.Changed = w.numberChanged
	return b
}

// showRows makes the combat box's rows again, one for each attack.
func (w *Window) showRows() {
	w.Rows = nil
	var objs []fyne.CanvasObject
	for _, x := range w.Character.Attacks {
		a := character.AttackOf(x)
		r := &AttackRow{Attack: a, Name: widget.NewLabel(a.Name)}
		r.Name.Truncation = fyne.TextTruncateEllipsis
		r.Roll = w.attackBox(&a.Roll)
		r.Roll.Label = "ROLL"
		r.DX = w.attackBox(&a.DX)
		r.DX.Label, r.DX.AllowHalf = "DX", true
		r.End = w.attackBox(&a.End)
		r.End.Label = "END"
		r.Edit = widget.NewButton("Edit", func() {
			if w.Attack == nil {
				w.EditAttack(w.Character, a, false, w.attackDone)
			}
		})
		r.Remove = widget.NewButton("Remove", func() {
			character.RemoveAttack(w.Character, &a.Attack)
			w.changed()
			w.showRows()
		})
		w.Rows = append(w.Rows, r)
		objs = append(objs, container.NewBorder(nil, nil, nil,
			container.NewHBox(r.Roll.Object(), r.DX.Object(), r.End.Object(), r.Edit, r.Remove), r.Name))
	}
	w.rowBox.Objects = objs
	w.rowBox.Refresh()
	w.showNumbers()
}

// ---- The file's characters ----

// characterLabel is character i's label in the Select: its number and
// name.
func (w *Window) characterLabel(i int) string {
	name := w.Characters[i].Name
	if name == "" {
		name = "(no name)"
	}
	return fmt.Sprintf("%d. %s", i+1, name)
}

// showCharacters lists the characters in the Select again, Current
// chosen, and the Character menu's items are active where they can do
// something.
func (w *Window) showCharacters() {
	n := len(w.Characters)
	l := make([]string, n)
	for i := range l {
		l[i] = w.characterLabel(i)
	}
	filling := w.filling
	w.filling = true
	w.CharacterIn.SetOptions(l)
	w.CharacterIn.SetSelectedIndex(w.Current)
	w.filling = filling
	w.RemoveCharacterItem.Disabled = n <= 1
	w.MoveUpItem.Disabled, w.PreviousItem.Disabled = w.Current == 0, w.Current == 0
	w.MoveDownItem.Disabled, w.NextItem.Disabled = w.Current == n-1, w.Current == n-1
	w.Menu.Refresh()
}

// showCharacter shows character i.
func (w *Window) showCharacter(i int) {
	w.Current, w.Character = i, w.Characters[i]
	character.Recompute(w.Character)
	w.showAll()
}

func (w *Window) characterChosen() {
	if w.filling {
		return
	}
	if i := w.CharacterIn.SelectedIndex(); i >= 0 && i != w.Current {
		w.showCharacter(i)
	}
}

// appendCharacter adds c after the last character, and shows it:
// Character > Add Character's new one, or Duplicate's copy.
func (w *Window) appendCharacter(c *character.Character) {
	w.Characters = append(w.Characters, c)
	w.Modified = true
	w.showCharacter(len(w.Characters) - 1)
}

// duplicateCharacter is Character > Duplicate Character: a copy of the
// one shown, last.
func (w *Window) duplicateCharacter() {
	if c := save.Copy(w.Character); c == nil {
		w.tell("The character couldn't be copied.")
	} else {
		w.appendCharacter(c)
	}
}

// removeCharacter is Character > Remove Character...: the one shown,
// once the user says so; not the only one, since a file holds at least
// one. The one after it is shown, or the one before if it was last.
func (w *Window) removeCharacter() {
	if len(w.Characters) <= 1 {
		return
	}
	name := w.Character.Name
	if name == "" {
		name = "this character"
	}
	w.Confirm("Remove "+name+" from the file?", "Remove", func(ok bool) {
		if !ok {
			return
		}
		i := w.Current
		w.Characters = append(w.Characters[:i:i], w.Characters[i+1:]...)
		w.Modified = true
		w.showCharacter(min(i, len(w.Characters)-1))
	})
}

// moveCharacter swaps the character shown with the one d after it, and
// it is still shown.
func (w *Window) moveCharacter(d int) {
	j := w.Current + d
	if j < 0 || j >= len(w.Characters) {
		return
	}
	w.Characters[w.Current], w.Characters[j] = w.Characters[j], w.Characters[w.Current]
	w.Current = j
	w.Modified = true
	w.showCharacters()
	w.showTitle()
}

// stepCharacter shows the character d after the one shown, if there is
// one: Next and Previous, which don't wrap.
func (w *Window) stepCharacter(d int) {
	if j := w.Current + d; j >= 0 && j < len(w.Characters) {
		w.showCharacter(j)
	}
}

// ---- The NPC type ----

// npcTypeLabels are the NPC type's choices: none, then each type with
// its page.
func npcTypeLabels() []string {
	l := []string{"none"}
	for k := entities.Heroic; k <= entities.Extra; k++ {
		l = append(l, rules.PageLabel(entities.NpcTypeName(k), rules.NpcTypesPage))
	}
	return l
}

// npcTypeChosen follows the NPC type chosen: the sheet's kind, so its
// Health, Endurance and TV, follow it (p. 123-124).
func (w *Window) npcTypeChosen() {
	if w.filling {
		return
	}
	if i := w.NpcTypeIn.SelectedIndex(); i <= 0 {
		character.SetNpcType(w.Character, false, 0)
	} else {
		character.SetNpcType(w.Character, true, entities.NpcType(i-1))
	}
	w.changed()
}

func (w *Window) showNpcType() {
	if w.Character.HasNpcType {
		w.NpcTypeIn.SetSelectedIndex(int(w.Character.NpcType) + 1)
	} else {
		w.NpcTypeIn.SetSelectedIndex(0)
	}
}

// ---- The transformed form, and the limits ----

// limitsChanged follows a change of the limits: changed, and the lists,
// which show Scaled Cost's points only under it.
func (w *Window) limitsChanged() {
	w.changed()
	w.Abilities.Refresh(w.Abilities.Selected())
	w.Weaknesses.Refresh(w.Weaknesses.Selected())
}

// editLimits is Character > Limits...: the settings dialog, which tells
// limitsChanged each change. Cancel puts the options back, and the title's
// mark as it was.
func (w *Window) editLimits() {
	if w.Settings != nil {
		return
	}
	w.wasModified = w.Modified
	w.EditSettings(w.Character, func(ok bool) {
		if !ok {
			w.Modified = w.wasModified
			w.showTitle()
		}
	})
}

// toggleTransformed is Character > Transformed form: the sheet shown,
// and computed, in the Transformation's form (p. 63), as Fukiko's is;
// the combat key transformed.
func (w *Window) toggleTransformed() {
	w.Character.Transformed = !w.Character.Transformed
	w.showTransformed()
	w.changed()
}

func (w *Window) showTransformed() {
	w.TransformedItem.Checked = w.Character.Transformed
	w.Menu.Refresh()
}

// ---- Everything shown ----

func (w *Window) showAll() {
	c := w.Character
	w.filling = true
	w.NameIn.SetText(c.Name)
	w.PlayerIn.SetText(c.Player)
	w.TaglineIn.SetText(c.Tagline)
	w.CombatNotes.SetText(c.CombatNotes)
	w.showNpcType()
	w.filling = false
	w.pointNumbers()
	w.Abilities.Show(&c.Sheet, nil, c)
	w.Weaknesses.Show(&c.Sheet, nil, c)
	w.showTransformed()
	w.showRows()
	dir := ""
	if w.Path != "" {
		dir = paths.Dir(w.Path)
	}
	w.Background.Show(c, dir)
	w.showCharacters()
	w.showTitle()
}

// ---- Files ----

// take shows c, from file p ("": none).
// take shows the characters cs, from file p ("": none), the first of
// them.
func (w *Window) take(cs []*character.Character, p string) {
	w.Characters, w.Current, w.Character, w.Path, w.Modified = cs, 0, cs[0], p, false
	character.Recompute(w.Character)
	w.showAll()
}

// Adopt shows c, just loaded by character.Load from the file at p, and
// says what the file gives that the rules compute differently (each is
// kept as an override): notices.
func (w *Window) Adopt(cs []*character.Character, p string, notices []string) {
	w.take(cs, p)
	if len(notices) > 0 {
		w.tell("Numbers the file gives differently from the rules, kept as overrides:\n" +
			strings.Join(notices, "\n"))
	}
}

// Load opens the character in the file at p; false, said why, if it
// can't.
func (w *Window) Load(p string) bool {
	cs, notices, err := character.LoadAll(p)
	if err != nil {
		w.tell(err.Error())
		return false
	}
	w.Adopt(cs, p, notices)
	return true
}

// saveTo saves to p; false, said why, if it can't. The portrait's path,
// relative to the file, is rebased to p's directory first (paths), and
// put back if the save fails.
func (w *Window) saveTo(p string) bool {
	old := make([]string, len(w.Characters))
	from := ""
	if w.Path != "" {
		from = paths.Dir(w.Path)
	}
	to := paths.Dir(p)
	for i, c := range w.Characters {
		old[i] = c.Portrait
		c.Portrait = paths.Rebase(old[i], from, to)
	}
	if err := save.SaveAll(w.Characters, p); err != nil {
		for i, c := range w.Characters {
			c.Portrait = old[i]
		}
		w.tell(err.Error())
		return false
	}
	w.Path, w.Modified = p, false
	w.showTitle()
	w.Background.SetDir(to)
	return true
}

// saveAs asks for a file and saves to it; then(saved).
func (w *Window) saveAs(then func(bool)) {
	w.ChoosePath(true, func(p string) { then(p != "" && w.saveTo(p)) })
}

// save saves to the file, or asks for one; then(saved).
func (w *Window) save(then func(bool)) {
	if w.Path == "" {
		w.saveAs(then)
	} else {
		then(w.saveTo(w.Path))
	}
}

// mayLeave calls then once the character may be put away: saved, or the
// user doesn't want it saved; never if the user cancels.
func (w *Window) mayLeave(then func()) {
	if !w.Modified {
		then()
		return
	}
	var name string
	switch {
	case w.Path != "":
		name = w.fileName()
	case len(w.Characters) > 1:
		name = "the characters"
	case w.Character.Name != "":
		name = w.Character.Name
	default:
		name = "the character"
	}
	w.Ask("Save the changes to "+name+" first?", func(a Answer) {
		switch a {
		case SaveFirst:
			w.save(func(ok bool) {
				if ok {
					then()
				}
			})
		case Discard:
			then()
		}
	})
}

func (w *Window) newCharacter() {
	w.mayLeave(func() { w.take(character.One(character.New()), "") })
}

func (w *Window) open() {
	w.mayLeave(func() {
		w.ChoosePath(false, func(p string) {
			if p != "" {
				w.Load(p)
			}
		})
	})
}

// Quit is File > Quit, and the window's close button: closes the window,
// which ends the program, once the changes are saved or let go.
func (w *Window) Quit() {
	w.mayLeave(func() {
		w.Closed = true
		w.Win.Close()
	})
}

// ---- Building ----

func shortcut(key fyne.KeyName, mods fyne.KeyModifier) fyne.Shortcut {
	return &desktop.CustomShortcut{KeyName: key, Modifier: fyne.KeyModifierShortcutDefault | mods}
}

func menuItem(label string, key fyne.KeyName, mods fyne.KeyModifier, action func()) *fyne.MenuItem {
	it := fyne.NewMenuItem(label, action)
	if key != "" {
		it.Shortcut = shortcut(key, mods)
	}
	return it
}

// buildMenu is the menu bar. Its keys are ova_cg_po's, and none is
// Ctrl+A, C, V, X, Z or Y, which the entries need (PLAN.md, "Decisions
// the port makes").
func (w *Window) buildMenu() {
	w.NewItem = menuItem("New", fyne.KeyN, 0, w.newCharacter)
	w.OpenItem = menuItem("Open...", fyne.KeyO, 0, w.open)
	w.SaveItem = menuItem("Save", fyne.KeyS, 0, func() { w.save(func(bool) {}) })
	w.SaveAsItem = menuItem("Save As...", fyne.KeyS, fyne.KeyModifierShift, func() { w.saveAs(func(bool) {}) })
	w.QuitItem = menuItem("Quit", fyne.KeyQ, 0, w.Quit)
	w.QuitItem.IsQuit = true
	w.LimitsItem = menuItem("Limits...", fyne.KeyE, 0, w.editLimits)
	w.TransformedItem = menuItem("Transformed form", fyne.KeyT, 0, w.toggleTransformed)
	w.AddCharacterItem = menuItem("Add Character", "", 0, func() { w.appendCharacter(character.New()) })
	w.DuplicateItem = menuItem("Duplicate Character", "", 0, w.duplicateCharacter)
	w.RemoveCharacterItem = menuItem("Remove Character...", "", 0, w.removeCharacter)
	w.MoveUpItem = menuItem("Move Up", "", 0, func() { w.moveCharacter(-1) })
	w.MoveDownItem = menuItem("Move Down", "", 0, func() { w.moveCharacter(1) })
	w.NextItem = menuItem("Next Character", fyne.KeyPageDown, 0, func() { w.stepCharacter(1) })
	w.PreviousItem = menuItem("Previous Character", fyne.KeyPageUp, 0, func() { w.stepCharacter(-1) })
	w.RulebookItem = menuItem("Open", "", 0, func() { w.openPage(1) })
	w.ChoosePdfItem = menuItem("Choose PDF...", "", 0, func() { w.chooseRulebook(func(bool) {}) })
	w.ViewerItem = menuItem("Viewer...", "", 0, w.chooseViewer)
	w.Menu = fyne.NewMainMenu(
		fyne.NewMenu("File", w.NewItem, w.OpenItem, w.SaveItem, w.SaveAsItem, fyne.NewMenuItemSeparator(),
			w.QuitItem),
		fyne.NewMenu("Character", w.LimitsItem, w.TransformedItem, fyne.NewMenuItemSeparator(),
			w.AddCharacterItem, w.DuplicateItem, w.RemoveCharacterItem, fyne.NewMenuItemSeparator(),
			w.MoveUpItem, w.MoveDownItem, fyne.NewMenuItemSeparator(), w.NextItem, w.PreviousItem),
		fyne.NewMenu("Rulebook", w.RulebookItem, fyne.NewMenuItemSeparator(), w.ChoosePdfItem, w.ViewerItem))
}

// textEntry is an entry whose every change goes into the character.
func (w *Window) textEntry(multi bool, set func(string)) *widget.Entry {
	var e *widget.Entry
	if multi {
		e = widget.NewMultiLineEntry()
		e.Wrapping = fyne.TextWrapWord
	} else {
		e = widget.NewEntry()
	}
	e.OnChanged = func(s string) {
		if !w.filling {
			set(s)
			w.textChanged()
		}
	}
	return e
}

// buildSheetTabs is the first two tabs: Character, and Attacks.
func (w *Window) buildSheetTabs() (sheet, attacks fyne.CanvasObject) {
	w.NameIn = w.textEntry(false, func(s string) { w.Character.Name = trim(s); w.showCharacters() })
	w.PlayerIn = w.textEntry(false, func(s string) {
		character.SetText(&w.Character.HasPlayer, &w.Character.Player, trim(s))
	})
	w.TaglineIn = w.textEntry(false, func(s string) {
		character.SetText(&w.Character.HasTagline, &w.Character.Tagline, trim(s))
	})
	w.CombatNotes = w.textEntry(true, func(s string) {
		character.SetText(&w.Character.HasCombatNotes, &w.Character.CombatNotes, s)
	})
	w.Abilities = newTraits(rules.Ability, "Abilities", 120, w.openPage, w.tell)
	w.Abilities.Changed = w.changed
	w.Abilities.OnSelect = w.showAbility
	w.Weaknesses = newTraits(rules.Weakness, "Weaknesses", 120, w.openPage, w.tell)
	w.Weaknesses.Changed = w.changed
	w.AbilityMods = widget.NewButton("Perks and Flaws...", w.editMods)
	w.AbilityLinked = widget.NewButton("Linked...", w.editLinked)
	w.AbilityEnd = NewNumberBox("END", false)
	w.AbilityEnd.Changed = w.numberChanged
	w.Warnings = newLines()
	w.rowBox = container.NewVBox()
	w.AddAttack = widget.NewButton("Add attack...", w.addAttack)
	w.NpcTypeIn = widget.NewSelect(npcTypeLabels(), func(string) { w.npcTypeChosen() })

	names := container.New(layout.NewFormLayout(),
		widget.NewLabel("Character Name"), container.NewBorder(nil, nil, nil,
			container.NewHBox(widget.NewLabel("Player Name"), sized(w.PlayerIn, 160, 0)), w.NameIn),
		widget.NewLabel("Tagline"), container.NewBorder(nil, nil, nil,
			container.NewHBox(widget.NewLabel("NPC Type"), sized(w.NpcTypeIn, 160, 0)), w.TaglineIn))
	totals := container.NewHBox(
		w.sheetBox(AbilityLevels, "Abilities"), w.sheetBox(WeaknessLevels, "Weaknesses"),
		w.sheetBox(Total, "Character Total"), w.sheetBox(AbilityPoints, "Ability points"),
		w.sheetBox(Points, "Scaled Cost points"))
	// The trait lists take what height the window has beyond its least.
	left := container.NewBorder(
		names,
		container.NewVBox(
			container.NewHBox(w.AbilityMods, w.AbilityEnd.Object(), layout.NewSpacer(), w.AbilityLinked),
			totals,
			bold("Warnings"),
			sized(w.Warnings.List, 0, 65),
			bold("Combat Notes"),
			sized(w.CombatNotes, 0, 80)),
		nil, nil,
		container.NewGridWithColumns(2, w.Abilities.Object(), w.Weaknesses.Object()))
	// The header's columns as wide as a row's (showRows): a number's
	// pair, and the Edit and Remove buttons.
	pad := theme.Padding()
	number := numberShownW + pad + overrideW()
	buttons := widget.NewButton("Edit", nil).MinSize().Width + pad + widget.NewButton("Remove", nil).MinSize().Width
	header := container.NewBorder(nil, nil, nil, container.NewHBox(
		sized(widget.NewLabel("ROLL"), number, 0), sized(widget.NewLabel("DX"), number, 0),
		sized(widget.NewLabel("END"), number, 0), sized(widget.NewLabel(""), buttons, 0)),
		widget.NewLabel("Name"))
	// Incorporeal Form's Defense under Barrier's: each is shown only
	// when the character has it.
	numbers := container.NewHBox(
		w.sheetBox(Defense, "DEFENSE"),
		container.NewVBox(w.sheetBox(Barrier, "Barrier DEFENSE"), w.sheetBox(Incorporeal, "Incorporeal DEFENSE")),
		w.sheetBox(Health, "HEALTH"), w.sheetBox(Endurance, "ENDURANCE"), w.sheetBox(Reserve, "Reserve"),
		w.sheetBox(TV, "TV"))
	right := container.NewBorder(
		container.NewVBox(bold("Attacks"), header),
		container.NewVBox(container.NewHBox(w.AddAttack), numbers),
		nil, nil,
		container.NewVScroll(w.rowBox))
	return left, right
}

// New is the window, not yet shown, in app a, with a new character.
func New(a fyne.App) *Window {
	w := &Window{App: a, Win: a.NewWindow(Title)}
	w.Tell = w.dialogTell
	w.Ask = w.dialogAsk
	w.Confirm = w.dialogConfirm
	w.ChoosePath = w.dialogPath
	w.ChoosePdf = w.dialogPdf
	w.AskText = w.dialogText
	w.ChooseImage = w.dialogImage
	w.OpenPage = w.openRulebook
	w.Viewer = viewer.Default()
	w.buildMenu()
	w.Win.SetMainMenu(w.Menu)
	w.Background = newBackground(w)
	sheet, attacks := w.buildSheetTabs()
	w.SheetTab = container.NewTabItem("Character", sheet)
	w.AttacksTab = container.NewTabItem("Attacks", attacks)
	w.BackgroundTab = container.NewTabItem("Background", w.Background.object)
	w.FluffTab = container.NewTabItem("Fluff", w.Background.fluff)
	w.Tabs = container.NewAppTabs(w.SheetTab, w.AttacksTab, w.BackgroundTab, w.FluffTab)
	// The characters' Select over the tab bar's empty right end: Fyne's
	// main menu holds no widget, as ova_cg_po's menu bar does its Choice.
	w.CharacterIn = widget.NewSelect(nil, func(string) { w.characterChosen() })
	w.Win.SetContent(container.NewStack(w.Tabs,
		container.NewVBox(container.NewHBox(layout.NewSpacer(), sized(w.CharacterIn, CharacterW, 0)))))
	w.Win.Resize(fyne.NewSize(Width, Height))
	w.Win.SetCloseIntercept(w.Quit)
	w.Win.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyEscape {
			w.Escape()
		}
	})
	w.take(character.One(character.New()), "")
	return w
}
