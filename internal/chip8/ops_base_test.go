package chip8

import (
	"errors"
	"testing"
)

// newTestMachine returns a modernChip8 machine with q applied to its
// default quirks.
func newTestMachine(q func(*Quirks)) *Machine {
	prof := DefaultProfile(ModernChip8)
	if q != nil {
		q(&prof.Quirks)
	}

	return New(prof)
}

// exec runs one opcode at the current PC.
func exec(m *Machine, op uint16) {
	m.mem[m.pc] = byte(op >> 8)
	m.mem[m.pc+1] = byte(op)
	m.step()
}

func TestOpcodeArithmetic(t *testing.T) {
	t.Parallel()

	type regs struct{ vx, vy byte }

	tests := []struct {
		name   string
		op     uint16 // X = 1, Y = 2 unless stated
		in     regs
		quirks func(*Quirks)
		wantVX byte
		wantVF byte
	}{
		{"6XNN load, VF unchanged", 0x61AB, regs{0, 0}, nil, 0xAB, 0x55},
		{"7XNN add wraps, VF unchanged", 0x7110, regs{0xF8, 0}, nil, 0x08, 0x55},
		{"8XY0 copy, VF unchanged", 0x8120, regs{1, 9}, nil, 9, 0x55},
		{"8XY1 or", 0x8121, regs{0xF0, 0x0F}, nil, 0xFF, 0x55},
		{"8XY1 or, logic quirk", 0x8121, regs{0xF0, 0x0F}, func(q *Quirks) { q.Logic = true }, 0xFF, 0},
		{"8XY2 and", 0x8122, regs{0xF3, 0x3F}, nil, 0x33, 0x55},
		{"8XY2 and, logic quirk", 0x8122, regs{0xF3, 0x3F}, func(q *Quirks) { q.Logic = true }, 0x33, 0},
		{"8XY3 xor", 0x8123, regs{0xFF, 0x0F}, nil, 0xF0, 0x55},
		{"8XY3 xor, logic quirk", 0x8123, regs{0xFF, 0x0F}, func(q *Quirks) { q.Logic = true }, 0xF0, 0},
		{"8XY4 add no carry", 0x8124, regs{0x7F, 0x80}, nil, 0xFF, 0},
		{"8XY4 add carry", 0x8124, regs{0x80, 0x80}, nil, 0x00, 1},
		{"8XY5 sub no borrow", 0x8125, regs{5, 3}, nil, 2, 1},
		{"8XY5 sub equal", 0x8125, regs{5, 5}, nil, 0, 1},
		{"8XY5 sub borrow", 0x8125, regs{3, 5}, nil, 0xFE, 0},
		{"8XY7 subn no borrow", 0x8127, regs{3, 5}, nil, 2, 1},
		{"8XY7 subn borrow", 0x8127, regs{5, 3}, nil, 0xFE, 0},
		{"8XY6 shr uses VY", 0x8126, regs{0xFF, 0x03}, nil, 0x01, 1},
		{"8XY6 shr, shift quirk uses VX", 0x8126, regs{0x02, 0x03}, func(q *Quirks) { q.Shift = true }, 0x01, 0},
		{"8XYE shl uses VY", 0x812E, regs{0x00, 0x81}, nil, 0x02, 1},
		{"8XYE shl, shift quirk uses VX", 0x812E, regs{0x40, 0x81}, func(q *Quirks) { q.Shift = true }, 0x80, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := newTestMachine(tt.quirks)
			m.v[1], m.v[2], m.v[flagReg] = tt.in.vx, tt.in.vy, 0x55
			exec(m, tt.op)

			if m.v[1] != tt.wantVX || m.v[flagReg] != tt.wantVF {
				t.Errorf("V1=%#02x VF=%#02x, want V1=%#02x VF=%#02x", m.v[1], m.v[flagReg], tt.wantVX, tt.wantVF)
			}
		})
	}
}

// TestFlagRegisterAsDestination checks that VF is written after the result,
// so the flag wins when X is F.
func TestFlagRegisterAsDestination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		op     uint16
		vf, v1 byte
		want   byte
	}{
		{"8FY4 carry", 0x8F14, 0xFF, 0x01, 1},
		{"8FY4 no carry", 0x8F14, 0x01, 0x01, 0},
		{"8FY5 no borrow", 0x8F15, 0x05, 0x01, 1},
		{"8FY7 borrow", 0x8F17, 0x05, 0x01, 0},
		{"8FY6 shifted-out bit", 0x8F16, 0x00, 0x03, 1},
		{"8FYE shifted-out bit", 0x8F1E, 0x00, 0x80, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := newTestMachine(nil)
			m.v[flagReg], m.v[1] = tt.vf, tt.v1
			exec(m, tt.op)

			if m.v[flagReg] != tt.want {
				t.Errorf("VF=%d, want %d", m.v[flagReg], tt.want)
			}
		})
	}
}

func TestOpcodeSkips(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		op       uint16
		v1, v2   byte
		key      int // key held down, or -1
		wantSkip bool
	}{
		{"3XNN equal", 0x3142, 0x42, 0, -1, true},
		{"3XNN not equal", 0x3142, 0x41, 0, -1, false},
		{"4XNN not equal", 0x4142, 0x41, 0, -1, true},
		{"4XNN equal", 0x4142, 0x42, 0, -1, false},
		{"5XY0 equal", 0x5120, 7, 7, -1, true},
		{"5XY0 not equal", 0x5120, 7, 8, -1, false},
		{"9XY0 not equal", 0x9120, 7, 8, -1, true},
		{"9XY0 equal", 0x9120, 7, 7, -1, false},
		{"EX9E key down", 0xE19E, 0xA, 0, 0xA, true},
		{"EX9E key up", 0xE19E, 0xA, 0, 0xB, false},
		{"EXA1 key up", 0xE1A1, 0xA, 0, 0xB, true},
		{"EXA1 key down", 0xE1A1, 0xA, 0, 0xA, false},
		{"EX9E uses low nibble", 0xE19E, 0x1A, 0, 0xA, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := newTestMachine(nil)
			m.v[1], m.v[2] = tt.v1, tt.v2

			if tt.key >= 0 {
				m.SetKey(uint8(tt.key), true)
			}

			exec(m, tt.op)

			want := uint16(ProgramStart + 2)
			if tt.wantSkip {
				want += 2
			}

			if m.pc != want {
				t.Errorf("PC=%#03x, want %#03x", m.pc, want)
			}
		})
	}
}

func TestOpcodeFlow(t *testing.T) {
	t.Parallel()

	t.Run("1NNN jump", func(t *testing.T) {
		t.Parallel()

		m := newTestMachine(nil)
		exec(m, 0x1ABC)

		if m.pc != 0xABC {
			t.Errorf("PC=%#03x, want 0xABC", m.pc)
		}
	})

	t.Run("2NNN call and 00EE return", func(t *testing.T) {
		t.Parallel()

		m := newTestMachine(nil)
		exec(m, 0x2400)

		if m.pc != 0x400 || m.sp != 1 || m.stack[0] != ProgramStart+2 {
			t.Fatalf("after CALL: PC=%#03x SP=%d stack[0]=%#03x", m.pc, m.sp, m.stack[0])
		}

		exec(m, 0x00EE)

		if m.pc != ProgramStart+2 || m.sp != 0 {
			t.Errorf("after RET: PC=%#03x SP=%d", m.pc, m.sp)
		}
	})

	jumps := []struct {
		name   string
		quirks func(*Quirks)
		want   uint16
	}{
		{"BNNN adds V0", nil, 0x345 + 0x10},
		{"BXNN with jump quirk adds VX", func(q *Quirks) { q.Jump = true }, 0x345 + 0x30},
	}
	for _, tt := range jumps {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := newTestMachine(tt.quirks)
			m.v[0], m.v[3] = 0x10, 0x30
			exec(m, 0xB345)

			if m.pc != tt.want {
				t.Errorf("PC=%#03x, want %#03x", m.pc, tt.want)
			}
		})
	}
}

func TestOpcodeIndexAndMemory(t *testing.T) {
	t.Parallel()

	t.Run("ANNN", func(t *testing.T) {
		t.Parallel()

		m := newTestMachine(nil)
		exec(m, 0xA123)

		if m.i != 0x123 {
			t.Errorf("I=%#03x, want 0x123", m.i)
		}
	})

	t.Run("FX1E adds VX, VF unchanged", func(t *testing.T) {
		t.Parallel()

		m := newTestMachine(nil)
		m.i, m.v[1], m.v[flagReg] = 0x0FFF, 2, 0x55
		exec(m, 0xF11E)

		if m.i != 0x1001 || m.v[flagReg] != 0x55 {
			t.Errorf("I=%#04x VF=%#02x, want I=0x1001 VF=0x55", m.i, m.v[flagReg])
		}
	})

	t.Run("FX29 font glyph", func(t *testing.T) {
		t.Parallel()

		m := newTestMachine(nil)
		m.v[1] = 0x1B // low nibble B
		exec(m, 0xF129)

		if want := uint16(FontAddr + 0xB*smallFontHeight); m.i != want {
			t.Errorf("I=%#03x, want %#03x", m.i, want)
		}
	})

	t.Run("FX33 BCD", func(t *testing.T) {
		t.Parallel()

		m := newTestMachine(nil)
		m.i, m.v[1] = 0x300, 254
		exec(m, 0xF133)

		if got := [3]byte{m.mem[0x300], m.mem[0x301], m.mem[0x302]}; got != [3]byte{2, 5, 4} {
			t.Errorf("BCD %v, want [2 5 4]", got)
		}
	})

	memModes := []struct {
		name   string
		quirks func(*Quirks)
		wantI  uint16
	}{
		{"I += X+1", nil, 0x303},
		{"I += X", func(q *Quirks) { q.MemoryIncrementByX = true }, 0x302},
		{"I unchanged", func(q *Quirks) { q.MemoryLeaveIUnchanged = true }, 0x300},
	}
	for _, tt := range memModes {
		t.Run("FX55 "+tt.name, func(t *testing.T) {
			t.Parallel()

			m := newTestMachine(tt.quirks)
			m.i = 0x300
			m.v[0], m.v[1], m.v[2], m.v[3] = 1, 2, 3, 4
			exec(m, 0xF255)

			if got := [4]byte(m.mem[0x300:0x304]); got != [4]byte{1, 2, 3, 0} {
				t.Errorf("memory %v, want [1 2 3 0]", got)
			}

			if m.i != tt.wantI {
				t.Errorf("I=%#03x, want %#03x", m.i, tt.wantI)
			}
		})

		t.Run("FX65 "+tt.name, func(t *testing.T) {
			t.Parallel()

			m := newTestMachine(tt.quirks)
			m.i = 0x300
			copy(m.mem[0x300:], []byte{9, 8, 7, 6})
			exec(m, 0xF265)

			if got := [4]byte(m.v[:4]); got != [4]byte{9, 8, 7, 0} {
				t.Errorf("V0..V3 %v, want [9 8 7 0]", got)
			}

			if m.i != tt.wantI {
				t.Errorf("I=%#03x, want %#03x", m.i, tt.wantI)
			}
		})
	}

	t.Run("addresses wrap at 4 KB", func(t *testing.T) {
		t.Parallel()

		m := newTestMachine(nil)
		m.i, m.v[0], m.v[1] = 0xFFF, 0xAA, 0xBB
		exec(m, 0xF155)

		if m.mem[0xFFF] != 0xAA || m.mem[0x000] != 0xBB {
			t.Errorf("mem[FFF]=%#02x mem[000]=%#02x, want 0xAA 0xBB", m.mem[0xFFF], m.mem[0])
		}
	})
}

func TestOpcodeTimersAndRandom(t *testing.T) {
	t.Parallel()

	m := newTestMachine(nil)
	m.v[1] = 42
	exec(m, 0xF115) // DT = 42
	exec(m, 0xF118) // ST = 42
	exec(m, 0xF207) // V2 = DT

	if m.dt != 42 || m.st != 42 || m.v[2] != 42 {
		t.Errorf("DT=%d ST=%d V2=%d, want 42 42 42", m.dt, m.st, m.v[2])
	}

	for range 50 {
		exec(m, 0xC30F) // V3 = rand & 0x0F

		if m.v[3] > 0x0F {
			t.Fatalf("CXNN result %#02x not masked by 0x0F", m.v[3])
		}
	}
}

func TestOpcodeDraw(t *testing.T) {
	t.Parallel()

	// A sprite of two rows, 0b10000001 each, drawn from I = 0x300.
	setup := func(wrap bool) *Machine {
		m := newTestMachine(func(q *Quirks) { q.Wrap = wrap })
		m.i = 0x300
		m.mem[0x300], m.mem[0x301] = 0x81, 0x81

		return m
	}
	on := func(m *Machine, x, y int) bool { return m.Display().Pixel(x, y) != 0 }

	t.Run("draws and reports no collision", func(t *testing.T) {
		t.Parallel()

		m := setup(false)
		m.v[1], m.v[2] = 10, 5
		exec(m, 0xD122)

		if !on(m, 10, 5) || !on(m, 17, 5) || !on(m, 10, 6) || on(m, 11, 5) || m.v[flagReg] != 0 {
			t.Errorf("unexpected pixels or VF=%d", m.v[flagReg])
		}
	})

	t.Run("XOR erases and reports collision", func(t *testing.T) {
		t.Parallel()

		m := setup(false)
		exec(m, 0xD122)
		exec(m, 0xD122)

		if on(m, 0, 0) || m.v[flagReg] != 1 {
			t.Errorf("pixel still on or VF=%d, want off and 1", m.v[flagReg])
		}
	})

	t.Run("start position wraps", func(t *testing.T) {
		t.Parallel()

		m := setup(false)
		m.v[1], m.v[2] = 64+3, 32+4
		exec(m, 0xD121)

		if !on(m, 3, 4) {
			t.Error("sprite at (67,36) should start at (3,4)")
		}
	})

	edges := []struct {
		name string
		wrap bool
	}{{"clips at edges", false}, {"wraps at edges, wrap quirk", true}}
	for _, tt := range edges {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := setup(tt.wrap)
			m.v[1], m.v[2] = 60, 31 // right column at x=67, second row at y=32
			exec(m, 0xD122)

			if !on(m, 60, 31) {
				t.Fatal("pixel inside the screen not drawn")
			}

			if on(m, 3, 31) != tt.wrap || on(m, 60, 0) != tt.wrap {
				t.Errorf("wrapped pixels on=%v,%v, want %v", on(m, 3, 31), on(m, 60, 0), tt.wrap)
			}
		})
	}

	t.Run("00E0 clears", func(t *testing.T) {
		t.Parallel()

		m := setup(false)
		exec(m, 0xD122)
		exec(m, 0x00E0)

		if on(m, 0, 0) {
			t.Error("display not cleared")
		}
	})
}

func TestUnknownOpcodes(t *testing.T) {
	t.Parallel()

	for _, op := range []uint16{0x0123, 0x5121, 0x9121, 0x8128, 0xE1FF, 0xF1FF} {
		m := newTestMachine(nil)
		exec(m, op)

		var oe *OpcodeError
		if !errors.As(m.State().Err, &oe) || !errors.Is(m.State().Err, ErrUnknownOpcode) {
			t.Errorf("%04X: err=%v, want OpcodeError", op, m.State().Err)

			continue
		}

		if oe.Op != op || oe.PC != ProgramStart || m.pc != ProgramStart {
			t.Errorf("%04X: OpcodeError{PC:%03X Op:%04X} PC=%03X", op, oe.PC, oe.Op, m.pc)
		}
	}
}
