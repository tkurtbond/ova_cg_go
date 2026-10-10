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

	"github.com/tkurtbond/ova_cg/internal/calc"
	"github.com/tkurtbond/ova_cg/internal/character"
	"github.com/tkurtbond/ova_cg/internal/entities"
	"github.com/tkurtbond/ova_cg/internal/rules"
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
	c, notices, err := character.Load(dataDir + name + ".yaml")
	if err != nil {
		t.Fatalf("loaded: %v", err)
	}
	if len(notices) != want {
		t.Errorf("notices: %d, want %d:\n    %s", len(notices), want, strings.Join(notices, "\n    "))
	}
	text := ToText(c)
	saved := savedDir + name + ".yaml"
	if err := Save(c, saved); err != nil {
		t.Fatalf("saved: %v", err)
	}
	c2, notices, err := character.Load(saved)
	if err != nil {
		t.Fatalf("the copy loaded: %v", err)
	}
	if len(notices) != 0 {
		t.Errorf("the copy has no notices: it lists its overrides:\n    %s", strings.Join(notices, "\n    "))
	}
	if text == "" || ToText(c2) != text {
		t.Errorf("the copy saves the same:\n%s\n----\n%s", text, ToText(c2))
	}
}

// TestFixtures: the samples' numbers are all the rules', given the keys
// the fixtures carry, but the book's two rulings, which their
// overrides: list. Sheet uses every key. po_ova_fmt's own fixtures'
// numbers were never the rules', so they come back as overrides.
func TestFixtures(t *testing.T) {
	for _, x := range []struct {
		name string
		want int
	}{
		{"Ai", 0}, {"Auren", 0}, {"Braun", 0}, {"Cora", 0}, {"Daisuke", 0}, {"Fukiko", 0},
		{"Karis", 0}, {"Miho", 0}, {"Natsuki", 0}, {"Raine", 0}, {"Shou", 0}, {"Yuu", 0},
		{"Sheet", 0},
		{"Escaping", 5}, {"Unsorted", 8}, {"Vehicle", 5},
	} {
		t.Run(x.name, func(t *testing.T) { roundTrip(t, x.name, x.want) })
	}
}

func TestErrors(t *testing.T) {
	for _, x := range []struct{ path, want string }{
		{dataDir + "Broken.yaml", "level"},
		{dataDir + "Fragment.yaml", "holds 2 characters"},
		{dataDir + "Options.yaml", "holds 2 characters"},
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
