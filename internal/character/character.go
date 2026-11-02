// Package character is the character being built: calc's sheet, with
// its names and text.
//
// ova_cg_po's PLAN.md, "Files" and "Modules". A Character embeds a
// calc.Sheet, and its Traits, Mods, Attacks and Spells embed calc's,
// with what calc doesn't need: names, details, notes, descriptions, the
// background and the fluff, and the flags saying which optional keys a
// file had, so that a file loaded and saved formats the same in ova_fmt.
// Each calc record's Owner is the record that embeds it.
//
// A file is read by package entities, ova_fmt_po's loader, so the
// generator reads exactly what ova_fmt_po reads, and then built into a
// Character. A number the file's overrides: key lists is an override.
// Every other number the file holds is compared with the one calc
// computes; one that differs is kept as an override too, with a notice
// saying so: the file is the user's (a GM's ruling, a hand edit, or a
// case the rules here don't cover), and nothing in it is lost.
//
// The port of ova_cg_po's OvaCharacter. Imports nothing of the window,
// so its tests need no display.
package character

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"syscall"

	"go.yaml.in/yaml/v3"

	"github.com/tkurtbond/ova_cg_go/internal/calc"
	"github.com/tkurtbond/ova_cg_go/internal/entities"
	"github.com/tkurtbond/ova_cg_go/internal/rules"
	"github.com/tkurtbond/ova_cg_go/internal/spawn"
)

// NFluff is how many fluff fields there are.
const NFluff = entities.NFluff

// Slot is one of the Defense values, in the order a combat: block's
// defense holds them (BestDefense).
type Slot int

const (
	DefenseSlot Slot = iota
	BarrierSlot
	IncorporealSlot
)

// Note is a headed note.
type Note struct {
	Header, Text string
}

// Character is the character (or a partner's sheet) being built.
type Character struct {
	calc.Sheet
	Label                         string // a partner's sheet: "Companion" or "Vehicle"; "" for the character
	Name                          string
	HasPlayer, HasPortrait        bool
	Player, Portrait              string // Portrait: the image's path, relative to the file
	HasTagline, HasSummary        bool
	HasBackground, HasAppearance  bool
	HasPersonality, HasOtherNotes bool
	HasCharacterHooks             bool
	Tagline, Summary              string
	Background, Appearance        string
	Personality, OtherNotes       string
	CharacterHooks                string
	// npc-type: entities.Heroic, Secondary or Extra; it sets the sheet's
	// kind (SetNpcType).
	HasNpcType     bool
	NpcType        entities.NpcType
	Count          int // count:, how many it stands for; kept for the save
	Notes          []*Note
	HasFluff       bool
	HasFluffField  [NFluff]bool
	Fluff          [NFluff]string
	HasOptions     bool // an options: block, even an empty one
	IsFragment     bool // an NPC Fragment, or a simple Companion
	HasCombat      bool // a combat: block: a Fragment or Companion may have none
	HasThreatValue bool
	// The file gave one Defense though Barrier or Incorporeal Form gives
	// another, as the book's Sample NPCs have it: the best, saved so, the
	// others left out (SingleDefense). singleSlot, the slot it was read
	// into, BestDefense's then.
	OmitAlternates bool
	singleSlot     Slot
	HasCombatNotes bool
	CombatNotes    string
}

// Trait is an Ability or Weakness.
type Trait struct {
	calc.Trait
	Name      string
	HasDetail bool
	Detail    string
	HasNote   bool
	Note      string
	HasHalfTV bool // HalfTV is written: threat-value-half; else the rules'
}

// Mod is a Perk or Flaw.
type Mod struct {
	calc.Mod
	Name      string
	HasDetail bool
	Detail    string
	IsFlaw    bool // which list it is in, whether or not rules has it
	HasCost   bool // its cost is written: chosen (Cancel, Requirement, Unique), or the file's
}

// Attack is an attack.
type Attack struct {
	calc.Attack
	Name           string
	HasAffinity    bool
	Affinity       string
	HasDescription bool
	Description    string
}

// Spell is a Magic's spell.
type Spell struct {
	calc.Spell
	Name, Ability string
}

// The records calc's lists hold, back to the character's.

// TraitOf is t's Trait.
func TraitOf(t *calc.Trait) *Trait { return t.Owner.(*Trait) }

// ModOf is m's Mod.
func ModOf(m *calc.Mod) *Mod { return m.Owner.(*Mod) }

// AttackOf is a's Attack.
func AttackOf(a *calc.Attack) *Attack { return a.Owner.(*Attack) }

// SpellOf is sp's Spell.
func SpellOf(sp *calc.Spell) *Spell { return sp.Owner.(*Spell) }

// SheetOf is s's Character.
func SheetOf(s *calc.Sheet) *Character { return s.Owner.(*Character) }

// ---- Endurance Use ----

// names says whether name, or name and detail as "Name (detail)", is
// target, ignoring case.
func names(name string, hasDetail bool, detail, target string) bool {
	target = entities.FoldCI(entities.Trim(target))
	if target == "" {
		return false
	}
	if entities.FoldCI(name) == target {
		return true
	}
	return hasDetail && entities.FoldCI(name+" ("+detail+")") == target
}

// linkList ties each Endurance Use in list to the Ability in list, or
// else the attack of s, that its detail names (ova_cg_po's PLAN.md,
// "Changes to ova_fmt": it needs no key of its own), setting their Use;
// one naming nothing ties to nothing.
func linkList(list []*calc.Trait, s *calc.Sheet) {
	for _, w := range list {
		wt := TraitOf(w)
		if w.Kind != rules.Weakness || !w.Item.Has(rules.CostUp) || !wt.HasDetail {
			continue
		}
		found := false
		for _, t := range list {
			x := TraitOf(t)
			if t.Kind == rules.Ability && names(x.Name, x.HasDetail, x.Detail, wt.Detail) {
				t.Use = w.Level
				found = true
				break
			}
		}
		for _, a := range s.Attacks {
			if found {
				break
			}
			if names(AttackOf(a).Name, false, "", wt.Detail) {
				a.Use = w.Level
				found = true
			}
		}
	}
}

// link ties every Endurance Use of s, its Transformations' forms and
// its partners to what it names.
func link(s *calc.Sheet) {
	for _, t := range s.Traits {
		t.Use = 0
		for _, f := range t.Form {
			f.Use = 0
		}
	}
	for _, a := range s.Attacks {
		a.Use = 0
	}
	linkList(s.Traits, s)
	for _, t := range s.Traits {
		linkList(t.Form, s)
		if t.Partner != nil {
			link(t.Partner)
		}
	}
}

// Recompute computes every number of c again, after a change: what
// calc.Compute does, with each Endurance Use tied first.
func Recompute(c *Character) {
	link(&c.Sheet)
	calc.Compute(&c.Sheet)
}

// ---- Building from what entities read ----

func newCharacter(k calc.SheetKind) *Character {
	c := &Character{HasCombat: true, HasThreatValue: true, Count: 1}
	c.Sheet = calc.Sheet{Kind: k, Owner: c}
	return c
}

// New is a new, empty character.
func New() *Character { return newCharacter(calc.Character) }

// kindOf is the sheet kind of an NPC type (p. 123-124): a Heroic NPC's
// is a character's, a Secondary's a Companion's, an Extra's its own; no
// type is a character's.
func kindOf(has bool, k entities.NpcType) calc.SheetKind {
	switch {
	case !has || k == entities.Heroic:
		return calc.Character
	case k == entities.Secondary:
		return calc.CompanionSheet
	}
	return calc.ExtraSheet
}

// SetNpcType sets c's NPC type (has false: none), and so its sheet's
// kind: Recompute after.
func SetNpcType(c *Character, has bool, k entities.NpcType) {
	if has && (k < entities.Heroic || k > entities.Extra) {
		panic("character.SetNpcType: no such NPC type")
	}
	c.HasNpcType = has
	c.NpcType = entities.Heroic
	if has {
		c.NpcType = k
	}
	c.Kind = kindOf(has, k)
}

// mods adds l's Perks (isFlaw false) or Flaws to *list.
func mods(l []*entities.PerkFlaw, isFlaw bool, list *[]*calc.Mod) {
	for _, pf := range l {
		times := 1
		if pf.HasCountsAs {
			times = pf.CountsAs
		}
		cost := 0
		if pf.HasCost {
			cost = pf.Cost
		}
		m := &Mod{Name: pf.Name, HasDetail: pf.HasDetail, Detail: pf.Detail, IsFlaw: isFlaw, HasCost: pf.HasCost}
		calc.InitMod(&m.Mod, m, rules.Find(pf.Name), times, cost)
		*list = append(*list, &m.Mod)
	}
}

func subTrait(x *entities.SubItem, k rules.Kind) *Trait {
	t := &Trait{Name: x.Name, HasDetail: x.HasDetail, Detail: x.Detail, HasNote: x.HasNote, Note: x.Note,
		HasHalfTV: x.HasThreatValueHalf}
	calc.InitTrait(&t.Trait, t, rules.Find(x.Name), k, x.Level)
	if x.HasThreatValueHalf {
		t.HalfTV = x.ThreatValueHalf
	}
	mods(x.Perks, false, &t.Mods)
	mods(x.Flaws, true, &t.Mods)
	return t
}

func subTraits(l []*entities.SubItem, k rules.Kind, list *[]*calc.Trait) {
	for _, x := range l {
		*list = append(*list, &subTrait(x, k).Trait)
	}
}

func attacks(x *entities.Combat, c *Character) {
	c.HasCombat, c.HasThreatValue = x.HasCombat, x.HasThreatValue
	c.HasCombatNotes, c.CombatNotes = x.HasNotes, x.Notes
	c.Transformed = x.Transformed
	for _, y := range x.Attacks {
		a := &Attack{Name: y.Name, HasAffinity: y.HasAffinity, Affinity: y.Affinity,
			HasDescription: y.HasDescription, Description: y.Description}
		calc.InitAttack(&a.Attack, a)
		a.Telekinetic, a.Transformed = y.Telekinesis, y.Transformed
		mods(y.Perks, false, &a.Mods)
		mods(y.Flaws, true, &a.Mods)
		c.Attacks = append(c.Attacks, &a.Attack)
	}
}

// partner is a Companion's sheet (a Secondary NPC's), or a Vehicle's (a
// character's).
func partner(x *entities.SubEntity) *Character {
	k := calc.Character
	if x.Label == "Companion" {
		k = calc.CompanionSheet
	}
	c := newCharacter(k)
	c.Label, c.Name, c.IsFragment = x.Label, x.Name, x.IsFragment
	subTraits(x.Abilities, rules.Ability, &c.Traits)
	subTraits(x.Weaknesses, rules.Weakness, &c.Traits)
	attacks(&x.Combat, c)
	return c
}

// Build is e's character, every number computed, none overridden: its
// Abilities, then its Weaknesses, each in e's (sorted) order, its
// attacks in the file's.
func Build(e *entities.Entity) *Character {
	c := New()
	c.Name = e.Name
	c.HasPlayer, c.Player = e.HasPlayer, e.Player
	c.HasPortrait, c.Portrait = e.HasPortrait, e.Portrait
	c.HasTagline, c.Tagline = e.HasTagline, e.Tagline
	c.HasSummary, c.Summary = e.HasSummary, e.Summary
	c.HasBackground, c.Background = e.HasBackground, e.Background
	c.HasAppearance, c.Appearance = e.HasAppearance, e.Appearance
	c.HasPersonality, c.Personality = e.HasPersonality, e.Personality
	c.HasCharacterHooks, c.CharacterHooks = e.HasCharacterHooks, e.CharacterHooks
	SetNpcType(c, e.HasNpcType, e.NpcType)
	c.Count = e.Count
	c.HasOtherNotes, c.OtherNotes = e.HasOtherNotes, e.OtherNotes
	c.IsFragment = e.IsFragment
	for _, a := range e.Abilities {
		t := subTrait(&a.SubItem, rules.Ability)
		subTraits(a.SubAbilities, rules.Ability, &t.Form)
		subTraits(a.SubWeaknesses, rules.Weakness, &t.Form)
		for _, sp := range a.Spells {
			s := &Spell{Name: sp.Name, Ability: sp.Ability}
			calc.InitSpell(&s.Spell, s, rules.Find(sp.Ability), sp.Level)
			t.Spells = append(t.Spells, &s.Spell)
		}
		if a.SubEntity != nil {
			t.Partner = &partner(a.SubEntity).Sheet
		}
		c.Traits = append(c.Traits, &t.Trait)
	}
	for _, w := range e.Weaknesses {
		t := &Trait{Name: w.Name, HasDetail: w.HasDetail, Detail: w.Detail}
		calc.InitTrait(&t.Trait, t, rules.Find(w.Name), rules.Weakness, w.Level)
		c.Traits = append(c.Traits, &t.Trait)
	}
	attacks(&e.Combat, c)
	for _, x := range e.Notes {
		c.Notes = append(c.Notes, &Note{Header: x.Header, Text: x.Text})
	}
	c.HasFluff = e.HasFluff
	c.HasFluffField = e.HasFluffField
	c.Fluff = e.Fluff
	c.HasOptions = e.HasOptions
	if e.HasOptions {
		o := &e.Options
		c.Options = calc.Options{
			BaseZero: o.HasBaseZero, BaseZeroRange: o.BaseZero,
			PowerCeiling: o.HasPowerCeiling, Ceiling: o.PowerCeiling,
			StartingLevels: o.HasStartingLevels, Starting: o.StartingLevels,
			ScaledCost: o.ScaledCost, HalfCostKnowledge: o.HalfCostKnowledge,
			CombinedEffectiveness: o.CombinedEffectiveness,
		}
	}
	Recompute(c)
	return c
}

// ---- The file's numbers ----

type adopter struct{ notices []string }

// BestDefense is c's best Defense value, its slot: the Defense, or an
// alternate c has that is higher than those before it.
func BestDefense(c *Character) Slot {
	best, value := DefenseSlot, c.Defense.Get()
	if c.HasBarrier && c.Barrier.Get() > value {
		best, value = BarrierSlot, c.Barrier.Get()
	}
	if c.HasIncorporeal && c.Incorporeal.Get() > value {
		best = IncorporealSlot
	}
	return best
}

// SingleDefense says whether c's defense is saved as one value, and
// which, slot: the file gave one (OmitAlternates), and the only value
// overridden, if any, is the one it was read into. It is saved, or else
// the best.
func SingleDefense(c *Character) (slot Slot, single bool) {
	n := 0
	slot = BestDefense(c)
	if c.Defense.Overridden {
		n++
		slot = DefenseSlot
	}
	if c.Barrier.Overridden {
		n++
		slot = BarrierSlot
	}
	if c.Incorporeal.Overridden {
		n++
		slot = IncorporealSlot
	}
	return slot, c.OmitAlternates && (n == 0 || n == 1 && slot == c.singleSlot)
}

// adopt takes the file's text for n: n's override if listed (the
// mapping's overrides: names it), or if it differs from the computed
// number, with a notice. Text that isn't a number for n is noted and
// dropped, the computed number kept.
func (ad *adopter) adopt(n *calc.Number, file string, allowHalf, allowPlus, listed bool, where, what string) {
	computed := n.ComputedText()
	t := *n
	t.Clear()
	if !t.Parse(file, allowHalf, allowPlus) || !t.Overridden {
		ad.notices = append(ad.notices, fmt.Sprintf("%s: %s is %s in the file, not a number: %s by the rules is used",
			where, what, file, computed))
		return
	}
	got := t.Text()
	switch {
	case listed:
		*n = t
	case got != computed:
		*n = t
		ad.notices = append(ad.notices, fmt.Sprintf("%s: %s is %s in the file, %s by the rules: kept as an override",
			where, what, got, computed))
	}
}

func (ad *adopter) adoptInt(n *calc.Number, i int, listed bool, where, what string) {
	ad.adopt(n, fmt.Sprint(i), false, false, listed, where, what)
}

func within(where, name string) string { return where + ": " + name }

// adoptDefense takes the file's value v for Defense slot slot of c.
func (ad *adopter) adoptDefense(c *Character, slot Slot, v string, listed bool, where string) {
	switch slot {
	case DefenseSlot:
		ad.adopt(&c.Defense, v, false, false, listed, where, "DEFENSE")
	case BarrierSlot:
		ad.adopt(&c.Barrier, v, false, false, listed, where, "Barrier's DEFENSE")
	case IncorporealSlot:
		ad.adopt(&c.Incorporeal, v, false, false, listed, where, "Incorporeal Form's DEFENSE")
	}
}

// adoptDefenses takes the Defense values of x's defense: one is the best
// of c's, two its Defense and its first alternate, three its Defense,
// Barrier's and Incorporeal Form's (ova_fmt allows no more).
func (ad *adopter) adoptDefenses(x *entities.Combat, c *Character, where string) {
	v := strings.SplitN(x.Defense, "/", 3)
	if len(v) == 1 {
		c.OmitAlternates = c.HasBarrier || c.HasIncorporeal
		c.singleSlot = BestDefense(c)
		ad.adoptDefense(c, c.singleSlot, v[0], x.Overrides.Has(entities.OvDefense), where)
		return
	}
	ad.adoptDefense(c, DefenseSlot, v[0], x.Overrides.Has(entities.OvDefense), where)
	if len(v) == 3 {
		ad.adoptDefense(c, BarrierSlot, v[1], x.Overrides.Has(entities.OvDefense2), where)
		ad.adoptDefense(c, IncorporealSlot, v[2], x.Overrides.Has(entities.OvDefense3), where)
		return
	}
	second := BarrierSlot
	if !c.HasBarrier && c.HasIncorporeal {
		second = IncorporealSlot
	}
	ad.adoptDefense(c, second, v[1], x.Overrides.Has(entities.OvDefense2), where)
}

// adoptCombat takes the numbers of a combat: block but TV.
func (ad *adopter) adoptCombat(x *entities.Combat, c *Character, where string) {
	if !x.HasCombat {
		return
	}
	for i, y := range x.Attacks {
		a := c.Attacks[i]
		at := within(where, y.Name)
		ad.adoptInt(&a.Roll, y.Roll, y.Overrides.Has(entities.OvRoll), at, "ROLL")
		ad.adopt(&a.DX, y.DX, true, false, y.Overrides.Has(entities.OvDx), at, "DX")
		ad.adoptInt(&a.End, y.Endurance, y.Overrides.Has(entities.OvEndurance), at, "END")
	}
	ad.adoptDefenses(x, c, where)
	ad.adoptInt(&c.Health, x.Health, x.Overrides.Has(entities.OvHealth), where, "HEALTH")
	ad.adoptInt(&c.Endurance, x.Endurance, x.Overrides.Has(entities.OvEndurance), where, "ENDURANCE")
}

func (ad *adopter) adoptSubItems(l []*entities.SubItem, ts []*calc.Trait, where string) {
	for i, x := range l {
		if x.HasEndurance {
			ad.adoptInt(&ts[i].End, x.Endurance, x.Overrides.Has(entities.OvEndurance), within(where, x.Name), "END")
		}
	}
}

// adoptNumbers takes every number but TV, which is computed from them,
// and then TV.
func (ad *adopter) adoptNumbers(e *entities.Entity, c *Character) {
	for i, a := range e.Abilities {
		t := c.Traits[i]
		if a.HasEndurance {
			ad.adoptInt(&t.End, a.Endurance, a.Overrides.Has(entities.OvEndurance), within(c.Name, a.Name), "END")
		}
		// The form holds the sub-abilities first, then the
		// sub-weaknesses, which have no END.
		ad.adoptSubItems(a.SubAbilities, t.Form, within(c.Name, a.Name))
		for j, x := range a.Spells {
			ad.adopt(&t.Spells[j].End, x.Endurance, false, true, x.Overrides.Has(entities.OvEndurance),
				within(c.Name, x.Name), "END")
		}
		if a.SubEntity != nil {
			p := SheetOf(t.Partner)
			ad.adoptSubItems(a.SubEntity.Abilities, p.Traits, p.Name)
			ad.adoptCombat(&a.SubEntity.Combat, p, p.Name)
		}
	}
	ad.adoptCombat(&e.Combat, c, c.Name)
	Recompute(c)
	for i, a := range e.Abilities {
		if se := a.SubEntity; se != nil && se.Combat.HasCombat && se.Combat.HasThreatValue {
			p := SheetOf(c.Traits[i].Partner)
			ad.adoptInt(&p.TV, se.Combat.ThreatValue, se.Combat.Overrides.Has(entities.OvThreatValue), p.Name, "TV")
		}
	}
	if e.Combat.HasCombat && e.Combat.HasThreatValue {
		ad.adoptInt(&c.TV, e.Combat.ThreatValue, e.Combat.Overrides.Has(entities.OvThreatValue), c.Name, "TV")
	}
	Recompute(c)
}

// ---- Files ----

// openError is how libfyaml says a file can't be opened: "PATH: error:
// TEXT".
func openError(path string, err error) error {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return fmt.Errorf("%s: error: %s", path, spawn.ErrorText(errno))
	}
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return fmt.Errorf("%s: error: %s", path, spawn.ErrorText(pe.Err))
	}
	return fmt.Errorf("%s: error: %v", path, err)
}

// ReadEntities reads every entity of the file at path, of every
// document in it, in order, as ova_fmt_po reads them. An error if it
// can't be read, or entities finds a problem in any of them.
func ReadEntities(path string) ([]*entities.Entity, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, openError(path, err)
	}
	defer f.Close()
	var l entities.Loader
	var es []*entities.Entity
	dec := yaml.NewDecoder(f)
	for {
		var doc yaml.Node
		if err := dec.Decode(&doc); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("%s: %v", path, err)
		}
		var root *yaml.Node
		if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
			root = doc.Content[0]
			for root.Kind == yaml.AliasNode {
				root = root.Alias
			}
		}
		if root == nil || root.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("expected a top-level YAML sequence of entities in %s", path)
		}
		for i, item := range root.Content {
			e, err := l.Load(item, i)
			if err != nil {
				return nil, err
			}
			es = append(es, e)
		}
	}
	if len(l.Errors) > 0 {
		msgs := make([]string, len(l.Errors))
		for i, le := range l.Errors {
			msgs[i] = le.Error()
		}
		return nil, errors.New(strings.Join(msgs, "\n"))
	}
	return es, nil
}

// ReadEntity reads the first entity of the file at path, and n, how many
// it holds; as ReadEntities.
func ReadEntity(path string) (first *entities.Entity, n int, err error) {
	es, err := ReadEntities(path)
	if err != nil || len(es) == 0 {
		return nil, 0, err
	}
	return es[0], len(es), nil
}

// FromEntity is the character of entity e, its numbers e's: each its
// mapping's overrides: lists is an override, and each other that differs
// from the rules' is kept as one too, with a notice.
func FromEntity(e *entities.Entity) (c *Character, notices []string) {
	c = Build(e)
	var ad adopter
	ad.adoptNumbers(e, c)
	return c, ad.notices
}

// One is c alone, as a file's characters.
func One(c *Character) []*Character { return []*Character{c} }

// LoadAll reads every character in the file at path, in order, as
// FromEntity builds them, with the notices of all of them (each names
// its character). An error if the file can't be read, entities finds a
// problem, or it holds none.
func LoadAll(path string) (cs []*Character, notices []string, err error) {
	es, err := ReadEntities(path)
	if err != nil {
		return nil, nil, err
	}
	if len(es) == 0 {
		return nil, nil, fmt.Errorf("%s holds no characters", path)
	}
	for _, e := range es {
		c, n := FromEntity(e)
		cs = append(cs, c)
		notices = append(notices, n...)
	}
	return cs, notices, nil
}

// Load reads the one character in the file at path, as LoadAll; an
// error as LoadAll's, or if the file holds more than one.
func Load(path string) (c *Character, notices []string, err error) {
	cs, notices, err := LoadAll(path)
	if err != nil {
		return nil, nil, err
	}
	if len(cs) != 1 {
		return nil, nil, fmt.Errorf("%s holds %d characters, not one", path, len(cs))
	}
	return cs[0], notices, nil
}

// ---- Editing (the window's) ----
//
// Every change the window makes goes through these, then Recompute.
// Traits and Mods are kept in the order the file's loader sorts them (by
// name, then detail, ignoring case), so the window lists them as ova_fmt
// prints them; attacks keep the order they were added in, as a file's
// do.

// NewTrait is a new Ability or Weakness (kind), named for it, or name if
// it is nil (a name the tables don't have); detail "" for none.
func NewTrait(it *rules.Item, kind rules.Kind, level int, name, detail string) *Trait {
	t := &Trait{Name: name, HasDetail: detail != "", Detail: detail}
	calc.InitTrait(&t.Trait, t, it, kind, level)
	if it != nil {
		t.Name = it.Name
	}
	return t
}

// InsertTrait puts t into *list in its place: Abilities before
// Weaknesses, each sorted.
func InsertTrait(list *[]*calc.Trait, t *Trait) {
	i := 0
	for i < len(*list) {
		p := (*list)[i]
		x := TraitOf(p)
		if !(p.Kind < t.Kind || p.Kind == t.Kind && !entities.Before(t.Name, t.Detail, x.Name, x.Detail)) {
			break
		}
		i++
	}
	*list = append(*list, nil)
	copy((*list)[i+1:], (*list)[i:])
	(*list)[i] = &t.Trait
}

func remove[T comparable](list *[]T, x T) {
	for i, y := range *list {
		if y == x {
			*list = append((*list)[:i:i], (*list)[i+1:]...)
			return
		}
	}
}

// RemoveTrait takes t out of *list; nothing if it isn't there.
func RemoveTrait(list *[]*calc.Trait, t *calc.Trait) { remove(list, t) }

func linked(it *rules.Item) rules.Link {
	if it == nil {
		return rules.NoLink
	}
	return it.Linked
}

// ChangeTrait changes t to item it at level with detail, and moves it to
// its new place in *list. Its Perks and Flaws stay; its linked content
// stays if it keeps the same kind (Magic, Arcane to Magic, Witchcraft
// keeps the spells), and is dropped if not, as the file couldn't hold
// it.
func ChangeTrait(list *[]*calc.Trait, t *Trait, it *rules.Item, level int, detail string) {
	RemoveTrait(list, &t.Trait)
	if linked(it) != linked(t.Item) {
		t.Form, t.Spells, t.Partner = nil, nil, nil
	}
	if it != t.Item {
		t.Item = it
		if it != nil {
			t.Name = it.Name
		}
		t.HasHalfTV = false
		t.HalfTV = it.Has(rules.HalfTV)
	}
	t.Level = level
	t.HasDetail, t.Detail = detail != "", detail
	InsertTrait(list, t)
}

// NewPartner is a new sheet for t, a Companion (a Secondary NPC's
// sheet, p. 52) or a Vehicle (a character's, p. 64): named t's detail,
// or what it is; with a combat block, not a Fragment.
func NewPartner(t *Trait) *Character {
	var c *Character
	switch linked(t.Item) {
	case rules.Partner:
		c = newCharacter(calc.CompanionSheet)
		c.Label = "Companion"
	case rules.Machine:
		c = newCharacter(calc.Character)
		c.Label = "Vehicle"
	default:
		panic("character: NewPartner of a trait with no partner")
	}
	c.Name = c.Label
	if t.HasDetail {
		c.Name = t.Detail
	}
	return c
}

// NewSpell is a new spell named name, mimicking ability (the Ability's
// name; it, rules', or nil) at level.
func NewSpell(name, ability string, it *rules.Item, level int) *Spell {
	sp := &Spell{Name: name, Ability: ability}
	calc.InitSpell(&sp.Spell, sp, it, level)
	return sp
}

// RemoveSpell takes sp out of t's spells.
func RemoveSpell(t *calc.Trait, sp *calc.Spell) { remove(&t.Spells, sp) }

// NewAttack is a new attack named name, with no Perks or Flaws.
func NewAttack(name string) *Attack {
	a := &Attack{Name: name}
	calc.InitAttack(&a.Attack, a)
	return a
}

// RemoveAttack takes a out of c's attacks.
func RemoveAttack(c *Character, a *calc.Attack) { remove(&c.Attacks, a) }

// NewMod is a new Perk, or Flaw if isFlaw, taken times times at cost
// each (a RangeCost's or UserCost's; else ignored).
func NewMod(it *rules.Item, isFlaw bool, times, cost int) *Mod {
	m := &Mod{Name: it.Name, IsFlaw: isFlaw, HasCost: it.CostKind != rules.FixedCost}
	calc.InitMod(&m.Mod, m, it, times, cost)
	return m
}

// InsertMod puts m into *list in its place: Perks before Flaws, each
// sorted.
func InsertMod(list *[]*calc.Mod, m *Mod) {
	i := 0
	for i < len(*list) {
		p := ModOf((*list)[i])
		if !(!p.IsFlaw && m.IsFlaw || p.IsFlaw == m.IsFlaw && !entities.Before(m.Name, m.Detail, p.Name, p.Detail)) {
			break
		}
		i++
	}
	*list = append(*list, nil)
	copy((*list)[i+1:], (*list)[i:])
	(*list)[i] = &m.Mod
}

// RemoveMod takes m out of *list.
func RemoveMod(list *[]*calc.Mod, m *calc.Mod) { remove(list, m) }

// CopyMods is a copy of list, each Mod copied: what a dialog puts back
// on Cancel.
func CopyMods(list []*calc.Mod) []*calc.Mod {
	var r []*calc.Mod
	for _, x := range list {
		m := *ModOf(x)
		c := &m
		c.Mod.Owner = c
		r = append(r, &c.Mod)
	}
	return r
}

// SetText sets a text field and its flag: "" is none.
func SetText(has *bool, field *string, s string) {
	*field = s
	*has = s != ""
}
