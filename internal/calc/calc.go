// Package calc computes the numbers on the sheet from the character.
//
// po_ova_cg's PLAN.md, "The rules the program computes" and "Limiting
// the character": each attack's ROLL, DX and END; the END of an Ability
// with Perks and Flaws or Endurance Use, and of each spell; DEFENSE and
// Barrier's second Defense; HEALTH, ENDURANCE, Endurance Reserve and TV;
// the running totals; and the warnings of broken limits.
//
// A character is given as a Sheet of Traits (its Abilities and
// Weaknesses) and Attacks, each Attack and Trait with its Perks and
// Flaws (Mods). The model (package character) embeds these types in its
// own, which hold what calc doesn't need (names, details, text); each
// record's Owner points back at the record that embeds it, so a warning
// about a *Trait leads the window to the character's.
//
// Every computed number is a Number, which also holds the user's
// override (po_ova_cg's PLAN.md, "Overriding a computed number").
// Compute reads the overrides of the numbers others are computed from
// (an attack's ROLL and DX, Defense, the totals) so that an override
// counts downstream, and sets every computed value; the overrides it
// leaves alone.
//
// The port of po_ova_cg's OvaCalc. Imports nothing of the window or the
// files, so its tests need no display.
package calc

import (
	"strconv"
	"strings"

	"github.com/tkurtbond/ova_cg/internal/rules"
)

// SheetKind says where Health and Endurance start, and TV's last step.
type SheetKind int

const (
	Character      SheetKind = iota // 40 and 40 (p. 19); a Vehicle's too (p. 64)
	CompanionSheet                  // 20 and 20 (p. 52); TV 4 less, at least 1, a Secondary NPC's (p. 123-124)
	MinionSheet                     // 10 and 10 (p. 58)
)

// Code is a warning's kind. Each names what it is about in its trait,
// attack, mod or spell; none for the sheet as a whole.
type Code int

const (
	UnknownItem         Code = iota // trait or mod: a name rules doesn't have
	LevelRange                      // trait: an Ability not +1 to +5, a Weakness not -1 to -3 (p. 18)
	TooOften                        // mod: taken more times than its text allows (p. 83)
	CostRange                       // mod: a chosen cost outside its range, or not in steps of 5 (p. 87-88)
	NoDamageIneffective             // attack: No Damage with Ineffective (p. 88)
	RedirectableAlone               // attack: Redirectable without Ranged (p. 86)
	SpellRange                      // spell: its level, or its Magic's, not 1 to 5: no cost (p. 58)
	TableRange                      // Tough, Vigorous or Endurance Reserve past +5 in all: counted as +5
	BaseZeroBroken                  // the Character Total (or points) beyond Base Zero's range (p. 21)
	StartingBroken                  // Abilities beyond the starting levels and the Weaknesses' (p. 21)
	CeilingBroken                   // Abilities beyond the Power Ceiling (p. 21)
	OverFive                        // attack, or none for Defense: Abilities give a roll more than +5 (p. 21)
)

const (
	combinedLimit = 5 // Combined Effectiveness: at most +5 to a roll
	characterBase = 40
	companionBase = 20
	minionBase    = 10
	companionTV   = 4 // taken off a Secondary NPC's TV
	maxLevel      = 5
	minWeakness   = -3
	costStep      = 5
)

// Number is a number on the sheet: the computed value, and the override
// the user typed, if any. A DX can be 1/2 (Half, value 0) and a spell's
// END "10+" (Plus). An override is the same three.
type Number struct {
	Value      int
	Half, Plus bool
	Overridden bool
	OvValue    int
	OvHalf     bool
	OvPlus     bool
}

// Mod is a Perk or Flaw taken, by an attack or an Ability. Cost is each
// time's cost where the user sets it: a RangeCost's or UserCost's, or an
// item the tables don't have (from the file); else ignored.
type Mod struct {
	Owner any
	Item  *rules.Item // nil: a name rules doesn't have
	Times int         // >= 1; ova_fmt's counts-as
	Cost  int
}

// Spell is a Magic Ability's spell, mimicking Item at Level.
type Spell struct {
	Owner any
	Item  *rules.Item
	Level int
	End   Number
}

// Trait is an Ability or Weakness, at its signed level.
type Trait struct {
	Owner   any
	Item    *rules.Item // nil: a name rules doesn't have
	Kind    rules.Kind  // rules.Ability or Weakness, with or without Item
	Level   int
	HalfTV  bool     // marked: half its level counts toward TV (p. 124)
	Mods    []*Mod   // its Perks and Flaws (p. 83)
	Use     int      // the level, -1 to -3, of an Endurance Use tied to it; 0 if none
	Form    []*Trait // a Transformation's form: counted when transformed (p. 63)
	Spells  []*Spell // a Magic Ability's
	Partner *Sheet   // a Companion's or Vehicle's own sheet
	HasEnd  bool     // it has Perks, Flaws or Endurance Use: End is its cost
	End     Number
}

// Attack is an attack in the combat box.
type Attack struct {
	Owner       any
	Mods        []*Mod
	Telekinetic bool // made with Telekinesis, which adds to its DX (p. 62)
	Transformed bool // made in the transformed form, as Miho's Mode SP
	Use         int  // an Endurance Use tied to it, as a Trait's
	Roll, DX    Number
	End         Number
	// What TV takes of it (po_ova_cg's PLAN.md, "The combat box"): its
	// ROLL and DX in effect, less what Accurate, Inaccurate, Effective
	// and Ineffective add; 1/2 and No Damage count as 0. InTV is false
	// for an attack made in a form the sheet isn't shown in.
	InTV         bool
	TVRoll, TVDX int
}

// Options are the limits the user turned on (p. 21; ova_fmt's options:).
type Options struct {
	BaseZero              bool
	BaseZeroRange         int
	PowerCeiling          bool
	Ceiling               int
	StartingLevels        bool
	Starting              int
	ScaledCost            bool
	HalfCostKnowledge     bool
	CombinedEffectiveness bool
}

// Warning is something the rules find wrong, and what it is about.
type Warning struct {
	Code   Code
	Trait  *Trait
	Attack *Attack
	Mod    *Mod
	Spell  *Spell
}

// Sheet is a character's (or a partner's) Traits and Attacks, and every
// number computed from them.
type Sheet struct {
	Owner       any
	Kind        SheetKind
	Traits      []*Trait // in the order added
	Attacks     []*Attack
	Transformed bool // shown in the transformed form, as Fukiko's sheet is
	Options     Options
	Defense     Number
	Barrier     Number
	Health      Number
	Endurance   Number
	Reserve     Number
	TV          Number
	HasBarrier  bool
	HasReserve  bool
	// The running totals: levels (Knowledge and Ineptitude halved if the
	// options say so), the Character Total, and Scaled Cost's points
	// (abilities' less weaknesses').
	AbilityLevels, WeaknessLevels, Total, Points Number
	AbilityPoints, WeaknessPoints                int        // both >= 0
	Warnings                                     []*Warning // from the last Compute, in the order found
}

// ---- Numbers ----

func (n *Number) setComputed(value int, half, plus bool) {
	n.Value, n.Half, n.Plus = value, half, plus
}

func (n *Number) set(value int) { n.setComputed(value, false, false) }

// Override overrides n with value (1/2 if half, value+ if plus).
func (n *Number) Override(value int, half, plus bool) {
	n.Overridden = true
	if half {
		value = 0
	}
	n.OvValue, n.OvHalf, n.OvPlus = value, half, plus
}

// Clear goes back to the computed value.
func (n *Number) Clear() {
	n.Overridden, n.OvValue, n.OvHalf, n.OvPlus = false, 0, false, false
}

// Get is the value in effect: the override if there is one; 1/2 is 0.
func (n *Number) Get() int {
	if n.Overridden {
		return n.OvValue
	}
	return n.Value
}

// IsHalf says whether the number in effect is 1/2.
func (n *Number) IsHalf() bool {
	if n.Overridden {
		return n.OvHalf
	}
	return n.Half
}

// IsPlus says whether the number in effect is written with a "+".
func (n *Number) IsPlus() bool {
	if n.Overridden {
		return n.OvPlus
	}
	return n.Plus
}

func format(value int, half, plus bool) string {
	s := strconv.Itoa(value)
	if half {
		s = "1/2"
	}
	if plus {
		s += "+"
	}
	return s
}

// Text is the number in effect as the sheet writes it: "4", "1/2", "10+".
func (n *Number) Text() string { return format(n.Get(), n.IsHalf(), n.IsPlus()) }

// ComputedText is the computed number, whatever the override.
func (n *Number) ComputedText() string { return format(n.Value, n.Half, n.Plus) }

// Parse sets n's override from what the user typed in its box: blank
// clears it; else an integer, "1/2" if allowHalf (a DX), or an integer
// and "+" if allowPlus (an END). False, n unchanged, for anything else.
func (n *Number) Parse(s string, allowHalf, allowPlus bool) bool {
	s = strings.Trim(s, " \t")
	if s == "" {
		n.Clear()
		return true
	}
	if allowHalf && s == "1/2" {
		n.Override(0, true, false)
		return true
	}
	plus := false
	if allowPlus && strings.HasSuffix(s, "+") {
		plus = true
		s = s[:len(s)-1]
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	// At most six digits, as po_ova_cg's: no overflow, and nothing on
	// the sheet is near a million.
	if len(s) == 0 || len(s) > 6 {
		return false
	}
	v := 0
	for _, c := range []byte(s) {
		if c < '0' || c > '9' {
			return false
		}
		v = 10*v + int(c-'0')
	}
	if neg {
		v = -v
	}
	n.Override(v, false, plus)
	return true
}

// ---- Building a sheet ----

// NewSheet is an empty sheet of kind k, with owner as its Owner.
func NewSheet(k SheetKind, owner any) *Sheet { return &Sheet{Kind: k, Owner: owner} }

// InitTrait sets t up: item may be nil, for a name rules doesn't have;
// kind then says which it is. A Healer or Magic Ability starts marked
// for TV.
func InitTrait(t *Trait, owner any, item *rules.Item, kind rules.Kind, level int) {
	*t = Trait{Owner: owner, Item: item, Kind: kind, Level: level, HalfTV: item.Has(rules.HalfTV)}
}

// InitMod sets m up; cost is each time's, used where the user sets it.
func InitMod(m *Mod, owner any, item *rules.Item, times, cost int) {
	*m = Mod{Owner: owner, Item: item, Times: times, Cost: cost}
}

// InitSpell sets sp up.
func InitSpell(sp *Spell, owner any, item *rules.Item, level int) {
	*sp = Spell{Owner: owner, Item: item, Level: level}
}

// InitAttack sets a up.
func InitAttack(a *Attack, owner any) { *a = Attack{Owner: owner} }

// ---- Warnings ----

func (s *Sheet) warn(c Code, t *Trait, a *Attack, m *Mod, sp *Spell) {
	s.Warnings = append(s.Warnings, &Warning{Code: c, Trait: t, Attack: a, Mod: m, Spell: sp})
}

// Warned says whether the last Compute gave a warning of code c.
func (s *Sheet) Warned(c Code) bool {
	for _, w := range s.Warnings {
		if w.Code == c {
			return true
		}
	}
	return false
}

// ---- Traits ----

func (t *Trait) has(f rules.Flag) bool { return t.Item.Has(f) }

// sum is the signed levels of s's Traits that have flag, with their
// forms' when form (a Transformation's form adds to the same Abilities).
func (s *Sheet) sum(f rules.Flag, form bool) int {
	n := 0
	for _, t := range s.Traits {
		if t.has(f) {
			n += t.Level
		}
		if form {
			for _, x := range t.Form {
				if x.has(f) {
					n += x.Level
				}
			}
		}
	}
	return n
}

// present says whether s has a Trait with flag (in its form too, when
// form).
func (s *Sheet) present(f rules.Flag, form bool) bool {
	for _, t := range s.Traits {
		if t.has(f) {
			return true
		}
		if form {
			for _, x := range t.Form {
				if x.has(f) {
					return true
				}
			}
		}
	}
	return false
}

// tableLevel is a table's level, 0 to +5; one past it is counted as +5,
// with a warning (two Tough Abilities, or the form's added to the
// base's).
func (s *Sheet) tableLevel(level int) int {
	switch {
	case level > maxLevel:
		s.warn(TableRange, nil, nil, nil, nil)
		return maxLevel
	case level < 0:
		s.warn(TableRange, nil, nil, nil, nil)
		return 0
	}
	return level
}

// ---- Perks and Flaws ----

func modHas(list []*Mod, f rules.Flag) bool {
	for _, m := range list {
		if m.Item.Has(f) {
			return true
		}
	}
	return false
}

// kindHas says whether list has a Mod of kind (rules.Perk or Flaw) with
// flag.
func kindHas(list []*Mod, k rules.Kind, f rules.Flag) bool {
	for _, m := range list {
		if m.Item != nil && m.Item.Kind == k && m.Item.Has(f) {
			return true
		}
	}
	return false
}

func modHasItem(list []*Mod, it *rules.Item) bool {
	for _, m := range list {
		if m.Item == it {
			return true
		}
	}
	return false
}

// net is the times the Perks with flag are taken, less the times the
// Flaws with it are: Accurate x2 and Inaccurate x3 make -1.
func net(list []*Mod, f rules.Flag) int {
	n := 0
	for _, m := range list {
		if m.Item.Has(f) {
			if m.Item.Kind == rules.Perk {
				n += m.Times
			} else {
				n -= m.Times
			}
		}
	}
	return n
}

// ModCost is what m adds to an END: its item's fixed cost, or its own,
// each time.
func ModCost(m *Mod) int {
	if m.Item != nil && m.Item.CostKind == rules.FixedCost {
		return m.Item.Cost * m.Times
	}
	return m.Cost * m.Times
}

// useCost is what Endurance Use at level adds (p. 70); none outside -1
// to -3.
func useCost(level int) int {
	if level < 0 && level >= minWeakness {
		return rules.EnduranceUseCost(level)
	}
	return 0
}

// endCost is an attack's or Ability's Endurance cost (p. 83, 70): its
// Perks', its Endurance Use's, less its Flaws', never below 0.
func endCost(list []*Mod, use int) int {
	n := useCost(use)
	for _, m := range list {
		n += ModCost(m)
	}
	if n < 0 {
		n = 0
	}
	return n
}

// checkMods gives the warnings of each Mod: unknown, taken too often, a
// cost out of its range. t or a is the owner.
func (s *Sheet) checkMods(list []*Mod, t *Trait, a *Attack) {
	for _, m := range list {
		it := m.Item
		if it == nil {
			s.warn(UnknownItem, t, a, m, nil)
			continue
		}
		if m.Times < 1 || m.Times > it.MaxTimes {
			s.warn(TooOften, t, a, m, nil)
		}
		if it.CostKind == rules.RangeCost && (m.Cost < it.MinCost || m.Cost > it.MaxCost || m.Cost%costStep != 0) {
			s.warn(CostRange, t, a, m, nil)
		}
	}
}

// ---- The combat box (p. 105-107) ----

var redirectable = rules.Find("Redirectable")

func (s *Sheet) computeAttack(a *Attack) {
	form := s.Transformed || a.Transformed
	// ROLL: 2 + Agile - Clumsy, or Iron-Willed - Weak-Willed for a Will
	// Attack, + Combat Expert, + Accurate - Inaccurate.
	var roll int
	if modHas(a.Mods, rules.WillAttack) {
		roll = 2 + s.sum(rules.Will, form)
	} else {
		roll = 2 + s.sum(rules.Agility, form)
	}
	roll += s.sum(rules.CombatSkill, form)
	if s.Options.CombinedEffectiveness && roll-2 > combinedLimit {
		s.warn(OverFive, nil, a, nil, nil)
	}
	netRoll := net(a.Mods, rules.RollEach)
	a.Roll.set(roll + netRoll)
	// DX: 1 + Attack + Strong - Weak (not if Ranged, unless
	// Strength-Powered) + Telekinesis if made with it + Effective -
	// Ineffective; 0 for No Damage, else 1/2 at 0 or less.
	dx := 1 + s.sum(rules.Damage, form)
	if !modHas(a.Mods, rules.RangedAttack) || modHas(a.Mods, rules.StrengthRanged) {
		dx += s.sum(rules.Strength, form)
	}
	if a.Telekinetic {
		dx += s.sum(rules.Telekinetic, form)
	}
	netDX := net(a.Mods, rules.DamageEach)
	switch {
	case modHas(a.Mods, rules.NoDamageDone):
		a.DX.set(0)
	case dx+netDX <= 0:
		a.DX.setComputed(0, true, false)
	default:
		a.DX.set(dx + netDX)
	}
	a.End.set(endCost(a.Mods, a.Use))
	// What TV takes: the numbers in effect, overrides included, less
	// what the Perks and Flaws added each time.
	a.InTV = !a.Transformed || s.Transformed
	a.TVRoll = a.Roll.Get() - netRoll
	switch {
	case !a.DX.Overridden:
		a.TVDX = dx
	case a.DX.OvHalf:
		a.TVDX = 0
	default:
		a.TVDX = a.DX.OvValue - netDX
	}
	if a.TVDX < 0 {
		a.TVDX = 0
	}
	s.checkMods(a.Mods, nil, a)
	if modHas(a.Mods, rules.NoDamageDone) && kindHas(a.Mods, rules.Flaw, rules.DamageEach) {
		s.warn(NoDamageIneffective, nil, a, nil, nil)
	}
	if redirectable != nil && modHasItem(a.Mods, redirectable) &&
		!modHas(a.Mods, rules.RangedAttack) && !modHas(a.Mods, rules.StrengthRanged) {
		s.warn(RedirectableAlone, nil, a, nil, nil)
	}
}

// computeTrait gives an Ability's or Weakness's own warnings and END,
// and its spells' costs (p. 58): by the Magic's level and the spell's,
// "+" for an Ability that spends Endurance of its own (Barrier).
func (s *Sheet) computeTrait(t *Trait) {
	if t.Item == nil {
		s.warn(UnknownItem, t, nil, nil, nil)
	}
	if t.Kind == rules.Ability && (t.Level < 1 || t.Level > maxLevel) ||
		t.Kind == rules.Weakness && (t.Level > -1 || t.Level < minWeakness) {
		s.warn(LevelRange, t, nil, nil, nil)
	}
	t.HasEnd = len(t.Mods) > 0 || t.Use != 0
	if t.HasEnd {
		t.End.set(endCost(t.Mods, t.Use))
	} else {
		t.End.set(0)
	}
	s.checkMods(t.Mods, t, nil)
	for _, sp := range t.Spells {
		if t.Level >= 1 && t.Level <= maxLevel && sp.Level >= 1 && sp.Level <= maxLevel {
			sp.End.setComputed(rules.SpellCost(t.Level, sp.Level), false, sp.Item.Has(rules.OwnCost))
		} else {
			sp.End.set(0)
			s.warn(SpellRange, t, nil, nil, sp)
		}
	}
}

// computeDefense: Defense (p. 105), and Barrier's (p. 51).
func (s *Sheet) computeDefense() {
	form := s.Transformed
	d := 2 + s.sum(rules.DefenseRoll, form)
	s.Defense.set(d)
	if s.Options.CombinedEffectiveness && d-2 > combinedLimit {
		s.warn(OverFive, nil, nil, nil, nil)
	}
	s.HasBarrier = s.present(rules.SecondDefense, form)
	if s.HasBarrier {
		d = 2 + s.sum(rules.SecondDefense, form)
		s.Barrier.set(d)
		if s.Options.CombinedEffectiveness && d-2 > combinedLimit {
			s.warn(OverFive, nil, nil, nil, nil)
		}
	} else {
		s.Barrier.set(0)
	}
}

// computeHealth: Health and Endurance (p. 107), from the sheet's base,
// + the Tough and Vigorous table (p. 63, 65), - 10 a level of Frail and
// Languid (p. 71, 73); and Endurance Reserve's separate store (p. 53).
func (s *Sheet) computeHealth() {
	form := s.Transformed
	base := characterBase
	switch s.Kind {
	case CompanionSheet:
		base = companionBase
	case MinionSheet:
		base = minionBase
	}
	s.Health.set(base + rules.ToughBonus(s.tableLevel(s.sum(rules.HealthTable, form))) +
		10*s.sum(rules.HealthTens, form))
	s.Endurance.set(base + rules.ToughBonus(s.tableLevel(s.sum(rules.EnduranceTable, form))) +
		10*s.sum(rules.EnduranceTens, form))
	s.HasReserve = s.present(rules.Reserve, form)
	if s.HasReserve {
		s.Reserve.set(rules.ReserveBonus(s.tableLevel(s.sum(rules.Reserve, form))))
	} else {
		s.Reserve.set(0)
	}
}

// halfTVLevel is half of a marked Ability's level, rounded up (p. 124).
func halfTVLevel(t *Trait) int {
	if t.HalfTV && t.Kind == rules.Ability && t.Level > 0 && !t.has(rules.NoTV) {
		return (t.Level + 1) / 2
	}
	return 0
}

// computeTV: TV (p. 124), the best ROLL, Defense and DX, without the
// Perks and Flaws that add to them each time (Attack.TVRoll, TVDX), +
// Armored + the signed levels that change Health and Endurance + half of
// each marked Ability's; a Companion's 4 less, at least 1. With no
// attack, the plain one: 2 + Agile + Combat Expert - Clumsy, 1 + Attack
// + Strong - Weak.
func (s *Sheet) computeTV() {
	form := s.Transformed
	any := false
	roll, dx := 0, 0
	for _, a := range s.Attacks {
		if a.InTV {
			if !any || a.TVRoll > roll {
				roll = a.TVRoll
			}
			if !any || a.TVDX > dx {
				dx = a.TVDX
			}
			any = true
		}
	}
	if !any {
		roll = 2 + s.sum(rules.Agility, form) + s.sum(rules.CombatSkill, form)
		dx = 1 + s.sum(rules.Damage, form) + s.sum(rules.Strength, form)
		if dx < 0 {
			dx = 0
		}
	}
	def := s.Defense.Get()
	if s.HasBarrier && s.Barrier.Get() > def {
		def = s.Barrier.Get()
	}
	tv := roll + def + dx + s.sum(rules.Armor, form) +
		s.sum(rules.HealthTable, form) + s.sum(rules.HealthTens, form) +
		s.sum(rules.EnduranceTable, form) + s.sum(rules.EnduranceTens, form)
	for _, t := range s.Traits {
		tv += halfTVLevel(t)
		if form {
			for _, f := range t.Form {
				tv += halfTVLevel(f)
			}
		}
	}
	if s.Kind == CompanionSheet {
		tv -= companionTV
		if tv < 1 {
			tv = 1
		}
	}
	s.TV.set(tv)
}

// ---- The totals and the limits (p. 21) ----

// scaled is Scaled Cost's points for a level's size (p. 21): 1, 2, 4,
// 8, 15.
func scaled(size int) int {
	switch {
	case size <= 0:
		return 0
	case size >= maxLevel:
		return 15
	}
	return 1 << (size - 1)
}

// halfUp is n halved, rounded away from 0: Knowledge 3 counts 2,
// Ineptitude -3 counts -2 (p. 57, "round up").
func halfUp(n int) int {
	if n >= 0 {
		return (n + 1) / 2
	}
	return -((1 - n) / 2)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// computeTotals: the totals of the character's own Traits (not a
// form's, which Transformation's level pays for, p. 63), then the limits
// on them.
func (s *Sheet) computeTotals() {
	o := &s.Options
	half := o.HalfCostKnowledge && (o.BaseZero || o.PowerCeiling || o.ScaledCost)
	abilities, weaknesses := 0, 0
	s.AbilityPoints, s.WeaknessPoints = 0, 0
	for _, t := range s.Traits {
		lv, pts := t.Level, scaled(abs(t.Level))
		if half && t.has(rules.HalfCost) {
			lv, pts = halfUp(lv), halfUp(pts)
		}
		if t.Kind == rules.Ability {
			abilities += lv
			s.AbilityPoints += pts
		} else {
			weaknesses += lv
			s.WeaknessPoints += pts
		}
	}
	s.AbilityLevels.set(abilities)
	s.WeaknessLevels.set(weaknesses)
	s.Total.set(s.AbilityLevels.Get() + s.WeaknessLevels.Get())
	s.Points.set(s.AbilityPoints - s.WeaknessPoints)
	var measure, spent int
	if o.ScaledCost {
		measure, spent = s.Points.Get(), s.AbilityPoints
	} else {
		measure, spent = s.Total.Get(), s.AbilityLevels.Get()
	}
	if o.BaseZero && abs(measure) > o.BaseZeroRange {
		s.warn(BaseZeroBroken, nil, nil, nil, nil)
	}
	if o.PowerCeiling && spent > o.Ceiling {
		s.warn(CeilingBroken, nil, nil, nil, nil)
	}
	if o.StartingLevels {
		if o.ScaledCost {
			measure = s.WeaknessPoints
		} else {
			measure = -s.WeaknessLevels.Get()
		}
		if spent > o.Starting+measure {
			s.warn(StartingBroken, nil, nil, nil, nil)
		}
	}
}

// Compute sets every number of s and of its partners' sheets, and their
// warnings.
func Compute(s *Sheet) {
	s.Warnings = nil
	for _, t := range s.Traits {
		s.computeTrait(t)
		for _, f := range t.Form {
			s.computeTrait(f)
		}
		if t.Partner != nil {
			Compute(t.Partner)
		}
	}
	s.computeTotals()
	for _, a := range s.Attacks {
		s.computeAttack(a)
	}
	s.computeDefense()
	s.computeHealth()
	s.computeTV()
}
