package chip8

// The CHIP-8 instruction set. Behaviour that differs between platforms is
// selected by m.profile.Quirks. References:
//   - Timendus, "CHIP-8 test suite", https://github.com/Timendus/chip8-test-suite
//   - chip-8-database quirks.json, https://github.com/chip-8/chip-8-database
//   - Matthew Mikolay, "Mastering CHIP-8", https://github.com/mattmikolay/chip-8/wiki/Mastering-CHIP%E2%80%908

// Sprite sizes.
const (
	spriteWidth   = 8  // DXYN with N > 0
	bigSpriteSize = 16 // DXY0 on SUPER-CHIP: width and height
	maxSpriteRows = bigSpriteSize
	maxPlaneBit   = 2 // the highest display plane, XO-CHIP's second
)

// b2u converts a flag to 0 or 1 for VF.
func b2u(b bool) byte {
	if b {
		return 1
	}

	return 0
}

// op00E0 (CLS) clears the display.
func (m *Machine) op00E0() { m.disp.clear(m.plane) }

// op00EE (RET) returns from a subroutine. An empty stack halts the machine.
func (m *Machine) op00EE() {
	if m.sp == 0 {
		m.halt(ErrStackUnderflow)

		return
	}

	m.sp--
	m.pc = m.stack[m.sp]
}

// op1NNN (JP NNN) jumps to NNN.
func (m *Machine) op1NNN(nnn uint16) { m.pc = nnn & m.memMask }

// op2NNN (CALL NNN) calls the subroutine at NNN. A full stack halts the
// machine.
func (m *Machine) op2NNN(nnn uint16) {
	if m.sp == stackDepth {
		m.halt(ErrStackOverflow)

		return
	}

	m.stack[m.sp] = m.pc
	m.sp++
	m.pc = nnn & m.memMask
}

// skip skips the next instruction: 2 bytes, or 4 for XO-CHIP's F000 NNNN.
func (m *Machine) skip() {
	if m.ops&opsXO != 0 && m.mem[m.pc&m.memMask] == 0xF0 && m.mem[(m.pc+1)&m.memMask] == 0x00 {
		m.pc = (m.pc + 2) & m.memMask
	}

	m.pc = (m.pc + 2) & m.memMask
}

// op3XNN (SE VX, NN) skips the next instruction if VX == NN.
func (m *Machine) op3XNN(x int, nn byte) {
	if m.v[x] == nn {
		m.skip()
	}
}

// op4XNN (SNE VX, NN) skips the next instruction if VX != NN.
func (m *Machine) op4XNN(x int, nn byte) {
	if m.v[x] != nn {
		m.skip()
	}
}

// op5XY0 (SE VX, VY) skips the next instruction if VX == VY.
func (m *Machine) op5XY0(x, y int) {
	if m.v[x] == m.v[y] {
		m.skip()
	}
}

// op6XNN (LD VX, NN) sets VX to NN.
func (m *Machine) op6XNN(x int, nn byte) { m.v[x] = nn }

// op7XNN (ADD VX, NN) adds NN to VX. VF is not changed.
func (m *Machine) op7XNN(x int, nn byte) { m.v[x] += nn }

// op8XY0 (LD VX, VY) sets VX to VY.
func (m *Machine) op8XY0(x, y int) { m.v[x] = m.v[y] }

// op8XY1 (OR VX, VY). On the COSMAC VIP (logic quirk) VF is reset to 0.
func (m *Machine) op8XY1(x, y int) {
	m.v[x] |= m.v[y]
	m.logicQuirk()
}

// op8XY2 (AND VX, VY). On the COSMAC VIP (logic quirk) VF is reset to 0.
func (m *Machine) op8XY2(x, y int) {
	m.v[x] &= m.v[y]
	m.logicQuirk()
}

// op8XY3 (XOR VX, VY). On the COSMAC VIP (logic quirk) VF is reset to 0.
func (m *Machine) op8XY3(x, y int) {
	m.v[x] ^= m.v[y]
	m.logicQuirk()
}

// logicQuirk resets VF after 8XY1/2/3 on platforms with the logic quirk.
func (m *Machine) logicQuirk() {
	if m.profile.Quirks.Logic {
		m.v[flagReg] = 0
	}
}

// op8XY4 (ADD VX, VY) adds VY to VX; VF is the carry. VF is written last, so
// the flag wins when X is F.
func (m *Machine) op8XY4(x, y int) {
	sum := uint16(m.v[x]) + uint16(m.v[y])
	m.v[x] = byte(sum)
	m.v[flagReg] = b2u(sum > 0xFF)
}

// op8XY5 (SUB VX, VY) sets VX to VX - VY; VF is 1 when there is no borrow.
func (m *Machine) op8XY5(x, y int) {
	vx, vy := m.v[x], m.v[y]
	m.v[x] = vx - vy
	m.v[flagReg] = b2u(vx >= vy)
}

// op8XY7 (SUBN VX, VY) sets VX to VY - VX; VF is 1 when there is no borrow.
func (m *Machine) op8XY7(x, y int) {
	vx, vy := m.v[x], m.v[y]
	m.v[x] = vy - vx
	m.v[flagReg] = b2u(vy >= vx)
}

// shiftSource is the register 8XY6/8XYE shift: VY on the COSMAC VIP, VX
// with the shift quirk (CHIP-48 and SUPER-CHIP).
func (m *Machine) shiftSource(x, y int) byte {
	if m.profile.Quirks.Shift {
		return m.v[x]
	}

	return m.v[y]
}

// op8XY6 (SHR) sets VX to the source shifted right by one; VF is the bit
// shifted out.
func (m *Machine) op8XY6(x, y int) {
	src := m.shiftSource(x, y)
	m.v[x] = src >> 1
	m.v[flagReg] = src & 1
}

// op8XYE (SHL) sets VX to the source shifted left by one; VF is the bit
// shifted out.
func (m *Machine) op8XYE(x, y int) {
	src := m.shiftSource(x, y)
	m.v[x] = src << 1
	m.v[flagReg] = src >> 7
}

// op9XY0 (SNE VX, VY) skips the next instruction if VX != VY.
func (m *Machine) op9XY0(x, y int) {
	if m.v[x] != m.v[y] {
		m.skip()
	}
}

// opANNN (LD I, NNN) sets I to NNN.
func (m *Machine) opANNN(nnn uint16) { m.i = nnn }

// opBNNN (JP V0, NNN) jumps to NNN + V0. With the jump quirk (CHIP-48,
// SUPER-CHIP) it is BXNN: jump to XNN + VX.
func (m *Machine) opBNNN(x int, nnn uint16) {
	reg := 0
	if m.profile.Quirks.Jump {
		reg = x
	}

	m.pc = (nnn + uint16(m.v[reg])) & m.memMask
}

// opCXNN (RND VX, NN) sets VX to a random byte AND NN.
func (m *Machine) opCXNN(x int, nn byte) { m.v[x] = m.rng.byte() & nn }

// opDXYN (DRW VX, VY, N) draws an 8×N sprite from memory at I at (VX, VY).
// On SUPER-CHIP, N = 0 is the large sprite (see spriteShape). VF is 1 if any
// pixel was turned off. With the vblank quirk (COSMAC VIP) drawing waits for
// the next frame, so the frame ends here; legacy SUPER-CHIP does that in
// lo-res only.
func (m *Machine) opDXYN(x, y, n int) {
	width, height := m.spriteShape(n)
	legacy := m.legacy()

	var (
		rows         [maxSpriteRows]uint16
		hit, clipped int
		offset       uint16
	)

	// Each selected plane has its own sprite data, one after the other. Only
	// XO-CHIP has more than one plane, or none.
	for plane := byte(1); plane <= maxPlaneBit; plane <<= 1 {
		if m.plane&plane == 0 {
			continue
		}

		offset += m.readSprite(rows[:height], width, offset)
		h, c := m.disp.drawSprite(int(m.v[x]), int(m.v[y]), rows[:height], width, plane, m.profile.Quirks.Wrap)
		hit += h
		clipped += c
	}

	if legacy && m.hires {
		m.v[flagReg] = byte(hit + clipped)
	} else {
		m.v[flagReg] = b2u(hit > 0)
	}

	if m.profile.Quirks.VBlank || (legacy && !m.hires) {
		m.stopFrame = true
	}
}

// spriteShape returns the size DXYN draws for its N. N = 0 draws nothing on
// CHIP-8, 16×16 on SUPER-CHIP, and 8×16 in lo-res on legacy SUPER-CHIP.
func (m *Machine) spriteShape(n int) (int, int) {
	switch {
	case n != 0 || m.ops&opsSuper == 0:
		return spriteWidth, n
	case m.legacy() && !m.hires:
		return spriteWidth, bigSpriteSize
	default:
		return bigSpriteSize, bigSpriteSize
	}
}

// readSprite reads sprite rows from memory at I + offset into rows, aligned
// to the top bit. It returns the bytes read.
func (m *Machine) readSprite(rows []uint16, width int, offset uint16) uint16 {
	addr := m.i + offset

	for r := range rows {
		rows[r] = uint16(m.mem[addr&m.memMask]) << 8
		addr++

		if width == bigSpriteSize {
			rows[r] |= uint16(m.mem[addr&m.memMask])
			addr++
		}
	}

	return addr - m.i - offset
}

// opEX9E (SKP VX) skips the next instruction if key VX is down.
func (m *Machine) opEX9E(x int) {
	m.keysRead |= 1 << (m.v[x] & 0xF)

	if m.keys[m.v[x]&0xF] {
		m.skip()
	}
}

// opEXA1 (SKNP VX) skips the next instruction if key VX is up.
func (m *Machine) opEXA1(x int) {
	m.keysRead |= 1 << (m.v[x] & 0xF)

	if !m.keys[m.v[x]&0xF] {
		m.skip()
	}
}

// opFX07 (LD VX, DT) sets VX to the delay timer.
func (m *Machine) opFX07(x int) { m.v[x] = m.dt }

// opFX0A (LD VX, K) waits for a key to be pressed and released, then stores
// it in VX. This is the COSMAC VIP's behaviour: a key already down when FX0A
// starts doesn't count, and the timers keep running while it waits. See
// https://laurencescotford.net/chip-8-on-the-cosmac-vip-keyboard-input/
func (m *Machine) opFX0A(x int) {
	m.anyKey = true

	if !m.waiting {
		m.waiting = true
		m.waitKey = noKey
		m.lastKeys = m.keys
	}

	if m.waitKey == noKey {
		for k := range NumKeys {
			if m.keys[k] && !m.lastKeys[k] {
				m.waitKey = k

				break
			}
		}

		m.lastKeys = m.keys
	}

	if m.waitKey != noKey && !m.keys[m.waitKey] {
		m.v[x] = byte(m.waitKey)
		m.waiting = false
		m.waitKey = noKey

		return
	}

	// Keep waiting: run this instruction again next frame.
	m.pc = (m.pc - 2) & m.memMask
	m.stopFrame = true
}

// opFX15 (LD DT, VX) sets the delay timer to VX.
func (m *Machine) opFX15(x int) { m.dt = m.v[x] }

// opFX18 (LD ST, VX) sets the sound timer to VX.
func (m *Machine) opFX18(x int) { m.st = m.v[x] }

// opFX1E (ADD I, VX) adds VX to I. VF is not changed.
func (m *Machine) opFX1E(x int) { m.i += uint16(m.v[x]) }

// opFX29 (LD F, VX) points I at the small-font glyph for the low nibble of VX.
func (m *Machine) opFX29(x int) { m.i = FontAddr + uint16(m.v[x]&0xF)*smallFontHeight }

// opFX33 (LD B, VX) stores VX as three decimal digits at I, I+1 and I+2.
func (m *Machine) opFX33(x int) {
	const hundred, ten = 100, 10

	v := m.v[x]
	m.mem[m.i&m.memMask] = v / hundred
	m.mem[(m.i+1)&m.memMask] = v / ten % ten
	m.mem[(m.i+2)&m.memMask] = v % ten
}

// opFX55 (LD [I], VX) stores V0..VX at I.
func (m *Machine) opFX55(x int) {
	for r := range x + 1 {
		m.mem[(m.i+uint16(r))&m.memMask] = m.v[r]
	}

	m.memoryQuirk(x)
}

// opFX65 (LD VX, [I]) loads V0..VX from I.
func (m *Machine) opFX65(x int) {
	for r := range x + 1 {
		m.v[r] = m.mem[(m.i+uint16(r))&m.memMask]
	}

	m.memoryQuirk(x)
}

// memoryQuirk updates I after FX55/FX65: I += X+1 on the COSMAC VIP, I += X
// on CHIP-48 and SUPER-CHIP 1.0, unchanged on SUPER-CHIP 1.1.
func (m *Machine) memoryQuirk(x int) {
	switch {
	case m.profile.Quirks.MemoryLeaveIUnchanged:
	case m.profile.Quirks.MemoryIncrementByX:
		m.i += uint16(x)
	default:
		m.i += uint16(x) + 1
	}
}
