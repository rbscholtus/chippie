package chip8

import (
	"errors"
	"strings"
	"testing"
)

func xo() *Machine { return New(DefaultProfile(XOChip)) }

func TestXOOpcodesExistOnlyOnXO(t *testing.T) {
	t.Parallel()

	for _, op := range []uint16{0x00D1, 0x5122, 0x5123, 0xF000, 0xF101, 0xF002, 0xF13A} {
		for _, p := range []Platform{ModernChip8, SuperChip} {
			m := New(DefaultProfile(p))
			exec(m, op)

			if !errors.Is(m.State().Err, ErrUnknownOpcode) {
				t.Errorf("%s %04X: err=%v, want unknown", p, op, m.State().Err)
			}

			if !strings.HasPrefix(Disassemble(p, op), "DATA") {
				t.Errorf("%s %04X: disassembly %q", p, op, Disassemble(p, op))
			}
		}

		m := xo()
		exec(m, op)

		if err := m.State().Err; err != nil {
			t.Errorf("xochip %04X: %v", op, err)
		}

		if strings.HasPrefix(Disassemble(XOChip, op), "DATA") {
			t.Errorf("xochip %04X: no disassembly", op)
		}
	}
}

func TestLongLoadAndSkip(t *testing.T) {
	t.Parallel()

	m := xo()
	m.Poke(ProgramStart+2, 0xBE)
	m.Poke(ProgramStart+3, 0xEF)
	exec(m, 0xF000)

	if m.i != 0xBEEF || m.pc != ProgramStart+4 {
		t.Errorf("I=%04X PC=%04X", m.i, m.pc)
	}

	// A skip steps over all four bytes.
	m = xo()
	m.Poke(ProgramStart+2, 0xF0)
	m.Poke(ProgramStart+3, 0x00)
	exec(m, 0x3000) // V0 == 0: skip

	if m.pc != ProgramStart+6 {
		t.Errorf("skip over F000: PC=%04X", m.pc)
	}

	// On SUPER-CHIP there's no such thing.
	s := schip(false)
	s.Poke(ProgramStart+2, 0xF0)
	s.Poke(ProgramStart+3, 0x00)
	exec(s, 0x3000)

	if s.pc != ProgramStart+4 {
		t.Errorf("superchip skip: PC=%04X", s.pc)
	}
}

func TestSaveLoadRange(t *testing.T) {
	t.Parallel()

	m := xo()
	m.i = 0x400
	m.v[2], m.v[3], m.v[4] = 0xA, 0xB, 0xC

	exec(m, 0x5242) // ascending

	if m.mem[0x400] != 0xA || m.mem[0x401] != 0xB || m.mem[0x402] != 0xC || m.mem[0x403] != 0 {
		t.Errorf("ascending save: % X", m.mem[0x400:0x404])
	}

	m.pc = ProgramStart
	exec(m, 0x5422) // descending: V4 first

	if m.mem[0x400] != 0xC || m.mem[0x402] != 0xA {
		t.Errorf("descending save: % X", m.mem[0x400:0x404])
	}

	m.v = [numRegisters]byte{}
	m.pc = ProgramStart
	exec(m, 0x5243)

	if m.v[2] != 0xC || m.v[4] != 0xA || m.i != 0x400 {
		t.Errorf("load: V2=%X V4=%X I=%03X", m.v[2], m.v[4], m.i)
	}

	// Addresses wrap at the top of the 64 KB.
	m = xo()
	m.i = 0xFFFF
	m.v[0], m.v[1] = 1, 2
	exec(m, 0x5012)

	if m.mem[0xFFFF] != 1 || m.mem[0] != 2 {
		t.Errorf("wrap: %d %d", m.mem[0xFFFF], m.mem[0])
	}
}

func TestPlanesDraw(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		plane uint16
		want  byte
	}{
		{"none", 0, 0},
		{"first", 1, 1},
		{"second", 2, 2},
		{"both", 3, 3},
	}

	for _, tt := range tests {
		m := xo()
		m.i = 0x400
		m.mem[0x400], m.mem[0x401] = 0x80, 0x80
		exec(m, 0xF001|tt.plane<<8)
		exec(m, 0xD011)

		if got := m.Display().Pixel(0, 0); got != tt.want {
			t.Errorf("%s: pixel %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestTwoPlanesUseConsecutiveData(t *testing.T) {
	t.Parallel()

	m := xo()
	m.i = 0x400
	m.mem[0x400], m.mem[0x401] = 0x80, 0x40 // plane 1: column 0; plane 2: column 1
	exec(m, 0xF301)
	exec(m, 0xD011)

	v := m.Display()
	if v.Pixel(0, 0) != 1 || v.Pixel(1, 0) != 2 {
		t.Errorf("pixels %d %d, want 1 2", v.Pixel(0, 0), v.Pixel(1, 0))
	}
}

func TestPlaneCollision(t *testing.T) {
	t.Parallel()

	m := xo()
	m.i = 0x400
	m.mem[0x400] = 0x80
	exec(m, 0xF201)
	exec(m, 0xD011)
	exec(m, 0xF101) // the other plane: no collision
	exec(m, 0xD011)

	if m.v[flagReg] != 0 {
		t.Errorf("VF=%d drawing on an empty plane", m.v[flagReg])
	}

	exec(m, 0xF201)
	exec(m, 0xD011)

	if m.v[flagReg] != 1 {
		t.Errorf("VF=%d, want a collision", m.v[flagReg])
	}
}

func TestClearAndScrollAffectSelectedPlanes(t *testing.T) {
	t.Parallel()

	setup := func() *Machine {
		m := xo()
		m.i = 0x400
		m.mem[0x400], m.mem[0x401] = 0x80, 0x80
		m.v[0], m.v[1] = 10, 10
		exec(m, 0xF301)
		exec(m, 0xD011) // pixel (10,10) on both planes

		return m
	}

	m := setup()
	exec(m, 0xF101)
	exec(m, 0x00E0)

	if got := m.Display().Pixel(10, 10); got != 2 {
		t.Errorf("clear plane 1: pixel %d, want 2", got)
	}

	m = setup()
	exec(m, 0xF201)
	exec(m, 0x00D3) // up, plane 2 only

	v := m.Display()
	if v.Pixel(10, 10) != 1 || v.Pixel(10, 7) != 2 {
		t.Errorf("scroll plane 2: stayed %d, moved %d", v.Pixel(10, 10), v.Pixel(10, 7))
	}
}

func TestXOBigSpriteAndFlags(t *testing.T) {
	t.Parallel()

	m := xo()
	if w, h := m.spriteShape(0); w != bigSpriteSize || h != bigSpriteSize {
		t.Errorf("DXY0 is %d×%d", w, h)
	}

	// All 16 flag registers exist on XO-CHIP.
	m.v[15] = 0x5A
	exec(m, 0xFF75)

	m.v[15] = 0
	exec(m, 0xFF85)

	if m.v[15] != 0x5A {
		t.Errorf("VF=%X after reading flags back", m.v[15])
	}
}

func TestLegacyQuirkDoesNothingOnXO(t *testing.T) {
	t.Parallel()

	p := DefaultProfile(XOChip)
	p.Quirks.LegacySuperChip = true

	m := New(p)
	if v := m.Display(); v.Width() != loresWidth || m.legacy() {
		t.Errorf("width %d, legacy %v", v.Width(), m.legacy())
	}
}

func TestSoundPattern(t *testing.T) {
	t.Parallel()

	m := xo()
	if s := m.Sound(); s.HasPattern || s.Pitch != defaultPitch {
		t.Errorf("before F002: %+v", s)
	}

	m.i = 0x400
	m.mem[0x400], m.mem[0x40F] = 0xAA, 0x55
	exec(m, 0xF002)

	m.v[4] = 112
	exec(m, 0xF43A)

	s := m.Sound()
	if !s.HasPattern || s.Pattern[0] != 0xAA || s.Pattern[15] != 0x55 || s.Pitch != 112 {
		t.Errorf("after F002: %+v", s)
	}

	m.Reset(m.profile)

	if s := m.Sound(); s.HasPattern || s.Pitch != defaultPitch || s.Pattern[0] != 0 {
		t.Errorf("after Reset: %+v", s)
	}
}
