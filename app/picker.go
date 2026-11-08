package app

import "github.com/rbscholtus/chippie/app/roms"

// picker is the M3 ROM list: a scrollable menu for keyboard, mouse and
// touch. The M4 library drawer replaces it. It holds no Ebitengine state, so
// it can be tested directly.
type picker struct {
	entries  []roms.Entry
	rows     []pickerRow
	selected int

	// Scrolling, in pixels: offset is how far the list is scrolled; viewH
	// is the visible height and rowH the height of one row.
	offset float64
	viewH  float64
	rowH   float64
}

// pickerRow is one line of the list as drawn: a group heading or an entry.
type pickerRow struct {
	heading string
	entry   int // index into entries, or -1 for a heading
}

func newPicker(entries []roms.Entry) *picker {
	p := &picker{entries: entries, rowH: 1}
	p.rows = buildRows(entries)

	return p
}

// buildRows returns the list as drawn: a heading before each group.
func buildRows(entries []roms.Entry) []pickerRow {
	out := make([]pickerRow, 0, len(entries)+2)
	group := roms.Group(255)

	for i, e := range entries {
		if e.Group != group {
			group = e.Group
			out = append(out, pickerRow{heading: groupName(group), entry: -1})
		}

		out = append(out, pickerRow{entry: i})
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
		if row.entry == i {
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

// entryAt returns the entry drawn at y pixels below the top of the list,
// or false for a heading or empty space.
func (p *picker) entryAt(y float64) (int, bool) {
	if y < 0 || y >= p.viewH {
		return 0, false
	}

	r := int((y + p.offset) / p.rowH)
	if r < 0 || r >= len(p.rows) || p.rows[r].entry < 0 {
		return 0, false
	}

	return p.rows[r].entry, true
}

// selectIndex selects the entry at i, if it exists.
func (p *picker) selectIndex(i int) bool {
	if i < 0 || i >= len(p.entries) {
		return false
	}

	p.selected = i

	return true
}

// current returns the selected entry.
func (p *picker) current() (roms.Entry, bool) {
	if len(p.entries) == 0 {
		return roms.Entry{}, false
	}

	return p.entries[p.selected], true
}

// selectID selects the entry with the given ID.
func (p *picker) selectID(id string) {
	for i, e := range p.entries {
		if e.ID == id {
			p.selected = i
			p.ensureVisible()

			return
		}
	}
}

func groupName(g roms.Group) string {
	switch g {
	case roms.Games:
		return "GAMES  (chip8Archive, CC0)"
	case roms.TestSuite:
		return "TEST SUITE  (Timendus, GPLv3)"
	default:
		return ""
	}
}
