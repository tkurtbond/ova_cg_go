package markdown

// ova_cg_po's TestFiles' Markdown checks: each fixture's sheet, one
// character's after another, saved as ../../obj/files/NAME.md, and its
// inline paragraphs as NAME.inline.md, which test/files.sh compares with
// ova_fmt_po -M's and -i's of the copy internal/save's test saves beside
// them; and Save failing, then working.

import (
	"os"
	"strings"
	"testing"

	"github.com/tkurtbond/ova_cg_go/internal/character"
)

const (
	dataDir  = "../../testdata/"
	savedDir = "../../obj/files/"
)

// The fixtures internal/save's TestFixtures and TestDefenses save.
var fixtures = []string{
	"Ai", "Auren", "Braun", "Cora", "Daisuke", "Fukiko", "Karis", "Miho", "Natsuki", "Raine", "Shou", "Yuu",
	"Sheet", "Defenses", "Wisp", "Escaping", "Unsorted", "Vehicle",
	"Acacia", "Zurkrieg", "Ancel", "Dr-Tomori", "Jiro", "Nazo", "Saspar", "Shadowman",
	"Beat-Cop", "Cat-Burglar", "Dragon", "Ebullient-Club-President", "Ghosts", "Goblin", "GPL-Agents",
	"Indebted-Yakuza", "Magical-Fairies", "Ninja", "Oni", "Sentry-Bots", "Shady-Teacher", "Shinto-Priest",
	"Snobbish-Upperclassman", "Spider-Tank", "Thugs", "Venomous-Killer", "Weathered-Salaryman",
	"Weird-Exchange-Student", "Wild-Beast", "Zombies", "Fragment", "Options", "two-documents",
	"defense-best", "defense-best-differs", "defense-two",
}

func TestMain(m *testing.M) {
	if err := os.MkdirAll(savedDir, 0o755); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestFixtures(t *testing.T) {
	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			cs, _, err := character.LoadAll(dataDir + name + ".yaml")
			if err != nil {
				t.Fatalf("loaded: %v", err)
			}
			var sheet, inline strings.Builder
			for _, c := range cs {
				s := Sheet(c)
				if s == "" {
					t.Fatalf("%s: no sheet", c.Name)
				}
				sheet.WriteString(s)
				inline.WriteString(Inline(c))
			}
			md := savedDir + name + ".md"
			os.Remove(md) // so files.sh never compares a file from before
			if err := Save(sheet.String(), md); err != nil {
				t.Errorf("its Markdown sheet saved: %v", err)
			}
			md = savedDir + name + ".inline.md"
			os.Remove(md)
			if err := Save(inline.String(), md); err != nil {
				t.Errorf("its inline paragraphs saved: %v", err)
			}
		})
	}
}

// TestSave: a file that can't be written is an error, and the next
// save, to a file that can, works.
func TestSave(t *testing.T) {
	if err := Save("# A", "../../obj/no/such/dir/sheet.md"); err == nil {
		t.Errorf("a Markdown sheet that can't be written: no error")
	}
	p := "../../obj/sheet.md"
	os.Remove(p)
	if err := Save("# B", p); err != nil {
		t.Fatalf("then one that can: %v", err)
	}
	if b, err := os.ReadFile(p); err != nil || string(b) != "# B" {
		t.Errorf("then one that can: written, all of it: %q, %v", b, err)
	}
	if _, err := os.Stat(p + ".tmp"); err == nil {
		t.Errorf("the .tmp file left behind")
	}
}
