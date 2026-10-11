// Package markdown is a character's sheet in Markdown, as ova_fmt_po -M
// writes it (ova_cg_po's OvaMarkdown, and the Markdown path of
// ova_fmt_po's OvaCommon, which ova_cg_po copies).
//
// The character is written as it would be saved and read back as
// ova_fmt_po reads a file's entity (save.ToEntity), then formatted as
// OvaCommon.ProcessEntity formats it with its format Markdown and its
// defaults: the minus sign U+2212, and no options: block (no -O).
// test/files.sh checks the sheet is ova_fmt_po -M's of the saved file,
// byte for byte. OvaCommon's comments explain each choice of layout;
// these only summarize.
package markdown

import (
	"os"
	"strconv"
	"strings"

	"github.com/tkurtbond/ova_cg_go/internal/character"
	"github.com/tkurtbond/ova_cg_go/internal/entities"
	"github.com/tkurtbond/ova_cg_go/internal/save"
)

// Sheet is c's sheet; "" if c couldn't be read back (a programmer's
// error).
func Sheet(c *character.Character) string {
	e := save.ToEntity(c)
	if e == nil {
		return ""
	}
	return Entity(e)
}

// Save writes text to the file at path, through path.tmp, which replaces
// it only once written, as save.SaveAll does.
func Save(text, path string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(text), 0o644); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// Entity is e's sheet: OvaCommon.ProcessEntity's.
func Entity(e *entities.Entity) string {
	var b strings.Builder
	if e.IsFragment {
		fragmentHeader(&b, e)
		fragmentAttributes(&b, e)
		characterHooks(&b, e)
		combatAndAttacks(&b, &e.Combat, "")
		linkedSections(&b, e)
	} else {
		headerAndProse(&b, e)
		if len(e.Abilities) > 0 {
			b.WriteString(bullet + bold("Abilities:") + " " + joinAbilities(e.Abilities) + ".\n\n")
		}
		if len(e.Weaknesses) > 0 {
			b.WriteString(bullet + bold("Weaknesses:") + " " + joinWeaknesses(e.Weaknesses) + ".\n\n")
		}
		combatAndAttacks(&b, &e.Combat, npcTypeText(e))
		linkedSections(&b, e)
	}
	tail(&b, e)
	return b.String()
}

// ---- Primitives ----

const bullet = "- "

// line writes s and a line feed.
func line(b *strings.Builder, s string) { b.WriteString(s); b.WriteByte('\n') }

func bold(s string) string   { return "**" + s + "**" }
func italic(s string) string { return "*" + s + "*" }

// sectionHeader is a section's label: bold, in Markdown.
func sectionHeader(s string) string { return bold(s) }

// ---- Numbers and text ----

func trimImg(n int) string { return strconv.Itoa(n) }

// numberImg is a number that may be negative, with U+2212 MINUS SIGN.
func numberImg(n int) string {
	if n < 0 {
		return "−" + trimImg(-n)
	}
	return trimImg(n)
}

// signedImg is a Level: always signed.
func signedImg(n int) string {
	if n >= 0 {
		return "+" + trimImg(n)
	}
	return numberImg(n)
}

func endWithPeriod(s string) string {
	if strings.HasSuffix(s, ".") {
		return s
	}
	return s + "."
}

// nameDetail is "Name (detail)", or "Name" without a detail.
func nameDetail(name string, hasDetail bool, detail string) string {
	if hasDetail {
		return name + " (" + detail + ")"
	}
	return name
}

// marked is s, then "\*" if k is in overrides: escaped, so it can't open
// emphasis.
func marked(s string, k entities.Override, overrides entities.Overrides) string {
	if overrides.Has(k) {
		return s + `\*`
	}
	return s
}

// defenseText is c's defense with a mark after each overridden value: a
// "2/4/6" Defense's are defense, defense-2 and defense-3.
func defenseText(c *entities.Combat) string {
	var r strings.Builder
	k := entities.OvDefense
	rest := c.Defense
	for {
		slash := strings.IndexByte(rest, '/')
		if slash < 0 || k == entities.OvDefense3 {
			break
		}
		r.WriteString(marked(rest[:slash], k, c.Overrides) + "/")
		rest = rest[slash+1:]
		k++
	}
	r.WriteString(marked(rest, k, c.Overrides))
	return r.String()
}

func join[T any](l []T, f func(T) string) string {
	parts := make([]string, len(l))
	for i, x := range l {
		parts[i] = f(x)
	}
	return strings.Join(parts, ", ")
}

// ---- Perks and Flaws ----

// perkFlaw is "Name: detail x2 (−5)", each part only if there.
func perkFlaw(pf *entities.PerkFlaw) string {
	r := pf.Name
	if pf.HasDetail {
		r += ": " + pf.Detail
	}
	if pf.HasCountsAs {
		r += " x" + trimImg(pf.CountsAs)
	}
	if pf.HasCost {
		r += " (" + numberImg(pf.Cost) + ")"
	}
	return r
}

// modifiersBracket is " [Perks: ...; Flaws: ...; 5 Endurance]", with only
// the parts there are; "" if none is.
func modifiersBracket(s *entities.SubItem) string {
	var parts []string
	if len(s.Perks) > 0 {
		parts = append(parts, "Perks: "+join(s.Perks, perkFlaw))
	}
	if len(s.Flaws) > 0 {
		parts = append(parts, "Flaws: "+join(s.Flaws, perkFlaw))
	}
	if s.HasEndurance {
		parts = append(parts, marked(trimImg(s.Endurance), entities.OvEndurance, s.Overrides)+" Endurance")
	}
	if len(parts) == 0 {
		return ""
	}
	return " [" + strings.Join(parts, "; ") + "]"
}

// attackClause is "(Affinity: ...; Perks: ...; Flaws: ...)", with only
// the clauses there are; "" if none is.
func attackClause(a *entities.Attack) string {
	var parts []string
	if a.HasAffinity {
		parts = append(parts, "Affinity: "+a.Affinity)
	}
	if len(a.Perks) > 0 {
		parts = append(parts, "Perks: "+join(a.Perks, perkFlaw))
	}
	if len(a.Flaws) > 0 {
		parts = append(parts, "Flaws: "+join(a.Flaws, perkFlaw))
	}
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, "; ") + ")"
}

// ---- Item lines ----

// subItemLine is "Name (detail) +L [Perks: ...; Flaws: ...] (note)": an
// Ability or a SubItem.
func subItemLine(s *entities.SubItem) string {
	r := nameDetail(s.Name, s.HasDetail, s.Detail) + " " + signedImg(s.Level) + modifiersBracket(s)
	if s.HasNote {
		r += " (" + s.Note + ")"
	}
	return r
}

// weaknessLine is "Name (detail) +L".
func weaknessLine(w *entities.Weakness) string {
	return nameDetail(w.Name, w.HasDetail, w.Detail) + " " + signedImg(w.Level)
}

// fragmentSubItemLine is an NPC Fragment's order, "Name +L (detail)
// [...] (note)".
func fragmentSubItemLine(s *entities.SubItem) string {
	r := s.Name + " " + signedImg(s.Level)
	if s.HasDetail {
		r += " (" + s.Detail + ")"
	}
	r += modifiersBracket(s)
	if s.HasNote {
		r += " (" + s.Note + ")"
	}
	return r
}

// fragmentWeaknessLine is "Name +L (detail)".
func fragmentWeaknessLine(w *entities.Weakness) string {
	r := w.Name + " " + signedImg(w.Level)
	if w.HasDetail {
		r += " (" + w.Detail + ")"
	}
	return r
}

// spellLine is "Name (Ability +L; E Endurance)".
func spellLine(s *entities.Spell) string {
	return s.Name + " (" + s.Ability + " " + signedImg(s.Level) + "; " +
		marked(s.Endurance, entities.OvEndurance, s.Overrides) + " Endurance)"
}

func joinSubItems(l []*entities.SubItem) string { return join(l, subItemLine) }

func joinAbilities(l []*entities.Ability) string {
	return join(l, func(a *entities.Ability) string { return subItemLine(&a.SubItem) })
}

func joinFragmentAbilities(l []*entities.Ability) string {
	return join(l, func(a *entities.Ability) string { return fragmentSubItemLine(&a.SubItem) })
}

func joinWeaknesses(l []*entities.Weakness) string { return join(l, weaknessLine) }

// semi is a, then "; " if both a and b are there, then b.
func semi(a, b string, haveA, haveB bool) string {
	if haveA && haveB {
		return a + "; " + b
	}
	return a + b
}

// ---- Combat ----

// attackLine is "**Name:** Roll **N dice** | **DX d** | **Endurance
// Cost: c** (clause). *Description*", with no bullet.
func attackLine(a *entities.Attack) string {
	r := bold(a.Name+":") + " Roll " + bold(marked(numberImg(a.Roll), entities.OvRoll, a.Overrides)+" dice") +
		" | " + bold("DX "+marked(a.DX, entities.OvDx, a.Overrides)) +
		" | " + bold("Endurance Cost: "+marked(trimImg(a.Endurance), entities.OvEndurance, a.Overrides))
	if clause := attackClause(a); clause != "" {
		r += " " + clause
	}
	r += "."
	if a.HasDescription {
		r += " " + italic(a.Description)
	}
	return r
}

// combatSummaryLine is "**Health:** H | **Endurance:** E | **Defense:** D
// | **Threat Value (TV):** T", an em dash for a missing TV; led by "**NPC
// Type:** T | " if npcType isn't "".
func combatSummaryLine(c *entities.Combat, npcType string) string {
	r := ""
	if npcType != "" {
		r = bold("NPC Type:") + " " + npcType + " | "
	}
	r += bold("Health:") + " " + marked(trimImg(c.Health), entities.OvHealth, c.Overrides) +
		" | " + bold("Endurance:") + " " + marked(trimImg(c.Endurance), entities.OvEndurance, c.Overrides) +
		" | " + bold("Defense:") + " " + defenseText(c) +
		" | " + bold("Threat Value (TV):") + " "
	if c.HasThreatValue {
		r += marked(trimImg(c.ThreatValue), entities.OvThreatValue, c.Overrides)
	} else {
		r += "—"
	}
	return r
}

// combatAndAttacks is the combat summary, then the attacks, each a
// sub-bullet, and the combat notes, a paragraph.
func combatAndAttacks(b *strings.Builder, c *entities.Combat, npcType string) {
	if !c.HasCombat {
		return
	}
	line(b, bullet+combatSummaryLine(c, npcType))
	b.WriteByte('\n')
	if len(c.Attacks) > 0 {
		line(b, bullet+bold("Attacks:"))
		b.WriteByte('\n')
		for _, a := range c.Attacks {
			line(b, "  - "+attackLine(a))
		}
		b.WriteByte('\n')
	}
	if c.HasNotes {
		line(b, bold("Combat Notes:")+" "+c.Notes)
		b.WriteByte('\n')
	}
}

// ---- Linked content ----

// subEntityFragmentStyle is a Companion or Vehicle with fragment: true.
func subEntityFragmentStyle(b *strings.Builder, se *entities.SubEntity) {
	if len(se.Abilities) == 0 && len(se.Weaknesses) == 0 {
		line(b, bold(se.Name)+".")
	} else {
		line(b, bold(se.Name)+" -- "+bold("Attributes:")+" "+
			semi(join(se.Abilities, fragmentSubItemLine), join(se.Weaknesses, fragmentSubItemLine),
				len(se.Abilities) > 0, len(se.Weaknesses) > 0)+".")
	}
	b.WriteByte('\n')
	combatAndAttacks(b, &se.Combat, "")
}

func hasLinked(a *entities.Ability) bool {
	return len(a.SubAbilities) > 0 || len(a.SubWeaknesses) > 0 || len(a.Spells) > 0 || a.SubEntity != nil
}

// linkedSection is a section for an Ability's linked content.
func linkedSection(b *strings.Builder, a *entities.Ability) {
	if !hasLinked(a) {
		return
	}
	se := a.SubEntity
	if se != nil && se.IsFragment {
		subEntityFragmentStyle(b, se)
		return
	}
	if se != nil {
		line(b, sectionHeader(se.Label+": "+se.Name))
	} else {
		line(b, sectionHeader(nameDetail(a.Name, a.HasDetail, a.Detail)))
	}
	b.WriteByte('\n')
	if len(a.SubAbilities) > 0 || len(a.SubWeaknesses) > 0 {
		line(b, endWithPeriod("Grants: "+semi(joinSubItems(a.SubAbilities), joinSubItems(a.SubWeaknesses),
			len(a.SubAbilities) > 0, len(a.SubWeaknesses) > 0)))
		b.WriteByte('\n')
	}
	if len(a.Spells) > 0 {
		line(b, endWithPeriod("Spells: "+join(a.Spells, spellLine)))
		b.WriteByte('\n')
	}
	if se != nil {
		if len(se.Abilities) > 0 {
			line(b, endWithPeriod("Abilities: "+joinSubItems(se.Abilities)))
			b.WriteByte('\n')
		}
		if len(se.Weaknesses) > 0 {
			line(b, endWithPeriod("Weaknesses: "+joinSubItems(se.Weaknesses)))
			b.WriteByte('\n')
		}
		combatAndAttacks(b, &se.Combat, "")
	}
}

func linkedSections(b *strings.Builder, e *entities.Entity) {
	for _, a := range e.Abilities {
		linkedSection(b, a)
	}
}

// ---- Header, prose and tail ----

// fragmentHeader: an NPC Fragment's name is in its background's prose;
// only without one is the bare name written, bold.
func fragmentHeader(b *strings.Builder, e *entities.Entity) {
	if e.HasBackground {
		line(b, e.Background)
	} else {
		line(b, bold(e.Name))
	}
	b.WriteByte('\n')
}

func npcTypeText(e *entities.Entity) string {
	if e.HasNpcType {
		return entities.NpcTypeName(e.NpcType)
	}
	return ""
}

func fragmentAttributes(b *strings.Builder, e *entities.Entity) {
	if len(e.Abilities) == 0 && len(e.Weaknesses) == 0 {
		return
	}
	label := bold("Attributes:")
	if e.HasNpcType {
		label = bold("Attributes (" + npcTypeText(e) + "):")
	}
	line(b, label+" "+semi(joinFragmentAbilities(e.Abilities), join(e.Weaknesses, fragmentWeaknessLine),
		len(e.Abilities) > 0, len(e.Weaknesses) > 0)+".")
	b.WriteByte('\n')
}

func characterHooks(b *strings.Builder, e *entities.Entity) {
	if e.HasCharacterHooks {
		line(b, sectionHeader("Character Hooks"))
		b.WriteByte('\n')
		line(b, e.CharacterHooks)
		b.WriteByte('\n')
	}
}

// title is "Name (concept)", or "Name", then ", played by Player" with a
// player:.
func title(e *entities.Entity) string {
	r := nameDetail(e.Name, e.HasFluff && e.HasFluffField[entities.Concept], e.Fluff[entities.Concept])
	if e.HasPlayer {
		r += ", played by " + e.Player
	}
	return r
}

// section writes a prose section: its header, then its text.
func section(b *strings.Builder, has bool, header, text string) {
	if has {
		line(b, sectionHeader(header))
		b.WriteByte('\n')
		line(b, text)
		b.WriteByte('\n')
	}
}

// headerAndProse is the title, "# Title", then the tagline, summary,
// background, appearance, personality and character hooks.
func headerAndProse(b *strings.Builder, e *entities.Entity) {
	line(b, "# "+title(e))
	b.WriteByte('\n')
	if e.HasTagline {
		line(b, italic(`"`+e.Tagline+`"`))
		b.WriteByte('\n')
	}
	if e.HasSummary {
		line(b, italic(e.Summary))
		b.WriteByte('\n')
	}
	section(b, e.HasBackground, "Background", e.Background)
	section(b, e.HasAppearance, "Appearance", e.Appearance)
	section(b, e.HasPersonality, "Personality", e.Personality)
	characterHooks(b, e)
}

var fluffLabels = [entities.NFluff]string{
	"Concept", "Birthday", "Age", "Gender", "Height", "Weight", "Hair", "Eyes", "Association",
	"Hobbies", "Favorite Food", "Allies", "Enemies", "Love Interests",
}

// fluffLine is "**Concept:** ..., **Age:** ...", every fluff field there
// is, without the period.
func fluffLine(e *entities.Entity) string {
	var parts []string
	for i := 0; i < entities.NFluff; i++ {
		if e.HasFluffField[i] {
			parts = append(parts, bold(fluffLabels[i]+":")+" "+e.Fluff[i])
		}
	}
	return strings.Join(parts, ", ")
}

// tail is the notes, the other notes and the fluff; the creation rules
// only with -O, which the sheet doesn't have.
func tail(b *strings.Builder, e *entities.Entity) {
	if len(e.Notes) > 0 {
		line(b, sectionHeader("Notes"))
		b.WriteByte('\n')
		for _, n := range e.Notes {
			line(b, bold(n.Header)+" -- "+n.Text)
			b.WriteByte('\n')
		}
	}
	section(b, e.HasOtherNotes, "Other Notes", e.OtherNotes)
	if e.HasFluff {
		line(b, sectionHeader("Fluff"))
		b.WriteByte('\n')
		line(b, endWithPeriod(fluffLine(e)))
		b.WriteByte('\n')
	}
}
