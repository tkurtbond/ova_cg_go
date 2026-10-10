package ui

// The attack dialog, and the Perks and Flaws editor of an Ability
// (po_ova_cg's OvaAttackDialog).
//
// po_ova_cg's PLAN.md, "The window": an attack's name, description and
// affinity, whether it is made with Telekinesis or in the transformed
// form, its Perks and its Flaws, each a list with a name, a count (where
// it can be taken more than once) and a cost (where the user chooses
// one: Cancel, Requirement, Unique), and its ROLL, DX and END, computed
// as the Perks and Flaws change, each with its override box. An Ability
// with Perks and Flaws is edited by the same dialog, without the
// attack's fields: its END alone.
//
// The dialog changes the model as the user works, so the numbers it
// shows are calc's; Cancel puts back what was there when it opened (a
// new attack, the window added, it takes out again). It doesn't wait:
// it shows, and OK or Cancel calls done.

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/tkurtbond/go_ova_cg/internal/calc"
	"github.com/tkurtbond/go_ova_cg/internal/character"
	"github.com/tkurtbond/go_ova_cg/internal/rules"
)

// maxTimes is the count's bound for an item taken as often as wanted.
const maxTimes = 10

// AttackDialog is the attack dialog, or an Ability's Perks and Flaws.
type AttackDialog struct {
	w                             *Window
	d                             *dialog.CustomDialog
	NameIn, AffinityIn            *widget.Entry
	DescriptionIn                 *widget.Entry
	Telekinetic, TransformedCheck *widget.Check
	Perks, Flaws                  *ModPanel
	Roll, DX, End                 *NumberBox
	OK, Cancel                    *widget.Button

	character   *character.Character
	attack      *character.Attack // what is edited: an attack,
	trait       *character.Trait  // or an Ability's Perks and Flaws
	isNew       bool
	savedAttack character.Attack
	savedEnd    calc.Number
	savedMods   []*calc.Mod
	done        func(changed bool)
	filling     bool
}

// ModPanel is the Perks' or the Flaws' list and editor.
type ModPanel struct {
	*Panel
	d      *AttackDialog
	isFlaw bool
	shown  []*character.Mod // shown[i]: line i's
}

func (d *AttackDialog) mods() *[]*calc.Mod {
	if d.attack != nil {
		return &d.attack.Mods
	}
	return &d.trait.Mods
}

func (d *AttackDialog) showNumbers() {
	if d.attack != nil {
		d.Roll.Display()
		d.DX.Display()
	}
	d.End.Display()
}

func (d *AttackDialog) recompute() {
	character.Recompute(d.character)
	d.showNumbers()
}

func (d *AttackDialog) numberChanged(b *NumberBox, ok bool) {
	if !ok {
		d.w.tell("Not a number for " + b.Label + ". The computed number is used.")
	}
	d.recompute()
}

func (d *AttackDialog) numberBox(label string) *NumberBox {
	b := NewNumberBox(label, true)
	b.Changed = d.numberChanged
	return b
}

// ---- The Perks and Flaws ----

// modCost is the cost m adds: its own, or its item's, each time.
func modCost(m *character.Mod) int {
	if m.Item == nil || m.Item.CostKind != rules.FixedCost {
		return m.Times * m.Cost
	}
	return m.Times * m.Item.Cost
}

func (p *ModPanel) fill() [][]string {
	p.shown = nil
	var rows [][]string
	for _, x := range *p.d.mods() {
		m := character.ModOf(x)
		if m.IsFlaw != p.isFlaw {
			continue
		}
		p.shown = append(p.shown, m)
		name := m.Name
		if m.HasDetail {
			name += " (" + m.Detail + ")"
		}
		if m.Times != 1 {
			name += " x" + strconv.Itoa(m.Times)
		}
		c := modCost(m)
		cost := strconv.Itoa(c)
		if c > 0 {
			cost = "+" + cost
		}
		rows = append(rows, []string{name, cost, rules.PageText(m.Item)})
	}
	return rows
}

func (p *ModPanel) lineOf(m *character.Mod) int {
	i := 0
	for _, l := range *p.d.mods() {
		if l == &m.Mod {
			break
		}
		if character.ModOf(l).IsFlaw == p.isFlaw {
			i++
		}
	}
	return i
}

// chosen sets the count's bounds and the cost's field, for the item
// chosen.
func (p *ModPanel) chosen() {
	p.helpChosen()
	it := p.Item()
	if it == nil {
		return
	}
	top := min(it.MaxTimes, maxTimes)
	p.Spinner.SetBounds(1, top)
	if top > 1 {
		p.Spinner.Enable()
	} else {
		p.Spinner.SetValue(1)
		p.Spinner.Disable()
	}
	if it.CostKind == rules.FixedCost {
		p.Text.SetText("")
		p.Text.Disable()
	} else {
		p.Text.Enable()
		if it.CostKind == rules.RangeCost && p.Text.Text == "" {
			p.Text.SetText(strconv.Itoa(it.MinCost))
		}
	}
}

func (p *ModPanel) load(i int) {
	m := p.shown[i]
	p.Choose(m.Item)
	p.Spinner.SetValue(m.Times)
	if m.Item == nil || m.Item.CostKind != rules.FixedCost {
		p.Text.SetText(strconv.Itoa(m.Cost))
	} else {
		p.Text.SetText("")
	}
}

// read is the controls' count and cost; ok false, said why, if the cost
// isn't an integer where one is needed.
func (p *ModPanel) read(it *rules.Item) (times, cost int, ok bool) {
	times = p.Spinner.Value()
	if it.CostKind == rules.FixedCost {
		return times, 0, true
	}
	var n calc.Number
	if !n.Parse(p.Text.Text, false, false) || !n.Overridden {
		p.d.w.tell("The cost must be a whole number, such as -10.")
		return 0, 0, false
	}
	return times, n.Get(), true
}

func (p *ModPanel) add() int {
	it := p.Item()
	if it == nil {
		p.d.w.tell("Choose a name first.")
		return -1
	}
	times, cost, ok := p.read(it)
	if !ok {
		return -1
	}
	m := character.NewMod(it, p.isFlaw, times, cost)
	character.InsertMod(p.d.mods(), m)
	p.d.recompute()
	return p.lineOf(m)
}

func (p *ModPanel) change(i int) int {
	m := p.shown[i]
	it := p.Item()
	if it == nil {
		it = m.Item
	}
	if it == nil {
		return -1 // a name the tables don't have: only Remove
	}
	times, cost, ok := p.read(it)
	if !ok {
		return -1
	}
	list := p.d.mods()
	character.RemoveMod(list, &m.Mod)
	if it != m.Item {
		m.Item, m.Name, m.HasDetail, m.Detail = it, it.Name, false, ""
	}
	m.Times, m.Cost, m.HasCost = times, cost, it.CostKind != rules.FixedCost
	character.InsertMod(list, m)
	p.d.recompute()
	return p.lineOf(m)
}

func (p *ModPanel) remove(i int) bool {
	character.RemoveMod(p.d.mods(), &p.shown[i].Mod)
	p.d.recompute()
	return true
}

func (d *AttackDialog) modPanel(isFlaw bool) *ModPanel {
	p := &ModPanel{d: d, isFlaw: isFlaw}
	kind, title := rules.Perk, "Perks"
	if isFlaw {
		kind, title = rules.Flaw, "Flaws"
	}
	p.Panel = newPanel(p, kind, title, "times", "cost", 200, d.w.openPage, 50, 50)
	p.Spinner.SetBounds(1, 1)
	p.Spinner.SetValue(1)
	return p
}

// ---- Opening and closing ----

func (d *AttackDialog) finish(changed bool) {
	d.d.Hide()
	d.w.Attack = nil
	if d.done != nil {
		d.done(changed)
	}
}

// Ok is OK: the attack's fields taken, the dialog closed. An attack
// needs a name: without one, said so, and the dialog stays.
func (d *AttackDialog) Ok() {
	if a := d.attack; a != nil {
		name := trim(d.NameIn.Text)
		if name == "" {
			d.w.tell("The attack needs a name.")
			return
		}
		a.Name = name
		character.SetText(&a.HasAffinity, &a.Affinity, trim(d.AffinityIn.Text))
		character.SetText(&a.HasDescription, &a.Description, d.DescriptionIn.Text)
		a.Telekinetic, a.Transformed = d.Telekinetic.Checked, d.TransformedCheck.Checked
	}
	character.Recompute(d.character)
	d.finish(true)
}

// CancelEdit is Cancel: everything put back as it was.
func (d *AttackDialog) CancelEdit() {
	if a := d.attack; a != nil {
		if d.isNew {
			character.RemoveAttack(d.character, &a.Attack)
		} else {
			*a = d.savedAttack
			a.Mods = d.savedMods
		}
	} else {
		d.trait.Mods, d.trait.End = d.savedMods, d.savedEnd
	}
	character.Recompute(d.character)
	d.finish(false)
}

func (d *AttackDialog) build(title string, isAttack bool) {
	d.NameIn = widget.NewEntry()
	d.AffinityIn = widget.NewEntry()
	d.DescriptionIn = widget.NewMultiLineEntry()
	d.DescriptionIn.Wrapping = fyne.TextWrapWord
	flags := func(bool) {
		if !d.filling && d.attack != nil {
			d.attack.Telekinetic, d.attack.Transformed = d.Telekinetic.Checked, d.TransformedCheck.Checked
			d.recompute()
		}
	}
	d.Telekinetic = widget.NewCheck("made with Telekinesis", flags)
	d.TransformedCheck = widget.NewCheck("made in the transformed form", flags)
	d.Perks = d.modPanel(false)
	d.Flaws = d.modPanel(true)
	d.Roll = d.numberBox("ROLL")
	d.DX = d.numberBox("DX")
	d.DX.AllowHalf = true
	d.End = d.numberBox("END")
	d.OK = widget.NewButton("OK", d.Ok)
	d.OK.Importance = widget.HighImportance
	d.Cancel = widget.NewButton("Cancel", d.CancelEdit)

	var top fyne.CanvasObject = layout.NewSpacer()
	numbers := container.NewHBox(d.End.Object())
	if isAttack {
		top = container.NewVBox(
			container.NewGridWithColumns(2,
				container.New(layout.NewFormLayout(),
					widget.NewLabel("Name"), d.NameIn, widget.NewLabel("Affinity"), d.AffinityIn),
				container.NewVBox(d.Telekinetic, d.TransformedCheck)),
			container.New(layout.NewFormLayout(), widget.NewLabel("Description"), sized(d.DescriptionIn, 0, 60)))
		numbers = container.NewHBox(d.Roll.Object(), d.DX.Object(), d.End.Object())
	}
	content := container.NewVBox(top,
		container.NewGridWithColumns(2, d.Perks.Object(), d.Flaws.Object()),
		numbers,
		container.NewHBox(layout.NewSpacer(), d.OK, d.Cancel))
	d.d = dialog.NewCustomWithoutButtons(title, content, d.w.Win)
	d.d.Resize(fyne.NewSize(760, d.d.MinSize().Height))
}

func (d *AttackDialog) show() {
	d.Perks.Refresh(-1)
	d.Flaws.Refresh(-1)
	character.Recompute(d.character)
	d.showNumbers()
	d.w.modal(d.d, d.CancelEdit)
}

// EditAttack edits a, an attack of c; isNew if it has just been added,
// so that Cancel takes it out again. done is called after OK or Cancel,
// changed true for OK.
func (w *Window) EditAttack(c *character.Character, a *character.Attack, isNew bool, done func(changed bool)) {
	d := &AttackDialog{w: w, character: c, attack: a, isNew: isNew, done: done}
	d.savedAttack = *a
	d.savedMods = character.CopyMods(a.Mods)
	w.Attack = d
	d.build("Attack", true)
	d.Roll.Target, d.DX.Target, d.End.Target = &a.Roll, &a.DX, &a.End
	d.filling = true
	d.NameIn.SetText(a.Name)
	d.AffinityIn.SetText(a.Affinity)
	d.DescriptionIn.SetText(a.Description)
	d.Telekinetic.SetChecked(a.Telekinetic)
	d.TransformedCheck.SetChecked(a.Transformed)
	d.filling = false
	d.show()
}

// EditMods edits t's Perks and Flaws, and its END: an Ability of c.
func (w *Window) EditMods(c *character.Character, t *character.Trait, done func(changed bool)) {
	d := &AttackDialog{w: w, character: c, trait: t, done: done}
	d.savedEnd = t.End
	d.savedMods = character.CopyMods(t.Mods)
	w.Attack = d
	title := "an Ability"
	if t.Item != nil {
		title = rules.Label(t.Item)
	}
	d.build(title, false)
	d.End.Target = &t.End
	d.show()
}
