package chip8

// The COSMAC VIP's two-page 64×64 mode ("Hires CHIP-8", DESIGN §6.3). A
// hires ROM starts at 0x200 with 1260, a jump over a machine-code patch that
// switches the VIP interpreter to 64×64; the CHIP-8 program starts at 0x2C0.
// Chippie doesn't run the patch: it applies its effect. See
// https://github.com/kripod/chip8-roms/blob/master/hires/%21hires_information.txt.

const (
	// hiresStartOp is the jump over the patch, as the first instruction.
	hiresStartOp = 0x1260
	// hiresEntry is where the CHIP-8 program starts, after the patch.
	hiresEntry = 0x2C0
	// hiresClearOp calls the patch's routine that clears the 64×64 screen.
	hiresClearOp = 0x0230
)

// startsHires reports whether the instruction at pc is a hires ROM's start.
func (m *Machine) startsHires(pc, op uint16) bool {
	return m.ops&opsHires != 0 && pc == ProgramStart && op == hiresStartOp
}

// startHires does what the patch does: the screen becomes 64×64 (and
// clear), and the program continues at 0x2C0.
func (m *Machine) startHires() {
	m.disp.setResolution(vipHiresSize, vipHiresSize)
	m.pc = hiresEntry
}

// executeHires0 runs 0230, which clears the whole 64×64 screen, and reports
// whether op was it.
func (m *Machine) executeHires0(op uint16) bool {
	if m.ops&opsHires == 0 || op != hiresClearOp {
		return false
	}

	m.op00E0()

	return true
}
