package character

// The book's sample characters (po_ova_cg's TestCalc, Samples12), and
// the editing the window does (TestModel). TestModel's save and load
// is in package save, which imports this one.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tkurtbond/ova_cg/internal/calc"
	"github.com/tkurtbond/ova_cg/internal/entities"
	"github.com/tkurtbond/ova_cg/internal/rules"
)

const dataDir = "../../testdata/"

func checkText(t *testing.T, label string, n *calc.Number, want string) {
	t.Helper()
	if got := n.Text(); got != want {
		t.Errorf("%s %s (got %s)", label, want, got)
	}
}

func checkInt(t *testing.T, label string, n *calc.Number, want int) {
	t.Helper()
	checkText(t, label, n, fmt.Sprint(want))
}

// checkCombat checks the numbers of a combat: block against s's.
func checkCombat(t *testing.T, sample string, c *entities.Combat, s *calc.Sheet) {
	t.Helper()
	for i, x := range c.Attacks {
		a := s.Attacks[i]
		checkInt(t, sample+": "+x.Name+" ROLL", &a.Roll, x.Roll)
		checkText(t, sample+": "+x.Name+" DX", &a.DX, x.DX)
		checkInt(t, sample+": "+x.Name+" END", &a.End, x.Endurance)
	}
	def := s.Defense.Text()
	if s.HasBarrier {
		def += "/" + s.Barrier.Text()
	}
	if def != c.Defense {
		t.Errorf("%s: DEFENSE %s (got %s)", sample, c.Defense, def)
	}
	checkInt(t, sample+": HEALTH", &s.Health, c.Health)
	checkInt(t, sample+": ENDURANCE", &s.Endurance, c.Endurance)
	if c.HasThreatValue {
		checkInt(t, sample+": TV", &s.TV, c.ThreatValue)
	}
}

func checkSubItems(t *testing.T, sample string, l []*entities.SubItem, ts []*calc.Trait) {
	t.Helper()
	for i, x := range l {
		if x.HasEndurance {
			if !ts[i].HasEnd {
				t.Errorf("%s: %s has an END", sample, x.Name)
			}
			checkInt(t, sample+": "+x.Name+" END", &ts[i].End, x.Endurance)
		}
	}
}

// checkEntity checks every number e's file holds against s's.
func checkEntity(t *testing.T, sample string, e *entities.Entity, s *calc.Sheet) {
	t.Helper()
	for i, a := range e.Abilities {
		tr := s.Traits[i]
		if a.HasEndurance {
			checkInt(t, sample+": "+a.Name+" END", &tr.End, a.Endurance)
		}
		checkSubItems(t, sample, a.SubAbilities, tr.Form)
		for j, x := range a.Spells {
			checkText(t, sample+": "+x.Name+" spell END", &tr.Spells[j].End, x.Endurance)
		}
		if se := a.SubEntity; se != nil && se.Combat.HasCombat {
			checkCombat(t, se.Name, &se.Combat, tr.Partner)
			if len(tr.Partner.Warnings) != 0 {
				t.Errorf("%s: no warnings (got %d)", se.Name, len(tr.Partner.Warnings))
			}
		}
	}
	checkCombat(t, sample, &e.Combat, s)
	if len(s.Warnings) != 0 {
		t.Errorf("%s: no warnings (got %d)", sample, len(s.Warnings))
	}
}

func attackIndex(c *entities.Combat, name string) int {
	for i, a := range c.Attacks {
		if a.Name == name {
			return i
		}
	}
	return -1
}

// TestSamples builds each of the twelve sample characters (ova_fmt's
// test-data/, pp. 23-47) and compares every number its file holds with
// calc's. The book's two rulings the rules don't reproduce are checked
// as overrides: the computed number first, then the book's.
func TestSamples(t *testing.T) {
	for _, name := range []string{"Ai", "Auren", "Braun", "Cora", "Daisuke", "Fukiko",
		"Karis", "Miho", "Natsuki", "Raine", "Shou", "Yuu"} {
		t.Run(name, func(t *testing.T) {
			e, n, err := ReadEntity(dataDir + name + ".yaml")
			if err != nil || n != 1 {
				t.Fatalf("loaded: n %d, %v", n, err)
			}
			c := Build(e)
			s := &c.Sheet
			switch name {
			case "Ai":
				ok := len(s.Attacks) > 0
				for _, a := range s.Attacks {
					ok = ok && a.Telekinetic && !a.Transformed
				}
				if !ok {
					t.Error("both attacks made with Telekinesis")
				}
			case "Fukiko":
				if !s.Transformed {
					t.Error("the sheet shown transformed")
				}
			case "Miho":
				i := attackIndex(&e.Combat, "Mode SP: Miho Ray")
				if i < 0 {
					t.Fatal("has Mode SP")
				}
				if !s.Attacks[i].Transformed {
					t.Error("Mode SP made transformed")
				}
				if s.Transformed {
					t.Error("the sheet not transformed")
				}
			case "Braun":
				// The rule: 2 - Clumsy 2 = 0, TV 0 + 2 + 0 = 2; the
				// book's -2 as an override makes TV 0.
				calc.Compute(s)
				checkInt(t, "First Day of Sale Shove! computed ROLL", &s.Attacks[0].Roll, 0)
				checkInt(t, "computed TV", &s.TV, 2)
				s.Attacks[0].Roll.Override(-2, false, false)
			case "Natsuki":
				i := attackIndex(&e.Combat, "Stiletto Kick")
				if i < 0 {
					t.Fatal("has Stiletto Kick")
				}
				calc.Compute(s)
				checkInt(t, "Stiletto Kick computed DX", &s.Attacks[i].DX, 6)
				s.Attacks[i].DX.Override(3, false, false)
			}
			calc.Compute(s)
			checkEntity(t, name, e, s)
		})
	}
}

// TestLoadNotices: Load keeps a file's number that differs from the
// rules' as an override, with a notice; the samples whose numbers are
// all the rules' load with none.
func TestLoadNotices(t *testing.T) {
	for _, name := range []string{"Ai", "Auren", "Braun", "Cora", "Daisuke", "Fukiko", "Karis", "Miho",
		"Natsuki", "Raine", "Shou", "Yuu"} {
		_, notices, err := Load(dataDir + name + ".yaml")
		if err != nil || len(notices) != 0 {
			t.Errorf("%s: loads with no notices: %v %q", name, err, notices)
		}
	}
	// Braun's -2 is listed in its overrides:.
	c, _, err := Load(dataDir + "Braun.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !c.Attacks[0].Roll.Overridden {
		t.Error("Braun: ROLL overridden")
	}
	checkInt(t, "Braun: TV from the override", &c.TV, 0)

	// A number not listed that differs from the rules'.
	b, err := os.ReadFile(dataDir + "Ai.yaml")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "Ai.yaml")
	if err := os.WriteFile(path, []byte(strings.Replace(string(b), "health: 20", "health: 25", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	c, notices, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Ai: HEALTH is 25 in the file, 20 by the rules: kept as an override"}
	if fmt.Sprint(notices) != fmt.Sprint(want) {
		t.Errorf("a changed HEALTH: %q", notices)
	}
	if !c.Health.Overridden || c.Health.Get() != 25 {
		t.Error("and it is an override")
	}

	for name, want := range map[string]string{"Broken": "level", "Fragment": "holds 2 characters",
		"Options": "holds 2 characters"} {
		if _, _, err := Load(dataDir + name + ".yaml"); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: refused, saying %q: %v", name, want, err)
		}
	}
	if _, _, err := Load(dataDir + "missing.yaml"); err == nil ||
		err.Error() != dataDir+"missing.yaml: error: No such file or directory" {
		t.Errorf("missing: %v", err)
	}
}

func item(t *testing.T, name string) *rules.Item {
	t.Helper()
	it := rules.Find(name)
	if it == nil {
		t.Fatalf("no item %q", name)
	}
	return it
}

// traits is the traits' names and details, in order: "Flight;
// Telekinesis; Endurance Use (Gust)".
func traits(list []*calc.Trait) string {
	var r []string
	for _, x := range list {
		t := TraitOf(x)
		s := t.Name
		if t.HasDetail {
			s += " (" + t.Detail + ")"
		}
		r = append(r, s)
	}
	return strings.Join(r, "; ")
}

func modNames(list []*calc.Mod) string {
	var r []string
	for _, m := range list {
		r = append(r, ModOf(m).Name)
	}
	return strings.Join(r, "; ")
}

func is(t *testing.T, got, want, label string) {
	t.Helper()
	if got != want {
		t.Errorf("%s\n    got: %s", label, got)
	}
}

// TestModel: what the window changes goes through these, then
// Recompute: the order traits and mods are kept in, removing and
// changing them, copying a list of mods, and each Endurance Use tied
// to the Ability or attack its detail names.
func TestModel(t *testing.T) {
	// ---- Traits: in the loader's order ----
	c := New()
	c.Name = "Model"
	use := NewTrait(item(t, "Endurance Use"), rules.Weakness, -2, "", "telekinesis")
	InsertTrait(&c.Traits, use)
	tk := NewTrait(item(t, "Telekinesis"), rules.Ability, 2, "", "")
	InsertTrait(&c.Traits, tk)
	fl := NewTrait(item(t, "Flight"), rules.Ability, 1, "", "")
	InsertTrait(&c.Traits, fl)
	InsertTrait(&c.Traits, NewTrait(nil, rules.Ability, 1, "Zany Gadgets", "b"))
	z := NewTrait(nil, rules.Ability, 1, "zany gadgets", "A")
	InsertTrait(&c.Traits, z)
	is(t, traits(c.Traits), "Flight; Telekinesis; zany gadgets (A); Zany Gadgets (b); Endurance Use (telekinesis)",
		"InsertTrait: Abilities before Weaknesses, each by name, then detail, ignoring case")
	RemoveTrait(&c.Traits, &z.Trait)
	is(t, traits(c.Traits), "Flight; Telekinesis; Zany Gadgets (b); Endurance Use (telekinesis)", "RemoveTrait")
	ChangeTrait(&c.Traits, fl, item(t, "Telekinesis"), 3, "small things")
	is(t, traits(c.Traits), "Telekinesis; Telekinesis (small things); Zany Gadgets (b); Endurance Use (telekinesis)",
		"ChangeTrait: renamed and moved to its new place")
	if fl.Item != item(t, "Telekinesis") || fl.Level != 3 {
		t.Error("ChangeTrait: the item and level")
	}
	ChangeTrait(&c.Traits, fl, item(t, "Flight"), 1, "")
	if fl.HasDetail || fl.Detail != "" {
		t.Error("ChangeTrait: a detail removed")
	}

	// ---- Endurance Use ----
	Recompute(c)
	if !tk.HasEnd || tk.End.Get() != 10 {
		t.Error("Endurance Use -2 ties to the Ability its detail names: END 10")
	}
	if fl.HasEnd {
		t.Error("and to no other")
	}
	ChangeTrait(&c.Traits, use, use.Item, -1, "Telekinesis (x)")
	Recompute(c)
	if tk.HasEnd {
		t.Error("a detail naming nothing ties to nothing")
	}
	ChangeTrait(&c.Traits, tk, tk.Item, 2, "x")
	Recompute(c)
	if !tk.HasEnd || tk.End.Get() != 5 {
		t.Error("a detail naming an Ability and its detail: Endurance Use -1, END 5")
	}
	a := NewAttack("Gust")
	c.Attacks = append(c.Attacks, &a.Attack)
	InsertMod(&a.Mods, NewMod(item(t, "Accurate"), false, 1, 0))
	ChangeTrait(&c.Traits, use, use.Item, -1, "gust")
	Recompute(c)
	if tk.HasEnd || a.End.Get() != 10 {
		t.Error("else the attack it names: Accurate's 5 and Endurance Use's 5")
	}

	// ---- Mods ----
	var list []*calc.Mod
	InsertMod(&list, NewMod(item(t, "Requirement"), true, 1, -10))
	perk := NewMod(item(t, "Ranged"), false, 1, 0)
	InsertMod(&list, perk)
	cancel := NewMod(item(t, "Cancel"), true, 1, -15)
	InsertMod(&list, cancel)
	m := NewMod(item(t, "Accurate"), false, 2, 0)
	InsertMod(&list, m)
	is(t, modNames(list), "Accurate; Ranged; Cancel; Requirement", "InsertMod: Perks before Flaws, each by name")
	if !cancel.HasCost || cancel.Cost != -15 || m.HasCost {
		t.Error("NewMod: a chosen cost is written, a fixed one isn't")
	}
	cp := CopyMods(list)
	RemoveMod(&list, &perk.Mod)
	m.Times = 3
	is(t, modNames(list), "Accurate; Cancel; Requirement", "RemoveMod")
	is(t, modNames(cp), "Accurate; Ranged; Cancel; Requirement", "CopyMods: a copy the list's changes don't reach")
	if cp[0] == &m.Mod || cp[0].Times != 2 || ModOf(cp[0]) == m {
		t.Error("CopyMods: each mod copied")
	}
}
