package rules

import "testing"

// The Quick List (p. 90) is a source independent of the entries'
// headings, from which the pages were taken: every name on it must be
// here, of the same kind, at the same place in the book's order, with
// the cost it gives. Its entries were generated from the PDF's text of
// p. 90 (po_ova_cg's PLAN.md, Phase 1), not typed.
var quickList = []struct {
	kind     Kind
	name     string
	costKind CostKind
	cost     int
	max      int
}{
	{Ability, "Agile", FixedCost, 0, 0},
	{Ability, "Armored", FixedCost, 0, 0},
	{Ability, "Art of Invisibility", FixedCost, 0, 0},
	{Ability, "Attack", FixedCost, 0, 0},
	{Ability, "Barrier", FixedCost, 0, 0},
	{Ability, "Beautiful", FixedCost, 0, 0},
	{Ability, "Charismatic", FixedCost, 0, 0},
	{Ability, "Combat Expert", FixedCost, 0, 0},
	{Ability, "Companion", FixedCost, 0, 0},
	{Ability, "Connected", FixedCost, 0, 0},
	{Ability, "Cute!", FixedCost, 0, 0},
	{Ability, "Dexterous", FixedCost, 0, 0},
	{Ability, "Dimensional Pocket", FixedCost, 0, 0},
	{Ability, "Endurance Reserve", FixedCost, 0, 0},
	{Ability, "Evasive", FixedCost, 0, 0},
	{Ability, "Famous", FixedCost, 0, 0},
	{Ability, "Flight", FixedCost, 0, 0},
	{Ability, "Gear", FixedCost, 0, 0},
	{Ability, "Healer", FixedCost, 0, 0},
	{Ability, "Heightened Sense", FixedCost, 0, 0},
	{Ability, "Illusionist", FixedCost, 0, 0},
	{Ability, "Incorporeal Form", FixedCost, 0, 0},
	{Ability, "Intimidating", FixedCost, 0, 0},
	{Ability, "Intuitive", FixedCost, 0, 0},
	{Ability, "Inventor", FixedCost, 0, 0},
	{Ability, "Invisibility", FixedCost, 0, 0},
	{Ability, "Iron-Willed", FixedCost, 0, 0},
	{Ability, "Knowledge", FixedCost, 0, 0},
	{Ability, "Life Support", FixedCost, 0, 0},
	{Ability, "Lucky", FixedCost, 0, 0},
	{Ability, "Magic, Arcane", FixedCost, 0, 0},
	{Ability, "Magic, Witchcraft", FixedCost, 0, 0},
	{Ability, "Minions", FixedCost, 0, 0},
	{Ability, "Passion", FixedCost, 0, 0},
	{Ability, "Perceptive", FixedCost, 0, 0},
	{Ability, "Performer", FixedCost, 0, 0},
	{Ability, "Pilot", FixedCost, 0, 0},
	{Ability, "Position of Power", FixedCost, 0, 0},
	{Ability, "Psychic", FixedCost, 0, 0},
	{Ability, "Quick", FixedCost, 0, 0},
	{Ability, "Resistance", FixedCost, 0, 0},
	{Ability, "Shape-Shifter", FixedCost, 0, 0},
	{Ability, "Sixth Sense", FixedCost, 0, 0},
	{Ability, "Smart", FixedCost, 0, 0},
	{Ability, "Spirit Medium", FixedCost, 0, 0},
	{Ability, "Strong", FixedCost, 0, 0},
	{Ability, "Telekinesis", FixedCost, 0, 0},
	{Ability, "Teleport", FixedCost, 0, 0},
	{Ability, "Time Freeze", FixedCost, 0, 0},
	{Ability, "Tough", FixedCost, 0, 0},
	{Ability, "Transformation", FixedCost, 0, 0},
	{Ability, "Unique Ability", FixedCost, 0, 0},
	{Ability, "Vehicle (Mecha)", FixedCost, 0, 0},
	{Ability, "Vigorous", FixedCost, 0, 0},
	{Ability, "Walking Arsenal", FixedCost, 0, 0},
	{Ability, "Wealthy", FixedCost, 0, 0},
	{Weakness, "Absent-Minded", FixedCost, 0, 0},
	{Weakness, "Accidental Transformation", FixedCost, 0, 0},
	{Weakness, "Ageism", FixedCost, 0, 0},
	{Weakness, "Airhead", FixedCost, 0, 0},
	{Weakness, "Amnesia", FixedCost, 0, 0},
	{Weakness, "Arrogant", FixedCost, 0, 0},
	{Weakness, "Awkward Size", FixedCost, 0, 0},
	{Weakness, "Bizarre Appearance", FixedCost, 0, 0},
	{Weakness, "Clumsy", FixedCost, 0, 0},
	{Weakness, "Code of Conduct", FixedCost, 0, 0},
	{Weakness, "Compulsion", FixedCost, 0, 0},
	{Weakness, "Coward", FixedCost, 0, 0},
	{Weakness, "Crybaby", FixedCost, 0, 0},
	{Weakness, "Dense", FixedCost, 0, 0},
	{Weakness, "Dependency", FixedCost, 0, 0},
	{Weakness, "Dull", FixedCost, 0, 0},
	{Weakness, "Easily Distracted", FixedCost, 0, 0},
	{Weakness, "Emotionless", FixedCost, 0, 0},
	{Weakness, "Endurance Use", FixedCost, 0, 0},
	{Weakness, "Fear", FixedCost, 0, 0},
	{Weakness, "Focus", FixedCost, 0, 0},
	{Weakness, "Frail", FixedCost, 0, 0},
	{Weakness, "Fussy", FixedCost, 0, 0},
	{Weakness, "Greedy", FixedCost, 0, 0},
	{Weakness, "Guardian", FixedCost, 0, 0},
	{Weakness, "Hatred", FixedCost, 0, 0},
	{Weakness, "Impaired Sense", FixedCost, 0, 0},
	{Weakness, "Impulsive", FixedCost, 0, 0},
	{Weakness, "Ineptitude", FixedCost, 0, 0},
	{Weakness, "Infamous", FixedCost, 0, 0},
	{Weakness, "Jittery", FixedCost, 0, 0},
	{Weakness, "Kind-Hearted", FixedCost, 0, 0},
	{Weakness, "Languid", FixedCost, 0, 0},
	{Weakness, "Lecherous", FixedCost, 0, 0},
	{Weakness, "Limited Uses", FixedCost, 0, 0},
	{Weakness, "Loner", FixedCost, 0, 0},
	{Weakness, "Love Interest", FixedCost, 0, 0},
	{Weakness, "Love Magnet", FixedCost, 0, 0},
	{Weakness, "Mute", FixedCost, 0, 0},
	{Weakness, "Naive", FixedCost, 0, 0},
	{Weakness, "Nosebleeder", FixedCost, 0, 0},
	{Weakness, "Oblivious", FixedCost, 0, 0},
	{Weakness, "Obsession", FixedCost, 0, 0},
	{Weakness, "Outcast", FixedCost, 0, 0},
	{Weakness, "Overconfident", FixedCost, 0, 0},
	{Weakness, "Pacifist", FixedCost, 0, 0},
	{Weakness, "Poor", FixedCost, 0, 0},
	{Weakness, "Quirk", FixedCost, 0, 0},
	{Weakness, "Restricted Freedom", FixedCost, 0, 0},
	{Weakness, "Rival", FixedCost, 0, 0},
	{Weakness, "Room for Two", FixedCost, 0, 0},
	{Weakness, "Rude", FixedCost, 0, 0},
	{Weakness, "Secret", FixedCost, 0, 0},
	{Weakness, "Sensitivity", FixedCost, 0, 0},
	{Weakness, "Servitude", FixedCost, 0, 0},
	{Weakness, "Short-Tempered", FixedCost, 0, 0},
	{Weakness, "Shy", FixedCost, 0, 0},
	{Weakness, "Slow", FixedCost, 0, 0},
	{Weakness, "Soft Spot", FixedCost, 0, 0},
	{Weakness, "Split Personality", FixedCost, 0, 0},
	{Weakness, "Stubborn", FixedCost, 0, 0},
	{Weakness, "Suppressed Power", FixedCost, 0, 0},
	{Weakness, "Trigger", FixedCost, 0, 0},
	{Weakness, "Unique Weakness", FixedCost, 0, 0},
	{Weakness, "Unlucky", FixedCost, 0, 0},
	{Weakness, "Vulnerability", FixedCost, 0, 0},
	{Weakness, "Wanted", FixedCost, 0, 0},
	{Weakness, "Weak", FixedCost, 0, 0},
	{Weakness, "Weak-Willed", FixedCost, 0, 0},
	{Perk, "Accurate", FixedCost, 5, 0},
	{Perk, "Area Effect", FixedCost, 10, 0},
	{Perk, "Armor Piercing", FixedCost, 5, 0},
	{Perk, "Barrier Buster", FixedCost, 5, 0},
	{Perk, "Blinding", FixedCost, 20, 0},
	{Perk, "Continued Effect", FixedCost, 10, 0},
	{Perk, "Defensive", FixedCost, 5, 0},
	{Perk, "Disarming", FixedCost, 10, 0},
	{Perk, "Effective", FixedCost, 5, 0},
	{Perk, "Fatiguing", FixedCost, 0, 0},
	{Perk, "Impairing", FixedCost, 10, 0},
	{Perk, "Multiple Targets", FixedCost, 10, 0},
	{Perk, "No Gesture", FixedCost, 5, 0},
	{Perk, "Overwhelming", FixedCost, 15, 0},
	{Perk, "Paralyzing", FixedCost, 10, 0},
	{Perk, "Ranged", FixedCost, 0, 0},
	{Perk, "Ranged, Str.-Powered", FixedCost, 5, 0},
	{Perk, "Reach", FixedCost, 5, 0},
	{Perk, "Redirectable", FixedCost, 5, 0},
	{Perk, "Strike Through", FixedCost, 5, 0},
	{Perk, "Stunning", FixedCost, 10, 0},
	{Perk, "Trap", FixedCost, 5, 0},
	{Perk, "Unique Perk", UserCost, 0, 0},
	{Perk, "Vampire", FixedCost, 5, 0},
	{Perk, "Will Attack", FixedCost, 0, 0},
	{Flaw, "Activation", FixedCost, -10, 0},
	{Flaw, "Ammunition", FixedCost, -5, 0},
	{Flaw, "Assisted", FixedCost, -15, 0},
	{Flaw, "Backlash", FixedCost, -5, 0},
	{Flaw, "Break", FixedCost, -10, 0},
	{Flaw, "Buildup", FixedCost, -5, 0},
	{Flaw, "Cancel", RangeCost, -20, 0},
	{Flaw, "Concentration", FixedCost, -20, 0},
	{Flaw, "Delayed", FixedCost, -5, 0},
	{Flaw, "Elaborate Gestures", FixedCost, -5, 0},
	{Flaw, "Feather Blow", FixedCost, -5, 0},
	{Flaw, "Finisher", FixedCost, -10, 0},
	{Flaw, "Inaccurate", FixedCost, -5, 0},
	{Flaw, "Ineffective", FixedCost, -5, 0},
	{Flaw, "Low Penetration", FixedCost, -5, 0},
	{Flaw, "No Damage", FixedCost, -20, 0},
	{Flaw, "Open to Attack", FixedCost, -25, 0},
	{Flaw, "Recoil", FixedCost, -10, 0},
	{Flaw, "Requirement", RangeCost, -20, -5},
	{Flaw, "Self-Only", FixedCost, -5, 0},
	{Flaw, "Unique Flaw", UserCost, 0, 0},
	{Flaw, "Unwieldy", FixedCost, -5, 0},
	{Flaw, "Weapon", FixedCost, -5, 0},
}

func TestQuickList(t *testing.T) {
	var next [4]int
	for _, q := range quickList {
		it := Find(q.name)
		ok := it != nil && it.Kind == q.kind && next[q.kind] < Count(q.kind) && Get(q.kind, next[q.kind]) == it
		if ok && q.kind >= Perk {
			ok = it.CostKind == q.costKind
			if ok && q.costKind == FixedCost {
				ok = it.Cost == q.cost
			} else if ok && q.costKind == RangeCost {
				ok = it.MinCost == q.cost && it.MaxCost == q.max
			}
		}
		if !ok {
			t.Errorf("the Quick List's %s doesn't match", q.name)
		}
		next[q.kind]++
	}
	for k := Ability; k <= Flaw; k++ {
		if next[k] != Count(k) {
			t.Errorf("kind %d: %d on the Quick List, %d here: nothing beyond the Quick List", k, next[k], Count(k))
		}
	}
	if Count(Ability) != 56 || Count(Weakness) != 69 || Count(Perk) != 25 || Count(Flaw) != 23 {
		t.Error("56 Abilities, 69 Weaknesses, 25 Perks, 23 Flaws")
	}
}

// pagesIn: each kind's pages lie within its part of the chapter and
// never go back, since the book lists each kind alphabetically.
func pagesIn(k Kind, from, to int) bool {
	last := from
	for i := 0; i < Count(k); i++ {
		it := Get(k, i)
		if it.Page < last || it.Page > to {
			return false
		}
		last = it.Page
	}
	return Get(k, 0).Page == from && Get(k, Count(k)-1).Page == to
}

func pageIs(name string, page int) bool {
	it := Find(name)
	return it != nil && it.Page == page
}

func TestPages(t *testing.T) {
	check := func(ok bool, label string) {
		if !ok {
			t.Error(label)
		}
	}
	check(pagesIn(Ability, 49, 65), "Abilities on pp. 49-65, in order")
	check(pagesIn(Weakness, 66, 82), "Weaknesses on pp. 66-82, in order")
	check(pagesIn(Perk, 83, 86), "Perks on pp. 83-86, in order")
	check(pagesIn(Flaw, 87, 89), "Flaws on pp. 87-89, in order")
	check(pageIs("Attack", 50) && pageIs("Short-Tempered", 79) && pageIs("Weapon", 89) &&
		pageIs("Ammunition", 87) && pageIs("Ranged, Strength-Powered", 86), "pages checked by eye and by hand")
	check(PdfIndex(50) == 52, "the PDF's page index is the printed page + 2")
}

func maxTimes(name string) int {
	if it := Find(name); it != nil {
		return it.MaxTimes
	}
	return -1
}

func linked(name string) Link {
	if it := Find(name); it != nil {
		return it.Linked
	}
	return -1
}

func TestRepeats(t *testing.T) {
	var unlimited, two, three, once int
	for k := Perk; k <= Flaw; k++ {
		for i := 0; i < Count(k); i++ {
			switch Get(k, i).MaxTimes {
			case Unlimited:
				unlimited++
			case 2:
				two++
			case 3:
				three++
			case 1:
				once++
			}
		}
	}
	if unlimited != 16 || two != 2 || three != 1 || once != 29 {
		t.Error("16 repeat freely, 2 at most twice, 1 at most three times, 29 once")
	}
	if maxTimes("Continued Effect") != 2 || maxTimes("Paralyzing") != 2 || maxTimes("Ammunition") != 3 {
		t.Error("the limited ones: Continued Effect, Paralyzing, Ammunition")
	}
	if maxTimes("Low Penetration") != Unlimited || maxTimes("Assisted") != Unlimited || maxTimes("Effective") != Unlimited {
		t.Error("Low Penetration, Assisted, Effective repeat")
	}
	if maxTimes("Cancel") != 1 || maxTimes("Requirement") != 1 || maxTimes("Unique Perk") != 1 {
		t.Error("Cancel, Requirement, Unique Perk are taken once")
	}
}

func TestFind(t *testing.T) {
	if it := Find("attack"); it == nil || it.Name != "Attack" {
		t.Error("Find ignores case")
	}
	if it := Find("Vehicle"); it == nil || it.Name != "Vehicle (Mecha)" {
		t.Error("Find: Vehicle is Vehicle (Mecha)")
	}
	if it := Find("ranged, str.-powered"); it == nil || it.Name != "Ranged, Strength-Powered" {
		t.Error("Find: the Quick List's Ranged, Str.-Powered")
	}
	if Find("Magic, Testing") != nil || Find("") != nil || Find("Attac") != nil || Find("Attacks") != nil {
		t.Error("Find: no near misses")
	}
}

func TestLabels(t *testing.T) {
	if s := Label(Find("Attack")); s != "Attack (p. 50)" {
		t.Errorf("Label: Attack (p. 50) (got %s)", s)
	}
	if s := PageText(Find("Attack")); s != "p. 50" {
		t.Errorf("PageText: p. 50 (got %s)", s)
	}
	if s := PageText(nil); s != "p. ?" {
		t.Errorf("PageText of an unknown item: p. ? (got %s)", s)
	}
	if s := LabelOf("Magic, Testing"); s != "Magic, Testing (p. ?)" {
		t.Errorf("LabelOf an unknown name keeps it, with p. ? (got %s)", s)
	}
	if s := LabelOf("vehicle"); s != "Vehicle (Mecha) (p. 64)" {
		t.Errorf("LabelOf a known name gives the book's name (got %s)", s)
	}
	if s := NameLabel(nil, "Zany"); s != "Zany (p. ?)" {
		t.Errorf("NameLabel of an unknown item (got %s)", s)
	}
}

func TestTables(t *testing.T) {
	if ToughBonus(0) != 0 || ToughBonus(1) != 10 || ToughBonus(4) != 40 || ToughBonus(5) != 60 {
		t.Error("Tough and Vigorous: 10, 20, 30, 40, 60")
	}
	if ReserveBonus(1) != 20 || ReserveBonus(5) != 100 {
		t.Error("Endurance Reserve: 20 a level")
	}
	if EnduranceUseCost(0) != 0 || EnduranceUseCost(-1) != 5 || EnduranceUseCost(-2) != 10 || EnduranceUseCost(-3) != 20 {
		t.Error("Endurance Use: 5, 10, 20")
	}
	rows := [5][5]int{{20, 30, 40, 50, 60}, {10, 20, 30, 40, 50}, {5, 10, 20, 30, 40}, {2, 5, 10, 20, 30}, {0, 2, 5, 10, 20}}
	for m := 1; m <= 5; m++ {
		for a := 1; a <= 5; a++ {
			if SpellCost(m, a) != rows[m-1][a-1] {
				t.Errorf("the Spell Endurance Cost table (p. 58): Magic %d, Ability %d", m, a)
			}
		}
	}
	if SpellCost(3, 2) != 10 {
		t.Error("Auren's Mana Shield: Barrier +2 at Magic 3 costs 10")
	}
}

func has(name string, f Flag) bool { return Find(name).Has(f) }

func TestFlags(t *testing.T) {
	check := func(ok bool, label string) {
		if !ok {
			t.Error(label)
		}
	}
	check(has("Agile", Agility) && has("Clumsy", Agility) && has("Combat Expert", CombatSkill) &&
		has("Iron-Willed", Will) && has("Weak-Willed", Will), "the Attack Roll's Abilities and Weaknesses")
	check(has("Evasive", DefenseRoll) && has("Quick", DefenseRoll) && has("Slow", DefenseRoll) &&
		has("Barrier", SecondDefense), "the Defense Roll's")
	check(has("Attack", Damage) && has("Strong", Strength) && has("Weak", Strength) &&
		has("Telekinesis", Telekinetic), "the Damage Multiplier's")
	check(has("Tough", HealthTable) && has("Frail", HealthTens) && has("Vigorous", EnduranceTable) &&
		has("Languid", EnduranceTens) && has("Endurance Reserve", Reserve), "Health's and Endurance's")
	check(has("Healer", HalfTV) && has("Magic, Arcane", HalfTV) && has("Magic, Witchcraft", HalfTV) &&
		has("Companion", NoTV) && has("Minions", NoTV) && has("Armored", Armor), "TV's")
	check(has("Knowledge", HalfCost) && has("Ineptitude", HalfCost) && has("Endurance Use", CostUp),
		"half cost, and Endurance Use")
	check(has("Accurate", RollEach) && has("Inaccurate", RollEach) && has("Effective", DamageEach) &&
		has("Ineffective", DamageEach) && has("Ranged", RangedAttack) &&
		has("Ranged, Strength-Powered", StrengthRanged) && has("Will Attack", WillAttack) &&
		has("No Damage", NoDamageDone), "the Perks' and Flaws'")
	check(has("Barrier", OwnCost) && !has("Flight", OwnCost), "Barrier spends Endurance of its own")
	check(linked("Transformation") == Form && linked("Companion") == Partner &&
		linked("Magic, Arcane") == Spells && linked("Magic, Witchcraft") == Spells &&
		linked("Vehicle (Mecha)") == Machine && linked("Attack") == NoLink, "linked content")
	check(Index(Find("Agile")) == 0 && Index(Find("Weapon")) == Count(Flaw)-1 && Index(nil) == -1, "Index")
}
