package chip8

// The SUPER-CHIP instructions. Platform opcode sets decide which exist; see
// https://chip-8.github.io/extensions/#super-chip-10 and
// third_party/chip8-test-suite/legacy-superchip.md.

// Scroll distances, in screen pixels.
const (
	scrollSideways = 4 // 00FB, 00FC

	// legacyMaxFlag is the highest register FX75/FX85 can reach on SUPER-CHIP.
	legacyMaxFlag = 7
)

// legacy reports whether the machine runs legacy SUPER-CHIP behaviour.
func (m *Machine) legacy() bool {
	return m.profile.Quirks.LegacySuperChip && m.profile.Platform.HasLegacyMode()
}

// executeSuper0 runs the 00NN instructions from the platform's optional
// groups, and reports whether op was one.
func (m *Machine) executeSuper0(op uint16) bool {
	switch {
	case m.ops&opsXO != 0 && op&0xFFF0 == 0x00D0:
		m.op00DN(int(op & 0xF))
	case m.ops&opsScroll != 0 && op&0xFFF0 == 0x00C0:
		m.op00CN(int(op & 0xF))
	case m.ops&opsScroll != 0 && op == 0x00FB:
		m.op00FB()
	case m.ops&opsScroll != 0 && op == 0x00FC:
		m.op00FC()
	case m.ops&opsSuper != 0 && op == 0x00FD:
		m.halt(ErrProgramExit)
	case m.ops&opsSuper != 0 && op == 0x00FE:
		m.setHires(false)
	case m.ops&opsSuper != 0 && op == 0x00FF:
		m.setHires(true)
	default:
		return false
	}

	return true
}

// executeSuperF runs the FXNN instructions from the platform's optional
// groups, and reports whether nn was one.
func (m *Machine) executeSuperF(x int, nn byte) bool {
	if m.ops&opsXO != 0 && m.executeXOF(x, nn) {
		return true
	}

	if m.ops&opsSuper == 0 {
		return false
	}

	switch nn {
	case 0x30:
		m.opFX30(x)
	case 0x75:
		m.opFX75(x)
	case 0x85:
		m.opFX85(x)
	default:
		return false
	}

	return true
}

// op00CN (SCD N) scrolls the screen down N screen pixels.
func (m *Machine) op00CN(n int) { m.disp.scroll(0, n, m.plane) }

// op00FB (SCR) scrolls the screen right 4 screen pixels.
func (m *Machine) op00FB() { m.disp.scroll(scrollSideways, 0, m.plane) }

// op00FC (SCL) scrolls the screen left 4 screen pixels.
func (m *Machine) op00FC() { m.disp.scroll(-scrollSideways, 0, m.plane) }

// setHires switches between lo-res (00FE) and hi-res (00FF). Modern
// interpreters clear the screen. Legacy SUPER-CHIP keeps it: only the size
// of a pixel changes.
func (m *Machine) setHires(on bool) {
	m.hires = on

	switch {
	case m.legacy() && on:
		m.disp.scale = 1
	case m.legacy():
		m.disp.scale = legacyLoresScale
	case on:
		m.disp.setResolution(hiresWidth, hiresHeight)
	default:
		m.disp.setResolution(loresWidth, loresHeight)
	}
}

// opFX30 (LD HF, VX) points I at the large-font glyph for the low nibble of
// VX.
func (m *Machine) opFX30(x int) { m.i = BigFontAddr + uint16(m.v[x]&0xF)*bigFontHeight }

// opFX75 (LD R, VX) stores V0..VX in the flag registers.
func (m *Machine) opFX75(x int) { copy(m.rpl[:m.flagCount(x)], m.v[:]) }

// opFX85 (LD VX, R) loads V0..VX from the flag registers.
func (m *Machine) opFX85(x int) { copy(m.v[:m.flagCount(x)], m.rpl[:]) }

// flagCount is how many flag registers FX75/FX85 with this X use. SUPER-CHIP
// has 8; XO-CHIP has 16.
func (m *Machine) flagCount(x int) int {
	if m.ops&opsXO != 0 {
		return x + 1
	}

	return min(x, legacyMaxFlag) + 1
}
