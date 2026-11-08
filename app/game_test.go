package app

import (
	"image"
	"testing"

	"github.com/rbscholtus/chippie/app/roms"
	"github.com/rbscholtus/chippie/chip8"
)

func testEntries() []roms.Entry {
	return []roms.Entry{
		{ID: "game/a", Title: "A", Group: roms.Games, Platform: chip8.ModernChip8},
		{ID: "game/b", Title: "B", Group: roms.Games, Platform: chip8.SuperChip},
		{ID: "test/c", Title: "C", Group: roms.TestSuite, Platform: chip8.ModernChip8},
	}
}

func TestPickerMoveWraps(t *testing.T) {
	t.Parallel()

	p := newPicker(testEntries())

	for _, step := range []struct{ delta, want int }{{1, 1}, {1, 2}, {1, 0}, {-1, 2}, {-4, 1}} {
		p.move(step.delta)

		if p.selected != step.want {
			t.Fatalf("move(%d) → %d, want %d", step.delta, p.selected, step.want)
		}
	}

	empty := newPicker(nil)
	empty.move(1)

	if _, ok := empty.current(); ok {
		t.Error("empty picker returned an entry")
	}
}

func TestPickerSelect(t *testing.T) {
	t.Parallel()

	p := newPicker(testEntries())

	p.selectID("test/c")

	if e, _ := p.current(); e.ID != "test/c" {
		t.Errorf("selectID: current %q", e.ID)
	}

	if p.selectIndex(3) || p.selectIndex(-1) {
		t.Error("selectIndex accepted an out-of-range index")
	}

	if !p.selectIndex(1) || p.selected != 1 {
		t.Error("selectIndex(1) failed")
	}
}

func TestPickerRowsHaveGroupHeadings(t *testing.T) {
	t.Parallel()

	rows := newPicker(testEntries()).rows()
	kinds := make([]byte, 0, len(rows))

	for _, r := range rows {
		if r.entry < 0 {
			kinds = append(kinds, 'H')
		} else {
			kinds = append(kinds, 'E')
		}
	}

	if string(kinds) != "HEEHE" {
		t.Errorf("rows %q, want HEEHE", kinds)
	}
}

func TestPickerVisibleRowsFollowSelection(t *testing.T) {
	t.Parallel()

	entries := make([]roms.Entry, 30)
	for i := range entries {
		entries[i] = roms.Entry{Group: roms.Games}
	}

	p := newPicker(entries)
	rows := p.rows() // 1 heading + 30 entries

	if got := p.visibleRows(rows, 40); got != 0 {
		t.Errorf("everything fits: first=%d, want 0", got)
	}

	p.selected = 29 // the last row

	if got := p.visibleRows(rows, 10); got != len(rows)-10 {
		t.Errorf("last entry: first=%d, want %d", got, len(rows)-10)
	}

	p.selected = 15 // row 16

	if got := p.visibleRows(rows, 10); got != 11 {
		t.Errorf("middle entry: first=%d, want 11", got)
	}
}

func TestShouldRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		s               state
		focused, halted bool
		want            bool
	}{
		{stateRunning, true, false, true},
		{stateRunning, false, false, false}, // tab hidden or window unfocused
		{stateRunning, true, true, false},   // halted on an error
		{statePicker, true, false, false},   // picker open: paused
		{stateLoading, true, false, false},
		{stateOff, true, false, false},
	}

	for _, tt := range tests {
		if got := shouldRun(tt.s, tt.focused, tt.halted); got != tt.want {
			t.Errorf("shouldRun(%d, focused=%v, halted=%v) = %v", tt.s, tt.focused, tt.halted, got)
		}
	}
}

func TestScreenRect(t *testing.T) {
	t.Parallel()

	for _, size := range []image.Point{{1024, 640}, {780, 1688}, {2560, 1440}, {300, 300}} {
		r := screenRect(size.X, size.Y)

		if !r.In(image.Rect(0, 0, size.X, size.Y)) || r.Empty() {
			t.Errorf("%v: rect %v outside the window", size, r)
		}

		if ratio := float64(r.Dx()) / float64(r.Dy()); ratio < 1.98 || ratio > 2.02 {
			t.Errorf("%v: aspect %.3f, want 2", size, ratio)
		}

		if cx := r.Min.X + r.Dx()/2; cx < size.X/2-1 || cx > size.X/2+1 {
			t.Errorf("%v: not centred horizontally (centre %d)", size, cx)
		}
	}
}
