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

// longPicker returns a picker with n game entries (plus one heading row),
// a 100-pixel view and 10-pixel rows.
func longPicker(n int) *picker {
	entries := make([]roms.Entry, n)
	for i := range entries {
		entries[i] = roms.Entry{Group: roms.Games}
	}

	p := newPicker(entries)
	p.setView(100, 10)

	return p
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

	rows := newPicker(testEntries()).rows
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

func TestPickerScrollClamps(t *testing.T) {
	t.Parallel()

	p := longPicker(30) // 31 rows × 10 px = 310 px in a 100 px view

	p.scroll(-50)

	if p.offset != 0 {
		t.Errorf("scrolled above the top: offset %v", p.offset)
	}

	p.scroll(1000)

	if p.offset != 210 {
		t.Errorf("offset %v after scrolling past the end, want 210", p.offset)
	}

	// A short list never scrolls.
	short := longPicker(3)
	short.scroll(50)

	if short.offset != 0 {
		t.Errorf("short list scrolled to %v", short.offset)
	}
}

func TestPickerKeyboardKeepsSelectionVisible(t *testing.T) {
	t.Parallel()

	p := longPicker(30)

	p.move(-1) // wraps to the last entry, row 30: y 300..310

	if p.offset != 210 {
		t.Errorf("last entry: offset %v, want 210", p.offset)
	}

	p.move(1) // wraps to the first entry, row 1: y 10..20

	if p.offset != 10 {
		t.Errorf("first entry: offset %v, want 10", p.offset)
	}
}

func TestPickerEntryAt(t *testing.T) {
	t.Parallel()

	p := longPicker(30)

	if _, ok := p.entryAt(5); ok {
		t.Error("the heading row should not be an entry")
	}

	if i, ok := p.entryAt(15); !ok || i != 0 {
		t.Errorf("y=15: entry %d %v, want 0", i, ok)
	}

	p.scroll(100) // now row 10 (entry 9) is at the top

	if i, ok := p.entryAt(5); !ok || i != 9 {
		t.Errorf("scrolled, y=5: entry %d %v, want 9", i, ok)
	}

	if _, ok := p.entryAt(150); ok {
		t.Error("a point below the view should not be an entry")
	}
}

func TestPointerTapAndDrag(t *testing.T) {
	t.Parallel()

	p := pointer{threshold: 8}

	// A small wobble is still a tap, reported where it started.
	p.press(image.Pt(100, 100))

	if dy := p.move(image.Pt(103, 104)); dy != 0 {
		t.Errorf("wobble scrolled by %d", dy)
	}

	if at, tap := p.release(image.Pt(103, 104)); !tap || at != image.Pt(100, 100) {
		t.Errorf("wobble: tap=%v at %v", tap, at)
	}

	// A longer move is a drag: it scrolls and never taps.
	p.press(image.Pt(100, 100))

	if dy := p.move(image.Pt(100, 130)); dy != 30 {
		t.Errorf("drag: dy %d, want 30", dy)
	}

	if dy := p.move(image.Pt(100, 120)); dy != -10 {
		t.Errorf("drag back: dy %d, want -10", dy)
	}

	if _, tap := p.release(image.Pt(100, 120)); tap {
		t.Error("a drag ended as a tap")
	}

	// Releasing without a press does nothing.
	if _, tap := p.release(image.Pt(1, 1)); tap {
		t.Error("release without press tapped")
	}
}

func TestLayoutFitsPhones(t *testing.T) {
	t.Parallel()

	// iPhone 15 portrait and landscape at 3×, and a desktop window at 1×.
	sizes := []struct {
		w, h  int
		scale float64
	}{{1179, 1977, 3}, {2556, 1029, 3}, {1024, 640, 1}}

	for _, sz := range sizes {
		window := image.Rect(0, 0, sz.w, sz.h)
		screen := screenRect(sz.w, sz.h)
		l := computeLayout(window, screen, sz.scale, textSize*lineSpacing*sz.scale, true)

		if !l.list.In(l.panel) || !l.panel.In(window) || !l.status.In(window) {
			t.Errorf("%dx%d: rectangles outside the window: %+v", sz.w, sz.h, l)
		}

		if l.rowH < touchRowSize*sz.scale {
			t.Errorf("%dx%d: touch rows %.0f px, want at least %.0f", sz.w, sz.h, l.rowH, touchRowSize*sz.scale)
		}

		// The picker panel uses most of the window, not just the picture.
		if l.panel.Dy() < sz.h*7/10 {
			t.Errorf("%dx%d: panel height %d is under 70%% of the window", sz.w, sz.h, l.panel.Dy())
		}
	}

	// In portrait on an iPhone, most of the M3 list (16 rows) is visible at
	// once; the rest is a short drag away.
	l := computeLayout(image.Rect(0, 0, 1179, 1977), screenRect(1179, 1977), 3, textSize*lineSpacing*3, true)
	if rows := float64(l.list.Dy()) / l.rowH; rows < 10 {
		t.Errorf("portrait iPhone shows %.1f rows, want at least 10", rows)
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
