package chip8

import (
	"errors"
	"strings"
	"testing"
)

func schip(legacy bool) *Machine {
	p := DefaultProfile(SuperChip)
	p.Quirks.LegacySuperChip = legacy

	return New(p)
}

// countLit returns how many screen pixels are on.
func countLit(m *Machine) int {
	n := 0

	v := m.Display()
	for y := range v.Height() {
		for x := range v.Width() {
			if v.Pixel(x, y) != 0 {
				n++
			}
		}
	}

	return n
}

func TestOpcodesExistPerPlatform(t *testing.T) {
	t.Parallel()

	tests := []struct {
		op   uint16
		want map[Platform]bool
	}{
		{0x00FF, map[Platform]bool{ModernChip8: false, Chip48: false, SuperChip1: true, SuperChip: true}},
		{0x00FB, map[Platform]bool{ModernChip8: false, Chip48: false, SuperChip1: false, SuperChip: true}},
		{0x00C1, map[Platform]bool{ModernChip8: false, Chip48: false, SuperChip1: false, SuperChip: true}},
		{0xF030, map[Platform]bool{ModernChip8: false, Chip48: false, SuperChip1: true, SuperChip: true}},
		{0xF075, map[Platform]bool{OriginalChip8: false, Chip48: false, SuperChip1: true, SuperChip: true}},
		{0x00DE, map[Platform]bool{SuperChip: false}},
	}

	for _, tt := range tests {
		for p, want := range tt.want {
			m := New(DefaultProfile(p))
			exec(m, tt.op)

			if got := !errors.Is(m.State().Err, ErrUnknownOpcode); got != want {
				t.Errorf("%s %04X: exists=%v, want %v (err %v)", p, tt.op, got, want, m.State().Err)
			}

			if got := strings.HasPrefix(Disassemble(p, tt.op), "DATA"); got == want {
				t.Errorf("%s %04X: disassembly %q", p, tt.op, Disassemble(p, tt.op))
			}
		}
	}
}

func TestExitHalts(t *testing.T) {
	t.Parallel()

	m := schip(false)
	exec(m, 0x00FD)

	if !errors.Is(m.State().Err, ErrProgramExit) {
		t.Errorf("err=%v, want ErrProgramExit", m.State().Err)
	}
}

func TestResolutionSwitch(t *testing.T) {
	t.Parallel()

	for _, legacy := range []bool{false, true} {
		m := schip(legacy)
		m.i = FontAddr
		exec(m, 0xD015) // a glyph at the origin

		before := countLit(m)
		if before == 0 {
			t.Fatal("nothing drawn")
		}

		exec(m, 0x00FF)

		v := m.Display()
		if v.Width() != hiresWidth || v.Height() != hiresHeight {
			t.Errorf("legacy=%v: hi-res is %d×%d", legacy, v.Width(), v.Height())
		}

		if got := countLit(m); (got == before) != legacy {
			t.Errorf("legacy=%v: %d pixels lit after switching, had %d", legacy, got, before)
		}
	}
}

func TestLegacyLoresIsDoubled(t *testing.T) {
	t.Parallel()

	m := schip(true)
	m.i = 0x300
	m.mem[0x300] = 0x80
	m.v[1], m.v[2] = 3, 2
	exec(m, 0xD121)

	v := m.Display()
	if v.Width() != hiresWidth || v.Height() != hiresHeight {
		t.Fatalf("legacy lo-res is %d×%d, want the full screen", v.Width(), v.Height())
	}

	for _, p := range [][2]int{{6, 4}, {7, 4}, {6, 5}, {7, 5}} {
		if v.Pixel(p[0], p[1]) != 1 {
			t.Errorf("pixel %v off", p)
		}
	}

	if n := countLit(m); n != 4 {
		t.Errorf("%d pixels lit, want 4", n)
	}
}

func TestLegacyLoresWaitsForVBlank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		legacy bool
		hires  bool
		wait   bool
	}{
		{"modern lo-res", false, false, false},
		{"modern hi-res", false, true, false},
		{"legacy lo-res", true, false, true},
		{"legacy hi-res", true, true, false},
	}

	for _, tt := range tests {
		m := schip(tt.legacy)
		m.setHires(tt.hires)
		m.stopFrame = false
		exec(m, 0xD011)

		if m.stopFrame != tt.wait {
			t.Errorf("%s: stopFrame=%v, want %v", tt.name, m.stopFrame, tt.wait)
		}
	}
}

func TestBigSpriteShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		p      Platform
		legacy bool
		hires  bool
		n      int
		w, h   int
	}{
		{"chip-8 N=0 draws nothing", ModernChip8, false, false, 0, spriteWidth, 0},
		{"modern lo-res", SuperChip, false, false, 0, bigSpriteSize, bigSpriteSize},
		{"modern hi-res", SuperChip, false, true, 0, bigSpriteSize, bigSpriteSize},
		{"legacy lo-res", SuperChip, true, false, 0, spriteWidth, bigSpriteSize},
		{"legacy hi-res", SuperChip, true, true, 0, bigSpriteSize, bigSpriteSize},
		{"ordinary sprite", SuperChip, false, true, 5, spriteWidth, 5},
	}

	for _, tt := range tests {
		prof := DefaultProfile(tt.p)
		prof.Quirks.LegacySuperChip = tt.legacy
		m := New(prof)
		m.hires = tt.hires

		if w, h := m.spriteShape(tt.n); w != tt.w || h != tt.h {
			t.Errorf("%s: %d×%d, want %d×%d", tt.name, w, h, tt.w, tt.h)
		}
	}
}

func TestBigSpriteDraws16Wide(t *testing.T) {
	t.Parallel()

	m := schip(false)
	m.setHires(true)
	m.i = 0x300
	m.mem[0x300], m.mem[0x301] = 0x80, 0x01 // first and last column of row 0

	exec(m, 0xD010)

	v := m.Display()
	if v.Pixel(0, 0) != 1 || v.Pixel(15, 0) != 1 || countLit(m) != 2 {
		t.Errorf("pixels (0,0)=%d (15,0)=%d lit=%d", v.Pixel(0, 0), v.Pixel(15, 0), countLit(m))
	}
}

func TestHiresCollisionFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		legacy bool
		wantVF byte
	}{
		{"modern is 0 or 1", false, 1},
		{"legacy counts rows hit plus rows clipped", true, 3},
	}

	for _, tt := range tests {
		m := schip(tt.legacy)
		m.setHires(true)
		m.i = 0x300
		m.mem[0x300], m.mem[0x301], m.mem[0x302] = 0x80, 0x80, 0x80
		m.v[1], m.v[2] = 0, 62 // two rows fit, the third is clipped

		exec(m, 0xD122) // draw the two rows
		m.pc = ProgramStart
		exec(m, 0xD123) // redraw: 2 rows collide and 1 is clipped

		if m.v[flagReg] != tt.wantVF {
			t.Errorf("%s: VF=%d, want %d", tt.name, m.v[flagReg], tt.wantVF)
		}
	}
}

func TestScroll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		legacy       bool
		hires        bool
		op           uint16
		wantX, wantY int
	}{
		{"modern lo-res right", false, false, 0x00FB, 14, 10},
		{"modern lo-res left", false, false, 0x00FC, 6, 10},
		{"modern lo-res down", false, false, 0x00C3, 10, 13},
		{"hi-res right", false, true, 0x00FB, 14, 10},
		{"hi-res down", false, true, 0x00C5, 10, 15},
		{"legacy lo-res right moves screen pixels", true, false, 0x00FB, 24, 20},
		{"legacy lo-res down moves screen pixels", true, false, 0x00C1, 20, 21},
	}

	for _, tt := range tests {
		m := schip(tt.legacy)
		m.setHires(tt.hires)
		m.i = 0x300
		m.mem[0x300] = 0x80
		m.v[1], m.v[2] = 10, 10
		exec(m, 0xD121)
		m.stopFrame = false
		exec(m, tt.op)

		if m.Display().Pixel(tt.wantX, tt.wantY) != 1 {
			t.Errorf("%s: pixel (%d,%d) off", tt.name, tt.wantX, tt.wantY)
		}
	}
}

func TestScrollDropsPixelsAtTheEdge(t *testing.T) {
	t.Parallel()

	m := schip(false)
	m.i = 0x300
	m.mem[0x300] = 0x80
	m.v[1], m.v[2] = 62, 31
	exec(m, 0xD121)
	exec(m, 0x00FB)

	if n := countLit(m); n != 0 {
		t.Errorf("%d pixels left after scrolling off the right edge", n)
	}

	exec(m, 0xD121)
	exec(m, 0x00C8)

	if n := countLit(m); n != 0 {
		t.Errorf("%d pixels left after scrolling off the bottom edge", n)
	}
}

func TestBigFont(t *testing.T) {
	t.Parallel()

	m := schip(false)
	m.v[3] = 0x1A // low nibble selects glyph A
	exec(m, 0xF330)

	if want := uint16(BigFontAddr + 0xA*bigFontHeight); m.i != want {
		t.Errorf("I=%03X, want %03X", m.i, want)
	}

	if m.Peek(BigFontAddr) != bigFont[0] {
		t.Error("big font not in memory")
	}

	if int(BigFontAddr)+len(bigFont) > ProgramStart {
		t.Error("big font overlaps the program")
	}
}

func TestFlagRegisters(t *testing.T) {
	t.Parallel()

	m := schip(false)
	for r := range numRegisters {
		m.v[r] = byte(r + 1)
	}

	exec(m, 0xFF75) // X > 7 stores V0..V7

	m.v = [numRegisters]byte{}
	exec(m, 0xFF85)

	if m.v[7] != 8 || m.v[8] != 0 {
		t.Errorf("V7=%d V8=%d, want 8 and 0", m.v[7], m.v[8])
	}

	m.Reset(m.profile)

	exec(m, 0xF185)

	if m.v[0] != 0 {
		t.Error("flags survived Reset")
	}
}
