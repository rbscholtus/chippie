package chip8

import "testing"

// twoPageMachine returns a twoPageChip8 machine that has taken the 1260 start,
// so it is in 64×64 mode at 0x2C0.
func twoPageMachine(t *testing.T, q func(*Quirks)) *Machine {
	t.Helper()

	prof := DefaultProfile(TwoPageChip8)
	if q != nil {
		q(&prof.Quirks)
	}

	m := New(prof)
	mustLoad(t, m, program(twoPageStartOp))
	m.step()

	if v := m.Display(); v.Width() != twoPageSize || v.Height() != twoPageSize || m.pc != twoPageEntry {
		t.Fatalf("after 1260: %d×%d at %03X, want 64×64 at 2C0", v.Width(), v.Height(), m.pc)
	}

	return m
}

func TestTwoPageStart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		p      Platform
		rom    []byte
		steps  int
		wantPC uint16
		wantH  int
	}{
		{"1260 at 0x200 starts 64×64", TwoPageChip8, program(twoPageStartOp), 1, twoPageEntry, twoPageSize},
		{"1260 elsewhere is a jump", TwoPageChip8, program(0x1204, 0, twoPageStartOp), 2, 0x260, loresHeight},
		{"other first instruction stays 64×32", TwoPageChip8, program(0x00E0), 1, ProgramStart + 2, loresHeight},
		{"originalChip8 jumps", OriginalChip8, program(twoPageStartOp), 1, 0x260, loresHeight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := New(DefaultProfile(tt.p))
			mustLoad(t, m, tt.rom)

			for range tt.steps {
				m.step()
			}

			if m.Halted() {
				t.Fatal(m.State().Err)
			}

			v := m.Display()
			if m.pc != tt.wantPC || v.Width() != loresWidth || v.Height() != tt.wantH {
				t.Errorf("PC=%03X %d×%d, want %03X 64×%d", m.pc, v.Width(), v.Height(), tt.wantPC, tt.wantH)
			}
		})
	}
}

func TestTwoPageResetGoesBackTo64x32(t *testing.T) {
	t.Parallel()

	m := twoPageMachine(t, nil)
	mustLoad(t, m, program(0x00E0))

	if v := m.Display(); v.Width() != loresWidth || v.Height() != loresHeight {
		t.Errorf("after LoadROM: %d×%d, want 64×32", v.Width(), v.Height())
	}
}

func TestTwoPage0230ClearsWholeScreen(t *testing.T) {
	t.Parallel()

	m := twoPageMachine(t, nil)
	for i := range m.disp.pix {
		m.disp.pix[i] = 1
	}

	exec(m, twoPageClearOp)

	if m.Halted() {
		t.Fatal(m.State().Err)
	}

	v := m.Display()

	for y := range twoPageSize {
		for x := range twoPageSize {
			if v.Pixel(x, y) != 0 {
				t.Fatalf("pixel (%d,%d) still lit after 0230", x, y)
			}
		}
	}
}

func TestTwoPageSpritesAtBottomEdge(t *testing.T) {
	t.Parallel()

	// The font's "0" is 5 rows; drawn at row 62, two fit above row 64.
	for _, wrap := range []bool{false, true} {
		m := twoPageMachine(t, func(q *Quirks) { q.Wrap = wrap })
		m.i = FontAddr
		m.v[1], m.v[2] = 0, 62
		exec(m, 0xD125)

		v := m.Display()
		if v.Pixel(0, 62) == 0 || v.Pixel(0, 63) == 0 {
			t.Errorf("wrap=%v: rows 62 and 63 should be drawn", wrap)
		}

		if got := v.Pixel(0, 0) != 0; got != wrap {
			t.Errorf("wrap=%v: row 0 drawn=%v", wrap, got)
		}

		// Row 32 is on screen in 64×64: nothing drawn there either way.
		if v.Pixel(0, loresHeight) != 0 {
			t.Errorf("wrap=%v: row 32 drawn", wrap)
		}
	}

	// A start row of 64 or more wraps to the top, as on every platform.
	m := twoPageMachine(t, nil)
	m.i = FontAddr
	m.v[1], m.v[2] = 0, twoPageSize+2
	exec(m, 0xD125)

	if m.Display().Pixel(0, 2) == 0 {
		t.Error("a sprite at row 66 should start at row 2")
	}
}
