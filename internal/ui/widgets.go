package ui

// What the window and its dialogs share: the number boxes, the ?
// buttons, a spinner, and the list editors (ova_cg_po's OvaWidgets).
//
// ova_cg_po's PLAN.md, "Overriding a computed number": beside every
// number the program computes is an empty entry box. A NumberBox is
// both: the computed number, shown read-only, and the box, whose text is
// the override in effect ("" for none). Where its number lives is its
// target, a pointer into the model, so the box reads and sets the model,
// never a copy.
//
// A HelpButton, "?", opens the rulebook at a page (PLAN.md, "Page
// numbers beside names"), through the window's OpenPage.
//
// A Panel is a list and its editor, the shape PLAN.md gives the
// Abilities and Weaknesses and the attack dialog's Perks and Flaws: a
// list of lines, under it a Select of every name of one kind
// (rules.Label: "Attack (p. 50)"), a Spinner, a text field, and Add,
// Change, Remove and ? buttons. What the lines are, and what the
// buttons do to the model, are its hooks'.

import (
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tkurtbond/ova_cg_go/internal/calc"
	"github.com/tkurtbond/ova_cg_go/internal/rules"
)

// Row is a row's height: an entry's or a button's.
const Row = 36

// numberShownW is the width of a NumberBox's number shown.
const numberShownW = 44

// overrideSamples are the widest texts an override box need show
// whole: three characters, as a DX's 1/2 or a spell's 10+ (a longer one
// still fits, scrolled).
var overrideSamples = []string{"-10", "10+", "1/2", "100"}

// overrideW is an override box's width: its widest sample in the
// theme's text, the entry's inner padding and border at both ends, and
// a padding's room for the cursor.
func overrideW() float32 {
	th := theme.Current()
	text := float32(0)
	for _, s := range overrideSamples {
		text = max(text, fyne.MeasureText(s, th.Size(theme.SizeNameText), fyne.TextStyle{}).Width)
	}
	return text + 2*th.Size(theme.SizeNameInnerPadding) + 2*th.Size(theme.SizeNameInputBorder) +
		th.Size(theme.SizeNamePadding)
}

// sized is o, given at least w by h.
func sized(o fyne.CanvasObject, w, h float32) fyne.CanvasObject {
	r := canvas.NewRectangle(color.Transparent)
	r.SetMinSize(fyne.NewSize(w, h))
	return container.NewStack(r, o)
}

// bold is a heading's label.
func bold(text string) *widget.Label {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

func enable(on bool, ws ...fyne.Disableable) {
	for _, w := range ws {
		if on {
			w.Enable()
		} else {
			w.Disable()
		}
	}
}

func visible(on bool, os ...fyne.CanvasObject) {
	for _, o := range os {
		if on {
			o.Show()
		} else {
			o.Hide()
		}
	}
}

func trim(s string) string { return strings.Trim(s, " \t\n\r") }

// ---- Spinner ----

// Spinner is a bounded integer, with - and + buttons: Fyne has none.
type Spinner struct {
	widget.BaseWidget
	Min, Max  int
	value     int
	text      *widget.Label
	minus     *widget.Button
	plus      *widget.Button
	disabled  bool
	OnChanged func(int) // after the user changes it
}

// NewSpinner is a spinner from min to max showing value.
func NewSpinner(min, max, value int) *Spinner {
	s := &Spinner{Min: min, Max: max}
	s.text = widget.NewLabel("")
	s.text.Alignment = fyne.TextAlignCenter
	s.minus = widget.NewButton("-", func() { s.step(-1) })
	s.plus = widget.NewButton("+", func() { s.step(1) })
	s.ExtendBaseWidget(s)
	s.SetValue(value)
	return s
}

func (s *Spinner) step(d int) {
	old := s.value
	s.SetValue(s.value + d)
	if s.value != old && s.OnChanged != nil {
		s.OnChanged(s.value)
	}
}

// Spin sets the value, as the user does: for tests.
func (s *Spinner) Spin(v int) {
	old := s.value
	s.SetValue(v)
	if s.value != old && s.OnChanged != nil {
		s.OnChanged(s.value)
	}
}

// Value is the spinner's value.
func (s *Spinner) Value() int { return s.value }

// SetValue sets the value, kept within the bounds.
func (s *Spinner) SetValue(v int) {
	s.value = max(s.Min, min(s.Max, v))
	s.text.SetText(strconv.Itoa(s.value))
	s.showButtons()
}

// SetBounds sets the bounds, the value kept within them.
func (s *Spinner) SetBounds(min, max int) {
	s.Min, s.Max = min, max
	s.SetValue(s.value)
}

func (s *Spinner) showButtons() {
	enable(!s.disabled && s.value > s.Min, s.minus)
	enable(!s.disabled && s.value < s.Max, s.plus)
}

// Enable lets the user change it.
func (s *Spinner) Enable() { s.disabled = false; s.showButtons() }

// Disable stops the user changing it.
func (s *Spinner) Disable() { s.disabled = true; s.showButtons() }

// Disabled says whether the user can't change it.
func (s *Spinner) Disabled() bool { return s.disabled }

// CreateRenderer is Fyne's.
func (s *Spinner) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewBorder(nil, nil, s.minus, s.plus, sized(s.text, 40, 0)))
}

// ---- NumberBox ----

// overrideEntry is a NumberBox's box: it takes its text on Enter, and on
// losing the focus.
type overrideEntry struct {
	widget.Entry
	leave func()
}

func newOverrideEntry(leave func()) *overrideEntry {
	e := &overrideEntry{leave: leave}
	e.ExtendBaseWidget(e)
	e.OnSubmitted = func(string) { e.leave() }
	return e
}

// FocusLost is Fyne's: the text is taken.
func (e *overrideEntry) FocusLost() {
	e.Entry.FocusLost()
	e.leave()
}

// NumberBox is a computed number and the box for its override.
type NumberBox struct {
	Label     string
	Shown     *widget.Label // the computed number
	Entry     *widget.Entry // the override: "" for none
	entry     *overrideEntry
	Target    *calc.Number // nil: none
	AllowHalf bool         // a DX's 1/2
	AllowPlus bool         // an END's 10+
	// After the user leaves the box having changed it: the override set
	// from its text, or (ok false) the text refused, and the box
	// emptied.
	Changed func(b *NumberBox, ok bool)
	box     *fyne.Container
	taken   string // the box's text as Display last set it
}

// NewNumberBox is a number box labelled label (beside it, or "" for
// none, or above it if above).
func NewNumberBox(label string, above bool) *NumberBox {
	b := &NumberBox{Label: label}
	b.Shown = widget.NewLabel("")
	b.Shown.Alignment = fyne.TextAlignTrailing
	b.entry = newOverrideEntry(b.Take)
	b.entry.SetPlaceHolder("")
	b.Entry = &b.entry.Entry
	pair := container.NewHBox(sized(b.Shown, numberShownW, 0), sized(b.entry, overrideW(), 0))
	switch {
	case label == "":
		b.box = container.NewHBox(pair)
	case above:
		b.box = container.NewVBox(widget.NewLabel(label), pair)
	default:
		b.box = container.NewHBox(widget.NewLabel(label), pair)
	}
	return b
}

// Object is what the box is laid out as.
func (b *NumberBox) Object() fyne.CanvasObject { return b.box }

// Display shows the target's number: computed, and its override.
func (b *NumberBox) Display() {
	b.taken = ""
	if n := b.Target; n != nil {
		b.Shown.SetText(n.ComputedText())
		if n.Overridden {
			b.taken = n.Text()
		}
	} else {
		b.Shown.SetText("")
	}
	b.Entry.SetText(b.taken)
}

// Take reads the box's text into its target's override (an empty box
// clears it); text that isn't a number for it is refused, and the
// override cleared. Nothing if the text is as Display left it.
func (b *NumberBox) Take() {
	if b.Target == nil || b.Entry.Text == b.taken {
		return
	}
	n := *b.Target
	ok := n.Parse(b.Entry.Text, b.AllowHalf, b.AllowPlus)
	if !ok {
		n.Clear()
	}
	*b.Target = n
	b.Display()
	if b.Changed != nil {
		b.Changed(b, ok)
	}
}

// Type types text into the box and leaves it, as the user does: for
// tests.
func (b *NumberBox) Type(text string) {
	b.Entry.SetText(text)
	b.Take()
}

// SetRed shows the computed number red, or not: a total a limit finds
// too high.
func (b *NumberBox) SetRed(on bool) {
	imp := widget.MediumImportance
	if on {
		imp = widget.DangerImportance
	}
	if b.Shown.Importance != imp {
		b.Shown.Importance = imp
		b.Shown.Refresh()
	}
}

// Red says whether the computed number is shown red.
func (b *NumberBox) Red() bool { return b.Shown.Importance == widget.DangerImportance }

// Enable lets the user override the number.
func (b *NumberBox) Enable() { b.Entry.Enable() }

// Disable stops the user overriding the number.
func (b *NumberBox) Disable() { b.Entry.Disable() }

// Disabled says whether the user can't override the number.
func (b *NumberBox) Disabled() bool { return b.Entry.Disabled() }

// Active says whether the user can override the number.
func (b *NumberBox) Active() bool { return !b.Entry.Disabled() }

// Show shows the box.
func (b *NumberBox) Show() { b.box.Show() }

// Hide hides the box.
func (b *NumberBox) Hide() { b.box.Hide() }

// Visible says whether the box is shown.
func (b *NumberBox) Visible() bool { return b.box.Visible() }

// Shows says whether the box shows n: its computed number, and its
// override or "": for tests.
func (b *NumberBox) Shows(n *calc.Number) bool {
	want := ""
	if n.Overridden {
		want = n.Text()
	}
	return b.Shown.Text == n.ComputedText() && b.Entry.Text == want
}

// notANumber is the message for text refused in b.
func notANumber(b *NumberBox) string {
	msg := "Not a number"
	if b.Label != "" {
		msg += " for " + b.Label
	}
	msg += ": a whole number"
	if b.AllowHalf {
		msg += ", or 1/2"
	}
	if b.AllowPlus {
		msg += ", which may end in +"
	}
	return msg + ". The computed number is used."
}

// ---- HelpButton ----

// HelpButton is a "?" button that opens the rulebook at its page.
type HelpButton struct {
	*widget.Button
	Page int // the printed page; 0: nothing to show
}

// NewHelpButton is a ? button showing page (0: none yet), through
// openPage.
func NewHelpButton(page int, openPage func(int)) *HelpButton {
	h := &HelpButton{Page: page}
	h.Button = widget.NewButton("?", func() {
		if h.Page > 0 {
			openPage(h.Page)
		}
	})
	return h
}

// ---- Lists ----

// lines is a list of lines, each of columns: the first the name, the
// others right of it, at widths; a column right of the name may be
// hidden.
type lines struct {
	List     *widget.List
	rows     [][]string
	widths   []float32
	hidden   []bool // hidden[i]: the column at widths[i]
	selected int    // -1: none
	quiet    bool
	OnSelect func(i int) // after the user selects line i
}

func newLines(widths ...float32) *lines {
	l := &lines{widths: widths, hidden: make([]bool, len(widths)), selected: -1}
	l.List = widget.NewList(
		func() int { return len(l.rows) },
		func() fyne.CanvasObject {
			name := widget.NewLabel("")
			name.Truncation = fyne.TextTruncateEllipsis
			var right []fyne.CanvasObject
			for _, w := range l.widths {
				c := widget.NewLabel("")
				c.Alignment = fyne.TextAlignTrailing
				right = append(right, sized(c, w, 0))
			}
			return container.NewBorder(nil, nil, nil, container.NewHBox(right...), name)
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			c := o.(*fyne.Container)
			row := l.rows[id]
			c.Objects[0].(*widget.Label).SetText(row[0])
			cols := c.Objects[1].(*fyne.Container).Objects
			for i, col := range cols {
				text := ""
				if i+1 < len(row) {
					text = row[i+1]
				}
				col.(*fyne.Container).Objects[1].(*widget.Label).SetText(text)
				visible(!l.hidden[i], col)
			}
			c.Objects[1].Refresh()
		})
	l.List.OnSelected = func(id widget.ListItemID) {
		l.selected = id
		if !l.quiet && l.OnSelect != nil {
			l.OnSelect(id)
		}
	}
	l.List.OnUnselected = func(id widget.ListItemID) {
		if l.selected == id {
			l.selected = -1
		}
	}
	return l
}

// Set shows rows, line sel selected (-1: none), without calling
// OnSelect.
func (l *lines) Set(rows [][]string, sel int) {
	l.rows = rows
	l.quiet = true
	l.List.UnselectAll()
	l.selected = -1
	l.List.Refresh()
	if sel >= 0 && sel < len(rows) {
		l.List.Select(sel)
	}
	l.quiet = false
}

// Count is how many lines there are.
func (l *lines) Count() int { return len(l.rows) }

// Selected is the line selected; -1 for none.
func (l *lines) Selected() int { return l.selected }

// ShowColumn shows or hides the column at widths[i], from the next Set.
func (l *lines) ShowColumn(i int, on bool) { l.hidden[i] = !on }

// Line is line i's columns shown joined by tabs; "" past the end: for
// tests.
func (l *lines) Line(i int) string {
	if i < 0 || i >= len(l.rows) {
		return ""
	}
	cols := []string{l.rows[i][0]}
	for j, s := range l.rows[i][1:] {
		if j >= len(l.hidden) || !l.hidden[j] {
			cols = append(cols, s)
		}
	}
	return strings.Join(cols, "\t")
}

// SelectedLine is the selected line's text; "" for none: for tests.
func (l *lines) SelectedLine() string { return l.Line(l.selected) }

// Select selects line i, as a click does: for tests. A line already
// selected is let go first, so OnSelect runs as FLTK's callback did.
func (l *lines) Select(i int) {
	l.List.UnselectAll()
	l.List.Select(i)
}

// Find is the last line holding text; -1 if none: for tests.
func (l *lines) Find(text string) int {
	for i := len(l.rows) - 1; i >= 0; i-- {
		if strings.Contains(l.Line(i), text) {
			return i
		}
	}
	return -1
}

// ---- Panel ----

// panelHooks is what a Panel's lines are, and what its buttons do to
// the model.
type panelHooks interface {
	// fill is the lines, from the model, in its order.
	fill() [][]string
	// load sets the controls from line i, the one the user selected.
	load(i int)
	// add and change (of line i) change the model from the controls,
	// and return the line the item will be on once filled again (a
	// change can move it), or -1 if they changed nothing; remove
	// whether it removed line i. Each leaves the model recomputed.
	add() int
	change(i int) int
	remove(i int) bool
	// chosen follows a change of the Select: the spinner's bounds and
	// the text field.
	chosen()
}

// BossDivider is the row of a Panel's Select between its kind's names and
// the Boss Abilities: a Select has no divider of its own, so it is a row
// that can't stay chosen.
const BossDivider = "── Boss Abilities ──"

// Panel is a list and its editor.
type Panel struct {
	Kind                rules.Kind // the Select's
	WithBoss            bool       // the Select offers the Boss Abilities too
	*lines                         // the list
	Choice              *widget.Select
	Spinner             *Spinner
	Text                *widget.Entry
	Add, Change, Remove *widget.Button
	Help                *HelpButton
	Changed             func() // after a change the panel made: the window's or dialog's
	hooks               panelHooks
	object              fyne.CanvasObject
	choosing            bool // Choose is setting the Select
}

func newPanel(hooks panelHooks, kind rules.Kind, title, spinLabel, textLabel string, height float32,
	openPage func(int), widths ...float32) *Panel {
	p := &Panel{Kind: kind, hooks: hooks}
	p.lines = newLines(widths...)
	p.lines.OnSelect = func(i int) { p.hooks.load(i) }
	p.Choice = widget.NewSelect(p.labels(), func(string) {
		switch {
		case p.choosing:
		case p.onDivider():
			p.Choose(nil)
		default:
			p.hooks.chosen()
		}
	})
	p.Choice.PlaceHolder = "(choose a name)"
	p.Help = NewHelpButton(0, openPage)
	p.Spinner = NewSpinner(1, 1, 1)
	p.Text = widget.NewEntry()
	p.Add = widget.NewButton("Add", func() { p.button(p.Add) })
	p.Change = widget.NewButton("Change", func() { p.button(p.Change) })
	p.Remove = widget.NewButton("Remove", func() { p.button(p.Remove) })
	p.object = container.NewBorder(
		bold(title),
		container.NewVBox(
			container.NewBorder(nil, nil, nil, p.Help, p.Choice),
			container.NewBorder(nil, nil,
				container.NewHBox(widget.NewLabel(spinLabel), p.Spinner, widget.NewLabel(textLabel)), nil, p.Text),
			container.NewHBox(p.Add, p.Change, p.Remove, layout.NewSpacer())),
		nil, nil,
		sized(p.List, 0, height))
	return p
}

// labels is the Select's rows: the names p offers, each with its page;
// BossDivider before the Boss Abilities, if WithBoss.
func (p *Panel) labels() []string {
	var l []string
	for i := range rules.OfferCount(p.Kind, p.WithBoss) {
		if p.WithBoss && i == rules.Count(p.Kind) {
			l = append(l, BossDivider)
		}
		l = append(l, rules.Label(rules.Offered(p.Kind, i, p.WithBoss)))
	}
	return l
}

// onDivider says whether the Select's chosen row is BossDivider.
func (p *Panel) onDivider() bool {
	return p.WithBoss && p.Choice.SelectedIndex() == rules.Count(p.Kind)
}

// OfferBoss offers the Boss Abilities, or stops: what is chosen stays,
// if it still can.
func (p *Panel) OfferBoss(on bool) {
	if on == p.WithBoss {
		return
	}
	it := p.Item()
	p.WithBoss = on
	p.Choice.SetOptions(p.labels())
	p.Choose(it)
}

// Object is what the panel is laid out as.
func (p *Panel) Object() fyne.CanvasObject { return p.object }

// Item is the name chosen; nil for none.
func (p *Panel) Item() *rules.Item {
	i := p.Choice.SelectedIndex()
	if p.onDivider() {
		return nil
	}
	if p.WithBoss && i > rules.Count(p.Kind) {
		i-- // past BossDivider
	}
	if i < 0 || i >= rules.OfferCount(p.Kind, p.WithBoss) {
		return nil
	}
	return rules.Offered(p.Kind, i, p.WithBoss)
}

// helpChosen turns the ? button to the chosen name's page.
func (p *Panel) helpChosen() {
	p.Help.Page = 0
	if it := p.Item(); it != nil {
		p.Help.Page = it.Page
	}
}

// Choose chooses it in the Select (none for nil, or an item the Select
// doesn't offer), and follows it as the user's choice is followed.
func (p *Panel) Choose(it *rules.Item) {
	p.choosing = true
	defer func() { p.choosing = false }()
	if i := rules.OfferIndex(it, p.Kind, p.WithBoss); i >= 0 {
		if i >= rules.Count(p.Kind) {
			i++ // past BossDivider
		}
		p.Choice.SetSelectedIndex(i)
	} else {
		p.Choice.ClearSelected()
	}
	p.choosing = false
	p.hooks.chosen()
}

// Pick chooses the item named name, as the user does: for tests.
func (p *Panel) Pick(name string) { p.Choose(rules.Find(name)) }

// Refresh fills the list again, line sel selected (-1: none) and the
// controls loaded from it.
func (p *Panel) Refresh(sel int) {
	p.lines.Set(p.hooks.fill(), sel)
	if p.Selected() >= 0 {
		p.hooks.load(p.Selected())
	}
}

// button is what Add, Change and Remove do: Add and Change leave their
// item's line selected; Remove, none. Change and Remove need a line
// selected.
func (p *Panel) button(b *widget.Button) {
	line := p.Selected()
	done := false
	switch {
	case b == p.Add:
		line = p.hooks.add()
		done = line >= 0
	case line < 0:
	case b == p.Change:
		line = p.hooks.change(line)
		done = line >= 0
	default:
		done = p.hooks.remove(line)
		line = -1
	}
	if done {
		p.Refresh(line)
		if p.Changed != nil {
			p.Changed()
		}
	}
}
