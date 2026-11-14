package app

import (
	"image"
	"testing"

	"github.com/rbscholtus/chippie/app/library"
	"github.com/rbscholtus/chippie/app/skin"
	"github.com/rbscholtus/chippie/chip8"
)

func TestNeighbourSkipsUnplayableAndWraps(t *testing.T) {
	t.Parallel()

	p := newPicker([]library.Entry{
		entry("a", "Alpha", chip8.ModernChip8),
		entry("b", "Bravo", chip8.MegaChip8), // can't play yet
		entry("c", "Charlie", chip8.SuperChip),
		entry("d", "Delta", chip8.XOChip),
	}, false)

	for _, tc := range []struct {
		from   string
		n      int
		want   string
		wantCh int
	}{
		{"a", 1, "c", 2}, // Bravo skipped
		{"c", 1, "d", 3},
		{"d", 1, "a", 1},  // wraps forward
		{"a", -1, "d", 3}, // wraps back
		{"a", 4, "c", 2},  // more than a full turn
		{"", 1, "a", 1},   // no game yet: the first
		{"", -1, "d", 3},  // or, backwards, the last
	} {
		e, ch, ok := p.neighbour(tc.from, tc.n)
		if !ok || e.SHA1 != tc.want || ch != tc.wantCh {
			t.Errorf("neighbour(%q, %d) = %q ch %d %v; want %q ch %d", tc.from, tc.n, e.SHA1, ch, ok, tc.want, tc.wantCh)
		}
	}

	if _, _, ok := newPicker(nil, false).neighbour("a", 1); ok {
		t.Error("an empty list has a neighbour")
	}
}

func TestControlAngles(t *testing.T) {
	t.Parallel()

	dial := skin.Control{ID: controlChannel, Kind: skin.Dial, Steps: 13}
	knob := skin.Control{ID: controlVolume, Kind: skin.Knob, MinAngle: -135, MaxAngle: 135}
	tint := skin.Control{ID: controlTint, Kind: skin.Knob, MinAngle: -60, MaxAngle: 60}

	s := controlState{channel: 1, volume: 1, tint: len(tints) - 1}

	if a := s.angle(dial); a != -360.0/13 {
		t.Errorf("dial after one step: %g°", a)
	}

	if a := s.angle(knob); a != 135 {
		t.Errorf("volume at full: %g°", a)
	}

	if a := s.angle(tint); a != 60 {
		t.Errorf("last tint: %g°", a)
	}

	if a := s.angle(skin.Control{ID: controlPower, Kind: skin.Rocker}); a != 0 {
		t.Errorf("rocker turned %g°", a)
	}
}

func TestTintsHaveFullPalettes(t *testing.T) {
	t.Parallel()

	if tints[0].palette != nil {
		t.Error("the first tint should be the ROM's own colours")
	}

	for _, ti := range tints[1:] {
		for i, c := range ti.palette {
			if c.A != 0xFF {
				t.Errorf("%s: colour %d is transparent", ti.name, i)
			}
		}
	}
}

func TestRemoteFitsItsArea(t *testing.T) {
	t.Parallel()

	for _, area := range []image.Rectangle{
		image.Rect(48, 1100, 1122, 3200), // portrait phone
		image.Rect(1800, 40, 2500, 1000), // landscape column
		image.Rect(0, 0, 400, 400),
	} {
		var r remote
		r.layout(area)

		all := r.keys[0x1].Union(r.keys[0xF])
		frame := all.Inset(-all.Dx() / remoteFrame)

		if !frame.In(area) {
			t.Errorf("%v: frame %v sticks out", area, frame)
		}

		if got := r.keyAt(r.keys[0xA].Min.Add(image.Pt(2, 2))); got != 1<<0xA {
			t.Errorf("%v: keyAt on key A = %016b", area, got)
		}
	}
}
