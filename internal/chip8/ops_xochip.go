package chip8

// The XO-CHIP instructions, as defined by Octo:
// https://johnearnest.github.io/Octo/docs/XO-ChipSpecification.html

// XO-CHIP audio.
const (
	patternBytes = 16 // F002 loads 128 one-bit samples
	defaultPitch = 64 // 4000 bits per second
)

// executeXOF runs the FXNN instructions only XO-CHIP has, and reports whether
// (x, nn) was one.
func (m *Machine) executeXOF(x int, nn byte) bool {
	switch {
	case x == 0 && nn == 0x00:
		m.opF000()
	case nn == 0x01:
		m.opFN01(x)
	case x == 0 && nn == 0x02:
		m.opF002()
	case nn == 0x3A:
		m.opFX3A(x)
	default:
		return false
	}

	return true
}

// op00DN (SCU N) scrolls the selected planes up N pixels.
func (m *Machine) op00DN(n int) { m.disp.scroll(0, -n, m.plane) }

// registerRange returns the step between registers walking from X to Y and
// how many registers that is, so 5XY2/5XY3 work in either direction.
func registerRange(x, y int) (int, int) {
	if x <= y {
		return 1, y - x + 1
	}

	return -1, x - y + 1
}

// op5XY2 (SAVE VX-VY) stores VX..VY at I, in that order. I doesn't change.
func (m *Machine) op5XY2(x, y int) {
	step, count := registerRange(x, y)
	for k := range count {
		m.mem[(m.i+uint16(k))&m.memMask] = m.v[x+k*step]
	}
}

// op5XY3 (LOAD VX-VY) loads VX..VY from I, in that order. I doesn't change.
func (m *Machine) op5XY3(x, y int) {
	step, count := registerRange(x, y)
	for k := range count {
		m.v[x+k*step] = m.mem[(m.i+uint16(k))&m.memMask]
	}
}

// opF000 (LD I, NNNN) loads I from the next two bytes, and steps over them.
func (m *Machine) opF000() {
	m.i = uint16(m.mem[m.pc&m.memMask])<<8 | uint16(m.mem[(m.pc+1)&m.memMask])
	m.pc = (m.pc + 2) & m.memMask
}

// opFN01 (PLANE N) selects the planes that drawing, clearing and scrolling
// affect: a mask from 0 (none) to 3 (both). N's higher bits are ignored.
func (m *Machine) opFN01(n int) { m.plane = byte(n) & 3 }

// opF002 (AUDIO) loads the 16-byte audio pattern from I.
func (m *Machine) opF002() {
	for k := range m.pattern {
		m.pattern[k] = m.mem[(m.i+uint16(k))&m.memMask]
	}

	m.patternLoaded = true
}

// opFX3A (PITCH VX) sets the audio pattern's playback pitch.
func (m *Machine) opFX3A(x int) { m.pitch = m.v[x] }
