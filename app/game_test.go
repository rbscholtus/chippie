package app

import (
	"fmt"
	"image"
	"math"
	"strings"
	"testing"

	"github.com/rbscholtus/chippie/app/library"
	"github.com/rbscholtus/chippie/chip8"
)

// entry makes a library entry; Group follows the platform.
func entry(sha1, title string, p chip8.Platform) library.Entry {
	e := library.LocalEntry(nil, sha1, title)
	e.Platform = p

	if !p.Implemented() {
		e.Group = library.ComingSoon
	}

	return e
}

func testEntries() []library.Entry {
	return []library.Entry{
		entry("aa", "Alpha", chip8.ModernChip8),
		entry("bb", "Bravo", chip8.ModernChip8),
		entry("cc", "Charlie", chip8.MegaChip8),
	}
}

// longPicker returns a picker with n playable entries (plus one heading
// row), a 100-pixel view and 10-pixel rows.
func longPicker(n int) *picker {
	entries := make([]library.Entry, n)
	for i := range entries {
		entries[i] = entry(strings.Repeat("a", i+1), "Game", chip8.ModernChip8)
	}

	p := newPicker(entries, false)
	p.setView(100, 10)

	return p
}

func TestPickerMoveWraps(t *testing.T) {
	t.Parallel()

	p := newPicker(testEntries(), false)

	for _, step := range []struct{ delta, want int }{{1, 1}, {1, 2}, {1, 0}, {-1, 2}, {-4, 1}} {
		p.move(step.delta)

		if p.selected != step.want {
			t.Fatalf("move(%d) → %d, want %d", step.delta, p.selected, step.want)
		}
	}

	empty := newPicker(nil, false)
	empty.move(1)

	if _, ok := empty.current(); ok {
		t.Error("empty picker returned an entry")
	}
}

func TestPickerRows(t *testing.T) {
	t.Parallel()

	kinds := func(p *picker) string {
		out := make([]byte, 0, len(p.rows))

		for _, r := range p.rows {
			out = append(out, "HEL"[r.kind])
		}

		return string(out)
	}

	if got := kinds(newPicker(testEntries(), false)); got != "HEEHE" {
		t.Errorf("rows %q, want HEEHE (heading, 2 playable, heading, 1 coming soon)", got)
	}

	p := newPicker(testEntries(), true)
	if got := kinds(p); got != "LHEEHE" {
		t.Errorf("with the load row: %q, want LHEEHE", got)
	}

	if !strings.HasPrefix(p.rows[1].heading, "PLAY  (2)") {
		t.Errorf("heading %q should count its entries", p.rows[1].heading)
	}
}

func TestPickerSearch(t *testing.T) {
	t.Parallel()

	p := newPicker(testEntries(), false)
	p.setView(100, 10)
	p.move(2)
	p.setQuery("BRA")

	if len(p.entries) != 1 || p.entries[0].Title != "Bravo" || p.selected != 0 || p.offset != 0 {
		t.Errorf("search: %d entries, selected %d", len(p.entries), p.selected)
	}

	p.setQuery("")

	if len(p.entries) != 3 {
		t.Errorf("cleared search: %d entries", len(p.entries))
	}

	// A refreshed library keeps the search and the selected ROM.
	p.selectSHA1("cc")
	p.setAll(append(testEntries(), entry("dd", "Delta", chip8.ModernChip8)))

	if e, _ := p.current(); e.SHA1 != "cc" || len(p.entries) != 4 {
		t.Errorf("after setAll: selected %q of %d", e.SHA1, len(p.entries))
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

func TestPickerRowAt(t *testing.T) {
	t.Parallel()

	p := longPicker(30)

	if r, ok := p.rowAt(5); !ok || r.kind != rowHeading {
		t.Errorf("y=5: %+v %v, want the heading", r, ok)
	}

	if r, ok := p.rowAt(15); !ok || r.kind != rowEntry || r.entry != 0 {
		t.Errorf("y=15: %+v, want entry 0", r)
	}

	p.scroll(100) // now row 10 (entry 9) is at the top

	if r, ok := p.rowAt(5); !ok || r.entry != 9 {
		t.Errorf("scrolled, y=5: %+v, want entry 9", r)
	}

	if _, ok := p.rowAt(150); ok {
		t.Error("a point below the view should not be a row")
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
		below := testTV(t, window, sz.scale, true).below
		l := computeLayout(window, below, sz.scale, textSize*lineSpacing*sz.scale, true)

		if !l.list.In(l.panel) || !l.search.In(l.panel) || !l.panel.In(window) ||
			!l.statusLeft.In(window) || !l.statusRight.In(window) || l.statusLeft.Overlaps(l.statusRight) {
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

	// In portrait on an iPhone, at least ten rows are visible at once, and
	// the details pane is left out (too narrow).
	phone := image.Rect(0, 0, 1179, 1977)

	l := computeLayout(phone, testTV(t, phone, 3, true).below, 3, textSize*lineSpacing*3, true)
	if rows := float64(l.list.Dy()) / l.rowH; rows < 10 || !l.details.Empty() {
		t.Errorf("portrait iPhone: %.1f rows, details %v", rows, l.details)
	}

	// A desktop window is wide enough for the details pane beside the list.
	desk := image.Rect(0, 0, 2048, 1280)

	d := computeLayout(desk, testTV(t, desk, 2, false).below, 2, textSize*lineSpacing*2, false)
	if d.details.Empty() || d.details.Overlaps(d.list) || d.details.Min.X < d.list.Max.X {
		t.Errorf("desktop: list %v, details %v", d.list, d.details)
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

// testTV lays out the real skin in a window.
func testTV(t *testing.T, window image.Rectangle, scale float64, remote bool) tvLayout {
	t.Helper()

	skins, err := loadSkins()
	if err != nil {
		t.Fatal(err)
	}

	sk := skins[0].skin

	return computeTV(sk, window, scale, textSize*lineSpacing*scale, remote)
}

func TestComputeTV(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		w, h   int
		scale  float64
		remote bool
	}{
		{1024, 640, 1, false},
		{2560, 1440, 2, false},
		{300, 300, 1, false},
		{1179, 1977, 3, true},
		{780, 1688, 2, true},
		{2556, 1029, 3, true},
		{1179, 1977, 3, false},
	} {
		window := image.Rect(0, 0, tc.w, tc.h)
		l := testTV(t, window, tc.scale, tc.remote)
		name := fmt.Sprintf("%dx%d remote=%v", tc.w, tc.h, tc.remote)

		if !l.cabinet.In(window) || !l.tube.In(l.cabinet) || !l.picture.In(l.tube) || l.picture.Empty() {
			t.Errorf("%s: cabinet %v, tube %v, picture %v don't nest in the window", name, l.cabinet, l.tube, l.picture)
		}

		if ratio := float64(l.picture.Dx()) / float64(l.picture.Dy()); ratio < 1.98 || ratio > 2.02 {
			t.Errorf("%s: picture aspect %.3f, want 2", name, ratio)
		}

		if l.below.Min.Y != l.cabinet.Max.Y || l.below.Max.Y > tc.h {
			t.Errorf("%s: status %v not right under the TV %v", name, l.below, l.cabinet)
		}

		if tc.remote != !l.remote.Empty() || l.remote.Overlaps(l.cabinet) || l.remote.Overlaps(l.below) {
			t.Errorf("%s: remote %v (TV %v, status %v)", name, l.remote, l.cabinet, l.below)
		}

		if tc.remote && tc.h > tc.w && min(l.remote.Dx(), l.remote.Dy())/4 < int(remoteMinKey*tc.scale) {
			t.Errorf("%s: remote %v leaves keys under %d pt", name, l.remote, remoteMinKey)
		}

		// Cabinet pixels map back and forth.
		x, y := l.toCabinet(l.cabinet.Max)
		if bx, by := l.fromCabinet(x, y); int(bx+0.5) != l.cabinet.Max.X || int(by+0.5) != l.cabinet.Max.Y {
			t.Errorf("%s: toCabinet/fromCabinet don't round-trip", name)
		}
	}
}

func TestPictureInRotated(t *testing.T) {
	t.Parallel()

	tube := image.Rect(0, 0, 1300, 1000)

	for _, tc := range []struct {
		rot  int
		tall bool
	}{{0, false}, {90, true}, {180, false}, {270, true}} {
		p := pictureIn(tube, tc.rot)
		if !p.In(tube) || (p.Dy() > p.Dx()) != tc.tall {
			t.Errorf("rotation %d: picture %v in tube %v", tc.rot, p, tube)
		}

		want := 2.0
		if tc.tall {
			want = 0.5
		}

		if r := float64(p.Dx()) / float64(p.Dy()); math.Abs(r-want) > 0.02 {
			t.Errorf("rotation %d: aspect %.3f, want %.1f", tc.rot, r, want)
		}
	}
}
