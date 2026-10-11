// Package rules holds the rulebook's names, pages and numbers: no
// descriptions.
//
// Every Ability, Weakness, Perk and Flaw of the OVA rulebook, in the
// book's order (alphabetical within each kind), with the page it is
// described on and what the calculations need of it; and the rulebook's
// small tables (Tough, Vigorous, Endurance Reserve, Endurance Use, spell
// costs). The rulebook's text stays out (AGENTS.md): to explain an item
// the program opens the book at its page.
//
// Pages are the book's printed numbers, which are also the PDF's page
// labels; the PDF's page index is the printed page + PdfPageOffset. The
// pages were checked against the PDF by the font of the entries'
// headings (tools/rulebook-headings.py, make check-pages), as po_ova_cg's
// were (its PLAN.md, Phase 1).
//
// The port of po_ova_cg's OvaRules. Imports nothing of the window or the
// files, so its tests need no display.
package rules

import (
	"math"
	"strconv"
)

// Kind is what an item is.
type Kind int

const (
	Ability Kind = iota
	Weakness
	Perk
	Flaw
)

// PdfPageOffset is the PDF page index less the printed page.
const PdfPageOffset = 2

// Flags is a set of Flag.
type Flags uint32

// Flag is something an item does to the sheet.
type Flag uint

// What an Ability or Weakness does to the sheet. Its signed level is
// what it adds: Agile +2 adds 2 dice, Clumsy -2 takes 2.
const (
	Agility        Flag = iota // Attack Rolls, but a Will Attack's: Agile, Clumsy (p. 105, 86)
	CombatSkill                // Attack Rolls, a Will Attack's too: Combat Expert
	Will                       // a Will Attack's roll, in Agility's place: Iron-Willed, Weak-Willed
	DefenseRoll                // Defense Rolls: Evasive, Quick, Slow (p. 105)
	Damage                     // the Damage Multiplier: Attack (p. 106)
	Strength                   // the Damage Multiplier, but a Ranged attack's: Strong, Weak (p. 85)
	HealthTable                // Health + the Tough table (p. 63)
	HealthTens                 // Health - 10 a level: Frail (p. 71)
	EnduranceTable             // Endurance + the Vigorous table (p. 65)
	EnduranceTens              // Endurance - 10 a level: Languid (p. 73)
	Armor                      // Armored: in TV (p. 124)
	SecondDefense              // a second Defense Roll, 2 + its level: Barrier (p. 51)
	Telekinetic                // its level adds to an attack made with it: Telekinesis (p. 62)
	HalfTV                     // adds half its level to TV unless unmarked: Healer, Magic (p. 124)
	NoTV                       // never in TV, which is its own: Companion, Minions (p. 124)
	HalfCost                   // may cost half under the limits: Knowledge, Ineptitude (p. 57)
	Reserve                    // a separate store of Endurance, 20 a level: Endurance Reserve (p. 53)
	CostUp                     // raises one Ability's Endurance cost: Endurance Use (p. 70)
	OwnCost                    // spends Endurance of its own as it is used, so a spell of it
	//                            costs more than the table's, "10+": Barrier (p. 51, 58)
	CastsWeaknesses // its spells cast Weaknesses, -1 to -3, at Arcane's costs: Witchcraft (p. 58)
	ThirdDefense    // another alternate Defense Roll, 2 + its level, written after
	//                 Barrier's: Incorporeal Form, its dice as the Defense Roll (p. 55)
)

// What a Perk or Flaw does to an attack's numbers, after the Abilities'
// and Weaknesses' flags, with room for more of those.
const (
	RollEach       Flag = 24 + iota // +1 die a time (Accurate), or -1 (Inaccurate)
	DamageEach                      // +1 DX a time (Effective), or -1 (Ineffective)
	RangedAttack                    // drops Strong and Weak (Ranged, p. 85)
	StrengthRanged                  // keeps them (Ranged, Strength-Powered, p. 86)
	WillAttack                      // Iron-Willed and Weak-Willed for Agile and Clumsy (p. 86)
	NoDamageDone                    // DX 0 (No Damage, p. 88)
)

// Has reports whether f is in the set.
func (s Flags) Has(f Flag) bool { return s&(1<<f) != 0 }

func flags(fs ...Flag) Flags {
	var s Flags
	for _, f := range fs {
		s |= 1 << f
	}
	return s
}

// Link is what linked content an Ability has.
type Link int

const (
	NoLink  Link = iota
	Spells       // a Magic's spells
	Form         // a Transformation's form
	Partner      // a Companion's sheet
	Machine      // a Vehicle's sheet
)

// CostKind is how a Perk's or Flaw's cost is set.
type CostKind int

const (
	FixedCost CostKind = iota // Cost, each time it is taken
	RangeCost                 // chosen from MinCost..MaxCost, in steps of 5 (Cancel, Requirement)
	UserCost                  // any the user and GM agree on (Unique Perk, Unique Flaw)
)

// Unlimited is MaxTimes of a Perk or Flaw taken as often as wanted.
const Unlimited = math.MaxInt32

// Pages of the rules themselves, for the window's "?" buttons.
const (
	LimitsPage          = 21 // Limiting the Character
	PerksFlawsPage      = 83 // Ability Perks and Flaws: how they cost
	AffinitiesPage      = 89
	QuickListPage       = 90
	SpellCostPage       = 58 // the Spell Endurance Cost table
	RollsPage           = 105
	DamagePage          = 106
	HealthEndurancePage = 107
	ThreatValuePage     = 124
	HalfCostPage        = 57 // Knowledge, half cost
	// Where the NPC types, Heroic, Secondary and Extra (ova_fmt's
	// npc-type:), are described: their Health and Endurance; TV's step
	// for them is on p. 124.
	NpcTypesPage = 123
)

// Item is an Ability, Weakness, Perk or Flaw. Read-only after init.
type Item struct {
	Name   string
	Kind   Kind
	Page   int
	Flags  Flags
	Linked Link // an Ability's linked content
	// A Perk's or Flaw's Endurance cost, each time it is taken
	// (negative for a Flaw); for a RangeCost one, MinCost..MaxCost; and
	// how often it may be taken.
	CostKind                         CostKind
	Cost, MinCost, MaxCost, MaxTimes int
}

// Has reports whether the item has flag f.
func (it *Item) Has(f Flag) bool { return it != nil && it.Flags.Has(f) }

var (
	items   []*Item
	byKind  [4][]*Item
	aliases = map[string]*Item{} // folded alias -> item
	// spellCost[magic-1][ability-1]
	spellCost = [5][5]int{
		{20, 30, 40, 50, 60},
		{10, 20, 30, 40, 50},
		{5, 10, 20, 30, 40},
		{2, 5, 10, 20, 30},
		{0, 2, 5, 10, 20},
	}
)

func add(name string, kind Kind, page int, fs Flags) *Item {
	it := &Item{Name: name, Kind: kind, Page: page, Flags: fs, MaxTimes: 1}
	items = append(items, it)
	byKind[kind] = append(byKind[kind], it)
	return it
}

func a(name string, page int, fs ...Flag) { add(name, Ability, page, flags(fs...)) }

func l(name string, page int, link Link, fs ...Flag) {
	add(name, Ability, page, flags(fs...)).Linked = link
}

func w(name string, page int, fs ...Flag) { add(name, Weakness, page, flags(fs...)) }

// pf is a Perk or Flaw of fixed cost, taken at most maxTimes.
func pf(name string, kind Kind, page, cost, maxTimes int, fs ...Flag) {
	it := add(name, kind, page, flags(fs...))
	it.Cost = cost
	it.MaxTimes = maxTimes
}

func rangeFlaw(name string, page, min, max int) {
	it := add(name, Flaw, page, 0)
	it.CostKind = RangeCost
	it.MinCost, it.MaxCost = min, max
}

func unique(name string, kind Kind, page int) {
	add(name, kind, page, 0).CostKind = UserCost
}

func alias(alias, name string) {
	for _, it := range items {
		if it.Name == name {
			aliases[fold(alias)] = it
			return
		}
	}
	panic("rules: no item " + name)
}

func init() {
	a("Agile", 49, Agility)
	a("Armored", 49, Armor)
	a("Art of Invisibility", 50)
	a("Attack", 50, Damage)
	a("Barrier", 51, SecondDefense, OwnCost)
	a("Beautiful", 51)
	a("Charismatic", 51)
	a("Combat Expert", 51, CombatSkill)
	l("Companion", 52, Partner, NoTV)
	a("Connected", 52)
	a("Cute!", 52)
	a("Dexterous", 52)
	a("Dimensional Pocket", 52)
	a("Endurance Reserve", 53, Reserve)
	a("Evasive", 53, DefenseRoll)
	a("Famous", 53)
	a("Flight", 54)
	a("Gear", 54)
	a("Healer", 54, HalfTV)
	a("Heightened Sense", 54)
	a("Illusionist", 54)
	a("Incorporeal Form", 55, ThirdDefense)
	a("Intimidating", 55)
	a("Intuitive", 56)
	a("Inventor", 56)
	a("Invisibility", 56)
	a("Iron-Willed", 56, Will)
	a("Knowledge", 57, HalfCost)
	a("Life Support", 57)
	a("Lucky", 57)
	l("Magic, Arcane", 57, Spells, HalfTV)
	l("Magic, Witchcraft", 58, Spells, HalfTV, CastsWeaknesses)
	a("Minions", 58, NoTV)
	a("Passion", 59)
	a("Perceptive", 59)
	a("Performer", 59)
	a("Pilot", 59)
	a("Position of Power", 59)
	a("Psychic", 59)
	a("Quick", 60, DefenseRoll)
	a("Resistance", 60)
	a("Shape-Shifter", 60)
	a("Sixth Sense", 61)
	a("Smart", 61)
	a("Spirit Medium", 61)
	a("Strong", 62, Strength)
	a("Telekinesis", 62, Telekinetic)
	a("Teleport", 62)
	a("Time Freeze", 63)
	a("Tough", 63, HealthTable)
	l("Transformation", 63, Form)
	a("Unique Ability", 64)
	l("Vehicle (Mecha)", 64, Machine)
	a("Vigorous", 65, EnduranceTable)
	a("Walking Arsenal", 65)
	a("Wealthy", 65)

	w("Absent-Minded", 66)
	w("Accidental Transformation", 66)
	w("Ageism", 66)
	w("Airhead", 66)
	w("Amnesia", 67)
	w("Arrogant", 67)
	w("Awkward Size", 67)
	w("Bizarre Appearance", 68)
	w("Clumsy", 68, Agility)
	w("Code of Conduct", 68)
	w("Compulsion", 69)
	w("Coward", 69)
	w("Crybaby", 69)
	w("Dense", 69)
	w("Dependency", 70)
	w("Dull", 70)
	w("Easily Distracted", 70)
	w("Emotionless", 70)
	w("Endurance Use", 70, CostUp)
	w("Fear", 70)
	w("Focus", 71)
	w("Frail", 71, HealthTens)
	w("Fussy", 71)
	w("Greedy", 71)
	w("Guardian", 72)
	w("Hatred", 72)
	w("Impaired Sense", 72)
	w("Impulsive", 72)
	w("Ineptitude", 72, HalfCost)
	w("Infamous", 73)
	w("Jittery", 73)
	w("Kind-Hearted", 73)
	w("Languid", 73, EnduranceTens)
	w("Lecherous", 73)
	w("Limited Uses", 74)
	w("Loner", 74)
	w("Love Interest", 74)
	w("Love Magnet", 75)
	w("Mute", 75)
	w("Naive", 75)
	w("Nosebleeder", 75)
	w("Oblivious", 76)
	w("Obsession", 76)
	w("Outcast", 76)
	w("Overconfident", 76)
	w("Pacifist", 77)
	w("Poor", 77)
	w("Quirk", 77)
	w("Restricted Freedom", 77)
	w("Rival", 77)
	w("Room for Two", 78)
	w("Rude", 78)
	w("Secret", 78)
	w("Sensitivity", 78)
	w("Servitude", 78)
	w("Short-Tempered", 79)
	w("Shy", 80)
	w("Slow", 80, DefenseRoll)
	w("Soft Spot", 80)
	w("Split Personality", 80)
	w("Stubborn", 80)
	w("Suppressed Power", 81)
	w("Trigger", 81)
	w("Unique Weakness", 81)
	w("Unlucky", 81)
	w("Vulnerability", 82)
	w("Wanted", 82)
	w("Weak", 82, Strength)
	w("Weak-Willed", 82, Will)

	// Costs from the Quick List (p. 90) and the headings; how often each
	// may be taken from its own text (po_ova_cg's PLAN.md, "Perk and
	// Flaw costs").
	pf("Accurate", Perk, 83, 5, Unlimited, RollEach)
	pf("Area Effect", Perk, 83, 10, Unlimited)
	pf("Armor Piercing", Perk, 83, 5, Unlimited)
	pf("Barrier Buster", Perk, 83, 5, 1)
	pf("Blinding", Perk, 84, 20, 1)
	pf("Continued Effect", Perk, 84, 10, 2)
	pf("Defensive", Perk, 84, 5, Unlimited)
	pf("Disarming", Perk, 84, 10, 1)
	pf("Effective", Perk, 84, 5, Unlimited, DamageEach)
	pf("Fatiguing", Perk, 84, 0, 1)
	pf("Impairing", Perk, 85, 10, 1)
	pf("Multiple Targets", Perk, 85, 10, Unlimited)
	pf("No Gesture", Perk, 85, 5, 1)
	pf("Overwhelming", Perk, 85, 15, 1)
	pf("Paralyzing", Perk, 85, 10, 2)
	pf("Ranged", Perk, 85, 0, 1, RangedAttack)
	pf("Ranged, Strength-Powered", Perk, 86, 5, 1, StrengthRanged)
	pf("Reach", Perk, 86, 5, Unlimited)
	pf("Redirectable", Perk, 86, 5, 1)
	pf("Strike Through", Perk, 86, 5, 1)
	pf("Stunning", Perk, 86, 10, 1)
	pf("Trap", Perk, 86, 5, 1)
	unique("Unique Perk", Perk, 86)
	pf("Vampire", Perk, 86, 5, Unlimited)
	pf("Will Attack", Perk, 86, 0, 1, WillAttack)

	pf("Activation", Flaw, 87, -10, 1)
	pf("Ammunition", Flaw, 87, -5, 3)
	pf("Assisted", Flaw, 87, -15, Unlimited)
	pf("Backlash", Flaw, 87, -5, Unlimited)
	pf("Break", Flaw, 87, -10, 1)
	pf("Buildup", Flaw, 87, -5, Unlimited)
	rangeFlaw("Cancel", 87, -20, 0)
	pf("Concentration", Flaw, 87, -20, Unlimited)
	pf("Delayed", Flaw, 88, -5, 1)
	pf("Elaborate Gestures", Flaw, 88, -5, 1)
	pf("Feather Blow", Flaw, 88, -5, 1)
	pf("Finisher", Flaw, 88, -10, 1)
	pf("Inaccurate", Flaw, 88, -5, Unlimited, RollEach)
	pf("Ineffective", Flaw, 88, -5, Unlimited, DamageEach)
	pf("Low Penetration", Flaw, 88, -5, Unlimited)
	pf("No Damage", Flaw, 88, -20, 1, NoDamageDone)
	pf("Open to Attack", Flaw, 88, -25, 1)
	pf("Recoil", Flaw, 88, -10, 1)
	rangeFlaw("Requirement", 88, -20, -5)
	pf("Self-Only", Flaw, 88, -5, 1)
	unique("Unique Flaw", Flaw, 88)
	pf("Unwieldy", Flaw, 89, -5, Unlimited)
	pf("Weapon", Flaw, 89, -5, 1)

	alias("Vehicle", "Vehicle (Mecha)")
	alias("Ranged, Str.-Powered", "Ranged, Strength-Powered") // the Quick List's (p. 90)
}

// fold is s in lower case, ASCII only: names are ASCII.
func fold(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// Count is how many items of a kind there are.
func Count(k Kind) int { return len(byKind[k]) }

// Get is the i'th item of a kind, 0 <= i < Count(k), in the book's order.
func Get(k Kind, i int) *Item { return byKind[k][i] }

// Index is it's place in its kind's order; -1 for nil.
func Index(it *Item) int {
	if it == nil {
		return -1
	}
	for i, x := range byKind[it.Kind] {
		if x == it {
			return i
		}
	}
	return -1
}

// Find is the item named name, ignoring case, or by an alias ("Vehicle"
// for "Vehicle (Mecha)"); nil if there is none. Names are unique across
// the four kinds.
func Find(name string) *Item {
	f := fold(name)
	for _, it := range items {
		if fold(it.Name) == f {
			return it
		}
	}
	return aliases[f]
}

// PageText is "p. 50"; "p. ?" for nil, an item the tables don't have.
func PageText(it *Item) string {
	if it == nil {
		return "p. ?"
	}
	return "p. " + strconv.Itoa(it.Page)
}

// Label is "Attack (p. 50)": an item's name with its page, as every
// name is shown.
func Label(it *Item) string { return it.Name + " (" + PageText(it) + ")" }

// PageLabel is "Secondary (p. 123)": a name that isn't one of the
// items', with the page it is described on.
func PageLabel(name string, page int) string { return name + " (p. " + strconv.Itoa(page) + ")" }

// LabelOf is name's label, as written, if the tables have it: "Magic,
// Testing (p. ?)" if they don't.
func LabelOf(name string) string {
	if it := Find(name); it != nil {
		return Label(it)
	}
	return name + " (" + PageText(nil) + ")"
}

// NameLabel is it's label, or name's with "(p. ?)" when it is nil: an
// item from a file the tables don't have keeps the file's name.
func NameLabel(it *Item, name string) string {
	if it != nil {
		return Label(it)
	}
	return name + " (" + PageText(nil) + ")"
}

// PdfIndex is the PDF's page index of a printed page.
func PdfIndex(page int) int { return page + PdfPageOffset }

// ToughBonus is Tough's bonus to Health (p. 63), and Vigorous's to
// Endurance (p. 65), the same table: 10, 20, 30, 40, 60 for +1 to +5;
// 0 for 0.
func ToughBonus(level int) int {
	if level < 0 || level > 5 {
		panic("rules: ToughBonus level out of range")
	}
	if level == 5 {
		return 60
	}
	return 10 * level
}

// ReserveBonus is Endurance Reserve's store (p. 53): 20 a level.
func ReserveBonus(level int) int {
	if level < 0 || level > 5 {
		panic("rules: ReserveBonus level out of range")
	}
	return 20 * level
}

// EnduranceUseCost is what Endurance Use adds to an Ability's cost
// (p. 70): 5, 10, 20 at -1, -2, -3; 0 at 0.
func EnduranceUseCost(level int) int {
	switch level {
	case 0:
		return 0
	case -1:
		return 5
	case -2:
		return 10
	case -3:
		return 20
	}
	panic("rules: EnduranceUseCost level out of range")
}

// SpellCost is the Spell Endurance Cost table (p. 58): casting a spell
// that mimics an Ability of level ability, at Magic level magic, both
// 1..5.
func SpellCost(magic, ability int) int {
	if magic < 1 || magic > 5 || ability < 1 || ability > 5 {
		panic("rules: SpellCost level out of range")
	}
	return spellCost[magic-1][ability-1]
}
