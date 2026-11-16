package chip8

import (
	"bytes"
	"testing"
)

// twoPageMachine returns a twoPageChip8 machine that has started the 2 Page
// Display of a self-patching ROM, so it is in 64×64 mode at 0x2C0.
func twoPageMachine(t *testing.T, q func(*Quirks)) *Machine {
	t.Helper()

	prof := DefaultProfile(TwoPageChip8)
	if q != nil {
		q(&prof.Quirks)
	}

	m := New(prof)
	mustLoad(t, m, TwoPageROM(true, nil))
	m.step()

	if v := m.Display(); v.Width() != twoPageSize || v.Height() != twoPageSize || m.pc != twoPageSelfPatchedProgram {
		t.Fatalf("after 1260: %d×%d at %03X, want 64×64 at 2C0", v.Width(), v.Height(), m.pc)
	}

	return m
}

func TestTwoPageStart(t *testing.T) {
	t.Parallel()

	// A patch with one byte changed is not the 2 Page Display.
	almost := TwoPageROM(false, nil)
	almost[0x30] ^= 0xFF

	// Padding other than "REVIVAL2011" (Hires Test has zeros) doesn't matter.
	zeros := TwoPageROM(true, nil)
	copy(zeros[0xB3:0xBE], make([]byte, 11))

	// Without the self-patching code's last 0230, the program starts at 0x260.
	noClear := TwoPageROM(true, nil)
	noClear[0xBF] = 0x00

	tests := []struct {
		name   string
		p      Platform
		rom    []byte
		wantPC uint16
		wantH  int
	}{
		{"article's layout starts at 0x260", TwoPageChip8, TwoPageROM(false, nil), twoPageProgram, twoPageSize},
		{"self-patching ROM starts at 0x2C0", TwoPageChip8, TwoPageROM(true, nil), twoPageSelfPatchedProgram, twoPageSize},
		{"any padding before 0x2BE", TwoPageChip8, zeros, twoPageSelfPatchedProgram, twoPageSize},
		{"no 0230 at 0x2BE", TwoPageChip8, noClear, twoPageProgram, twoPageSize},
		{"1260 without the patch is a jump", TwoPageChip8, program(twoPageStartOp), twoPageProgram, loresHeight},
		{"a changed patch is a jump", TwoPageChip8, almost, twoPageProgram, loresHeight},
		{"other first instruction stays 64×32", TwoPageChip8, program(0x00E0), ProgramStart + 2, loresHeight},
		{"originalChip8 just jumps", OriginalChip8, TwoPageROM(true, nil), twoPageProgram, loresHeight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := New(DefaultProfile(tt.p))
			mustLoad(t, m, tt.rom)
			m.step()

			if m.Halted() {
				t.Fatal(m.State().Err)
			}

			v := m.Display()
			if m.pc != tt.wantPC || v.Width() != loresWidth || v.Height() != tt.wantH {
				t.Errorf("PC=%03X %d×%d, want %03X 64×%d", m.pc, v.Width(), v.Height(), tt.wantPC, tt.wantH)
			}

			if got := m.twoPage; got != (tt.wantH == twoPageSize) {
				t.Errorf("twoPage = %v", got)
			}
		})
	}
}

// TestTwoPageArticleTestProgram runs the test program printed with the
// article (VIPER 1/3, p. 12), at 0x260: 16 squares of 4×4 pixels down the
// diagonal of the 64×64 screen. (kripod's "Hires Test" is the same program,
// moved to 0x2C0.)
func TestTwoPageArticleTestProgram(t *testing.T) {
	t.Parallel()

	prog := program(
		0x6000, 0x6100, 0xA273, 0xD014, 0x7004, 0x7104, 0x3140, 0x1266, 0x1270,
		0x01F0, 0xF0F0, 0xF0D4,
	)

	m := New(DefaultProfile(TwoPageChip8))
	mustLoad(t, m, TwoPageROM(false, prog))

	for range 200 {
		m.step()
	}

	if m.Halted() {
		t.Fatal(m.State().Err)
	}

	v := m.Display()

	for i := range 16 {
		x := 4 * i
		if v.Pixel(x, x) == 0 || v.Pixel(x+3, x+3) == 0 {
			t.Errorf("square %d at (%d,%d) not drawn", i, x, x)
		}

		if i < 15 && v.Pixel(x+4, x) != 0 {
			t.Errorf("pixel right of square %d lit", i)
		}
	}
}

func TestTwoPageResetGoesBackTo64x32(t *testing.T) {
	t.Parallel()

	m := twoPageMachine(t, nil)
	mustLoad(t, m, program(0x00E0))

	if v := m.Display(); v.Width() != loresWidth || v.Height() != loresHeight || m.twoPage {
		t.Errorf("after LoadROM: %d×%d twoPage=%v, want 64×32 and false", v.Width(), v.Height(), m.twoPage)
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

// Test0230BeforeTwoPageIsMachineCode: without the 2 Page Display started,
// 0230 calls machine code, which Chippie doesn't run.
func Test0230BeforeTwoPageIsMachineCode(t *testing.T) {
	t.Parallel()

	m := New(DefaultProfile(TwoPageChip8))
	mustLoad(t, m, program(twoPageClearOp))
	m.step()

	if !m.Halted() {
		t.Error("0230 ran without the 2 Page Display")
	}

	// The disassembler sees one instruction; in a 2 Page Display program,
	// 0230 always runs after the start.
	if got := Disassemble(TwoPageChip8, twoPageClearOp); got != "CLS" {
		t.Errorf("Disassemble(0230) = %q, want CLS", got)
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

// TestTwoPagePatchIsTheArticles checks the patch against the article's
// listing (VIPER 1/3, p. 12, "0200 1260 GO 0260" to "0248 BBD4").
func TestTwoPagePatchIsTheArticles(t *testing.T) {
	t.Parallel()

	listing := program(
		0x1260, 0x017A, 0x4270, 0x2278, 0x2252, 0xC419, 0xF800, 0xA09B, 0xFA0E, 0xB0E2,
		0xE280, 0xE220, 0xA0E2, 0x3C15, 0x80E2, 0x20A0, 0x341C, 0x9832, 0x29AB, 0x2B8B,
		0xB888, 0x3203, 0x7B28, 0x3004, 0xF802, 0xAE9B, 0xBFF8, 0x00AF, 0xF800, 0x5F1F,
		0x8F3A, 0x382E, 0x8E3A, 0x38D4, 0x019B, 0xFF01, 0xBBD4,
	)

	if !bytes.Equal(listing[2:], twoPagePatch) || twoPageSelfPatchAddr != ProgramStart+len(listing) {
		t.Error("twoPagePatch differs from the article's 0x202–0x249")
	}
}
