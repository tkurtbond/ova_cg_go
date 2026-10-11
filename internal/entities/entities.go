// Package entities reads an OVA entity once from its YAML.
//
// The port of po_ova_fmt's OvaEntities (copied into po_ova_cg), itself
// the Ada Ova_Fmt.Entities: each entity (a character, or an NPC
// Fragment) is decoded from its YAML mapping once, by Load, into plain
// records. The Ada package's comments describe the schema and the
// reasons for its shape; the notes here say only what differs.
//
// An Ability can carry linked content: a Transformation's granted
// sub-abilities and sub-weaknesses, a Magic ability's spells, or a
// Companion's or Vehicle's whole sub-entity. That nesting is one level
// deep, so a SubEntity holds SubItems, never Abilities.
//
// Errors come in two kinds, as in the Ada version:
//
//   - A required field that is missing or malformed is recorded in the
//     Loader's Errors, with a placeholder in its place, and loading goes
//     on, so that every such problem in a file is reported together.
//     Each has a path, and an approximate location: that of its
//     mapping's own "name" scalar, when there is one.
//   - Anything else (an optional field that is malformed, YAML of the
//     wrong shape) ends loading: Load returns an error, and the caller
//     abandons the file.
//
// The YAML is read with go.yaml.in/yaml/v3 into nodes, but scalars are
// converted with polibfyaml's grammar (parseInt, parseBool), and paths
// are written in its ByPath syntax ("/1/abilities/2"), so this reads
// exactly what po_ova_fmt reads, and says it the same way.
package entities

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// The fluff: fields, in the order every format writes them (the Ada
// Fluff_Block's field order); index Fluff and HasFluffField with these.
const (
	Concept = iota
	Birthday
	Age
	Gender
	Height
	Weight
	Hair
	Eyes
	Association
	Hobbies
	FavoriteFood
	Allies
	Enemies
	LoveInterests
	NFluff
)

// FluffKeys are the fluff fields' YAML keys, by the constants above.
var FluffKeys = [NFluff]string{
	"concept", "birthday", "age", "gender", "height", "weight", "hair",
	"eyes", "association", "hobbies", "favorite-food", "allies", "enemies",
	"love-interests",
}

// Override is a number an overrides: list can name (the Ada
// Override_Key).
type Override uint

const (
	OvRoll Override = iota
	OvDx
	OvEndurance
	OvDefense
	OvDefense2
	OvDefense3
	OvHealth
	OvThreatValue
	nOverrides
)

var overrideKeys = [nOverrides]string{
	"roll", "dx", "endurance", "defense", "defense-2", "defense-3", "health", "threat-value",
}

// NpcType is an npc-type: (rulebook p. 133, "Using NPC Type"), an NPC's
// importance to the story: the Ada NPC_Kind. Matched ignoring case;
// anything else is a recorded error. See NpcTypeName.
type NpcType int

const (
	Heroic NpcType = iota
	Secondary
	Extra
	NNpcTypes
)

var npcTypeNames = [NNpcTypes]string{"Heroic", "Secondary", "Extra"}

// NpcTypeName is the rulebook's capitalization of npc-type k ("Heroic").
func NpcTypeName(k NpcType) string { return npcTypeNames[k] }

// KeyName is k's YAML spelling ("threat-value").
func KeyName(k Override) string { return overrideKeys[k] }

// Overrides is a set of Override.
type Overrides uint32

// Has says whether k is in the set.
func (o Overrides) Has(k Override) bool { return o&(1<<k) != 0 }

// With is the set with k in it too.
func (o Overrides) With(k Override) Overrides { return o | 1<<k }

func overridesOf(ks ...Override) Overrides {
	var o Overrides
	for _, k := range ks {
		o = o.With(k)
	}
	return o
}

// Named is what a sorted item has: a name and an optional detail.
type Named struct {
	Name      string
	HasDetail bool
	Detail    string
}

// PerkFlaw is a Perk or a Flaw: a plain name, or a mapping with a
// detail, a repeat count (counts-as) and a chosen cost, each optional.
type PerkFlaw struct {
	Named
	HasCountsAs bool
	CountsAs    int
	HasCost     bool
	Cost        int
}

// Attack is an attack of a combat: block.
type Attack struct {
	Name           string
	Perks, Flaws   []*PerkFlaw // sorted
	HasAffinity    bool
	Affinity       string
	HasDescription bool
	Description    string // trimmed
	Roll           int
	DX             string // text: "1/2" happens
	Endurance      int
	Telekinesis    bool // the generator's inputs, not printed
	Transformed    bool
	Overrides      Overrides // of OvRoll, OvDx, OvEndurance
}

// Combat is a combat: block. HasCombat is false when there is none (a
// Companion may have no combat stats).
type Combat struct {
	HasCombat      bool
	Defense        string // text: "4/5" happens
	Health         int
	Endurance      int
	HasThreatValue bool
	ThreatValue    int
	Attacks        []*Attack // in document order
	HasNotes       bool
	Notes          string // the Combat Notes, trimmed
	Transformed    bool   // the generator's input, not printed
	Overrides      Overrides
}

// SubItem is a linked sub-ability or sub-weakness, or a Companion's or
// Vehicle's own ability or weakness.
type SubItem struct {
	Named
	Level              int
	Perks, Flaws       []*PerkFlaw // sorted
	HasEndurance       bool
	Endurance          int
	HasNote            bool
	Note               string
	HasThreatValueHalf bool
	ThreatValueHalf    bool
	Overrides          Overrides // of OvEndurance
}

// Spell is a Magic ability's spell.
type Spell struct {
	Name, Ability string
	Level         int
	Endurance     string    // text: "10+" happens
	Overrides     Overrides // of OvEndurance
}

// SubEntity is a Companion or Vehicle, nested under the Ability that
// grants it. Label is "Companion" or "Vehicle", for the headings.
type SubEntity struct {
	Label, Name           string
	IsFragment            bool
	Abilities, Weaknesses []*SubItem // sorted
	Combat                Combat
}

// Ability is a top-level Ability: a SubItem's fields, and its linked
// content. SubEntity is nil if it grants none.
type Ability struct {
	SubItem
	SubAbilities, SubWeaknesses []*SubItem // sorted
	Spells                      []*Spell   // in document order
	SubEntity                   *SubEntity
}

// Weakness is a top-level Weakness.
type Weakness struct {
	Named
	Level int
}

// Note is a headed note.
type Note struct {
	Header, Text string // Text trimmed
}

// Options is an options: block: the character-creation limiting rules.
type Options struct {
	HasBaseZero           bool
	BaseZero              int
	HasPowerCeiling       bool
	PowerCeiling          int
	HasStartingLevels     bool
	StartingLevels        int
	ScaledCost            bool
	HalfCostKnowledge     bool
	CombinedEffectiveness bool
}

// Entity is a character or an NPC Fragment.
type Entity struct {
	Name                          string // required
	HasPlayer, HasPortrait        bool
	Player, Portrait              string // untrimmed, as in the Ada version
	HasTagline, HasSummary        bool
	HasBackground, HasAppearance  bool
	HasPersonality, HasOtherNotes bool
	HasCharacterHooks             bool
	Tagline, Summary              string // trimmed
	Background, Appearance        string
	Personality, OtherNotes       string
	CharacterHooks                string
	HasNpcType                    bool
	NpcType                       NpcType     // Heroic, Secondary or Extra
	Abilities                     []*Ability  // sorted
	Weaknesses                    []*Weakness // sorted
	Combat                        Combat
	Notes                         []*Note // in document order
	HasFluff                      bool
	HasFluffField                 [NFluff]bool
	Fluff                         [NFluff]string // untrimmed, as in the Ada version
	HasOptions                    bool
	Options                       Options
	IsFragment                    bool // an NPC Fragment
}

// LoadError is a required field that is missing or malformed.
type LoadError struct {
	Message, Path string
	HasLocation   bool
	Line, Column  int // 1-based
}

func (e *LoadError) Error() string { return e.Path + ": " + e.Message }

// Loader reads entities, collecting the LoadErrors of every one read.
type Loader struct {
	Errors []*LoadError
}

// fatal ends a Load with its message.
type fatal struct{ msg string }

func (f *fatal) Error() string { return f.msg }

// ---- Nodes ----

// node is a YAML node and its path.
type node struct {
	n    *yaml.Node
	path string
}

func deref(n *yaml.Node) *yaml.Node {
	for n != nil && n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	return n
}

func (x node) isScalar() bool   { return x.n.Kind == yaml.ScalarNode }
func (x node) isSequence() bool { return x.n.Kind == yaml.SequenceNode }
func (x node) isMapping() bool  { return x.n.Kind == yaml.MappingNode }

// value is m's value for key; ok false if there is none. A key with no
// value ("key:" and nothing) counts as none, as libfyaml gives no node
// for it.
func (x node) value(key string) (node, bool) {
	if !x.isMapping() {
		return node{}, false
	}
	c := x.n.Content
	for i := 0; i+1 < len(c); i += 2 {
		k := deref(c[i])
		if k.Kind == yaml.ScalarNode && k.Value == key {
			v := deref(c[i+1])
			if v.Kind == yaml.ScalarNode && v.Style == 0 && v.Value == "" && v.Tag == "!!null" {
				return node{}, false
			}
			return node{v, x.path + "/" + key}, true
		}
	}
	return node{}, false
}

func (x node) items() []node {
	var r []node
	for i, c := range x.n.Content {
		r = append(r, node{deref(c), x.path + "/" + strconv.Itoa(i)})
	}
	return r
}

// ---- Scalars, as polibfyaml converts them ----

// digitRun is the length of the run digit {["_"] digit} of base-digits
// at s[i], or 0 if s[i] isn't one.
func digitRun(s string, i, base int) int {
	val := func(c byte) int {
		switch {
		case c >= '0' && c <= '9':
			return int(c - '0')
		case c >= 'a' && c <= 'f':
			return int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			return int(c-'A') + 10
		}
		return 99
	}
	j := i
	if j >= len(s) || val(s[j]) >= base {
		return 0
	}
	j++
	for j < len(s) {
		if val(s[j]) < base {
			j++
		} else if s[j] == '_' && j+1 < len(s) && val(s[j+1]) < base {
			j += 2
		} else {
			break
		}
	}
	return j - i
}

// parseInt converts s as polibfyaml's IntegerValue does: [+-], then
// decimal, 0x hex, 0o octal or 0b binary digits, with single "_"s
// between them, within 32 bits.
func parseInt(s string) (int, error) {
	i, neg := 0, false
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		neg = s[0] == '-'
		i = 1
	}
	base := 10
	if len(s)-i >= 2 && s[i] == '0' {
		switch s[i+1] {
		case 'x':
			base, i = 16, i+2
		case 'o':
			base, i = 8, i+2
		case 'b':
			base, i = 2, i+2
		}
	}
	if i == len(s) || digitRun(s, i, base) != len(s)-i {
		return 0, fmt.Errorf("not a valid integer: %q", s)
	}
	v, err := strconv.ParseInt(strings.ReplaceAll(s[i:], "_", ""), base, 64)
	if err == nil && neg {
		v = -v
	}
	if err != nil || v < -1<<31 || v > 1<<31-1 {
		return 0, fmt.Errorf("integer out of range: %q", s)
	}
	return int(v), nil
}

// parseBool converts s as polibfyaml's BooleanValue does.
func parseBool(s string) (bool, error) {
	switch s {
	case "true", "True", "TRUE":
		return true, nil
	case "false", "False", "FALSE":
		return false, nil
	}
	return false, fmt.Errorf("not a valid boolean: %q", s)
}

// ---- Errors ----

// record records an error for key in mapping m: the Ada Record_Error.
// The location is that of m's own "name" scalar, if it has one: the
// missing key itself has no position to report.
func (l *Loader) record(m node, key, message string) {
	e := &LoadError{Message: message, Path: m.path + "/" + key}
	if name, ok := m.value("name"); ok && name.isScalar() {
		e.HasLocation, e.Line, e.Column = true, name.n.Line, name.n.Column
	}
	l.Errors = append(l.Errors, e)
}

func notScalar(key string) string { return `key "` + key + `" is not a scalar value` }

func failAt(n node, text string) error { return &fatal{n.path + ": " + text} }

// ---- Fields ----

// requireString is the text of the scalar for key, which is required.
// Records an error, and gives "", if it is missing or isn't a scalar.
func (l *Loader) requireString(m node, key string) string {
	v, ok := m.value(key)
	switch {
	case !ok:
		l.record(m, key, `missing required key "`+key+`"`)
	case !v.isScalar():
		l.record(m, key, notScalar(key))
	default:
		return v.n.Value
	}
	return ""
}

// requireInteger is the integer for key, which is required. Records an
// error, and gives 0, if it is missing or malformed.
func (l *Loader) requireInteger(m node, key string) int {
	v, ok := m.value(key)
	switch {
	case !ok:
		l.record(m, key, `missing required key "`+key+`"`)
	case !v.isScalar():
		l.record(m, key, notScalar(key))
	default:
		i, err := parseInt(v.n.Value)
		if err != nil {
			l.record(m, key, err.Error())
		}
		return i
	}
	return 0
}

// optString is the text of the scalar for key, if there is such a key,
// which has says. A value that isn't a scalar fails the Load.
func optString(m node, key string) (has bool, s string, err error) {
	v, ok := m.value(key)
	if !ok {
		return false, "", nil
	}
	if !v.isScalar() {
		return true, "", &fatal{notScalar(key)}
	}
	return true, v.n.Value, nil
}

// optInteger is the integer for key, if there is such a key. A
// malformed one fails the Load.
func optInteger(m node, key string) (has bool, i int, err error) {
	v, ok := m.value(key)
	if !ok {
		return false, 0, nil
	}
	if !v.isScalar() {
		return true, 0, &fatal{notScalar(key)}
	}
	i, err = parseInt(v.n.Value)
	if err != nil {
		return true, 0, &fatal{err.Error()}
	}
	return true, i, nil
}

// optBoolean sets *b from key, if there is such a key; *b is unchanged
// if there isn't. A malformed one fails the Load.
func optBoolean(m node, key string, b *bool) error {
	v, ok := m.value(key)
	if !ok {
		return nil
	}
	if !v.isScalar() {
		return &fatal{notScalar(key)}
	}
	x, err := parseBool(v.n.Value)
	if err != nil {
		return &fatal{err.Error()}
	}
	*b = x
	return nil
}

// sequence is the sequence for key, ok false if there's no such key;
// anything else there fails the Load.
func sequence(m node, key string) (s node, ok bool, err error) {
	s, ok = m.value(key)
	if ok && !s.isSequence() {
		return s, ok, failAt(s, "not a sequence")
	}
	return s, ok, nil
}

// mapping is the mapping for key, ok false if there's no such key;
// anything else there fails the Load.
func mapping(m node, key string) (s node, ok bool, err error) {
	s, ok = m.value(key)
	if ok && !s.isMapping() {
		return s, ok, failAt(s, "not a mapping")
	}
	return s, ok, nil
}

// loadList reads the items of the sequence for key (none if absent),
// each a mapping read by load.
func loadList[T any](m node, key string, load func(node) (T, error)) ([]T, error) {
	seq, ok, err := sequence(m, key)
	if err != nil || !ok {
		return nil, err
	}
	var r []T
	for _, item := range seq.items() {
		if !item.isMapping() {
			return nil, failAt(item, "not a mapping")
		}
		x, err := load(item)
		if err != nil {
			return nil, err
		}
		r = append(r, x)
	}
	return r, nil
}

// ---- Sorting ----

// FoldCI is s with every byte lowered as the Ada To_Lower lowers it,
// which names are sorted by: A..Z, and also the Latin-1 capitals C0..DE
// but D7. In UTF-8 those are lead bytes, so this doesn't lower non-ASCII
// letters, but it does move names starting with some of them relative
// to others, as in the Ada version.
func FoldCI(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' || c >= 0xC0 && c <= 0xDE && c != 0xD7 {
			b[i] = c + 32
		}
	}
	return string(b)
}

// Before says whether name an and detail ad sort before bn, bd: by name,
// then detail, folded.
func Before(an, ad, bn, bd string) bool {
	a, b := FoldCI(an), FoldCI(bn)
	if a != b {
		return a < b
	}
	return FoldCI(ad) < FoldCI(bd)
}

// sortByName sorts l by name and then detail, ignoring case: the Ada
// Ability_CI_Less and friends. An insertion sort: stable, and the lists
// are short.
func sortByName[T any](l []T, named func(T) *Named) {
	for i := 1; i < len(l); i++ {
		x := l[i]
		nx := named(x)
		j := i
		for j > 0 {
			ny := named(l[j-1])
			if !Before(nx.Name, nx.Detail, ny.Name, ny.Detail) {
				break
			}
			l[j] = l[j-1]
			j--
		}
		l[j] = x
	}
}

// ---- overrides: ----

// loadOverrides reads m's overrides: list, if any: the Ada
// Load_Overrides. A list that isn't a sequence, or an item that isn't
// one of the allowed key names, is a recorded error, like a missing
// required key.
func (l *Loader) loadOverrides(m node, allowed Overrides) Overrides {
	var o Overrides
	seq, ok := m.value("overrides")
	if !ok {
		return o
	}
	if !seq.isSequence() {
		l.record(m, "overrides", "not a sequence of key names")
		return o
	}
	for _, item := range seq.items() {
		if !item.isScalar() {
			l.record(m, "overrides", "an item is not a key name")
			continue
		}
		found := false
		for k := Override(0); k < nOverrides; k++ {
			if allowed.Has(k) && item.n.Value == overrideKeys[k] {
				o = o.With(k)
				found = true
				break
			}
		}
		// defense-2's name before there could be a third value.
		if !found && allowed.Has(OvDefense2) && item.n.Value == "barrier" {
			o = o.With(OvDefense2)
			found = true
		}
		if !found {
			var names []string
			for k := Override(0); k < nOverrides; k++ {
				if allowed.Has(k) {
					names = append(names, overrideKeys[k])
				}
			}
			l.record(m, "overrides", `"`+item.n.Value+`" is not one of `+strings.Join(names, ", "))
		}
	}
	return o
}

// loadNpcType reads m's npc-type:, if any: the Ada Load_NPC_Type.
// Anything but one of the three names is a recorded error, like an
// overrides: name: silently dropping it would lose the type from the
// output. A value that isn't a scalar fails the Load, as any optional
// string's does.
func (l *Loader) loadNpcType(m node) (bool, NpcType, error) {
	there, raw, err := optString(m, "npc-type")
	if err != nil || !there {
		return false, Heroic, err
	}
	for k := Heroic; k < NNpcTypes; k++ {
		if FoldCI(Trim(raw)) == FoldCI(NpcTypeName(k)) {
			return true, k, nil
		}
	}
	l.record(m, "npc-type", `"`+raw+`" is not one of Heroic, Secondary, Extra`)
	return false, Heroic, nil
}

// ---- Perks and Flaws ----

// loadPerkFlaw reads a Perk or Flaw: a scalar (just the name) or a
// mapping.
func (l *Loader) loadPerkFlaw(n node) (*PerkFlaw, error) {
	pf := &PerkFlaw{}
	switch {
	case n.isScalar():
		pf.Name = n.n.Value
		return pf, nil
	case n.isMapping():
		var err error
		pf.Name = l.requireString(n, "name")
		if pf.HasDetail, pf.Detail, err = optString(n, "detail"); err != nil {
			return nil, err
		}
		if pf.HasCountsAs, pf.CountsAs, err = optInteger(n, "counts-as"); err != nil {
			return nil, err
		}
		if pf.HasCost, pf.Cost, err = optInteger(n, "cost"); err != nil {
			return nil, err
		}
		return pf, nil
	}
	// The Ada version's own message, which has no path.
	return nil, &fatal{"unrecognized Perk/Flaw shape"}
}

// loadPerkFlaws reads the Perks or Flaws for key (none if absent),
// sorted.
func (l *Loader) loadPerkFlaws(m node, key string) ([]*PerkFlaw, error) {
	seq, ok, err := sequence(m, key)
	if err != nil || !ok {
		return nil, err
	}
	var r []*PerkFlaw
	for _, item := range seq.items() {
		pf, err := l.loadPerkFlaw(item)
		if err != nil {
			return nil, err
		}
		r = append(r, pf)
	}
	sortByName(r, func(p *PerkFlaw) *Named { return &p.Named })
	return r, nil
}

// ---- Attacks and combat: blocks ----

func (l *Loader) loadAttack(m node) (*Attack, error) {
	a := &Attack{}
	var err error
	a.Name = l.requireString(m, "name")
	if a.Perks, err = l.loadPerkFlaws(m, "perks"); err != nil {
		return nil, err
	}
	if a.Flaws, err = l.loadPerkFlaws(m, "flaws"); err != nil {
		return nil, err
	}
	if a.HasAffinity, a.Affinity, err = optString(m, "affinity"); err != nil {
		return nil, err
	}
	if a.HasDescription, a.Description, err = optString(m, "description"); err != nil {
		return nil, err
	}
	a.Description = Trim(a.Description)
	a.Roll = l.requireInteger(m, "roll")
	a.DX = l.requireString(m, "dx")
	a.Endurance = l.requireInteger(m, "endurance")
	if err = optBoolean(m, "telekinesis", &a.Telekinesis); err != nil {
		return nil, err
	}
	if err = optBoolean(m, "transformed", &a.Transformed); err != nil {
		return nil, err
	}
	a.Overrides = l.loadOverrides(m, overridesOf(OvRoll, OvDx, OvEndurance))
	return a, nil
}

// loadCombat reads the combat: block m into c. m has no "name" of its
// own, so these errors have a path but no location, as in the Ada
// version.
func (l *Loader) loadCombat(m node, c *Combat) error {
	var err error
	c.HasCombat = true
	c.Defense = l.requireString(m, "defense")
	if strings.Count(c.Defense, "/") > 2 {
		l.record(m, "defense", `"`+c.Defense+`" has more than three values`)
	}
	c.Health = l.requireInteger(m, "health")
	c.Endurance = l.requireInteger(m, "endurance")
	if c.HasThreatValue, c.ThreatValue, err = optInteger(m, "threat-value"); err != nil {
		return err
	}
	if c.Attacks, err = loadList(m, "attacks", l.loadAttack); err != nil {
		return err
	}
	if c.HasNotes, c.Notes, err = optString(m, "notes"); err != nil {
		return err
	}
	if err = optBoolean(m, "transformed", &c.Transformed); err != nil {
		return err
	}
	c.Notes = Trim(c.Notes)
	c.Overrides = l.loadOverrides(m, overridesOf(OvEndurance, OvDefense, OvDefense2, OvDefense3, OvHealth, OvThreatValue))
	return nil
}

// optCombat reads the combat: block, if there is one, into c.
func (l *Loader) optCombat(m node, c *Combat) error {
	*c = Combat{}
	n, ok, err := mapping(m, "combat")
	if err != nil || !ok {
		return err
	}
	return l.loadCombat(n, c)
}

// ---- Sub-items, spells, sub-entities ----

// loadSubItemFields reads the fields an Ability and a SubItem share.
func (l *Loader) loadSubItemFields(m node, s *SubItem) error {
	var err error
	s.Name = l.requireString(m, "name")
	s.Level = l.requireInteger(m, "level")
	if s.HasDetail, s.Detail, err = optString(m, "detail"); err != nil {
		return err
	}
	if s.Perks, err = l.loadPerkFlaws(m, "perks"); err != nil {
		return err
	}
	if s.Flaws, err = l.loadPerkFlaws(m, "flaws"); err != nil {
		return err
	}
	if s.HasEndurance, s.Endurance, err = optInteger(m, "endurance"); err != nil {
		return err
	}
	if s.HasNote, s.Note, err = optString(m, "note"); err != nil {
		return err
	}
	_, s.HasThreatValueHalf = m.value("threat-value-half")
	if err = optBoolean(m, "threat-value-half", &s.ThreatValueHalf); err != nil {
		return err
	}
	s.Overrides = l.loadOverrides(m, overridesOf(OvEndurance))
	return nil
}

func (l *Loader) loadSubItem(m node) (*SubItem, error) {
	s := &SubItem{}
	return s, l.loadSubItemFields(m, s)
}

// loadSubItems reads the SubItems for key (none if absent), sorted.
func (l *Loader) loadSubItems(m node, key string) ([]*SubItem, error) {
	r, err := loadList(m, key, l.loadSubItem)
	if err != nil {
		return nil, err
	}
	sortByName(r, func(s *SubItem) *Named { return &s.Named })
	return r, nil
}

func (l *Loader) loadSpell(m node) (*Spell, error) {
	s := &Spell{}
	s.Name = l.requireString(m, "name")
	s.Ability = l.requireString(m, "ability")
	s.Level = l.requireInteger(m, "level")
	s.Endurance = l.requireString(m, "endurance")
	s.Overrides = l.loadOverrides(m, overridesOf(OvEndurance))
	return s, nil
}

func (l *Loader) loadSubEntity(m node, label string) (*SubEntity, error) {
	se := &SubEntity{Label: label}
	var err error
	se.Name = l.requireString(m, "name")
	if err = optBoolean(m, "fragment", &se.IsFragment); err != nil {
		return nil, err
	}
	if se.Abilities, err = l.loadSubItems(m, "abilities"); err != nil {
		return nil, err
	}
	if se.Weaknesses, err = l.loadSubItems(m, "weaknesses"); err != nil {
		return nil, err
	}
	if err = l.optCombat(m, &se.Combat); err != nil {
		return nil, err
	}
	return se, nil
}

// ---- Abilities, weaknesses, notes ----

func (l *Loader) loadAbility(m node) (*Ability, error) {
	a := &Ability{}
	var err error
	if err = l.loadSubItemFields(m, &a.SubItem); err != nil {
		return nil, err
	}
	if a.SubAbilities, err = l.loadSubItems(m, "abilities"); err != nil {
		return nil, err
	}
	if a.SubWeaknesses, err = l.loadSubItems(m, "weaknesses"); err != nil {
		return nil, err
	}
	if a.Spells, err = loadList(m, "spells", l.loadSpell); err != nil {
		return nil, err
	}
	n, ok, err := mapping(m, "companion")
	if err != nil {
		return nil, err
	}
	if ok {
		a.SubEntity, err = l.loadSubEntity(n, "Companion")
		return a, err
	}
	n, ok, err = mapping(m, "vehicle")
	if err != nil {
		return nil, err
	}
	if ok {
		a.SubEntity, err = l.loadSubEntity(n, "Vehicle")
	}
	return a, err
}

func (l *Loader) loadWeakness(m node) (*Weakness, error) {
	w := &Weakness{}
	var err error
	w.Name = l.requireString(m, "name")
	w.Level = l.requireInteger(m, "level")
	if w.HasDetail, w.Detail, err = optString(m, "detail"); err != nil {
		return nil, err
	}
	return w, nil
}

// loadNote reads a note. A note has no "name" of its own, so its errors
// have a path but no location.
func (l *Loader) loadNote(m node) (*Note, error) {
	n := &Note{}
	n.Header = l.requireString(m, "header")
	n.Text = Trim(l.requireString(m, "text"))
	return n, nil
}

// ---- fluff: and options: ----

func loadFluff(m node, e *Entity) error {
	for i := 0; i < NFluff; i++ {
		var err error
		if e.HasFluffField[i], e.Fluff[i], err = optString(m, FluffKeys[i]); err != nil {
			return err
		}
	}
	return nil
}

func loadOptions(m node, o *Options) error {
	var err error
	if o.HasBaseZero, o.BaseZero, err = optInteger(m, "base-zero"); err != nil {
		return err
	}
	if o.HasPowerCeiling, o.PowerCeiling, err = optInteger(m, "power-ceiling"); err != nil {
		return err
	}
	if o.HasStartingLevels, o.StartingLevels, err = optInteger(m, "starting-levels"); err != nil {
		return err
	}
	if err = optBoolean(m, "scaled-cost", &o.ScaledCost); err != nil {
		return err
	}
	if err = optBoolean(m, "half-cost-knowledge", &o.HalfCostKnowledge); err != nil {
		return err
	}
	return optBoolean(m, "combined-effectiveness", &o.CombinedEffectiveness)
}

// ---- Entities ----

// prose reads a prose field, trimmed, as the Ada Load_Entity does.
func prose(m node, key string, has *bool, s *string) error {
	var err error
	*has, *s, err = optString(m, key)
	*s = Trim(*s)
	return err
}

func (l *Loader) loadEntity(m node, e *Entity) error {
	var err error
	e.Name = l.requireString(m, "name")
	if err = optBoolean(m, "fragment", &e.IsFragment); err != nil {
		return err
	}
	if e.HasPlayer, e.Player, err = optString(m, "player"); err != nil {
		return err
	}
	for _, p := range []struct {
		key string
		has *bool
		s   *string
	}{
		{"tagline", &e.HasTagline, &e.Tagline},
		{"summary", &e.HasSummary, &e.Summary},
		{"background", &e.HasBackground, &e.Background},
		{"appearance", &e.HasAppearance, &e.Appearance},
		{"personality", &e.HasPersonality, &e.Personality},
		{"character-hooks", &e.HasCharacterHooks, &e.CharacterHooks},
	} {
		if err = prose(m, p.key, p.has, p.s); err != nil {
			return err
		}
	}
	if e.HasNpcType, e.NpcType, err = l.loadNpcType(m); err != nil {
		return err
	}
	if err = prose(m, "other-notes", &e.HasOtherNotes, &e.OtherNotes); err != nil {
		return err
	}
	if e.HasPortrait, e.Portrait, err = optString(m, "portrait"); err != nil {
		return err
	}
	if e.Abilities, err = loadList(m, "abilities", l.loadAbility); err != nil {
		return err
	}
	sortByName(e.Abilities, func(a *Ability) *Named { return &a.Named })
	if e.Weaknesses, err = loadList(m, "weaknesses", l.loadWeakness); err != nil {
		return err
	}
	sortByName(e.Weaknesses, func(w *Weakness) *Named { return &w.Named })
	if err = l.optCombat(m, &e.Combat); err != nil {
		return err
	}
	if e.Notes, err = loadList(m, "notes", l.loadNote); err != nil {
		return err
	}
	n, ok, err := mapping(m, "fluff")
	if err != nil {
		return err
	}
	e.HasFluff = ok
	if ok {
		if err = loadFluff(n, e); err != nil {
			return err
		}
	}
	n, ok, err = mapping(m, "options")
	if err != nil {
		return err
	}
	e.HasOptions = ok
	if ok {
		return loadOptions(n, &e.Options)
	}
	return nil
}

// Load reads an entity from item, the i'th item of a document's
// top-level sequence. A required field that is missing or malformed is
// added to l.Errors, and loading goes on: check l.Errors once the whole
// file is loaded. Anything else wrong returns an error; the caller
// abandons the file.
func (l *Loader) Load(item *yaml.Node, i int) (*Entity, error) {
	m := node{deref(item), "/" + strconv.Itoa(i)}
	if !m.isMapping() {
		return nil, failAt(m, "not a mapping")
	}
	e := &Entity{}
	if err := l.loadEntity(m, e); err != nil {
		var f *fatal
		if errors.As(err, &f) {
			return nil, errors.New(f.msg)
		}
		return nil, err
	}
	return e, nil
}

// Trim is s without leading and trailing blanks, tabs, line feeds and
// carriage returns: the Ada Trim_Whitespace. Not just blanks, because a
// YAML literal block scalar ("background: |") ends with a newline.
func Trim(s string) string { return strings.Trim(s, " \t\n\r") }
