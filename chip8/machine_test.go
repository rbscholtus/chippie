package chip8

import (
	"errors"
	"testing"
)

// program returns ROM bytes for a list of opcodes.
func program(ops ...uint16) []byte {
	rom := make([]byte, 0, 2*len(ops))
	for _, op := range ops {
		rom = append(rom, byte(op>>8), byte(op))
	}

	return rom
}

func mustLoad(t *testing.T, m *Machine, rom []byte) {
	t.Helper()

	if err := m.LoadROM(rom); err != nil {
		t.Fatal(err)
	}
}

func TestPlatformTable(t *testing.T) {
	t.Parallel()

	for p := range numPlatforms {
		info := p.info()
		if info.id == "" || info.name == "" || info.ticks <= 0 || info.maxWidth <= 0 || info.planes <= 0 {
			t.Errorf("%d: incomplete entry %+v", p, *info)
		}

		if size := info.memSize; size&(size-1) != 0 {
			t.Errorf("%s: memory size %d is not a power of two", p, size)
		}

		if got, ok := ParsePlatform(p.String()); !ok || got != p {
			t.Errorf("ParsePlatform(%q) = %v, %v", p.String(), got, ok)
		}
	}

	if got := Platforms(); len(got) != int(numPlatforms) || got[0] != OriginalChip8 || got[len(got)-1] != XOChip {
		t.Errorf("Platforms() = %v", got)
	}

	if _, ok := ParsePlatform("nope"); ok {
		t.Error("ParsePlatform accepted an unknown id")
	}

	if Platform(200).Implemented() || Platform(200).String() != "unknown" {
		t.Error("out-of-range platform should be unknown and unimplemented")
	}

	for _, p := range []Platform{ModernChip8, OriginalChip8} {
		if !p.Implemented() {
			t.Errorf("%s should be implemented", p)
		}
	}
}

func TestMemorySizedPerPlatform(t *testing.T) {
	t.Parallel()

	m := New(DefaultProfile(ModernChip8))
	if len(m.mem) != 4096 {
		t.Fatalf("modernChip8 memory %d bytes, want 4096", len(m.mem))
	}

	before := &m.mem[0]
	m.mem[0x300] = 1
	m.Reset(DefaultProfile(OriginalChip8))

	if len(m.mem) != 4096 || &m.mem[0] != before {
		t.Error("same-size Reset should reuse the memory buffer")
	}

	if m.mem[0x300] != 0 {
		t.Error("Reset should clear memory")
	}

	if m.mem[FontAddr] != smallFont[0] {
		t.Error("Reset should load the font")
	}

	m.Reset(DefaultProfile(XOChip))

	if len(m.mem) != 64<<10 {
		t.Errorf("xochip memory %d bytes, want 65536", len(m.mem))
	}
}

func TestLoadROM(t *testing.T) {
	t.Parallel()

	m := New(DefaultProfile(ModernChip8))
	mustLoad(t, m, program(0x1234))

	if m.Peek(ProgramStart) != 0x12 || m.Peek(ProgramStart+1) != 0x34 || m.pc != ProgramStart {
		t.Error("ROM not loaded at 0x200")
	}

	if err := m.LoadROM(make([]byte, 4096-ProgramStart+1)); !errors.Is(err, ErrROMTooLarge) {
		t.Errorf("oversized ROM: err=%v, want ErrROMTooLarge", err)
	}

	if err := m.LoadROM(make([]byte, 4096-ProgramStart)); err != nil {
		t.Errorf("ROM filling memory exactly: %v", err)
	}

	x := New(DefaultProfile(SuperChip))
	if err := x.LoadROM(program(0x1200)); !errors.Is(err, ErrUnsupportedPlatform) {
		t.Errorf("superchip: err=%v, want ErrUnsupportedPlatform", err)
	}
}

func TestTimersTickOncePerFrame(t *testing.T) {
	t.Parallel()

	m := New(DefaultProfile(ModernChip8))
	// V0 = 3; DT = ST = V0; loop forever.
	mustLoad(t, m, program(0x6003, 0xF015, 0xF018, 0x1206))

	m.RunFrame() // timers tick (still 0), then they are set to 3

	if s := m.State(); s.DelayTimer != 3 || s.SoundTimer != 3 || !m.Sound().Buzzer {
		t.Fatalf("after frame 1: DT=%d ST=%d", s.DelayTimer, s.SoundTimer)
	}

	for want := byte(2); ; want-- {
		m.RunFrame()

		if s := m.State(); s.DelayTimer != want || s.SoundTimer != want {
			t.Fatalf("DT=%d ST=%d, want %d", s.DelayTimer, s.SoundTimer, want)
		}

		if want == 0 {
			break
		}
	}

	if m.Sound().Buzzer {
		t.Error("buzzer on with sound timer 0")
	}

	m.RunFrame()

	if s := m.State(); s.DelayTimer != 0 {
		t.Errorf("DT went below zero: %d", s.DelayTimer)
	}
}

func TestTicksPerFrame(t *testing.T) {
	t.Parallel()

	prof := DefaultProfile(ModernChip8)
	prof.TicksPerFrame = 5
	m := New(prof)
	mustLoad(t, m, program(0x7101, 0x1200)) // V1++; jump back

	m.RunFrame()

	if m.v[1] != 3 { // instructions 1, 3, 5 are the add
		t.Errorf("V1=%d after 5 ticks, want 3", m.v[1])
	}

	prof.TicksPerFrame = 0 // falls back to the platform default (12)
	m.Reset(prof)
	mustLoad(t, m, program(0x7101, 0x1200))
	m.RunFrame()

	if m.v[1] != 6 {
		t.Errorf("V1=%d with default ticks, want 6", m.v[1])
	}
}

func TestSetSpeedAndQuirksWhileRunning(t *testing.T) {
	t.Parallel()

	m := New(DefaultProfile(ModernChip8))
	mustLoad(t, m, program(0x7101, 0x1200)) // V1++; jump back

	m.SetTicksPerFrame(2)
	m.RunFrame()

	if m.v[1] != 1 {
		t.Errorf("V1=%d after 2 ticks, want 1", m.v[1])
	}

	m.SetQuirks(Quirks{Logic: true})

	if p := m.Profile(); !p.Quirks.Logic || p.TicksPerFrame != 2 || p.Platform != ModernChip8 {
		t.Errorf("profile %+v", p)
	}
}

func TestVBlankEndsFrameAfterDraw(t *testing.T) {
	t.Parallel()

	// Draw, then V1++ forever.
	rom := program(0xD001, 0x7101, 0x1202)

	for _, tt := range []struct {
		p      Platform
		wantV1 byte
	}{
		{OriginalChip8, 0}, // the frame ends after DXYN
		{ModernChip8, 6},   // 11 more instructions, every other one an add
	} {
		m := New(DefaultProfile(tt.p))
		mustLoad(t, m, rom)
		m.RunFrame()

		if m.v[1] != tt.wantV1 {
			t.Errorf("%s: V1=%d after one frame, want %d", tt.p, m.v[1], tt.wantV1)
		}
	}
}

func TestFX0AWaitsForPressAndRelease(t *testing.T) {
	t.Parallel()

	m := New(DefaultProfile(ModernChip8))
	// DT = 10; V1 = key; loop.
	mustLoad(t, m, program(0x600A, 0xF015, 0xF10A, 0x1206))

	m.SetKey(0x5, true) // held before FX0A starts: must not count
	m.RunFrame()
	m.RunFrame()

	if !m.State().WaitingKey {
		t.Fatal("FX0A returned for a key that was already down")
	}

	m.SetKey(0x5, false)
	m.RunFrame()
	m.SetKey(0x7, true) // a new press
	m.RunFrame()

	if !m.State().WaitingKey {
		t.Fatal("FX0A returned before the key was released")
	}

	m.SetKey(0x7, false)
	m.RunFrame()

	if s := m.State(); s.WaitingKey || s.V[1] != 0x7 {
		t.Fatalf("after release: waiting=%v V1=%#x, want false 0x7", s.WaitingKey, s.V[1])
	}

	if dt := m.State().DelayTimer; dt != 10-4 { // four frames since DT was set
		t.Errorf("DT=%d: timers must keep running while FX0A waits", dt)
	}
}

func TestStackErrors(t *testing.T) {
	t.Parallel()

	m := New(DefaultProfile(ModernChip8))
	mustLoad(t, m, program(0x00EE))
	m.RunFrame()

	if !errors.Is(m.State().Err, ErrStackUnderflow) || !m.Halted() {
		t.Errorf("RET on empty stack: err=%v", m.State().Err)
	}

	m = New(DefaultProfile(ModernChip8))
	mustLoad(t, m, program(0x2200)) // calls itself forever

	for range 3 {
		m.RunFrame()
	}

	if !errors.Is(m.State().Err, ErrStackOverflow) {
		t.Errorf("deep recursion: err=%v", m.State().Err)
	}

	pc := m.pc
	m.RunFrame()

	if m.pc != pc {
		t.Error("a halted machine kept running")
	}
}

func TestDeterministic(t *testing.T) {
	t.Parallel()

	// V0 = random; draw font glyph V0 at (V1, V2); V1 += 5; loop.
	rom := program(0xC00F, 0xF029, 0xD125, 0x7105, 0x1200)
	run := func(seed uint64) (CPUState, string) {
		prof := DefaultProfile(ModernChip8)
		prof.Seed = seed
		m := New(prof)
		mustLoad(t, m, rom)

		for range 30 {
			m.RunFrame()
		}

		v := m.Display()
		pix := make([]byte, 0, v.Width()*v.Height())

		for y := range v.Height() {
			for x := range v.Width() {
				pix = append(pix, v.Pixel(x, y))
			}
		}

		return m.State(), string(pix)
	}

	s1, d1 := run(1)
	s2, d2 := run(1)

	if s1 != s2 || d1 != d2 {
		t.Error("same seed gave different results")
	}

	if _, d3 := run(2); d3 == d1 {
		t.Error("different seeds gave identical displays")
	}
}

func TestDisplayView(t *testing.T) {
	t.Parallel()

	v := New(DefaultProfile(ModernChip8)).Display()
	if v.Width() != 64 || v.Height() != 32 {
		t.Errorf("display %d×%d, want 64×32", v.Width(), v.Height())
	}

	if v.Pixel(-1, 0) != 0 || v.Pixel(64, 0) != 0 || v.Pixel(0, 32) != 0 {
		t.Error("out-of-range pixels should read as off")
	}
}

func TestKeysRead(t *testing.T) {
	t.Parallel()

	m := New(DefaultProfile(ModernChip8))
	// V0 = 5; skip if key 5; V0 = 0x17 (low nibble 7); skip if not key 7; wait for a key.
	mustLoad(t, m, program(0x6005, 0xE09E, 0x6017, 0xE0A1, 0x0000, 0xF10A))

	m.RunFrame()

	mask, anyKey := m.KeysRead()
	if mask != 1<<5|1<<7 || !anyKey {
		t.Errorf("KeysRead = %016b, %v; want keys 5 and 7, and any key", mask, anyKey)
	}

	m.Reset(m.Profile())

	if mask, anyKey := m.KeysRead(); mask != 0 || anyKey {
		t.Error("Reset should forget the keys")
	}
}

func TestSetKeyIgnoresInvalid(t *testing.T) {
	t.Parallel()

	m := New(DefaultProfile(ModernChip8))
	m.SetKey(16, true) // must not panic or set anything

	if m.keys != [NumKeys]bool{} {
		t.Error("invalid key changed the keypad")
	}
}

func FuzzRunFrame(f *testing.F) {
	f.Add(program(0x00E0, 0x1200))
	f.Add(program(0x2200))
	f.Add(program(0xF10A))
	f.Add([]byte{0xFF, 0xFF, 0xD0, 0x0F, 0xF0, 0x65})

	f.Fuzz(func(_ *testing.T, rom []byte) {
		for _, p := range []Platform{ModernChip8, OriginalChip8} {
			m := New(DefaultProfile(p))
			if err := m.LoadROM(rom); err != nil {
				return
			}

			for k := range uint8(NumKeys) {
				m.SetKey(k, k%3 == 0)
			}

			for range 20 {
				m.RunFrame()
			}
		}
	})
}
