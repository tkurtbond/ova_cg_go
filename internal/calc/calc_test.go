package calc

// The hand-made cases of po_ova_cg's TestCalc: a case for each Perk and
// Flaw that changes ROLL or DX, the floor at 0, 1/2 DX, Endurance Use,
// spells, Defense, Health and Endurance, TV, each limit and warning, and
// overrides counting downstream. The sample characters, the oracle, are
// checked in package character, which builds them from their files.

import (
	"strconv"
	"testing"

	"github.com/tkurtbond/go_ova_cg/internal/rules"
)

type checker struct {
	t      *testing.T
	sample string
}

func (c *checker) ok(ok bool, label string) {
	c.t.Helper()
	if !ok {
		c.t.Errorf("%s: %s", c.sample, label)
	}
}

// text checks n's text in effect against want.
func (c *checker) text(n *Number, want, label string) {
	c.t.Helper()
	if got := n.Text(); got != want {
		c.t.Errorf("%s: %s %s (got %s)", c.sample, label, want, got)
	}
}

func (c *checker) int(n *Number, want int, label string) {
	c.t.Helper()
	c.text(n, strconv.Itoa(want), label)
}

func (c *checker) is(got, want int, label string) {
	c.t.Helper()
	if got != want {
		c.t.Errorf("%s: %s %d (got %d)", c.sample, label, want, got)
	}
}

// warnedAbout says whether s has a warning of code about attack a (nil:
// about none).
func warnedAbout(s *Sheet, code Code, a *Attack) bool {
	for _, w := range s.Warnings {
		if w.Code == code && w.Attack == a {
			return true
		}
	}
	return false
}

func item(name string) *rules.Item {
	it := rules.Find(name)
	if it == nil {
		panic("no item " + name)
	}
	return it
}

// tr is name at level, added to list (a sheet's traits or a form).
func tr(list *[]*Trait, name string, level int) *Trait {
	it := item(name)
	t := &Trait{}
	InitTrait(t, t, it, it.Kind, level)
	*list = append(*list, t)
	return t
}

// md is name taken times, at cost each time where the user sets it.
func md(mods *[]*Mod, name string, times, cost int) {
	m := &Mod{}
	InitMod(m, m, item(name), times, cost)
	*mods = append(*mods, m)
}

func newAttack(s *Sheet) *Attack {
	a := &Attack{}
	InitAttack(a, a)
	s.Attacks = append(s.Attacks, a)
	return a
}

// TestPlain: no Abilities: the plain attack, the starting numbers.
func TestPlain(t *testing.T) {
	c := &checker{t, "plain"}
	s := NewSheet(Character, nil)
	Compute(s)
	c.int(&s.TV, 5, "no attack: TV 2 + 2 + 1")
	a := newAttack(s)
	Compute(s)
	c.int(&a.Roll, 2, "ROLL")
	c.int(&a.DX, 1, "DX")
	c.int(&a.End, 0, "END")
	c.int(&s.Defense, 2, "DEFENSE")
	c.ok(!s.HasBarrier, "no Barrier")
	c.int(&s.Health, 40, "HEALTH")
	c.int(&s.Endurance, 40, "ENDURANCE")
	c.ok(!s.HasReserve, "no Reserve")
	c.int(&s.TV, 5, "TV")
}

// TestRolls: ROLL: Agile, Clumsy, Combat Expert; Accurate and Inaccurate
// each time; a Will Attack's Iron-Willed and Weak-Willed. Attack and
// Strong don't add.
func TestRolls(t *testing.T) {
	c := &checker{t, "rolls"}
	s := NewSheet(Character, nil)
	tr(&s.Traits, "Agile", 3)
	tr(&s.Traits, "Combat Expert", 2)
	tr(&s.Traits, "Clumsy", -1)
	tr(&s.Traits, "Iron-Willed", 1)
	tr(&s.Traits, "Attack", 4)
	tr(&s.Traits, "Strong", 2)
	plain := newAttack(s)
	acc := newAttack(s)
	md(&acc.Mods, "Accurate", 2, 0)
	inacc := newAttack(s)
	md(&inacc.Mods, "Inaccurate", 3, 0)
	both := newAttack(s)
	md(&both.Mods, "Accurate", 2, 0)
	md(&both.Mods, "Inaccurate", 3, 0)
	will := newAttack(s)
	md(&will.Mods, "Will Attack", 1, 0)
	Compute(s)
	c.int(&plain.Roll, 6, "2 + Agile 3 + Combat Expert 2 - Clumsy 1")
	c.int(&acc.Roll, 8, "Accurate x2")
	c.int(&acc.End, 10, "Accurate x2 END")
	c.int(&inacc.Roll, 3, "Inaccurate x3")
	c.int(&inacc.End, 0, "Inaccurate x3 END, floored")
	c.int(&both.Roll, 5, "Accurate x2, Inaccurate x3")
	c.int(&both.End, 0, "Accurate x2, Inaccurate x3 END, floored")
	c.int(&will.Roll, 5, "Will Attack: 2 + Iron-Willed 1 + Combat Expert 2")
	c.int(&will.End, 0, "Will Attack END")
	c.is(acc.TVRoll, 6, "TV's roll leaves out Accurate")
	c.is(inacc.TVRoll, 6, "TV's roll leaves out Inaccurate")
	s = NewSheet(Character, nil)
	tr(&s.Traits, "Agile", 2)
	tr(&s.Traits, "Weak-Willed", -2)
	will = newAttack(s)
	md(&will.Mods, "Will Attack", 1, 0)
	Compute(s)
	c.int(&will.Roll, 0, "Will Attack: 2 - Weak-Willed 2, Agile not counted")
}

// TestDamage: DX: Attack, Strong, Weak; Ranged and Ranged,
// Strength-Powered; Telekinesis; Effective and Ineffective; No Damage;
// 1/2.
func TestDamage(t *testing.T) {
	c := &checker{t, "DX"}
	s := NewSheet(Character, nil)
	tr(&s.Traits, "Attack", 2)
	tr(&s.Traits, "Strong", 3)
	tr(&s.Traits, "Agile", 2)
	melee := newAttack(s)
	ranged := newAttack(s)
	md(&ranged.Mods, "Ranged", 1, 0)
	strRanged := newAttack(s)
	md(&strRanged.Mods, "Ranged, Strength-Powered", 1, 0)
	both := newAttack(s)
	md(&both.Mods, "Ranged", 1, 0)
	md(&both.Mods, "Ranged, Strength-Powered", 1, 0)
	eff := newAttack(s)
	md(&eff.Mods, "Effective", 2, 0)
	ineff := newAttack(s)
	md(&ineff.Mods, "Ineffective", 3, 0)
	half := newAttack(s)
	md(&half.Mods, "Ineffective", 6, 0)
	none := newAttack(s)
	md(&none.Mods, "Area Effect", 2, 0)
	md(&none.Mods, "No Damage", 1, 0)
	Compute(s)
	c.int(&melee.DX, 6, "1 + Attack 2 + Strong 3")
	c.int(&ranged.DX, 3, "Ranged drops Strong")
	c.int(&ranged.End, 0, "Ranged END")
	c.int(&strRanged.DX, 6, "Ranged, Strength-Powered keeps Strong")
	c.int(&strRanged.End, 5, "Ranged, Strength-Powered END")
	c.int(&both.DX, 6, "Ranged with Ranged, Strength-Powered keeps Strong")
	c.int(&eff.DX, 8, "Effective x2")
	c.int(&eff.End, 10, "Effective x2 END")
	c.int(&ineff.DX, 3, "Ineffective x3")
	c.text(&half.DX, "1/2", "Ineffective x6: 0 is 1/2")
	c.ok(half.DX.IsHalf() && half.DX.Get() == 0, "1/2 is half")
	c.int(&none.DX, 0, "No Damage: 0, not 1/2")
	c.int(&none.End, 0, "No Damage, Area Effect x2 END")
	c.is(eff.TVDX, 6, "TV's DX leaves out Effective")
	c.is(half.TVDX, 6, "TV's DX leaves out Ineffective")
	c.is(none.TVDX, 6, "TV's DX leaves out No Damage")
	c.int(&s.TV, 12, "TV: roll 4 + Defense 2 + DX 6")

	s = NewSheet(Character, nil)
	tr(&s.Traits, "Attack", 1)
	tr(&s.Traits, "Weak", -2)
	tr(&s.Traits, "Telekinesis", 3)
	melee = newAttack(s)
	ranged = newAttack(s)
	md(&ranged.Mods, "Ranged", 1, 0)
	tk := newAttack(s)
	md(&tk.Mods, "Ranged", 1, 0)
	tk.Telekinetic = true
	plainTk := newAttack(s)
	plainTk.Telekinetic = true
	Compute(s)
	c.text(&melee.DX, "1/2", "1 + Attack 1 - Weak 2: 0 is 1/2")
	c.is(melee.TVDX, 0, "TV's DX of a 1/2 is 0")
	c.int(&ranged.DX, 2, "Ranged drops Weak")
	c.int(&tk.DX, 5, "made with Telekinesis 3, Ranged")
	c.int(&plainTk.DX, 3, "made with Telekinesis 3, not Ranged")
	c.is(tk.TVDX, 5, "TV's DX keeps Telekinesis")
}

// TestEndurance: END: Endurance Use, on an attack and on an Ability; the
// floor; a chosen cost; Unique Perk.
func TestEndurance(t *testing.T) {
	c := &checker{t, "END"}
	s := NewSheet(Character, nil)
	a := newAttack(s)
	a.Use = -2
	md(&a.Mods, "Accurate", 1, 0)
	b := newAttack(s)
	b.Use = -1
	md(&b.Mods, "Effective", 2, 0)
	md(&b.Mods, "Weapon", 1, 0)
	cc := newAttack(s)
	md(&cc.Mods, "Requirement", 1, -15)
	md(&cc.Mods, "Area Effect", 1, 0)
	md(&cc.Mods, "Cancel", 1, -5)
	d := newAttack(s)
	md(&d.Mods, "Unique Perk", 1, 15)
	md(&d.Mods, "Ammunition", 3, 0)
	barrier := tr(&s.Traits, "Barrier", 3)
	barrier.Use = -1
	md(&barrier.Mods, "Elaborate Gestures", 1, 0)
	flight := tr(&s.Traits, "Flight", 2)
	flight.Use = -3
	plain := tr(&s.Traits, "Smart", 2)
	Compute(s)
	c.int(&a.End, 15, "Endurance Use -2 + Accurate")
	c.int(&b.End, 10, "Endurance Use -1 + Effective x2 - Weapon")
	c.int(&cc.End, 0, "Area Effect - Requirement 15 - Cancel 5, floored")
	c.int(&d.End, 0, "Unique Perk 15 - Ammunition x3")
	c.ok(barrier.HasEnd, "Barrier with Endurance Use -1 has an END")
	c.int(&barrier.End, 0, "Barrier: Endurance Use -1 - Elaborate Gestures")
	c.int(&flight.End, 20, "Flight: Endurance Use -3")
	c.ok(!plain.HasEnd, "Smart has no END")
	c.is(len(s.Warnings), 0, "no warnings")
}

// TestSpells: spells (p. 58): the table's corners and middle, "+" for
// Barrier, a level out of the table.
func TestSpells(t *testing.T) {
	c := &checker{t, "spells"}
	sp := func(t *Trait, name string, level int) *Spell {
		x := &Spell{}
		InitSpell(x, x, item(name), level)
		t.Spells = append(t.Spells, x)
		return x
	}
	s := NewSheet(Character, nil)
	magic := tr(&s.Traits, "Magic, Arcane", 3)
	a := sp(magic, "Barrier", 2)
	b := sp(magic, "Flight", 2)
	cc := sp(magic, "Quick", 6)
	low := tr(&s.Traits, "Magic, Witchcraft", 1)
	d := sp(low, "Strong", 5)
	e := sp(low, "Flight", 1)
	Compute(s)
	c.text(&a.End, "10+", "Magic 3, Barrier 2")
	c.text(&b.End, "10", "Magic 3, Flight 2")
	c.int(&d.End, 60, "Magic 1, Strong 5")
	c.int(&e.End, 20, "Magic 1, Flight 1")
	c.ok(s.Warned(SpellRange), "Quick 6 is out of the table")
	c.is(len(s.Warnings), 1, "one warning")
	magic.Level = 5
	cc.Level = 1
	Compute(s)
	c.int(&cc.End, 0, "Magic 5, Quick 1")
	c.text(&a.End, "2+", "Magic 5, Barrier 2")
}

// TestStats: Defense, Barrier, Health, Endurance, Endurance Reserve, by
// sheet kind.
func TestStats(t *testing.T) {
	c := &checker{t, "stats"}
	s := NewSheet(Character, nil)
	tr(&s.Traits, "Evasive", 2)
	tr(&s.Traits, "Quick", 3)
	tr(&s.Traits, "Slow", -1)
	tr(&s.Traits, "Barrier", 4)
	tr(&s.Traits, "Tough", 5)
	tr(&s.Traits, "Frail", -2)
	tr(&s.Traits, "Vigorous", 3)
	tr(&s.Traits, "Languid", -1)
	tr(&s.Traits, "Endurance Reserve", 2)
	Compute(s)
	c.int(&s.Defense, 6, "DEFENSE 2 + Evasive 2 + Quick 3 - Slow 1")
	c.ok(s.HasBarrier, "Barrier")
	c.int(&s.Barrier, 6, "Barrier's Defense 2 + 4")
	c.int(&s.Health, 80, "HEALTH 40 + Tough 5's 60 - Frail 2's 20")
	c.int(&s.Endurance, 60, "ENDURANCE 40 + Vigorous 3's 30 - Languid 1's 10")
	c.ok(s.HasReserve, "Reserve")
	c.int(&s.Reserve, 40, "Endurance Reserve 2")

	s = NewSheet(CompanionSheet, nil)
	tr(&s.Traits, "Tough", 1)
	Compute(s)
	c.int(&s.Health, 30, "a Companion's HEALTH 20 + Tough 1")
	c.int(&s.Endurance, 20, "a Companion's ENDURANCE")
	c.int(&s.TV, 2, "a Companion's TV 2 + 2 + 1 + Tough 1 - 4, at least 1")
	tr(&s.Traits, "Frail", -3)
	Compute(s)
	c.int(&s.TV, 1, "a Companion's TV, at least 1")
	s = NewSheet(MinionSheet, nil)
	Compute(s)
	c.int(&s.Health, 10, "Minions' HEALTH")
	c.int(&s.Endurance, 10, "Minions' ENDURANCE")
}

// TestThreatValue: TV (p. 124): Armored, the signed Health and Endurance
// levels, the marked Abilities' halves; Companion never counted.
func TestThreatValue(t *testing.T) {
	c := &checker{t, "TV"}
	s := NewSheet(Character, nil)
	tr(&s.Traits, "Armored", 2)
	tr(&s.Traits, "Tough", 1)
	tr(&s.Traits, "Languid", -1)
	healer := tr(&s.Traits, "Healer", 3)
	tr(&s.Traits, "Magic, Arcane", 1)
	companion := tr(&s.Traits, "Companion", 3)
	smart := tr(&s.Traits, "Smart", 3)
	Compute(s)
	c.ok(healer.HalfTV, "Healer starts marked")
	c.ok(!companion.HalfTV, "Companion doesn't")
	c.int(&s.TV, 10, "5 + Armored 2 + Tough 1 - Languid 1 + Healer 3/2 + Magic 1/2")
	companion.HalfTV = true
	Compute(s)
	c.int(&s.TV, 10, "a marked Companion still doesn't count")
	healer.HalfTV = false
	smart.HalfTV = true
	Compute(s)
	c.int(&s.TV, 10, "Healer unmarked, Smart 3 marked")
	smart.HalfTV = false
	Compute(s)
	c.int(&s.TV, 8, "neither")
}

// TestTransformation: (p. 63) the form counts when the sheet is shown
// transformed, or for an attack made in it, which TV then leaves out.
func TestTransformation(t *testing.T) {
	c := &checker{t, "Transformation"}
	s := NewSheet(Character, nil)
	tr(&s.Traits, "Agile", 1)
	tr(&s.Traits, "Tough", 4)
	x := tr(&s.Traits, "Transformation", 2)
	tr(&x.Form, "Combat Expert", 2)
	tr(&x.Form, "Barrier", 1)
	tr(&x.Form, "Tough", 3)
	base := newAttack(s)
	formed := newAttack(s)
	formed.Transformed = true
	Compute(s)
	c.int(&base.Roll, 3, "ROLL 2 + Agile 1")
	c.int(&formed.Roll, 5, "made transformed, + Combat Expert 2")
	c.ok(!s.HasBarrier, "no Barrier untransformed")
	c.int(&s.Health, 80, "HEALTH: Tough 4")
	c.int(&s.TV, 10, "TV leaves out the transformed attack: 3 + 2 + 1 + Tough 4")
	c.is(len(s.Warnings), 0, "no warnings")
	c.int(&s.AbilityLevels, 7, "the form's levels aren't in the totals")
	s.Transformed = true
	Compute(s)
	c.int(&base.Roll, 5, "transformed: ROLL")
	c.ok(s.HasBarrier, "transformed: Barrier")
	c.int(&s.Health, 100, "transformed: Tough 4 + 3 counted as 5")
	c.ok(s.Warned(TableRange), "transformed: Tough past 5 warned of")
	c.int(&s.TV, 16, "transformed: TV 5 + Barrier 3 + 1 + Tough 7")
}

// TestOverrides: overrides, and what is computed from them.
func TestOverrides(t *testing.T) {
	c := &checker{t, "overrides"}
	s := NewSheet(Character, nil)
	tr(&s.Traits, "Attack", 2)
	tr(&s.Traits, "Evasive", 1)
	tr(&s.Traits, "Barrier", 1)
	a := newAttack(s)
	md(&a.Mods, "Accurate", 2, 0)
	md(&a.Mods, "Effective", 1, 0)
	b := newAttack(s)
	Compute(s)
	c.int(&s.TV, 8, "TV 2 + Defense 3 + DX 3")
	a.Roll.Override(10, false, false)
	Compute(s)
	c.int(&a.Roll, 10, "ROLL overridden")
	c.ok(a.Roll.ComputedText() == "4", "still computed")
	c.int(&s.TV, 14, "TV: overridden ROLL 10 less Accurate x2")
	a.DX.Override(0, true, false)
	b.DX.Override(1, false, false)
	Compute(s)
	c.text(&a.DX, "1/2", "DX overridden 1/2")
	c.int(&s.TV, 12, "TV: DX 1/2 and 1")
	b.DX.Override(7, false, false)
	Compute(s)
	c.int(&s.TV, 18, "TV: DX overridden 7")
	s.Barrier.Override(6, false, false)
	Compute(s)
	c.int(&s.TV, 21, "TV: Barrier's Defense overridden 6")
	s.Defense.Override(9, false, false)
	Compute(s)
	c.int(&s.TV, 24, "TV: Defense overridden 9")
	s.TV.Override(1, false, false)
	Compute(s)
	c.int(&s.TV, 1, "TV overridden")
	s.TV.Clear()
	s.Defense.Clear()
	s.Barrier.Clear()
	a.Roll.Clear()
	a.DX.Clear()
	b.DX.Clear()
	Compute(s)
	c.int(&s.TV, 8, "all cleared")
	s.AbilityLevels.Override(10, false, false)
	Compute(s)
	c.int(&s.Total, 10, "Character Total from the overridden Abilities' 10")
	s.Options.BaseZero = true
	s.Options.BaseZeroRange = 5
	Compute(s)
	c.ok(s.Warned(BaseZeroBroken), "and Base Zero checks it")
	s.AbilityLevels.Clear()
	Compute(s)
	c.ok(!s.Warned(BaseZeroBroken), "cleared: Base Zero kept")

	c.sample = "Parse"
	var n Number
	c.ok(n.Parse("12", false, false) && n.Overridden && n.Get() == 12, "12")
	c.ok(n.Parse(" -2 ", false, false) && n.Get() == -2, "' -2 '")
	c.ok(n.Parse("  ", false, false) && !n.Overridden, "blank clears")
	c.ok(n.Parse("1/2", true, false) && n.IsHalf(), "1/2 for a DX")
	c.ok(!n.Parse("1/2", false, true) && n.IsHalf(), "1/2 only for a DX")
	c.ok(n.Parse("10+", false, true) && n.IsPlus(), "10+ for an END")
	c.ok(n.Text() == "10+", "10+ written back")
	c.ok(!n.Parse("10+", true, false), "10+ only for an END")
	c.ok(!n.Parse("x", true, true), "x refused")
	c.ok(!n.Parse("+", true, true), "+ refused")
	c.ok(!n.Parse("-", true, true), "- refused")
	c.ok(!n.Parse("4 5", true, true), "4 5 refused")
	c.ok(!n.Parse("1234567", true, true), "seven digits refused")
}

// TestLimits: the limits (p. 21) and the totals they check.
func TestLimits(t *testing.T) {
	c := &checker{t, "limits"}
	s := NewSheet(Character, nil)
	tr(&s.Traits, "Agile", 4)
	tr(&s.Traits, "Smart", 4)
	tr(&s.Traits, "Weak", -1)
	Compute(s)
	c.int(&s.AbilityLevels, 8, "Abilities")
	c.int(&s.WeaknessLevels, -1, "Weaknesses")
	c.int(&s.Total, 7, "Character Total")
	c.int(&s.Points, 15, "Scaled Cost points 8 + 8 - 1")
	c.is(len(s.Warnings), 0, "no limits, no warnings")
	s.Options.BaseZero = true
	s.Options.BaseZeroRange = 5
	Compute(s)
	c.ok(s.Warned(BaseZeroBroken), "Base Zero: 7 is past 5")
	tr(&s.Traits, "Poor", -2)
	Compute(s)
	c.ok(!s.Warned(BaseZeroBroken), "Base Zero: 5 isn't")
	s.Options.ScaledCost = true
	Compute(s)
	c.int(&s.Points, 13, "Scaled Cost points 16 - 1 - 2")
	c.ok(s.Warned(BaseZeroBroken), "Base Zero on Scaled Cost's points: 13 is past 5")

	s = NewSheet(Character, nil)
	tr(&s.Traits, "Agile", 5)
	tr(&s.Traits, "Smart", 5)
	tr(&s.Traits, "Quick", 5)
	tr(&s.Traits, "Poor", -3)
	tr(&s.Traits, "Rude", -3)
	tr(&s.Traits, "Shy", -3)
	tr(&s.Traits, "Mute", -1)
	s.Options.PowerCeiling = true
	s.Options.Ceiling = 15
	s.Options.StartingLevels = true
	s.Options.Starting = 5
	Compute(s)
	c.ok(!s.Warned(CeilingBroken), "Power Ceiling 15, 15 levels")
	c.ok(!s.Warned(StartingBroken), "starting 5 + Weaknesses 10")
	know := tr(&s.Traits, "Knowledge", 1)
	Compute(s)
	c.ok(s.Warned(CeilingBroken), "Power Ceiling 15, 16 levels")
	c.ok(s.Warned(StartingBroken), "starting 5 + Weaknesses 10, 16 levels")
	s.Options.HalfCostKnowledge = true
	know.Level = 2
	Compute(s)
	c.int(&s.AbilityLevels, 16, "Knowledge 2 at half cost")
	know.Level = 3
	tr(&s.Traits, "Ineptitude", -3)
	Compute(s)
	c.int(&s.AbilityLevels, 17, "Knowledge 3 at half cost, rounded up")
	c.int(&s.WeaknessLevels, -12, "Ineptitude -3 at half cost, rounded up")
	s.Options.ScaledCost = true
	know.Level = 5
	Compute(s)
	c.is(s.AbilityPoints, 53, "Scaled Cost: Knowledge 5 at half cost, 8")
	c.is(s.WeaknessPoints, 15, "Scaled Cost: Ineptitude -3 at half cost, 2")
	c.ok(s.Warned(CeilingBroken), "Power Ceiling on Scaled Cost's points")
	s.Options.PowerCeiling = false
	s.Options.StartingLevels = false
	s.Options.ScaledCost = false
	Compute(s)
	c.int(&s.AbilityLevels, 20, "no limit, no half cost")
	c.is(len(s.Warnings), 0, "no limit, no warnings")

	s = NewSheet(Character, nil)
	tr(&s.Traits, "Agile", 4)
	tr(&s.Traits, "Combat Expert", 2)
	tr(&s.Traits, "Evasive", 3)
	tr(&s.Traits, "Quick", 2)
	a := newAttack(s)
	Compute(s)
	c.is(len(s.Warnings), 0, "Combined Effectiveness off")
	s.Options.CombinedEffectiveness = true
	Compute(s)
	c.ok(warnedAbout(s, OverFive, a), "Combined Effectiveness: ROLL +6")
	c.ok(!warnedAbout(s, OverFive, nil), "Combined Effectiveness: Defense +5 is all right")
	tr(&s.Traits, "Quick", 1)
	Compute(s)
	c.ok(warnedAbout(s, OverFive, nil), "Combined Effectiveness: Defense +6")

	c.sample = "warnings"
	s = NewSheet(Character, nil)
	a = newAttack(s)
	md(&a.Mods, "Paralyzing", 2, 0)
	md(&a.Mods, "Requirement", 1, -10)
	md(&a.Mods, "Cancel", 1, 0)
	md(&a.Mods, "Redirectable", 1, 0)
	md(&a.Mods, "Ranged", 1, 0)
	md(&a.Mods, "No Damage", 1, 0)
	md(&a.Mods, "Effective", 1, 0)
	Compute(s)
	c.is(len(s.Warnings), 0, "within the limits")
	b := newAttack(s)
	md(&b.Mods, "Paralyzing", 3, 0)
	md(&b.Mods, "Requirement", 1, -25)
	md(&b.Mods, "Cancel", 1, -7)
	md(&b.Mods, "Redirectable", 1, 0)
	md(&b.Mods, "No Damage", 1, 0)
	md(&b.Mods, "Ineffective", 1, 0)
	md(&b.Mods, "Ammunition", 4, 0)
	Compute(s)
	c.ok(warnedAbout(s, TooOften, b), "Paralyzing x3, Ammunition x4")
	c.ok(warnedAbout(s, CostRange, b), "Requirement -25, Cancel -7")
	c.ok(warnedAbout(s, RedirectableAlone, b), "Redirectable not Ranged")
	c.ok(warnedAbout(s, NoDamageIneffective, b), "No Damage with Ineffective")
	c.is(len(s.Warnings), 6, "six warnings")
	s = NewSheet(Character, nil)
	tr(&s.Traits, "Agile", 6)
	tr(&s.Traits, "Weak", 0)
	tr(&s.Traits, "Poor", -4)
	unknown := &Trait{}
	InitTrait(unknown, unknown, nil, rules.Ability, 2)
	s.Traits = append(s.Traits, unknown)
	Compute(s)
	c.ok(s.Warned(UnknownItem), "an unknown Ability")
	c.int(&s.AbilityLevels, 8, "an unknown Ability counts in the totals")
	c.is(len(s.Warnings), 4, "Agile 6, Weak 0, Poor -4")
}
