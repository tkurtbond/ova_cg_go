package ui

// Character > Limits...: the rules that limit a character, saved in its
// options (po_ova_cg's OvaSettings).
//
// po_ova_cg's PLAN.md, "Limiting the character (p. 21)": Base Zero, a
// Power Ceiling, starting levels, Scaled Cost, Knowledge and Ineptitude
// at half cost, Combined Effectiveness, each turned on here and saved in
// the character's options: block. calc applies them; this dialog only
// sets them.
//
// It edits the character as the user works, as the attack dialog does,
// so the totals and warnings behind it follow each change (the window's
// changed); Cancel puts back the options there were. It is modal but
// doesn't wait: OK and Cancel call done.
//
// A number turned off keeps what it was, so turning it on again brings
// it back; one never set starts at p. 21's example: Base Zero's 5, 5
// starting levels and a Power Ceiling of 15.

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/tkurtbond/go_ova_cg/internal/calc"
	"github.com/tkurtbond/go_ova_cg/internal/character"
	"github.com/tkurtbond/go_ova_cg/internal/rules"
)

const (
	baseZeroDefault = 5
	ceilingDefault  = 15
	startingDefault = 5
)

// SettingsDialog is Character > Limits....
type SettingsDialog struct {
	w                                      *Window
	d                                      *dialog.CustomDialog
	BaseZero, PowerCeiling, StartingLevels *widget.Check
	ScaledCost, HalfCost, Combined         *widget.Check
	BaseZeroRange, Ceiling, Starting       *Spinner
	OK, Cancel                             *widget.Button
	character                              *character.Character
	old                                    calc.Options // as they were, for Cancel
	oldHas                                 bool
	done                                   func(ok bool)
	building                               bool
}

// same says whether a and b are the same limits: a rule's number counts
// only while the rule is on, since only then is it used or saved.
func same(a, b *calc.Options) bool {
	return a.BaseZero == b.BaseZero && (!a.BaseZero || a.BaseZeroRange == b.BaseZeroRange) &&
		a.PowerCeiling == b.PowerCeiling && (!a.PowerCeiling || a.Ceiling == b.Ceiling) &&
		a.StartingLevels == b.StartingLevels && (!a.StartingLevels || a.Starting == b.Starting) &&
		a.ScaledCost == b.ScaledCost && a.HalfCostKnowledge == b.HalfCostKnowledge &&
		a.CombinedEffectiveness == b.CombinedEffectiveness
}

// showActive makes the spinners active only with their rule, half cost
// only with one of the rules it applies under (p. 57).
func (d *SettingsDialog) showActive() {
	enable(d.BaseZero.Checked, d.BaseZeroRange)
	enable(d.PowerCeiling.Checked, d.Ceiling)
	enable(d.StartingLevels.Checked, d.Starting)
	enable(d.BaseZero.Checked || d.PowerCeiling.Checked || d.ScaledCost.Checked, d.HalfCost)
}

// take sets the character's options from the dialog. The block is saved
// while any rule is on, or if the file had one and nothing changed.
func (d *SettingsDialog) take() {
	o := calc.Options{
		BaseZero: d.BaseZero.Checked, BaseZeroRange: d.BaseZeroRange.Value(),
		PowerCeiling: d.PowerCeiling.Checked, Ceiling: d.Ceiling.Value(),
		StartingLevels: d.StartingLevels.Checked, Starting: d.Starting.Value(),
		ScaledCost: d.ScaledCost.Checked, HalfCostKnowledge: d.HalfCost.Checked,
		CombinedEffectiveness: d.Combined.Checked,
	}
	c := d.character
	if same(&o, &c.Options) {
		return
	}
	c.Options = o
	c.HasOptions = o.BaseZero || o.PowerCeiling || o.StartingLevels || o.ScaledCost || o.HalfCostKnowledge ||
		o.CombinedEffectiveness || d.oldHas && same(&o, &d.old)
	d.w.changed()
}

func (d *SettingsDialog) changeAction() {
	if d.building {
		return
	}
	d.showActive()
	d.take()
}

func (d *SettingsDialog) finish(accepted bool) {
	d.d.Hide()
	d.w.Settings = nil
	if d.done != nil {
		d.done(accepted)
	}
}

// Ok is OK: the limits kept.
func (d *SettingsDialog) Ok() { d.finish(true) }

// CancelEdit is Cancel: the options put back.
func (d *SettingsDialog) CancelEdit() {
	c := d.character
	if !same(&c.Options, &d.old) || c.HasOptions != d.oldHas {
		c.Options, c.HasOptions = d.old, d.oldHas
		d.w.changed()
	}
	d.finish(false)
}

func (d *SettingsDialog) check(label string, on bool) *widget.Check {
	c := widget.NewCheck(label, func(bool) { d.changeAction() })
	c.SetChecked(on)
	return c
}

// spinner is a spinner from low to high showing value, or def if the
// rule is off.
func (d *SettingsDialog) spinner(low, high int, on bool, value, def int) *Spinner {
	if !on {
		value = def
	}
	s := NewSpinner(low, high, value)
	s.OnChanged = func(int) { d.changeAction() }
	return s
}

// EditSettings edits c's limits; done is called on OK (ok) or Cancel.
func (w *Window) EditSettings(c *character.Character, done func(ok bool)) {
	d := &SettingsDialog{w: w, character: c, done: done, old: c.Options, oldHas: c.HasOptions}
	w.Settings = d
	o := c.Options
	d.building = true
	d.BaseZero = d.check("Base Zero: the Character Total within", o.BaseZero)
	d.BaseZeroRange = d.spinner(0, 99, o.BaseZero, o.BaseZeroRange, baseZeroDefault)
	d.StartingLevels = d.check("Starting levels, plus the Weaknesses'", o.StartingLevels)
	d.Starting = d.spinner(0, 999, o.StartingLevels, o.Starting, startingDefault)
	d.PowerCeiling = d.check("Power Ceiling: the Abilities at most", o.PowerCeiling)
	d.Ceiling = d.spinner(1, 999, o.PowerCeiling, o.Ceiling, ceilingDefault)
	d.ScaledCost = d.check("Scaled Cost: +1 to +5 cost 1, 2, 4, 8, 15", o.ScaledCost)
	d.HalfCost = d.check("Knowledge and Ineptitude at half cost", o.HalfCostKnowledge)
	d.Combined = d.check("Combined Effectiveness: at most +5 to a roll", o.CombinedEffectiveness)
	d.building = false
	d.OK = widget.NewButton("OK", d.Ok)
	d.OK.Importance = widget.HighImportance
	d.Cancel = widget.NewButton("Cancel", d.CancelEdit)
	help := func(page int) fyne.CanvasObject { return NewHelpButton(page, w.openPage) }
	row := func(c *widget.Check, label string, s *Spinner, page int) fyne.CanvasObject {
		var right fyne.CanvasObject = layout.NewSpacer()
		if s != nil {
			right = container.NewHBox(widget.NewLabel(label), s)
		}
		return container.NewBorder(nil, nil, nil, container.NewHBox(right, help(page)), c)
	}
	content := container.NewVBox(
		row(d.BaseZero, "+/-", d.BaseZeroRange, rules.LimitsPage),
		row(d.StartingLevels, "levels", d.Starting, rules.LimitsPage),
		row(d.PowerCeiling, "levels", d.Ceiling, rules.LimitsPage),
		row(d.ScaledCost, "", nil, rules.LimitsPage),
		row(d.HalfCost, "", nil, rules.HalfCostPage),
		row(d.Combined, "", nil, rules.LimitsPage),
		container.NewHBox(layout.NewSpacer(), d.OK, d.Cancel))
	d.d = dialog.NewCustomWithoutButtons("Limits", content, w.Win)
	d.d.Resize(fyne.NewSize(600, d.d.MinSize().Height))
	d.showActive()
	d.w.modal(d.d, d.CancelEdit)
}
