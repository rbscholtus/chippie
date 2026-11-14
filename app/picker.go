package app

import (
	"strconv"

	"github.com/rbscholtus/chippie/app/library"
)

// rowKind is what a line of the list is.
type rowKind uint8

const (
	rowHeading  rowKind = iota // a group heading
	rowEntry                   // a ROM
	rowLoadFile                // "Load a ROM from this device…"
)

// pickerRow is one line of the list as drawn.
type pickerRow struct {
	kind    rowKind
	heading string
	entry   int // index into entries, for rowEntry
}

// picker is the library drawer's list: search, group headings and a
// scrollable list for keyboard, mouse and touch. It holds no Ebitengine
// state, so it can be tested directly.
type picker struct {
	all      []library.Entry // the whole library
	entries  []library.Entry // after the search
	rows     []pickerRow
	selected int // index into entries
	query    string
	family   library.Family         // the platform filter; AllFamilies: none
	counts   map[library.Family]int // entries per family for the query, for the filter's labels
	canLoad  bool                   // offer "Load a ROM from this device"

	// Scrolling, in pixels: offset is how far the list is scrolled; viewH
	// is the visible height and rowH the height of one row.
	offset float64
	viewH  float64
	rowH   float64
}

func newPicker(all []library.Entry, canLoad bool) *picker {
	p := &picker{all: all, canLoad: canLoad, rowH: 1}
	p.refilter()

	return p
}

// setAll replaces the library (after a catalogue refresh), keeping the
// search and, if it's still there, the selected ROM.
func (p *picker) setAll(all []library.Entry) {
	sel, _ := p.current()
	p.all = all
	p.refilter()

	if sel.SHA1 != "" {
		p.selectSHA1(sel.SHA1)
	}
}

// markOffline marks cached ROMs as playable offline.
func (p *picker) markOffline(cached map[string]bool) {
	library.MarkOffline(p.all, cached)
	library.MarkOffline(p.entries, cached)
}

// setQuery filters the list and goes back to its top.
func (p *picker) setQuery(q string) {
	if q == p.query {
		return
	}

	p.query = q
	p.refilter()
	p.selected, p.offset = 0, 0
}

// setFamily filters the list to one platform family (AllFamilies: none)
// and goes back to its top.
func (p *picker) setFamily(f library.Family) {
	if f == p.family {
		return
	}

	p.family = f
	p.refilter()
	p.selected, p.offset = 0, 0
}

func (p *picker) refilter() {
	matching := library.Filter(p.all, p.query)
	p.counts = library.FamilyCounts(matching)
	p.entries = library.InFamily(matching, p.family)
	p.rows = buildRows(p.entries, p.canLoad)
	p.selected = min(p.selected, max(len(p.entries)-1, 0))
	p.scroll(0)
}

// buildRows returns the list as drawn: the load-file row, then a heading
// with a count before each group.
func buildRows(entries []library.Entry, canLoad bool) []pickerRow {
	out := make([]pickerRow, 0, len(entries)+3)
	if canLoad {
		out = append(out, pickerRow{kind: rowLoadFile})
	}

	counts := map[library.Group]int{}
	for _, e := range entries {
		counts[e.Group]++
	}

	group := library.Group(255)

	for i, e := range entries {
		if e.Group != group {
			group = e.Group
			out = append(out, pickerRow{kind: rowHeading, heading: group.Name() + "  (" + strconv.Itoa(counts[group]) + ")"})
		}

		out = append(out, pickerRow{kind: rowEntry, entry: i})
	}

	return out
}

// setView sets the visible height and the row height, and keeps the scroll
// position in range.
func (p *picker) setView(viewH, rowH float64) {
	p.viewH, p.rowH = viewH, max(rowH, 1)
	p.scroll(0)
}

// maxOffset is the furthest the list can scroll.
func (p *picker) maxOffset() float64 {
	return max(0, float64(len(p.rows))*p.rowH-p.viewH)
}

// scroll moves the list by dy pixels (positive scrolls down), clamped.
func (p *picker) scroll(dy float64) {
	p.offset = min(max(p.offset+dy, 0), p.maxOffset())
}

// rowOf returns the row index of entry i.
func (p *picker) rowOf(i int) int {
	for r, row := range p.rows {
		if row.kind == rowEntry && row.entry == i {
			return r
		}
	}

	return 0
}

// ensureVisible scrolls just enough to show the selected entry.
func (p *picker) ensureVisible() {
	top := float64(p.rowOf(p.selected)) * p.rowH
	bottom := top + p.rowH

	switch {
	case top < p.offset:
		p.offset = top
	case bottom > p.offset+p.viewH:
		p.offset = bottom - p.viewH
	}

	p.scroll(0)
}

// move changes the selection by delta, wrapping at the ends, and scrolls to
// it.
func (p *picker) move(delta int) {
	n := len(p.entries)
	if n == 0 {
		return
	}

	p.selected = ((p.selected+delta)%n + n) % n
	p.ensureVisible()
}

// rowAt returns the row drawn at y pixels below the top of the list.
func (p *picker) rowAt(y float64) (pickerRow, bool) {
	if y < 0 || y >= p.viewH {
		return pickerRow{}, false
	}

	r := int((y + p.offset) / p.rowH)
	if r < 0 || r >= len(p.rows) {
		return pickerRow{}, false
	}

	return p.rows[r], true
}

// current returns the selected entry.
func (p *picker) current() (library.Entry, bool) {
	if len(p.entries) == 0 {
		return library.Entry{}, false
	}

	return p.entries[p.selected], true
}

// selectSHA1 selects the entry with the given SHA-1, if it's in the list.
func (p *picker) selectSHA1(sha1 string) {
	for i, e := range p.entries {
		if e.SHA1 == sha1 {
			p.selected = i
			p.ensureVisible()

			return
		}
	}
}
