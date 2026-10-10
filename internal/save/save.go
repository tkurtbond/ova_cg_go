// Package save writes a character as YAML in ova_fmt's format.
//
// po_ova_cg's PLAN.md, "Files". The document is built as yaml.v3 nodes,
// never written as text here, so the quoting is the emitter's: text is
// tagged !!str, so the emitter quotes it wherever plain YAML would read
// it as something else (#1 fan, true, 10). Text of more than one line
// is a literal block, as in ova_fmt's fixtures; any other is plain where
// that is safe, else quoted. Numbers, booleans and keys are plain.
//
// One document, a sequence of one entity, its keys in the order of
// ova_fmt's fixtures. The numbers written are the ones in effect: an
// override where there is one, which the mapping's overrides: key then
// lists. Only keys both ova_fmts read are written (AGENTS.md).
//
// A failed save leaves the old file as it was: the document goes to
// NAME.tmp, which then replaces NAME.
//
// The port of po_ova_cg's OvaSave.
package save

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/tkurtbond/ova_cg/internal/calc"
	"github.com/tkurtbond/ova_cg/internal/character"
	"github.com/tkurtbond/ova_cg/internal/entities"
	"github.com/tkurtbond/ova_cg/internal/rules"
)

// ---- Scalars ----

func text(s string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
	if strings.Contains(s, "\n") {
		n.Style = yaml.LiteralStyle
	}
	return n
}

func plain(s string) *yaml.Node { return &yaml.Node{Kind: yaml.ScalarNode, Value: s} }

func integer(i int) *yaml.Node { return plain(fmt.Sprint(i)) }

func boolean(b bool) *yaml.Node {
	if b {
		return plain("true")
	}
	return plain("false")
}

// num is n in effect: "4", "1/2", "10+".
func num(n *calc.Number) *yaml.Node { return plain(n.Text()) }

// ---- Mappings ----

func mapping() *yaml.Node  { return &yaml.Node{Kind: yaml.MappingNode} }
func sequence() *yaml.Node { return &yaml.Node{Kind: yaml.SequenceNode} }

func pair(m *yaml.Node, key string, v *yaml.Node) { m.Content = append(m.Content, plain(key), v) }

func appendTo(seq, v *yaml.Node) { seq.Content = append(seq.Content, v) }

// over adds k to *set if n is overridden.
func over(set *entities.Overrides, k entities.Override, n *calc.Number) {
	if n.Overridden {
		*set = set.With(k)
	}
}

// overrides is "overrides: [...]", the keys in set in entities' order,
// if any.
func overrides(m *yaml.Node, set entities.Overrides) {
	if set == 0 {
		return
	}
	seq := sequence()
	seq.Style = yaml.FlowStyle
	for k := entities.OvRoll; k <= entities.OvThreatValue; k++ {
		if set.Has(k) {
			appendTo(seq, plain(entities.KeyName(k)))
		}
	}
	pair(m, "overrides", seq)
}

func textPair(m *yaml.Node, key string, has bool, s string) {
	if has {
		pair(m, key, text(s))
	}
}

// ---- Perks and Flaws ----

// modNode is a plain name, as the fixtures write most; a mapping when it
// has a detail, a count or a cost to say.
func modNode(x *calc.Mod) *yaml.Node {
	m := character.ModOf(x)
	if !m.HasDetail && m.Times == 1 && !m.HasCost {
		return text(m.Name)
	}
	n := mapping()
	pair(n, "name", text(m.Name))
	textPair(n, "detail", m.HasDetail, m.Detail)
	if m.Times != 1 {
		pair(n, "counts-as", integer(m.Times))
	}
	if m.HasCost {
		pair(n, "cost", integer(m.Cost))
	}
	return n
}

// mods is the Perks (flaws false) or Flaws of list, under key, if any.
func mods(m *yaml.Node, key string, list []*calc.Mod, flaws bool) {
	var seq *yaml.Node
	for _, x := range list {
		if character.ModOf(x).IsFlaw == flaws {
			if seq == nil {
				seq = sequence()
			}
			appendTo(seq, modNode(x))
		}
	}
	if seq != nil {
		pair(m, key, seq)
	}
}

// ---- Traits ----

func traitNode(x *calc.Trait) *yaml.Node {
	t := character.TraitOf(x)
	n := mapping()
	pair(n, "name", text(t.Name))
	pair(n, "level", integer(t.Level))
	textPair(n, "detail", t.HasDetail, t.Detail)
	mods(n, "perks", t.Mods, false)
	mods(n, "flaws", t.Mods, true)
	if t.HasEnd || t.End.Overridden {
		pair(n, "endurance", num(&t.End))
	}
	textPair(n, "note", t.HasNote, t.Note)
	if t.HasHalfTV {
		pair(n, "threat-value-half", boolean(t.HalfTV))
	}
	var set entities.Overrides
	over(&set, entities.OvEndurance, &t.End)
	overrides(n, set)
	traits(n, "abilities", t.Form, rules.Ability)
	traits(n, "weaknesses", t.Form, rules.Weakness)
	if len(t.Spells) > 0 {
		seq := sequence()
		for _, sp := range t.Spells {
			s := character.SpellOf(sp)
			sn := mapping()
			pair(sn, "name", text(s.Name))
			pair(sn, "ability", text(s.Ability))
			pair(sn, "level", integer(s.Level))
			pair(sn, "endurance", num(&s.End))
			var set entities.Overrides
			over(&set, entities.OvEndurance, &s.End)
			overrides(sn, set)
			appendTo(seq, sn)
		}
		pair(n, "spells", seq)
	}
	if t.Partner != nil {
		p := character.SheetOf(t.Partner)
		pair(n, entities.FoldCI(p.Label), partner(p)) // "companion" or "vehicle"
	}
	return n
}

// traits is the Traits of list of kind, under key, if any.
func traits(m *yaml.Node, key string, list []*calc.Trait, kind rules.Kind) {
	var seq *yaml.Node
	for _, t := range list {
		if t.Kind == kind {
			if seq == nil {
				seq = sequence()
			}
			appendTo(seq, traitNode(t))
		}
	}
	if seq != nil {
		pair(m, key, seq)
	}
}

// ---- Combat ----

func combat(m *yaml.Node, c *character.Character) {
	if !c.HasCombat {
		return
	}
	n := mapping()
	def := c.Defense.Text()
	if c.HasBarrier || c.Barrier.Overridden {
		def += "/" + c.Barrier.Text()
	}
	pair(n, "defense", plain(def))
	pair(n, "health", num(&c.Health))
	pair(n, "endurance", num(&c.Endurance))
	if c.HasThreatValue {
		pair(n, "threat-value", num(&c.TV))
	}
	if c.Transformed {
		pair(n, "transformed", boolean(true))
	}
	var set entities.Overrides
	over(&set, entities.OvDefense, &c.Defense)
	over(&set, entities.OvBarrier, &c.Barrier)
	over(&set, entities.OvHealth, &c.Health)
	over(&set, entities.OvEndurance, &c.Endurance)
	if c.HasThreatValue {
		over(&set, entities.OvThreatValue, &c.TV)
	}
	overrides(n, set)
	textPair(n, "notes", c.HasCombatNotes, c.CombatNotes)
	if len(c.Attacks) > 0 {
		seq := sequence()
		for _, x := range c.Attacks {
			a := character.AttackOf(x)
			an := mapping()
			pair(an, "name", text(a.Name))
			mods(an, "perks", a.Mods, false)
			textPair(an, "affinity", a.HasAffinity, a.Affinity)
			mods(an, "flaws", a.Mods, true)
			textPair(an, "description", a.HasDescription, a.Description)
			pair(an, "roll", num(&a.Roll))
			pair(an, "dx", num(&a.DX))
			pair(an, "endurance", num(&a.End))
			if a.Telekinetic {
				pair(an, "telekinesis", boolean(true))
			}
			if a.Transformed {
				pair(an, "transformed", boolean(true))
			}
			var set entities.Overrides
			over(&set, entities.OvRoll, &a.Roll)
			over(&set, entities.OvDx, &a.DX)
			over(&set, entities.OvEndurance, &a.End)
			overrides(an, set)
			appendTo(seq, an)
		}
		pair(n, "attacks", seq)
	}
	pair(m, "combat", n)
}

// ---- Entities ----

// partner is a Companion's or Vehicle's mapping.
func partner(c *character.Character) *yaml.Node {
	m := mapping()
	pair(m, "name", text(c.Name))
	if c.IsFragment {
		pair(m, "fragment", boolean(true))
	}
	traits(m, "abilities", c.Traits, rules.Ability)
	traits(m, "weaknesses", c.Traits, rules.Weakness)
	combat(m, c)
	return m
}

func entity(c *character.Character) *yaml.Node {
	m := mapping()
	pair(m, "name", text(c.Name))
	if c.IsFragment {
		pair(m, "fragment", boolean(true))
	}
	textPair(m, "player", c.HasPlayer, c.Player)
	textPair(m, "portrait", c.HasPortrait, c.Portrait)
	textPair(m, "tagline", c.HasTagline, c.Tagline)
	textPair(m, "summary", c.HasSummary, c.Summary)
	textPair(m, "background", c.HasBackground, c.Background)
	textPair(m, "appearance", c.HasAppearance, c.Appearance)
	textPair(m, "personality", c.HasPersonality, c.Personality)
	textPair(m, "other-notes", c.HasOtherNotes, c.OtherNotes)
	if c.HasOptions {
		o := mapping()
		op := &c.Options
		if op.BaseZero {
			pair(o, "base-zero", integer(op.BaseZeroRange))
		}
		if op.PowerCeiling {
			pair(o, "power-ceiling", integer(op.Ceiling))
		}
		if op.StartingLevels {
			pair(o, "starting-levels", integer(op.Starting))
		}
		if op.ScaledCost {
			pair(o, "scaled-cost", boolean(true))
		}
		if op.HalfCostKnowledge {
			pair(o, "half-cost-knowledge", boolean(true))
		}
		if op.CombinedEffectiveness {
			pair(o, "combined-effectiveness", boolean(true))
		}
		pair(m, "options", o)
	}
	traits(m, "abilities", c.Traits, rules.Ability)
	traits(m, "weaknesses", c.Traits, rules.Weakness)
	combat(m, c)
	if len(c.Notes) > 0 {
		seq := sequence()
		for _, note := range c.Notes {
			n := mapping()
			pair(n, "header", text(note.Header))
			pair(n, "text", text(note.Text))
			appendTo(seq, n)
		}
		pair(m, "notes", seq)
	}
	if c.HasFluff {
		f := mapping()
		for i := range character.NFluff {
			if c.HasFluffField[i] {
				pair(f, entities.FluffKeys[i], text(c.Fluff[i]))
			}
		}
		pair(m, "fluff", f)
	}
	return m
}

// document is c's document.
func document(c *character.Character) *yaml.Node {
	root := sequence()
	appendTo(root, entity(c))
	return &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
}

func encode(c *character.Character) ([]byte, error) {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(document(c)); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// ToText is c as YAML text, for the tests.
func ToText(c *character.Character) string {
	b, err := encode(c)
	if err != nil {
		return ""
	}
	return string(b)
}

// Save writes c to the file at path, through path.tmp, which replaces it
// only once written. An error if it couldn't be, the old file left as it
// was.
func Save(c *character.Character, path string) error {
	b, err := encode(c)
	if err != nil {
		return fmt.Errorf("the character's YAML couldn't be built: %v", err)
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("couldn't replace %s with %s: %v", path, tmp, err)
	}
	return nil
}
