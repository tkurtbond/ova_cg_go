package ui

// An Ability's linked content: a Transformation's form, a Magic's
// spells, a Companion or Vehicle (ova_cg_po's OvaLinkedDialog).
//
// ova_cg_po's PLAN.md, "The window": an Ability with linked content has
// a button that opens this dialog, in one of three shapes
// (rules.Item.Linked):
//
//   - Form, a Transformation's (p. 63): the form's Abilities and
//     Weaknesses, worth 2 levels a level of Transformation, each line
//     with its page; the selected Ability's Perks and Flaws and END.
//   - Spells, a Magic's (p. 57-58): each spell's name, the Ability it
//     mimics, its level, and its END, computed from the Magic's level and
//     its own, with an override box. Witchcraft's spells are curses: the
//     Weakness each casts, at -1 to -3 (p. 58; rules.CastsWeaknesses).
//   - A partner's sheet, a Companion's (p. 52) or a Vehicle's (p. 64):
//     its name, whether it is a Fragment, whether it has combat stats,
//     its Abilities and Weaknesses, its attacks (edited in the attack
//     dialog) and its numbers, each with its override box.
//
// It edits the model as the user works, as the window's first tab does,
// and has Close, no Cancel: a change is in the character at once, and
// the window's changed recomputes it and shows it. It is modal but
// doesn't wait: it shows, and Close calls done.

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/tkurtbond/ova_cg_go/internal/calc"
	"github.com/tkurtbond/ova_cg_go/internal/character"
	"github.com/tkurtbond/ova_cg_go/internal/rules"
)

// A partner's numbers, LinkedDialog.Numbers' indices: the first six of
// the window's, then Incorporeal Form's Defense.
const (
	partnerIncorporeal = TV + 1
	nPartnerNumbers    = partnerIncorporeal + 1
)

// LinkedDialog is an Ability's linked content.
type LinkedDialog struct {
	w     *Window
	d     *dialog.CustomDialog
	Mode  rules.Link // Form, Spells, Partner or Machine
	Info  *widget.Label
	Close *widget.Button

	// A form's, or a partner's:
	Abilities, Weaknesses *TraitPanel
	Mods                  *widget.Button // the selected Ability's Perks and Flaws
	End                   *NumberBox     // its END

	// A Magic's:
	Spells                             *lines
	SpellName                          *widget.Entry
	SpellAbility                       *widget.Select
	SpellKind                          *widget.Label // SpellAbility's label: Ability, or Weakness
	SpellLevel                         *Spinner
	SpellHelp                          *HelpButton
	AddSpell, ChangeSpell, RemoveSpell *widget.Button
	SpellEnd                           *NumberBox
	CastKind                           rules.Kind // what the spells mimic: Ability, or Weakness for Witchcraft

	// A partner's:
	PartnerName                         *widget.Entry
	Fragment, Combat                    *widget.Check
	Attacks                             *lines
	AddAttack, EditAttack, RemoveAttack *widget.Button
	Numbers                             [nPartnerNumbers]*NumberBox

	character *character.Character // the window's, recomputed after a change
	trait     *character.Trait     // the Ability whose content this is
	partner   *character.Character // its partner's sheet, in Partner and Machine
	done      func()
	filling   bool
}

// changed follows a change here: the character recomputed and shown,
// here and in the window.
func (d *LinkedDialog) changed() {
	d.w.changed()
	d.showAll()
}

func (d *LinkedDialog) numberChanged(b *NumberBox, ok bool) {
	if !ok {
		d.w.tell("Not a number for " + b.Label + ". The computed number is used.")
	}
	d.changed()
}

func (d *LinkedDialog) numberBox(label string, above bool) *NumberBox {
	b := NewNumberBox(label, above)
	b.Changed = d.numberChanged
	return b
}

// ---- Abilities and Weaknesses: a form's or a partner's ----

// showAbility shows the selected Ability's END and Perks and Flaws.
func (d *LinkedDialog) showAbility(t *character.Trait) {
	if t == nil {
		d.End.Target = nil
		d.End.Disable()
		d.Mods.Disable()
	} else {
		d.End.Target = &t.End
		enable(t.HasEnd, d.End)
		d.Mods.Enable()
	}
	d.End.Display()
}

func (d *LinkedDialog) editMods() {
	t := d.Abilities.SelectedTrait()
	if t == nil || d.w.Attack != nil {
		return
	}
	c := d.character
	if d.partner != nil {
		c = d.partner
	}
	d.w.EditMods(c, t, func(changed bool) {
		if changed {
			d.changed()
		}
		d.Abilities.Refresh(d.Abilities.Selected())
	})
}

// panels is the two panels, each list h high, and the Ability's controls
// under them.
func (d *LinkedDialog) panels(h float32) fyne.CanvasObject {
	d.Abilities = newTraits(rules.Ability, "Abilities", h, d.w.openPage, d.w.tell)
	d.Abilities.OnSelect = d.showAbility
	d.Abilities.Changed = d.changed
	d.Weaknesses = newTraits(rules.Weakness, "Weaknesses", h, d.w.openPage, d.w.tell)
	d.Weaknesses.Changed = d.changed
	d.Mods = widget.NewButton("Perks and Flaws...", d.editMods)
	d.End = d.numberBox("END", false)
	return container.NewVBox(
		container.NewGridWithColumns(2, d.Abilities.Object(), d.Weaknesses.Object()),
		container.NewHBox(d.Mods, d.End.Object()))
}

// showFormLevels shows the form's levels, against what the
// Transformation pays for: 2 a level (p. 63).
func (d *LinkedDialog) showFormLevels() {
	n := 0
	for _, f := range d.trait.Form {
		n += f.Level
	}
	levels := strconv.Itoa(n)
	if n > 0 {
		levels = "+" + levels
	}
	d.Info.SetText("The form's Abilities and Weaknesses come to " + levels + " levels; Transformation +" +
		strconv.Itoa(d.trait.Level) + " pays for " + strconv.Itoa(2*d.trait.Level) + " (p. 63).")
	imp := widget.MediumImportance
	if n > 2*d.trait.Level {
		imp = widget.DangerImportance
	}
	d.Info.Importance = imp
	d.Info.Refresh()
}

// ---- Spells ----

// spellAt is the spell on line i; nil for none.
func (d *LinkedDialog) spellAt(i int) *character.Spell {
	if i < 0 || i >= len(d.trait.Spells) {
		return nil
	}
	return character.SpellOf(d.trait.Spells[i])
}

// showSpells lists the spells again, line sel selected (-1: none), and
// its END.
func (d *LinkedDialog) showSpells(sel int) {
	var rows [][]string
	for _, sp := range d.trait.Spells {
		x := character.SpellOf(sp)
		sign := ""
		if x.Level > 0 {
			sign = "+"
		}
		rows = append(rows, []string{x.Name,
			x.Ability + " " + sign + strconv.Itoa(x.Level) + " (" + rules.PageText(x.Item) + ")",
			"END " + x.End.Text()})
	}
	d.Spells.Set(rows, sel)
	if x := d.spellAt(sel); x != nil {
		d.SpellEnd.Target = &x.End
		d.SpellEnd.Enable()
	} else {
		d.SpellEnd.Target = nil
		d.SpellEnd.Disable()
	}
	d.SpellEnd.Display()
}

// spellItem is the Ability (or Weakness) chosen; nil for none.
func (d *LinkedDialog) spellItem() *rules.Item {
	i := d.SpellAbility.SelectedIndex()
	if i < 0 || i >= rules.Count(d.CastKind) {
		return nil
	}
	return rules.Get(d.CastKind, i)
}

// chooseSpellItem chooses it in the Select; none for nil, or an item of
// the other kind (a spell the Magic can't cast, from a file).
func (d *LinkedDialog) chooseSpellItem(it *rules.Item) {
	d.filling = true
	if it != nil && it.Kind == d.CastKind {
		d.SpellAbility.SetSelectedIndex(rules.Index(it))
	} else {
		d.SpellAbility.ClearSelected()
	}
	d.filling = false
	d.SpellHelp.Page = 0
	if it != nil {
		d.SpellHelp.Page = it.Page
	}
}

// PickSpellAbility chooses the Ability (or Weakness) named name, as the
// user does: for tests.
func (d *LinkedDialog) PickSpellAbility(name string) {
	d.SpellAbility.SetSelectedIndex(rules.Index(rules.Find(name)))
}

func (d *LinkedDialog) spellSelected(i int) {
	if x := d.spellAt(i); x != nil {
		d.SpellName.SetText(x.Name)
		d.chooseSpellItem(x.Item)
		d.SpellLevel.SetValue(x.Level)
		d.showSpells(i)
	}
}

// spellButton is what Add, Change and Remove do: Add puts the spell
// last; Change and Remove need one selected. A spell needs a name and an
// Ability (a Weakness, Witchcraft's); one changed keeps one the Select
// doesn't have if none is chosen.
func (d *LinkedDialog) spellButton(b *widget.Button) {
	line := d.Spells.Selected()
	x := d.spellAt(line)
	name, it := trim(d.SpellName.Text), d.spellItem()
	switch {
	case b == d.RemoveSpell:
		if x == nil {
			return
		}
		character.RemoveSpell(&d.trait.Trait, &x.Spell)
		line = -1
	case name == "" || it == nil && (b == d.AddSpell || x == nil || x.Item != nil):
		if d.CastKind == rules.Weakness {
			d.w.tell("A curse needs a name and the Weakness it casts.")
		} else {
			d.w.tell("A spell needs a name and the Ability it mimics.")
		}
		return
	case b == d.AddSpell:
		sp := character.NewSpell(name, it.Name, it, d.SpellLevel.Value())
		d.trait.Spells = append(d.trait.Spells, &sp.Spell)
		line = len(d.trait.Spells) - 1
	case x == nil:
		return
	default:
		x.Name = name
		if it != nil {
			x.Item, x.Ability = it, it.Name
		}
		x.Level = d.SpellLevel.Value()
	}
	d.w.changed()
	d.showSpells(line)
}

// ---- A partner's sheet ----

// attackAt is the partner's attack on line i; nil for none.
func (d *LinkedDialog) attackAt(i int) *character.Attack {
	if i < 0 || i >= len(d.partner.Attacks) {
		return nil
	}
	return character.AttackOf(d.partner.Attacks[i])
}

func (d *LinkedDialog) showAttacks(sel int) {
	var rows [][]string
	for _, a := range d.partner.Attacks {
		rows = append(rows, []string{character.AttackOf(a).Name,
			"ROLL " + a.Roll.Text(), "DX " + a.DX.Text(), "END " + a.End.Text()})
	}
	d.Attacks.Set(rows, sel)
}

func (d *LinkedDialog) attackDone(bool) { d.changed() }

func (d *LinkedDialog) attackButton(b *widget.Button) {
	if d.w.Attack != nil {
		return
	}
	if b == d.AddAttack {
		a := character.NewAttack("New attack")
		d.partner.Attacks = append(d.partner.Attacks, &a.Attack)
		character.Recompute(d.character)
		d.w.EditAttack(d.partner, a, true, d.attackDone)
		return
	}
	a := d.attackAt(d.Attacks.Selected())
	if a == nil {
		return
	}
	if b == d.EditAttack {
		d.w.EditAttack(d.partner, a, false, d.attackDone)
	} else {
		character.RemoveAttack(d.partner, &a.Attack)
		d.changed()
	}
}

// ---- Everything shown ----

func (d *LinkedDialog) showAll() {
	if d.w.Linked != d {
		return
	}
	switch d.Mode {
	case rules.Form:
		d.showFormLevels()
		d.End.Display()
	case rules.Spells:
		d.showSpells(d.Spells.Selected())
	case rules.Partner, rules.Machine:
		p := d.partner
		d.End.Display()
		d.showAttacks(d.Attacks.Selected())
		for _, b := range d.Numbers {
			b.Display()
		}
		// With no combat stats, nothing of combat is saved: all of it
		// inactive. Barrier's and Incorporeal Form's Defense and the
		// Reserve only with them.
		enable(p.HasCombat, d.AddAttack, d.EditAttack, d.RemoveAttack)
		for i, b := range d.Numbers {
			enable(p.HasCombat && (i != Barrier || p.HasBarrier) && (i != partnerIncorporeal || p.HasIncorporeal) &&
				(i != Reserve || p.HasReserve), b)
		}
	}
}

// ---- Opening and closing ----

// CloseDialog is Close: an attack dialog above this one closes first.
func (d *LinkedDialog) CloseDialog() {
	if d.w.Attack != nil {
		return
	}
	d.d.Hide()
	d.w.Linked = nil
	if d.done != nil {
		d.done()
	}
}

func (d *LinkedDialog) buildForm() fyne.CanvasObject {
	o := d.panels(300)
	d.Abilities.Show(&d.character.Sheet, &d.trait.Trait, d.character)
	d.Weaknesses.Show(&d.character.Sheet, &d.trait.Trait, d.character)
	return o
}

func (d *LinkedDialog) buildSpells() fyne.CanvasObject {
	d.Spells = newLines(320, 90)
	d.Spells.OnSelect = d.spellSelected
	d.SpellName = widget.NewEntry()
	d.CastKind = rules.Ability
	d.SpellKind = widget.NewLabel("Ability")
	if d.trait.Item.Flags.Has(rules.CastsWeaknesses) {
		d.CastKind = rules.Weakness
		d.SpellKind.SetText("Weakness")
	}
	labels := make([]string, rules.Count(d.CastKind))
	for i := range labels {
		labels[i] = rules.Label(rules.Get(d.CastKind, i))
	}
	d.SpellAbility = widget.NewSelect(labels, func(string) {
		if !d.filling {
			d.chooseSpellItem(d.spellItem())
		}
	})
	d.SpellHelp = NewHelpButton(0, d.w.openPage)
	if d.CastKind == rules.Weakness {
		d.SpellLevel = NewSpinner(-3, -1, -1)
	} else {
		d.SpellLevel = NewSpinner(1, 5, 1)
	}
	d.SpellEnd = d.numberBox("END", false)
	d.SpellEnd.AllowPlus = true
	d.AddSpell = widget.NewButton("Add", func() { d.spellButton(d.AddSpell) })
	d.ChangeSpell = widget.NewButton("Change", func() { d.spellButton(d.ChangeSpell) })
	d.RemoveSpell = widget.NewButton("Remove", func() { d.spellButton(d.RemoveSpell) })
	o := container.NewVBox(
		bold("Spells"),
		sized(d.Spells.List, 0, 300),
		container.NewGridWithColumns(2,
			container.NewBorder(nil, nil, widget.NewLabel("Name"), nil, d.SpellName),
			container.NewBorder(nil, nil, d.SpellKind, d.SpellHelp, d.SpellAbility)),
		container.NewHBox(widget.NewLabel("level"), d.SpellLevel, d.SpellEnd.Object()),
		container.NewHBox(d.AddSpell, d.ChangeSpell, d.RemoveSpell))
	d.showSpells(-1)
	return o
}

func (d *LinkedDialog) buildPartner() fyne.CanvasObject {
	p := d.partner
	d.PartnerName = widget.NewEntry()
	d.PartnerName.SetText(p.Name)
	d.PartnerName.OnChanged = func(s string) {
		if name := trim(s); name != "" {
			p.Name = name
		} else {
			p.Name = p.Label // the file needs one
		}
		d.changed()
	}
	d.Fragment = widget.NewCheck("a Fragment", nil)
	d.Fragment.SetChecked(p.IsFragment)
	d.Fragment.OnChanged = func(on bool) { p.IsFragment = on; d.changed() }
	d.Combat = widget.NewCheck("combat stats", nil)
	d.Combat.SetChecked(p.HasCombat)
	d.Combat.OnChanged = func(on bool) { p.HasCombat = on; d.changed() }
	panels := d.panels(170)
	d.Abilities.Show(&p.Sheet, nil, d.character)
	d.Weaknesses.Show(&p.Sheet, nil, d.character)
	d.Attacks = newLines(100, 100, 100)
	d.AddAttack = widget.NewButton("Add...", func() { d.attackButton(d.AddAttack) })
	d.EditAttack = widget.NewButton("Edit...", func() { d.attackButton(d.EditAttack) })
	d.RemoveAttack = widget.NewButton("Remove", func() { d.attackButton(d.RemoveAttack) })
	for i, n := range []*calc.Number{&p.Defense, &p.Barrier, &p.Health, &p.Endurance, &p.Reserve, &p.TV,
		&p.Incorporeal} {
		b := d.numberBox([]string{"DEFENSE", "Barrier DEFENSE", "HEALTH", "ENDURANCE", "Reserve", "TV",
			"Incorporeal DEFENSE"}[i], true)
		b.Target = n
		d.Numbers[i] = b
	}
	// Incorporeal Form's Defense under Barrier's, as in the window.
	boxes := []fyne.CanvasObject{d.Numbers[Defense].Object(),
		container.NewVBox(d.Numbers[Barrier].Object(), d.Numbers[partnerIncorporeal].Object()),
		d.Numbers[Health].Object(), d.Numbers[Endurance].Object(), d.Numbers[Reserve].Object(),
		d.Numbers[TV].Object()}
	o := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabel("Name"), container.NewHBox(d.Fragment, d.Combat),
			d.PartnerName),
		panels,
		bold("Attacks"),
		sized(d.Attacks.List, 0, 100),
		container.NewHBox(d.AddAttack, d.EditAttack, d.RemoveAttack),
		container.NewHBox(boxes...))
	d.showAttacks(-1)
	return o
}

// EditLinked edits t's linked content, t an Ability of c (or of its
// form, or its partner's) with some (t.Item.Linked isn't NoLink). A
// Companion or Vehicle gets its sheet now if it has none. done is called
// on Close.
func (w *Window) EditLinked(c *character.Character, t *character.Trait, done func()) {
	d := &LinkedDialog{w: w, character: c, trait: t, done: done, Mode: t.Item.Linked}
	if d.Mode == rules.Partner || d.Mode == rules.Machine {
		if t.Partner == nil {
			t.Partner = &character.NewPartner(t).Sheet
			w.changed()
		}
		d.partner = character.SheetOf(t.Partner)
	}
	w.Linked = d
	d.Info = widget.NewLabel("")
	d.Info.Wrapping = fyne.TextWrapWord
	var body fyne.CanvasObject
	switch d.Mode {
	case rules.Form:
		body = d.buildForm()
	case rules.Spells:
		if t.Item.Flags.Has(rules.CastsWeaknesses) {
			d.Info.SetText("Magic +" + strconv.Itoa(t.Level) +
				": a curse casts a Weakness; its END is by the Magic's level and the Weakness's size (p. 58).")
		} else {
			d.Info.SetText("Magic +" + strconv.Itoa(t.Level) +
				": a spell's END is by the Magic's level and its own (p. 58), more for Barrier.")
		}
		body = d.buildSpells()
	default:
		if d.Mode == rules.Partner {
			d.Info.SetText("A Companion's sheet: a Secondary NPC's (p. 52, 123).")
		} else {
			d.Info.SetText("A Vehicle's sheet (p. 64).")
		}
		body = d.buildPartner()
	}
	d.Close = widget.NewButton("Close", d.CloseDialog)
	content := container.NewVBox(d.Info, body, container.NewHBox(layout.NewSpacer(), d.Close))
	d.d = dialog.NewCustomWithoutButtons(rules.Label(t.Item), content, w.Win)
	d.d.Resize(fyne.NewSize(800, d.d.MinSize().Height))
	d.showAll()
	d.w.modal(d.d, d.CloseDialog)
}
