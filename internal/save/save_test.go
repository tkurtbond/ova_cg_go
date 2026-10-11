package save

// po_ova_cg's TestFiles, and TestModel's save and load. Each fixture
// holding one character is loaded, saved into ../../obj/files/ (for
// test/files.sh, which formats both with po_ova_fmt and compares), and
// loaded again: the second load must give the same YAML, and no
// notices, since the copy's overrides: keys list every number kept as
// an override. The notices are the numbers the first load kept as
// overrides, those the file differs from the rules on without listing
// them. Then the files that can't be loaded, a failed save, and a
// character with text plain YAML can't hold.

import (
	"os"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/tkurtbond/go_ova_cg/internal/calc"
	"github.com/tkurtbond/go_ova_cg/internal/character"
	"github.com/tkurtbond/go_ova_cg/internal/entities"
	"github.com/tkurtbond/go_ova_cg/internal/rules"
)

const (
	dataDir  = "../../testdata/"
	savedDir = "../../obj/files/"
)

func TestMain(m *testing.M) {
	if err := os.MkdirAll(savedDir, 0o755); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// roundTrip loads name, saves it, loads the copy; want: the notices
// expected.
func roundTrip(t *testing.T, name string, want int) {
	c, notices, err := character.LoadAll(dataDir + name + ".yaml")
	if err != nil {
		t.Fatalf("loaded: %v", err)
	}
	if len(notices) != want {
		t.Errorf("notices: %d, want %d:\n    %s", len(notices), want, strings.Join(notices, "\n    "))
	}
	text := AllToText(c)
	saved := savedDir + name + ".yaml"
	if err := SaveAll(c, saved); err != nil {
		t.Fatalf("saved: %v", err)
	}
	c2, notices, err := character.LoadAll(saved)
	if err != nil {
		t.Fatalf("the copy loaded: %v", err)
	}
	if len(notices) != 0 {
		t.Errorf("the copy has no notices: it lists its overrides:\n    %s", strings.Join(notices, "\n    "))
	}
	if text == "" || AllToText(c2) != text {
		t.Errorf("the copy saves the same:\n%s\n----\n%s", text, AllToText(c2))
	}
}

// TestFixtures: the samples' numbers are all the rules', given the keys
// the fixtures carry, but the book's two rulings, which their
// overrides: list. Sheet uses every key; Defenses and Wisp, the three
// Defense values, and Incorporeal Form's alone. po_ova_fmt's own
// fixtures' numbers were never the rules', so they come back as
// overrides. The Sample NPCs: a Secondary NPC's sheet is a Companion's
// (npc-type:), so Dr. Tomori's and Jiro's numbers are the rules'; Nazo's
// and Shadowman's single Defense is their best, as the book prints it,
// but Nazo's TV is the book's own (po_ova_cg's CHARACTER-DIFFERENCES.md).
// Acacia and Zurkrieg hold more than one entity (TestSeveral); Acacia's
// notice is its Lock Heaven's Door's END. The NPC Fragments have no
// combat: block, so nothing to compare. Fragment and Options hold two
// Fragments each, and two-documents a character in each of two YAML
// documents.
func TestFixtures(t *testing.T) {
	for _, x := range []struct {
		name string
		want int
	}{
		{"Ai", 0}, {"Auren", 0}, {"Braun", 0}, {"Cora", 0}, {"Daisuke", 0}, {"Fukiko", 0},
		{"Karis", 0}, {"Miho", 0}, {"Natsuki", 0}, {"Raine", 0}, {"Shou", 0}, {"Yuu", 0},
		{"Sheet", 0}, {"Defenses", 0}, {"Wisp", 0},
		{"Escaping", 5}, {"Unsorted", 8}, {"Vehicle", 5},
		{"Acacia", 1}, {"Zurkrieg", 0}, {"Ancel", 0}, {"Dr-Tomori", 0}, {"Jiro", 0}, {"Nazo", 1}, {"Saspar", 0}, {"Shadowman", 0},
		{"Beat-Cop", 0}, {"Cat-Burglar", 0}, {"Dragon", 0}, {"Ebullient-Club-President", 0},
		{"Ghosts", 0}, {"Goblin", 0}, {"GPL-Agents", 0}, {"Indebted-Yakuza", 0},
		{"Magical-Fairies", 0}, {"Ninja", 0}, {"Oni", 0}, {"Sentry-Bots", 0}, {"Shady-Teacher", 0},
		{"Shinto-Priest", 0}, {"Snobbish-Upperclassman", 0}, {"Spider-Tank", 0}, {"Thugs", 0},
		{"Venomous-Killer", 0}, {"Weathered-Salaryman", 0}, {"Weird-Exchange-Student", 0},
		{"Wild-Beast", 0}, {"Zombies", 0},
		{"Fragment", 0}, {"Options", 0}, {"two-documents", 0},
	} {
		t.Run(x.name, func(t *testing.T) { roundTrip(t, x.name, x.want) })
	}
}

func TestErrors(t *testing.T) {
	for _, x := range []struct{ path, want string }{
		{dataDir + "Broken.yaml", "level"},
		{dataDir + "no-such-file.yaml", "no-such-file.yaml"},
	} {
		c, _, err := character.Load(x.path)
		if c != nil || err == nil || !strings.Contains(err.Error(), x.want) {
			t.Errorf("%s: refused, saying %s: %v", x.path, x.want, err)
		}
	}
	c := character.New()
	c.Name = "Nobody"
	if err := Save(c, "../../obj/no-such-dir/x.yaml"); err == nil || err.Error() == "" {
		t.Error("a save to a missing directory fails, with a message")
	}
}

// names is the characters' names, "|" between them.
func names(cs []*character.Character) string {
	l := make([]string, len(cs))
	for i, c := range cs {
		l[i] = c.Name
	}
	return strings.Join(l, "|")
}

// TestSeveral: files of several characters (po_ova_cg's PLAN.md,
// "Several characters in a file"): every one loaded, in order, each its
// notices, and saved in that order; Load still takes exactly one.
func TestSeveral(t *testing.T) {
	cs, notices, err := character.LoadAll(dataDir + "Acacia.yaml")
	if err != nil || names(cs) != "Acacia|Sazza and Fraz|High Fructo Cezar" {
		t.Errorf("Acacia.yaml: her and her two demons, in the file's order: %v %q", err, names(cs))
	}
	if len(notices) != 1 || !strings.HasPrefix(notices[0], "Acacia: Lock Heaven's Door: END") {
		t.Errorf("its one notice names its character: %q", notices)
	}
	cs, _, err = character.LoadAll(dataDir + "two-documents.yaml")
	if err != nil || names(cs) != "First Of Two|Second Of Two" {
		t.Errorf("two-documents.yaml: a character from each document: %v %q", err, names(cs))
	}
	err = SaveAll(cs, "../../obj/two-documents.yaml")
	cs2, _, err2 := character.LoadAll("../../obj/two-documents.yaml")
	if err != nil || err2 != nil || names(cs2) != "First Of Two|Second Of Two" ||
		strings.Count(AllToText(cs), "---") > 0 {
		t.Errorf("saved as one document, the two in order: %v %v %q", err, err2, names(cs2))
	}
	cs, notices, err = character.LoadAll(dataDir + "Zurkrieg.yaml")
	if err != nil || names(cs) != "Zurkrieg|Men in Black" || !cs[0].HasCombat || !cs[1].IsFragment ||
		cs[1].HasCombat || len(notices) != 0 {
		t.Errorf("Zurkrieg.yaml: him, with combat stats, and his Men in Black, a Fragment; no notices: %v", err)
	}
	c, _, err := character.Load(dataDir + "Zurkrieg.yaml")
	if c != nil || err == nil || !strings.Contains(err.Error(), "holds 2 characters, not one") {
		t.Errorf("Load, for one character, refuses it, saying how many it holds: %v", err)
	}
}

// TestCopy: a copy, for Duplicate Character, saves the same and shares
// nothing.
func TestCopy(t *testing.T) {
	c, _, err := character.Load(dataDir + "Fukiko.yaml")
	if err != nil {
		t.Fatal(err)
	}
	d := Copy(c)
	if d == nil || d == c || ToText(d) != ToText(c) {
		t.Fatal("a copy, the same when saved")
	}
	d.Health.Override(55, false, false)
	character.Recompute(d)
	if c.Health.Overridden || len(d.Traits) == 0 || d.Traits[0] == c.Traits[0] {
		t.Error("the copy's HEALTH overridden, the first's not: they share nothing")
	}
}

// holdsLine says whether text holds the line line.
func holdsLine(text, line string) bool {
	return strings.HasPrefix(text, line+"\n") || strings.Contains(text, "\n"+line+"\n")
}

// savesAs loads name and saves it: its text must hold the line want.
func savesAs(t *testing.T, name, want string) {
	t.Helper()
	c, _, err := character.Load(dataDir + name + ".yaml")
	if err != nil {
		t.Fatalf("%s: loaded: %v", name, err)
	}
	if text := ToText(c); !holdsLine(text, want) {
		t.Errorf("%s: saved with %s:\n%s", name, want, text)
	}
}

// TestDefenses: up to three Defense values (po_ova_cg's PLAN.md, Phase
// 11): one where the rules give more is the best; two, the Defense and
// the first alternate; an overridden alternate the character doesn't
// have is saved where it is read back.
func TestDefenses(t *testing.T) {
	t.Run("defense-best", func(t *testing.T) { roundTrip(t, "defense-best", 0) })
	savesAs(t, "defense-best", "    defense: 4")
	t.Run("defense-best-differs", func(t *testing.T) { roundTrip(t, "defense-best-differs", 1) })
	savesAs(t, "defense-best-differs", "    defense: 3")
	c, _, err := character.Load(dataDir + "defense-best-differs.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if slot, single := character.SingleDefense(c); !c.Incorporeal.Overridden || c.Barrier.Overridden ||
		c.Defense.Overridden || !single || slot != character.IncorporealSlot {
		t.Error("defense-best-differs: read as Incorporeal Form's, the best by the rules, and saved as one")
	}
	t.Run("defense-two", func(t *testing.T) { roundTrip(t, "defense-two", 0) })
	savesAs(t, "defense-two", "    defense: 2/3/4")

	// One value, but another overridden since: all of them saved.
	c, _, err = character.Load(dataDir + "defense-best.yaml")
	if err != nil {
		t.Fatal(err)
	}
	c.Defense.Override(3, false, false)
	character.Recompute(c)
	text := ToText(c)
	if _, single := character.SingleDefense(c); single || !holdsLine(text, "    defense: 3/4/4") ||
		!holdsLine(text, "    overrides: [defense]") {
		t.Errorf("defense-best, its DEFENSE overridden: saved with all three:\n%s", text)
	}

	// An overridden Incorporeal Form's Defense the character doesn't have:
	// Barrier's is saved before it, so it is read back as the third.
	c = character.New()
	c.Name = "Ghostless"
	c.Incorporeal.Override(7, false, false)
	character.Recompute(c)
	text = ToText(c)
	if !holdsLine(text, "    defense: 2/0/7") || !holdsLine(text, "    overrides: [defense-3]") {
		t.Errorf("an overridden Incorporeal Form's DEFENSE without it: saved third, after Barrier's 0:\n%s", text)
	}
	const path = "../../obj/ghostless.yaml"
	if err := Save(c, path); err != nil {
		t.Fatal(err)
	}
	c2, notices, err := character.Load(path)
	if err != nil || len(notices) != 0 || !c2.Incorporeal.Overridden || c2.Incorporeal.Get() != 7 ||
		c2.Barrier.Overridden {
		t.Errorf("and read back as Incorporeal Form's, with no notice: %v %q", err, notices)
	}
}

// TestAwkwardText: text plain YAML can't hold, saved and loaded back.
func TestAwkwardText(t *testing.T) {
	const path = savedDir + "awkward.yaml"
	c := character.New()
	c.Name = `#1 "Ace"`
	c.HasTagline, c.Tagline = true, "'Til the end: - and | >"
	c.HasBackground, c.Background = true, "  Indented first line,\nthen a second; # not a comment"
	note := &character.Note{Header: "? Header", Text: "@line one\n\n%line three"}
	c.Notes = []*character.Note{note}
	c.HasFluff = true
	c.HasFluffField[entities.Hobbies], c.Fluff[entities.Hobbies] = true, "#1 fan"
	calc.Compute(&c.Sheet)
	if err := Save(c, path); err != nil {
		t.Fatalf("saved: %v", err)
	}
	c2, notices, err := character.Load(path)
	if err != nil {
		t.Fatalf("loaded: %v", err)
	}
	if c2.Name != c.Name {
		t.Errorf("the name: %q", c2.Name)
	}
	if c2.Tagline != c.Tagline {
		t.Errorf("the tagline: %q", c2.Tagline)
	}
	// The file keeps the background's first blanks; entities trims it
	// on loading, as po_ova_fmt and the Ada ova_fmt do.
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Background string `yaml:"background"`
	}
	var docs []yaml.Node
	if err := yaml.Unmarshal(b, &docs); err != nil || len(docs) != 1 || docs[0].Decode(&doc) != nil ||
		doc.Background != c.Background {
		t.Errorf("the file holds the background exactly, its first blanks too: %q %v", doc.Background, err)
	}
	if c2.Background != entities.Trim(c.Background) {
		t.Errorf("the background loads trimmed: %q", c2.Background)
	}
	if len(c2.Notes) != 1 || *c2.Notes[0] != *note {
		t.Errorf("the note: %+v", c2.Notes)
	}
	if !c2.HasFluffField[entities.Hobbies] || c2.Fluff[entities.Hobbies] != "#1 fan" {
		t.Error("hobbies #1 fan")
	}
	if len(notices) != 0 {
		t.Errorf("no notices: %q", notices)
	}
}

// TestModelSaved: each Endurance Use tied to the Ability or attack its
// detail names counts in that one's END, and survives a save and a load
// with no notice (TestModel's end).
func TestModelSaved(t *testing.T) {
	const path = savedDir + "model.yaml"
	c := character.New()
	c.Name = "Model"
	use := character.NewTrait(rules.Find("Endurance Use"), rules.Weakness, -1, "", "telekinesis (x)")
	character.InsertTrait(&c.Traits, use)
	tk := character.NewTrait(rules.Find("Telekinesis"), rules.Ability, 2, "", "x")
	character.InsertTrait(&c.Traits, tk)
	character.InsertTrait(&c.Traits, character.NewTrait(rules.Find("Flight"), rules.Ability, 1, "", ""))
	character.InsertTrait(&c.Traits, character.NewTrait(nil, rules.Ability, 1, "Zany Gadgets", "b"))
	a := character.NewAttack("Gust")
	c.Attacks = append(c.Attacks, &a.Attack)
	character.InsertMod(&a.Mods, character.NewMod(rules.Find("Accurate"), false, 1, 0))
	character.Recompute(c)
	if !tk.HasEnd || tk.End.Get() != 5 {
		t.Fatal("Telekinesis (x) has END 5")
	}
	if err := Save(c, path); err != nil {
		t.Fatalf("saved: %v", err)
	}
	c2, notices, err := character.Load(path)
	if err != nil || len(notices) != 0 {
		t.Fatalf("loaded again with no notices: the END it wrote is the rules': %v %q", err, notices)
	}
	if ToText(c2) != ToText(c) {
		t.Error("and it saves the same")
	}
}

// TestNpcTypeSaved: the NPC type sets the sheet's kind, and it and the
// hooks are saved and loaded back (TestModel's).
func TestNpcTypeSaved(t *testing.T) {
	c := character.New()
	c.Name = "Typed"
	character.InsertTrait(&c.Traits, character.NewTrait(rules.Find("Tough"), rules.Ability, 1, "", ""))
	character.Recompute(c)
	if c.HasNpcType || c.Kind != calc.Character || c.Health.Get() != 50 {
		t.Error("no NPC type: a character's sheet, HEALTH 40 + Tough 1")
	}
	character.SetNpcType(c, true, entities.Secondary)
	character.Recompute(c)
	if c.Kind != calc.CompanionSheet || c.Health.Get() != 30 || c.Endurance.Get() != 20 {
		t.Error("Secondary: a Companion's sheet, HEALTH 20 + Tough 1, ENDURANCE 20")
	}
	character.SetNpcType(c, true, entities.Extra)
	character.Recompute(c)
	if c.Kind != calc.ExtraSheet || c.Health.Get() != 20 || c.TV.Get() != 1 {
		t.Error("Extra: HEALTH 10 + Tough 1, TV 1")
	}
	character.SetNpcType(c, true, entities.Heroic)
	character.Recompute(c)
	if c.Kind != calc.Character || c.Health.Get() != 50 {
		t.Error("Heroic: a character's sheet")
	}
	character.SetNpcType(c, true, entities.Secondary)
	c.HasCharacterHooks, c.CharacterHooks = true, "Hooks."
	character.Recompute(c)
	if text := ToText(c); !strings.Contains(text, "npc-type: Secondary") ||
		!strings.Contains(text, "character-hooks: Hooks.") {
		t.Errorf("the type and the hooks saved:\n%s", text)
	}
	const path = savedDir + "model.yaml"
	if err := Save(c, path); err != nil {
		t.Fatal(err)
	}
	c2, notices, err := character.Load(path)
	if err != nil || len(notices) != 0 || !c2.HasNpcType || c2.NpcType != entities.Secondary ||
		c2.Kind != calc.CompanionSheet || c2.CharacterHooks != "Hooks." {
		t.Errorf("loaded again: the type, so the sheet's kind, and the hooks; no notices: %v %q", err, notices)
	}
	character.SetNpcType(c, false, 0)
	character.Recompute(c)
	if c.HasNpcType || c.Kind != calc.Character || strings.Contains(ToText(c), "npc-type") {
		t.Error("the type taken off: a character's sheet, nothing saved")
	}
}

// TestEndZero: an Ability's END of 0 isn't written unless it is an
// override (TestModel's).
func TestEndZero(t *testing.T) {
	c := character.New()
	c.Name = "Free"
	c.HasCombat = false
	tr := character.NewTrait(rules.Find("Attack"), rules.Ability, 3, "", "")
	character.InsertMod(&tr.Mods, character.NewMod(rules.Find("Armor Piercing"), false, 1, 0))
	character.InsertMod(&tr.Mods, character.NewMod(rules.Find("Ranged"), false, 1, 0))
	character.InsertMod(&tr.Mods, character.NewMod(rules.Find("Ineffective"), true, 1, 0))
	character.InsertMod(&tr.Mods, character.NewMod(rules.Find("Weapon"), true, 1, 0))
	character.InsertTrait(&c.Traits, tr)
	character.Recompute(c)
	if !tr.HasEnd || tr.End.Get() != 0 {
		t.Fatal("Perks and Flaws costing nothing: END 0")
	}
	if strings.Contains(ToText(c), "endurance") {
		t.Error("an END of 0 isn't written")
	}
	tr.End.Override(0, false, false)
	if !strings.Contains(ToText(c), "endurance: 0") {
		t.Error("unless it is an override")
	}
	tr.End.Clear()
	tr.End.Override(5, false, false)
	if !strings.Contains(ToText(c), "endurance: 5") {
		t.Error("an override of 5 is")
	}
}
