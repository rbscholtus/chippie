package app

import "github.com/rbscholtus/chippie/app/roms"

// picker is the M3 ROM list: a simple keyboard- and mouse-driven menu. The
// M4 library drawer replaces it. It holds no Ebitengine state, so it can be
// tested directly.
type picker struct {
	entries  []roms.Entry
	selected int
}

func newPicker(entries []roms.Entry) *picker {
	return &picker{entries: entries}
}

// move changes the selection by delta, wrapping at the ends.
func (p *picker) move(delta int) {
	n := len(p.entries)
	if n == 0 {
		return
	}

	p.selected = ((p.selected+delta)%n + n) % n
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

			return
		}
	}
}

// pickerRow is one line of the list as drawn: a group heading or an entry.
type pickerRow struct {
	heading string
	entry   int // index into entries, or -1 for a heading
}

// rows returns the list as drawn: a heading before each group.
func (p *picker) rows() []pickerRow {
	out := make([]pickerRow, 0, len(p.entries)+2)
	group := roms.Group(255)

	for i, e := range p.entries {
		if e.Group != group {
			group = e.Group
			out = append(out, pickerRow{heading: groupName(group), entry: -1})
		}

		out = append(out, pickerRow{entry: i})
	}

	return out
}

// visibleRows returns the first row to show so that the selected entry is
// in view when only max rows fit.
func (p *picker) visibleRows(rows []pickerRow, maxRows int) int {
	if maxRows <= 0 || len(rows) <= maxRows {
		return 0
	}

	sel := 0

	for i, r := range rows {
		if r.entry == p.selected {
			sel = i

			break
		}
	}

	first := max(sel-maxRows/2, 0)

	return min(first, len(rows)-maxRows)
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
