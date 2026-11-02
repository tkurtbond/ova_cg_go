package ui

// A list of Abilities or Weaknesses and its editor: a character's, a
// form's, a partner's (ova_cg_po's OvaTraitPanel).
//
// ova_cg_po's PLAN.md, "The window": the Abilities and the Weaknesses
// are each a list of lines ("+3 Attack (Kaze Satsujinsha)", and "p. 50"
// at the right; under Scaled Cost its points between, "4 pts") with a
// Panel's controls. The same panel edits the
// character's own (the window), a Transformation's form, and a
// Companion's or Vehicle's (the linked dialog): its list is a sheet's
// traits, or, given an owner, the owner's form.
//
// After each change the panel recomputes top, the character the list
// belongs to, so what is shown is calc's; then calls Changed, the
// window's or the dialog's. OnSelect tells which trait the user chose
// (nil after Remove), for the controls that act on one.

import (
	"strconv"

	"github.com/tkurtbond/ova_cg_go/internal/calc"
	"github.com/tkurtbond/ova_cg_go/internal/character"
	"github.com/tkurtbond/ova_cg_go/internal/rules"
)

// TraitPanel is a list of Abilities or Weaknesses and its editor.
type TraitPanel struct {
	*Panel
	sheet    *calc.Sheet // whose traits are listed,
	owner    *calc.Trait // or, if not nil, whose form
	top      *character.Character
	shown    []*character.Trait // shown[i]: line i's
	OnSelect func(t *character.Trait)
	tell     func(string)
}

// newTraits is a panel for kind's traits: levels +1 to +5 for an
// Ability, -1 to -3 for a Weakness.
func newTraits(kind rules.Kind, title string, height float32, openPage func(int), tell func(string)) *TraitPanel {
	p := &TraitPanel{tell: tell}
	p.Panel = newPanel(p, kind, title, "level", "detail", height, openPage, 50, 50)
	if kind == rules.Ability {
		p.Spinner.SetBounds(1, 5)
		p.Spinner.SetValue(1)
	} else {
		p.Spinner.SetBounds(-3, -1)
		p.Spinner.SetValue(-1)
	}
	return p
}

func (p *TraitPanel) list() *[]*calc.Trait {
	if p.owner != nil {
		return &p.owner.Form
	}
	return &p.sheet.Traits
}

// TraitAt is the trait on line i; nil for none.
func (p *TraitPanel) TraitAt(i int) *character.Trait {
	if i < 0 || i >= len(p.shown) {
		return nil
	}
	return p.shown[i]
}

// SelectedTrait is the selected trait; nil for none.
func (p *TraitPanel) SelectedTrait() *character.Trait { return p.TraitAt(p.Selected()) }

// traitLine is t's line: "+2 Flight (on a broom)", its Scaled Cost
// points ("2 pts", shown only under it), and its page.
func traitLine(t *character.Trait) []string {
	s := strconv.Itoa(t.Level)
	if t.Level > 0 {
		s = "+" + s
	}
	s += " " + t.Name
	if t.HasDetail {
		s += " (" + t.Detail + ")"
	}
	pts := strconv.Itoa(t.Points) + " pts"
	if t.Points == 1 {
		pts = "1 pt"
	}
	return []string{s, pts, rules.PageText(t.Item)}
}

// ShowsPoints says whether the lines show Scaled Cost's points: a
// sheet's own traits under it, not a form's, which no total counts
// (p. 63).
func (p *TraitPanel) ShowsPoints() bool {
	return p.sheet != nil && p.owner == nil && p.sheet.Options.ScaledCost
}

func (p *TraitPanel) fill() [][]string {
	p.shown = nil
	p.ShowColumn(0, p.ShowsPoints())
	if p.sheet == nil {
		return nil
	}
	p.OfferBoss(p.offersBoss())
	var rows [][]string
	for _, t := range *p.list() {
		if t.Kind == p.Kind {
			x := character.TraitOf(t)
			p.shown = append(p.shown, x)
			rows = append(rows, traitLine(x))
		}
	}
	return rows
}

// lineOf is the line t is on.
func (p *TraitPanel) lineOf(t *calc.Trait) int {
	i := 0
	for _, l := range *p.list() {
		if l == t {
			break
		}
		if l.Kind == p.Kind {
			i++
		}
	}
	return i
}

func (p *TraitPanel) selectTrait(t *character.Trait) {
	if p.OnSelect != nil {
		p.OnSelect(t)
	}
}

func (p *TraitPanel) load(i int) {
	t := p.shown[i]
	p.Choose(t.Item)
	p.Spinner.SetValue(t.Level)
	p.Text.SetText(t.Detail)
	p.selectTrait(t)
}

// chosen: a Boss Ability has no level of its own (ova_cg_po's PLAN.md,
// "Boss NPCs"), so its spinner is 1 and off.
func (p *TraitPanel) chosen() {
	p.helpChosen()
	if p.Item().Has(rules.BossOnly) {
		p.Spinner.SetValue(1)
		p.Spinner.Disable()
	} else {
		p.Spinner.Enable()
	}
}

// offersBoss says whether the list is a Boss's own Abilities.
func (p *TraitPanel) offersBoss() bool {
	return p.Kind == rules.Ability && p.sheet != nil && p.owner == nil && p.sheet.Kind == calc.BossSheet
}

func (p *TraitPanel) add() int {
	it := p.Item()
	if it == nil {
		p.tell("Choose a name first.")
		return -1
	}
	t := character.NewTrait(it, p.Kind, p.Spinner.Value(), "", trim(p.Text.Text))
	character.InsertTrait(p.list(), t)
	character.Recompute(p.top)
	return p.lineOf(&t.Trait)
}

// change: a name the tables don't have stays, if none is chosen.
func (p *TraitPanel) change(i int) int {
	t := p.shown[i]
	it := p.Item()
	if it == nil {
		it = t.Item
	}
	character.ChangeTrait(p.list(), t, it, p.Spinner.Value(), trim(p.Text.Text))
	character.Recompute(p.top)
	return p.lineOf(&t.Trait)
}

func (p *TraitPanel) remove(i int) bool {
	character.RemoveTrait(p.list(), &p.shown[i].Trait)
	p.selectTrait(nil)
	character.Recompute(p.top)
	return true
}

// Show lists sheet's traits (owner nil) or owner's form, recomputing top
// after a change; nothing selected.
func (p *TraitPanel) Show(sheet *calc.Sheet, owner *calc.Trait, top *character.Character) {
	p.sheet, p.owner, p.top = sheet, owner, top
	p.Refresh(-1)
	p.selectTrait(nil)
}

// ListShown says whether the lines are the model's, in its order: for
// tests.
func (p *TraitPanel) ListShown() bool {
	var want [][]string
	for _, t := range *p.list() {
		if t.Kind == p.Kind {
			want = append(want, traitLine(character.TraitOf(t)))
		}
	}
	if len(want) != p.Count() {
		return false
	}
	for i, w := range want {
		line := w[0] + "\t"
		if p.ShowsPoints() {
			line += w[1] + "\t"
		}
		if p.Line(i) != line+w[2] {
			return false
		}
	}
	return true
}
