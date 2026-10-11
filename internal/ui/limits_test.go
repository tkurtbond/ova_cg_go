package ui

// ova_cg_po's TestLimits: Character > Limits..., and the rulebook and
// viewer kept in the preferences (a file of this test's own, never the
// user's).

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tkurtbond/ova_cg_go/internal/character"
	"github.com/tkurtbond/ova_cg_go/internal/prefs"
	"github.com/tkurtbond/ova_cg_go/internal/rules"
	"github.com/tkurtbond/ova_cg_go/internal/viewer"
)

// reds is which of the Abilities' levels, the Total, the points and the
// Abilities' points are red: "ATPa" for all four, "-T--" for the Total
// alone.
func reds(w *Window) string {
	s := []byte("----")
	if w.Numbers[AbilityLevels].Red() {
		s[0] = 'A'
	}
	if w.Numbers[Total].Red() {
		s[1] = 'T'
	}
	if w.Numbers[Points].Red() {
		s[2] = 'P'
	}
	if w.Numbers[AbilityPoints].Red() {
		s[3] = 'a'
	}
	return string(s)
}

func addAbility(w *Window, name string, level int) {
	w.Abilities.Pick(name)
	w.Abilities.Spinner.SetValue(level)
	w.Abilities.Text.SetText("")
	w.Abilities.Add.OnTapped()
}

func addWeakness(w *Window, name string, level int) {
	w.Weaknesses.Pick(name)
	w.Weaknesses.Spinner.SetValue(level)
	w.Weaknesses.Text.SetText("")
	w.Weaknesses.Add.OnTapped()
}

// lineListed says whether one of p's lines is head, a tab, points and a tab
// if any are given, and name's page.
func lineListed(p *TraitPanel, head, points, name string) bool {
	want := head + "\t"
	if points != "" {
		want += points + "\t"
	}
	want += rules.PageText(rules.Find(name))
	for i := 0; i < p.Count(); i++ {
		if p.Line(i) == want {
			return true
		}
	}
	return false
}

func warned(w *Window, text string) bool { return w.Warnings.Find(text) >= 0 }

func modified(w *Window) bool { return strings.Contains(w.Win.Title(), " *") }

func turn(c interface{ SetChecked(bool) }, on bool) { c.SetChecked(on) }

func TestLimits(t *testing.T) {
	w, f := newWindow(t)
	f.answer = Discard
	w.Win.Show()
	check(t, w.LimitsItem.Label == "Limits...", "Character > Limits...")

	// fresh is a new character with Attack +3, unmodified; Limits...
	// opened.
	fresh := func() *SettingsDialog {
		w.NewItem.Action()
		addAbility(w, "Attack", 3)
		w.Adopt(character.One(w.Character), "", nil)
		w.LimitsItem.Action()
		return w.Settings
	}

	// ---- Limits... ----
	d := fresh()
	c := w.Character
	check(t, d != nil && !d.BaseZero.Checked && !d.PowerCeiling.Checked && !d.StartingLevels.Checked &&
		!d.ScaledCost.Checked && !d.HalfCost.Checked && !d.Combined.Checked, "a new character: no limits")
	check(t, d.BaseZeroRange.Value() == 5 && d.Starting.Value() == 5 && d.Ceiling.Value() == 15 &&
		d.BaseZeroRange.Disabled() && d.Ceiling.Disabled() && d.Starting.Disabled() && d.HalfCost.Disabled(),
		"their numbers p. 21's example, inactive; half cost inactive with none of its rules")
	check(t, c.Total.Value == 3 && reds(w) == "----" && !modified(w), "Attack +3: a total of 3, nothing red")

	turn(d.BaseZero, true)
	check(t, c.Options.BaseZero && c.Options.BaseZeroRange == 5 && c.HasOptions && !d.BaseZeroRange.Disabled() &&
		!d.HalfCost.Disabled() && modified(w) && strings.Contains(saved(w), "base-zero: 5") && reds(w) == "----",
		"Base Zero on: its range 5, saved, the title marked; 3 within it")
	d.BaseZeroRange.Spin(2)
	check(t, c.Options.BaseZeroRange == 2 && reds(w) == "-T--" && warned(w, "Base Zero") &&
		strings.Contains(saved(w), "base-zero: 2"), "range 2: the Total red, and the warning")
	d.BaseZeroRange.Spin(3)
	check(t, reds(w) == "----" && !warned(w, "Base Zero"), "range 3: not red")
	turn(d.ScaledCost, true)
	check(t, c.Options.ScaledCost && c.Points.Value == 4 && w.Numbers[Points].Visible() && reds(w) == "--P-" &&
		strings.Contains(saved(w), "scaled-cost: true"),
		"Scaled Cost: 4 points, shown, beyond 3: the points red, not the Total")
	check(t, c.AbilityPoints.Value == 4 && w.Numbers[AbilityPoints].Visible(), "and the Abilities' 4 points shown")
	turn(d.PowerCeiling, true)
	d.Ceiling.Spin(3)
	check(t, reds(w) == "--Pa" && warned(w, "Power Ceiling"),
		"Power Ceiling 3 on Scaled Cost: the Abilities' points red, not their levels")
	d.Ceiling.Spin(15)
	turn(d.PowerCeiling, false)
	turn(d.ScaledCost, false)
	turn(d.BaseZero, false)
	check(t, !c.Options.BaseZero && d.BaseZeroRange.Disabled() && d.HalfCost.Disabled() &&
		d.BaseZeroRange.Value() == 3 && reds(w) == "----" && !strings.Contains(saved(w), "base-zero"),
		"Base Zero off: its number kept, inactive, not saved")

	turn(d.PowerCeiling, true)
	check(t, c.Options.PowerCeiling && c.Options.Ceiling == 15 && reds(w) == "----" &&
		strings.Contains(saved(w), "power-ceiling: 15"), "Power Ceiling 15: 3 under it")
	d.Ceiling.Spin(2)
	check(t, reds(w) == "A---" && warned(w, "Power Ceiling"), "Power Ceiling 2: the Abilities red, and the warning")
	turn(d.PowerCeiling, false)
	turn(d.StartingLevels, true)
	check(t, c.Options.StartingLevels && c.Options.Starting == 5 && reds(w) == "----" &&
		strings.Contains(saved(w), "starting-levels: 5"), "5 starting levels: 3 within them")
	d.Starting.Spin(2)
	check(t, reds(w) == "A---" && warned(w, "starting levels"), "2 starting levels: the Abilities red")
	turn(d.StartingLevels, false)

	addAbility(w, "Knowledge", 3)
	turn(d.PowerCeiling, true)
	turn(d.HalfCost, true)
	check(t, c.Options.HalfCostKnowledge && c.AbilityLevels.Value == 5 &&
		strings.Contains(saved(w), "half-cost-knowledge: true"), "half cost: Knowledge +3 counts 2 (p. 57)")
	turn(d.PowerCeiling, false)
	check(t, c.Options.HalfCostKnowledge && d.HalfCost.Disabled() && c.AbilityLevels.Value == 6,
		"with none of its rules: kept, inactive, and Knowledge counts 3")
	turn(d.Combined, true)
	check(t, c.Options.CombinedEffectiveness && strings.Contains(saved(w), "combined-effectiveness: true"),
		"Combined Effectiveness on, saved")
	check(t, lineListed(w.Abilities, "+3 Attack", "", "Attack"), "no points in the lists without Scaled Cost")
	turn(d.ScaledCost, true)
	check(t, lineListed(w.Abilities, "+3 Attack", "4 pts", "Attack") && lineListed(w.Abilities, "+3 Knowledge", "2 pts", "Knowledge"),
		"Scaled Cost: each Ability's points listed, Knowledge +3's halved")
	addWeakness(w, "Weak", -1)
	check(t, lineListed(w.Weaknesses, "-1 Weak", "1 pt", "Weak"), "and each Weakness's")
	turn(d.ScaledCost, false)
	check(t, lineListed(w.Abilities, "+3 Attack", "", "Attack") && lineListed(w.Weaknesses, "-1 Weak", "", "Weak"),
		"Scaled Cost off: the points gone from the lists")

	d.Cancel.OnTapped()
	check(t, w.Settings == nil && !c.HasOptions && !c.Options.HalfCostKnowledge &&
		!c.Options.CombinedEffectiveness && !c.Options.PowerCeiling && !strings.Contains(saved(w), "options:") &&
		reds(w) == "----" && c.AbilityLevels.Value == 6 && !modified(w),
		"Cancel: no limits again, nothing saved, the title unmarked")

	d = fresh()
	c = w.Character
	turn(d.BaseZero, true)
	d.BaseZeroRange.Spin(1)
	d.OK.OnTapped()
	check(t, w.Settings == nil && c.Options.BaseZero && strings.Contains(saved(w), "base-zero: 1") && modified(w) &&
		reds(w) == "-T--", "OK: kept, saved, the title marked, the Total red")
	w.LimitsItem.Action()
	d = w.Settings
	check(t, d.BaseZero.Checked && d.BaseZeroRange.Value() == 1 && !d.BaseZeroRange.Disabled(),
		"opened again: as left")
	turn(d.BaseZero, false)
	d.Cancel.OnTapped()
	check(t, w.Settings == nil && c.Options.BaseZero && reds(w) == "-T--" && modified(w),
		"Cancel: as before, the title still marked from before")

	w.NewItem.Action()
	c = w.Character
	c.HasOptions = true // as a file with an empty options: block loads
	w.LimitsItem.Action()
	d = w.Settings
	turn(d.BaseZero, true)
	turn(d.BaseZero, false)
	check(t, c.HasOptions && strings.Contains(saved(w), "options:"),
		"an empty options: block, changed and changed back: kept")
	turn(d.Combined, true)
	turn(d.Combined, false)
	turn(d.BaseZero, true)
	d.OK.OnTapped()
	ok := w.Load(dataDir + "Sheet.yaml")
	w.LimitsItem.Action()
	d = w.Settings
	check(t, ok && !d.BaseZero.Checked && d.PowerCeiling.Checked && d.Ceiling.Value() == 30 &&
		d.StartingLevels.Checked && d.Starting.Value() == 20 && d.HalfCost.Checked && !d.HalfCost.Disabled() &&
		!d.ScaledCost.Checked, "Sheet's: Power Ceiling 30, 20 starting levels, half cost")
	d.CancelEdit()

	// ---- The rulebook and the viewer, kept ----
	path := filepath.Join(t.TempDir(), "ova_cg_go.prefs")
	w.UsePrefs(prefs.Open(path))
	check(t, w.SavedRulebook == "", "no rulebook kept")
	pdfAsks, pdfAnswer := 0, dataDir+"Sheet.yaml" // any file stands for the PDF
	w.ChoosePdf = func(then func(string)) { pdfAsks++; then(pdfAnswer) }
	textAnswer := ""
	w.AskText = func(prompt, text string, then func(string, bool)) { then(textAnswer, textAnswer != "") }
	kept := func(name string) string { return prefs.Open(path).Get(name) }
	w.ChoosePdfItem.Action()
	check(t, pdfAsks == 1 && w.RulebookOption == w.SavedRulebook &&
		strings.HasSuffix(kept("rulebook"), "/testdata/Sheet.yaml") && strings.HasPrefix(kept("rulebook"), "/"),
		"Choose PDF...: used from now on, and kept, absolute")

	viewerIs := func(name string) bool { return w.Viewer.Name() == name }
	textAnswer = "evince"
	w.ViewerItem.Action()
	check(t, viewerIs("evince") && kept("viewer") == "evince", "Viewer...: evince, kept")
	textAnswer = "cat %f"
	w.ViewerItem.Action()
	check(t, viewerIs("cat %f") && kept("viewer") == "cat %f", "a command, kept")
	f.tells, textAnswer = 0, "nonsense"
	w.ViewerItem.Action()
	check(t, viewerIs("cat %f") && kept("viewer") == "cat %f" && f.tells == 1,
		"not a viewer: said so, nothing changed")
	textAnswer = ""
	w.ViewerItem.Action()
	check(t, viewerIs("cat %f"), "cancelled: nothing changed")

	w.Viewer, w.RulebookOption = viewer.Default(), ""
	w.UsePrefs(prefs.Open(path))
	check(t, viewerIs("cat %f") && strings.HasSuffix(w.SavedRulebook, "/testdata/Sheet.yaml"),
		"read back on the next run: the viewer and the rulebook")
	w.Viewer, w.ViewerOption = viewer.Default(), true
	w.UsePrefs(prefs.Open(path))
	check(t, !viewerIs("cat %f"), "--viewer's wins over the one kept")
	w.ViewerOption = false
	p := prefs.Open(path)
	p.Set("viewer", "nonsense")
	w.Viewer, _ = viewer.Parse("mupdf")
	w.UsePrefs(p)
	check(t, viewerIs("mupdf"), "a viewer kept that isn't one: ignored")

	// Not found: asked for once, then opened with it.
	w.OpenPage = w.openRulebook
	w.Viewer, _ = viewer.Parse("true %f %p")
	w.RulebookOption = "no/such/rulebook.pdf"
	pdfAsks, f.tells, pdfAnswer = 0, 0, ""
	w.RulebookItem.Action()
	check(t, pdfAsks == 1 && f.tells == 1, "the rulebook not found: asked for, and cancelled, said where it isn't")
	pdfAnswer, f.tells = dataDir+"Sheet.yaml", 0
	w.RulebookItem.Action()
	check(t, pdfAsks == 2 && f.tells == 0 && strings.HasSuffix(w.RulebookOption, "/testdata/Sheet.yaml") &&
		strings.HasSuffix(kept("rulebook"), "/testdata/Sheet.yaml"), "chosen: opened, and kept")
	w.RulebookItem.Action()
	check(t, pdfAsks == 2, "found the next time: not asked again")
}
