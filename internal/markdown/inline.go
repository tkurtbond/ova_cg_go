package markdown

// The inline format, ova_fmt_po -i's (its OvaInline, which ova_cg_po
// copies, and the Ada ova_fmt's Format_Inline): each entity as one
// Markdown paragraph, a stat block for an adventure's cast list:
//
//	**Name** (Secondary). Summary. Background. *Ability +2 (detail),
//	...; Weakness −1, ...* Attack (Perks; Flaws): Roll 4, DX 3, End
//	0. Defense 3. Health 20, Endurance 20. TV 6.
//
// Abilities and Weaknesses, a Companion's or Vehicle's and those a
// Transformation grants too, are "Name Level (detail)", the NPC Fragment
// order. The prose that describes the entity runs together, as do its
// combat notes; the tagline, character hooks, notes, fluff, options, the
// player and the portrait are left out. An NPC Fragment's background
// names it already, so it leads instead of the bold name. An Ability's
// linked content follows as a paragraph of its own. Each paragraph is
// filled to 70 columns, counting characters, not bytes, and never breaks
// a line before a word that would start a Markdown block there. A
// count: over 1 makes the lead "**Name** (three Secondaries)." and the
// end "TV 6 each.": one stat block standing for a group.
// test/files.sh checks it is ova_fmt_po -i's of the saved file, byte for
// byte.

import (
	"strings"
	"unicode/utf8"

	"github.com/tkurtbond/ova_cg_go/internal/character"
	"github.com/tkurtbond/ova_cg_go/internal/entities"
	"github.com/tkurtbond/ova_cg_go/internal/save"
)

const inlineWidth = 70

// Inline is c as an inline paragraph; "" if c couldn't be read back (a
// programmer's error).
func Inline(c *character.Character) string {
	e := save.ToEntity(c)
	if e == nil {
		return ""
	}
	return InlineEntity(e)
}

// InlineEntity is e's paragraphs: OvaInline.ProcessEntity's.
func InlineEntity(e *entities.Entity) string {
	var b strings.Builder
	var p string
	group := e.Count > 1
	if e.IsFragment && e.HasBackground {
		p = addProse(p, e.Background)
		switch {
		case e.HasNpcType && group:
			p = add(p, npcTypeText(e)+" NPCs ("+countText(e.Count)+").")
		case e.HasNpcType:
			p = add(p, npcTypeText(e)+" NPC.")
		case group:
			p = add(p, "("+countText(e.Count)+").")
		}
	} else {
		if group {
			r := countText(e.Count)
			if e.HasNpcType {
				r += " " + npcTypePlural(e)
			}
			p = bold(e.Name) + " (" + r + ")."
		} else if e.HasNpcType {
			p = bold(e.Name) + " (" + npcTypeText(e) + ")."
		} else {
			p = bold(e.Name + ".")
		}
		if e.HasSummary {
			p = addProse(p, e.Summary)
		}
		if e.HasBackground {
			p = addProse(p, e.Background)
		}
		if e.HasAppearance {
			p = addProse(p, e.Appearance)
		}
		if e.HasPersonality {
			p = addProse(p, e.Personality)
		}
	}
	if t := traits(joinFragmentAbilities(e.Abilities), join(e.Weaknesses, fragmentWeaknessLine)); t != "" {
		p = add(p, t)
	}
	p = addCombat(p, &e.Combat, group)
	paragraph(&b, p)
	for _, a := range e.Abilities {
		linked(&b, a)
	}
	return b.String()
}

// add is p, then a blank unless p is empty, then s.
func add(p, s string) string {
	if p == "" {
		return s
	}
	return p + " " + s
}

// sentence is s ended with a period, unless it ends a sentence already:
// with ".", "!" or "?", perhaps then a closing quote or parenthesis.
func sentence(s string) string {
	t := strings.TrimRight(s, `"')`)
	if strings.HasSuffix(t, ".") || strings.HasSuffix(t, "!") || strings.HasSuffix(t, "?") {
		return s
	}
	return s + "."
}

// addProse is p, then the prose s, on one line, as a sentence.
func addProse(p, s string) string {
	return add(p, sentence(strings.ReplaceAll(s, "\n", " ")))
}

// ---- Filling ----

// opensBlock is whether the word w, at the start of a line, would make
// it something other than a paragraph's text: a heading, a block quote,
// a list item (a lone "-", "+" or "*", or digits and a "." or ")"), a
// thematic break or a setext underline (nothing but "-", "=", "*" or
// "_").
func opensBlock(w string) bool {
	if w[0] == '#' || w[0] == '>' || w == "+" {
		return true
	}
	if strings.Trim(w, "-=*_") == "" {
		return true
	}
	digits := strings.TrimLeft(w, "0123456789")
	return len(digits) < len(w) && (digits == "." || digits == ")")
}

// paragraph writes p filled to inlineWidth columns, then an empty line.
// Words are split at blanks, and a run of blanks is one. A word that
// would open a Markdown block at the start of a line stays on the line
// of the word before it, so the break comes before that one instead. A
// word, or words kept together, longer than inlineWidth has a line of
// its own.
func paragraph(b *strings.Builder, p string) {
	words := strings.FieldsFunc(p, func(r rune) bool { return r == ' ' })
	col := 0
	for i := 0; i < len(words); {
		// Take in the words that can't start a line.
		group := words[i]
		for i++; i < len(words) && opensBlock(words[i]); i++ {
			group += " " + words[i]
		}
		w := utf8.RuneCountInString(group)
		if col > 0 {
			if col+1+w > inlineWidth {
				b.WriteByte('\n')
				col = 0
			} else {
				b.WriteByte(' ')
				col++
			}
		}
		b.WriteString(group)
		col += w
	}
	b.WriteString("\n\n")
}

// ---- Parts ----

// traits is "*abilities; weaknesses.*", "" if there are neither.
func traits(abilities, weaknesses string) string {
	if abilities == "" && weaknesses == "" {
		return ""
	}
	return italic(semi(abilities, weaknesses, abilities != "", weaknesses != "") + ".")
}

// inlineAttack is "Name (Affinity: a; perks; flaws): Roll N, DX d, End
// c.", then its description, with only the parenthesized parts there
// are.
func inlineAttack(a *entities.Attack) string {
	var parts []string
	if a.HasAffinity {
		parts = append(parts, "Affinity: "+a.Affinity)
	}
	if len(a.Perks) > 0 {
		parts = append(parts, join(a.Perks, perkFlaw))
	}
	if len(a.Flaws) > 0 {
		parts = append(parts, join(a.Flaws, perkFlaw))
	}
	r := a.Name
	if len(parts) > 0 {
		r += " (" + strings.Join(parts, "; ") + ")"
	}
	r += ": Roll " + marked(numberImg(a.Roll), entities.OvRoll, a.Overrides) +
		", DX " + marked(a.DX, entities.OvDx, a.Overrides) +
		", End " + marked(trimImg(a.Endurance), entities.OvEndurance, a.Overrides) + "."
	if a.HasDescription {
		r = addProse(r, a.Description)
	}
	return r
}

// countWords are the counts prose spells out.
var countWords = [...]string{"", "one", "two", "three", "four", "five", "six",
	"seven", "eight", "nine", "ten", "eleven", "twelve"}

// countText is n as a word, up to twelve; past that its digits: the Ada
// Count_Text.
func countText(n int) string {
	if n < len(countWords) {
		return countWords[n]
	}
	return trimImg(n)
}

// npcTypePlural is e's NPC type's plural, as a count of them has it:
// "Secondaries".
func npcTypePlural(e *entities.Entity) string {
	if e.NpcType == entities.Secondary {
		return "Secondaries"
	}
	return npcTypeText(e) + "s"
}

// addCombat is p, then c's attacks, "Defense D. Health H, Endurance E.
// TV T." and its notes; p if there is no combat.
func addCombat(p string, c *entities.Combat, group bool) string {
	if !c.HasCombat {
		return p
	}
	for _, a := range c.Attacks {
		p = add(p, inlineAttack(a))
	}
	s := "Defense " + defenseText(c) +
		". Health " + marked(trimImg(c.Health), entities.OvHealth, c.Overrides) +
		", Endurance " + marked(trimImg(c.Endurance), entities.OvEndurance, c.Overrides) + "."
	if c.HasThreatValue {
		s += " TV " + marked(trimImg(c.ThreatValue), entities.OvThreatValue, c.Overrides)
		if group {
			s += " each"
		}
		s += "."
	}
	p = add(p, s)
	if c.HasNotes {
		p = addProse(p, c.Notes)
	}
	return p
}

// linked writes a paragraph for a's linked content, led by its title in
// bold: "**Companion: Name.**", or the Ability's name and detail.
func linked(b *strings.Builder, a *entities.Ability) {
	if !hasLinked(a) {
		return
	}
	se := a.SubEntity
	var p string
	if se != nil {
		p = bold(se.Label + ": " + se.Name + ".")
	} else {
		p = bold(nameDetail(a.Name, a.HasDetail, a.Detail) + ".")
	}
	if len(a.SubAbilities) > 0 || len(a.SubWeaknesses) > 0 {
		p = add(p, sentence("Grants: "+semi(join(a.SubAbilities, fragmentSubItemLine),
			join(a.SubWeaknesses, fragmentSubItemLine), len(a.SubAbilities) > 0, len(a.SubWeaknesses) > 0)))
	}
	if len(a.Spells) > 0 {
		p = add(p, sentence("Spells: "+join(a.Spells, spellLine)))
	}
	if se != nil {
		if t := traits(join(se.Abilities, fragmentSubItemLine), join(se.Weaknesses, fragmentSubItemLine)); t != "" {
			p = add(p, t)
		}
		p = addCombat(p, &se.Combat, false)
	}
	paragraph(b, p)
}
