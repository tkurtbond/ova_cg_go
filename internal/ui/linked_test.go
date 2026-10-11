package ui

// ova_cg_po's TestLinked: a Transformation's form, a Magic's spells, a
// Companion's and a Vehicle's sheets. Each kind of linked content
// opened from the window's button, as the user opens it; shown with
// each name's page; edited, live, and checked in the model, in the
// numbers computed from it, and in what is saved; and the window's
// Character > Transformed form.

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/widget"

	"github.com/tkurtbond/ova_cg_go/internal/calc"
	"github.com/tkurtbond/ova_cg_go/internal/character"
	"github.com/tkurtbond/ova_cg_go/internal/rules"
)

// click selects l's last line holding text, as a click does.
func click(l *lines, text string) { l.Select(l.Find(text)) }

func named(list []*calc.Trait, name string) *character.Trait {
	for _, t := range list {
		if x := character.TraitOf(t); x.Name == name {
			return x
		}
	}
	return nil
}

func partnerShown(d *LinkedDialog, p *character.Character) bool {
	for i, n := range []*calc.Number{&p.Defense, &p.Barrier, &p.Health, &p.Endurance, &p.Reserve, &p.TV,
		&p.Incorporeal} {
		if !d.Numbers[i].Shows(n) {
			return false
		}
	}
	return true
}

func TestLinked(t *testing.T) {
	w, f := newWindow(t)
	f.answer = Discard
	w.Win.Show()
	has := func(s string) bool { return strings.Contains(saved(w), s) }
	openLinked := func(name string) *LinkedDialog {
		click(w.Abilities.lines, name)
		w.AbilityLinked.OnTapped()
		return w.Linked
	}

	// ---- The button ----
	ok := load(w, "Fukiko.yaml")
	c := w.Character
	click(w.Abilities.lines, "Cute!")
	check(t, ok && w.AbilityLinked.Disabled(), "an Ability with no linked content: the button inactive")
	click(w.Abilities.lines, "Transformation")
	check(t, !w.AbilityLinked.Disabled() && w.AbilityLinked.Text == "Form...", "Transformation: Form...")
	click(w.Abilities.lines, "Companion")
	check(t, w.AbilityLinked.Text == "Companion...", "Companion: Companion...")

	// ---- A Transformation's form ----
	tr := named(c.Traits, "Transformation")
	d := openLinked("Transformation")
	if d == nil || d.Mode != rules.Form {
		t.Fatal("Form...: the dialog, for the form")
	}
	check(t, d.Abilities.ListShown() && d.Weaknesses.ListShown() && d.Abilities.Count() == 4,
		"the form's Abilities and Weaknesses listed, with pages")
	check(t, d.Info.Text == "The form's Abilities and Weaknesses come to +8 levels; Transformation +4 pays for 8 (p. 63)." &&
		d.Info.Importance != widget.DangerImportance, "its levels against what Transformation +4 pays for")
	before := c.Health.Value
	d.Abilities.Pick("Tough")
	d.Abilities.Spinner.SetValue(1)
	d.Abilities.Add.OnTapped()
	check(t, named(tr.Form, "Tough") != nil && d.Abilities.ListShown() && modified(w),
		"a form Ability added: in the form, listed, the character modified")
	check(t, c.Transformed && c.Health.Value == before+10, "Fukiko shown transformed: her HEALTH counts the form's Tough")
	check(t, d.Info.Text == "The form's Abilities and Weaknesses come to +9 levels; Transformation +4 pays for 8 (p. 63)." &&
		d.Info.Importance == widget.DangerImportance, "the levels again, over what it pays for: in red")
	check(t, has("- name: Tough"), "saved")
	click(d.Abilities.lines, "Barrier")
	check(t, !d.Mods.Disabled() && d.End.Active(), "the form's Barrier: its Perks and Flaws, its END")
	d.Mods.OnTapped()
	check(t, w.Attack != nil, "Perks and Flaws...: the dialog over this one")
	d.Close.OnTapped()
	check(t, w.Linked != nil, "Close waits for the dialog above it")
	w.Attack.CancelEdit()
	d.Close.OnTapped()
	check(t, w.Linked == nil, "Close")

	// ---- Character > Transformed form ----
	check(t, w.TransformedItem.Checked, "Character > Transformed form: on, as Fukiko's file says")
	w.TransformedItem.Action()
	check(t, !c.Transformed && c.Health.Value == before && w.Numbers[Health].Shows(&c.Health) &&
		!has("transformed: true"),
		"off: the form not counted, in the combat box too, and not saved as shown transformed")
	w.TransformedItem.Action()
	check(t, c.Transformed && c.Health.Value == before+10 && w.Numbers[Health].Shows(&c.Health) &&
		has("transformed: true"), "on again")

	// ---- A Companion's sheet ----
	tr = named(c.Traits, "Companion")
	p := character.SheetOf(tr.Partner)
	d = openLinked("Companion")
	check(t, d.Mode == rules.Partner && d.PartnerName.Text == "Azyrus" && d.Fragment.Checked && !d.Combat.Checked,
		"Companion...: Azyrus, a Fragment, no combat stats")
	check(t, d.Abilities.ListShown() && d.Weaknesses.ListShown(), "its Abilities and Weaknesses, with pages")
	check(t, d.AddAttack.Disabled() && !d.Numbers[Health].Active() && partnerShown(d, p),
		"no combat stats: its attacks and numbers inactive, but computed: a Companion's 20 less Frail")
	d.PartnerName.SetText("Azyrus II")
	check(t, p.Name == "Azyrus II" && has("name: Azyrus II"), "renamed, and saved")
	d.PartnerName.SetText(" ")
	check(t, p.Name == "Companion", "a blank name: Companion, since the file needs one")
	d.Combat.SetChecked(true)
	check(t, p.HasCombat && !d.AddAttack.Disabled() && d.Numbers[Health].Active() && !d.Numbers[Barrier].Active() &&
		!d.Numbers[Reserve].Active() && !d.Numbers[partnerIncorporeal].Active(),
		"combat stats on: its attacks and numbers, Barrier's, Incorporeal Form's and Reserve's not, having none")
	before = p.Health.Value
	d.Abilities.Pick("Tough")
	d.Abilities.Add.OnTapped()
	check(t, p.Health.Value == before+10 && partnerShown(d, p), "a Companion's Tough +1: its HEALTH 10 more, shown")
	d.Numbers[Health].Type("35")
	check(t, p.Health.Overridden && p.Health.Get() == 35 && partnerShown(d, p), "its HEALTH overridden")
	check(t, has("health: 35") && has("overrides: [health]"),
		"and saved, in its combat block, listed in overrides")
	d.AddAttack.OnTapped()
	if w.Attack == nil {
		t.Fatal("Add...: the attack dialog, for the Companion")
	}
	w.Attack.NameIn.SetText("Claw")
	w.Attack.Ok()
	check(t, len(p.Attacks) == 1 && d.Attacks.Count() == 1 && len(c.Attacks) == 4,
		"OK: the Companion's attack, not Fukiko's")
	check(t, strings.HasPrefix(d.Attacks.Line(0), "Claw\tROLL "), "listed, with its numbers")
	check(t, has("name: Claw"), "and saved")
	d.Attacks.Select(0)
	d.RemoveAttack.OnTapped()
	check(t, len(p.Attacks) == 0 && !has("name: Claw"), "Remove")
	d.Close.OnTapped()

	// ---- A Magic's spells ----
	load(w, "Auren.yaml")
	c = w.Character
	tr = named(c.Traits, "Magic, Arcane")
	d = openLinked("Magic, Arcane")
	check(t, w.AbilityLinked.Text == "Spells..." && d.Mode == rules.Spells && d.Spells.Count() == len(tr.Spells) &&
		len(tr.Spells) == 6, "Spells...: Auren's six spells")
	check(t, d.Spells.Line(0) == "Mana Shield\tBarrier +2 (p. 51)\tEND 10+",
		"each with its Ability's page and its END: Barrier's 10+")
	check(t, d.CastKind == rules.Ability && d.SpellKind.Text == "Ability" &&
		d.SpellLevel.Min == 1 && d.SpellLevel.Max == 5, "Arcane's spells: Abilities, +1 to +5")
	n := len(tr.Spells)
	d.SpellName.SetText("Glow")
	d.PickSpellAbility("Healer")
	check(t, d.SpellHelp.Page == 54, "the Ability chosen: ? at its page")
	d.SpellLevel.SetValue(1)
	d.AddSpell.OnTapped()
	i := d.Spells.Selected()
	check(t, len(tr.Spells) == n+1 && i == n && d.Spells.Line(i) == "Glow\tHealer +1 (p. 54)\tEND 5",
		"Add: last, selected, its END by Magic +3 and its level 1: 5 (p. 58)")
	check(t, has("name: Glow") && has("ability: Healer") && modified(w), "saved")
	d.SpellLevel.SetValue(2)
	d.ChangeSpell.OnTapped()
	check(t, d.Spells.Line(n) == "Glow\tHealer +2 (p. 54)\tEND 10", "Change: level 2, END 10")
	d.SpellEnd.Type("15")
	check(t, d.Spells.Line(n) == "Glow\tHealer +2 (p. 54)\tEND 15" && d.SpellEnd.Active(),
		"its END overridden: 15, listed")
	f.tells = 0
	d.SpellName.SetText(" ")
	d.AddSpell.OnTapped()
	check(t, f.tells == 1 && len(tr.Spells) == n+1, "a spell with no name: said so, not added")
	d.Spells.Select(n)
	check(t, d.SpellName.Text == "Glow", "a spell selected: its name, Ability and level to edit")
	d.RemoveSpell.OnTapped()
	check(t, len(tr.Spells) == n && !has("name: Glow") && d.Spells.Count() == n, "Remove")
	d.Close.OnTapped()

	// A change to an Ability of another kind drops what the file couldn't
	// hold.
	click(w.Abilities.lines, "Magic, Arcane")
	w.Abilities.Pick("Magic, Witchcraft")
	w.Abilities.Change.OnTapped()
	check(t, len(tr.Spells) == n, "Magic, Arcane to Magic, Witchcraft: the spells kept")
	w.Abilities.Pick("Flight")
	w.Abilities.Change.OnTapped()
	check(t, tr.Spells == nil && !has("spells:"), "to Flight: the spells dropped")

	// ---- Witchcraft's curses: Weaknesses (p. 58) ----
	ok = load(w, "Cora.yaml")
	c = w.Character
	tr = named(c.Traits, "Magic, Witchcraft")
	d = openLinked("Magic, Witchcraft")
	check(t, ok && d.CastKind == rules.Weakness && d.SpellKind.Text == "Weakness" &&
		d.SpellLevel.Min == -3 && d.SpellLevel.Max == -1 && d.SpellLevel.Value() == -1 && d.Spells.Count() == 0,
		"Cora's Witchcraft: its spells are Weaknesses, -1 to -3, none yet")
	d.SpellName.SetText("Hex")
	d.PickSpellAbility("Frail")
	check(t, d.SpellHelp.Page == 71, "the Weakness chosen: ? at its page, Frail's 71")
	d.SpellLevel.SetValue(-2)
	d.AddSpell.OnTapped()
	check(t, len(tr.Spells) == 1 && d.Spells.Line(0) == "Hex\tFrail -2 (p. 71)\tEND 20",
		"Add: Frail -2, its END by Witchcraft +2 and its size 2: 20")
	check(t, has("ability: Frail") && has("level: -2"), "saved")
	f.tells = 0
	d.SpellName.SetText(" ")
	d.AddSpell.OnTapped()
	check(t, f.tells == 1 && f.told == "A curse needs a name and the Weakness it casts." && len(tr.Spells) == 1,
		"a curse with no name: said so, not added")
	d.Close.OnTapped()
	d = openLinked("Magic, Witchcraft")
	d.Spells.Select(0)
	check(t, d.SpellName.Text == "Hex" && d.SpellAbility.SelectedIndex() == rules.Index(rules.Find("Frail")) &&
		d.SpellLevel.Value() == -2, "opened again and selected: its name, Weakness and level")
	d.Close.OnTapped()

	// ---- A new Companion, and a Vehicle ----
	w.NewItem.Action()
	c = w.Character
	w.Abilities.Pick("Companion")
	w.Abilities.Text.SetText("Rex")
	w.Abilities.Add.OnTapped()
	tr = named(c.Traits, "Companion")
	check(t, tr != nil && tr.Partner == nil, "a Companion added: no sheet yet")
	w.AbilityLinked.OnTapped()
	d = w.Linked
	check(t, tr.Partner != nil && tr.Partner.Kind == calc.CompanionSheet && d.PartnerName.Text == "Rex" &&
		d.Combat.Checked, "Companion...: its sheet made, a Companion's, named its detail, with combat stats")
	p = character.SheetOf(tr.Partner)
	check(t, p.Health.Value == 20 && p.Endurance.Value == 20 && partnerShown(d, p), "a Companion's 20 and 20 (p. 52)")
	check(t, has("companion:") && has("name: Rex"), "saved as companion:")
	d.Close.OnTapped()
	w.Abilities.Pick("Vehicle (Mecha)")
	w.Abilities.Text.SetText("")
	w.Abilities.Add.OnTapped()
	d = openLinked("Vehicle (Mecha)")
	tr = named(c.Traits, "Vehicle (Mecha)")
	check(t, tr.Partner != nil && tr.Partner.Kind == calc.Character && d.Mode == rules.Machine &&
		d.PartnerName.Text == "Vehicle" && has("vehicle:") && has("name: Vehicle"),
		"a Vehicle with no detail: a character's sheet, named Vehicle, saved as vehicle:")
	d.Close.OnTapped()
	load(w, "Vehicle.yaml")
	c = w.Character
	d = openLinked("Vehicle")
	check(t, w.AbilityLinked.Text == "Vehicle..." && d.Mode == rules.Machine && d.PartnerName.Text == "Thunderhawk" &&
		d.Attacks.Count() == 1, "Vehicle...: Thunderhawk's sheet, its attack")
	tr = named(c.Traits, "Vehicle")
	p = character.SheetOf(tr.Partner)
	check(t, partnerShown(d, p) && p.Endurance.Value == 40, "its numbers: a character's 40 (p. 64)")
	d.Close.OnTapped()
}
