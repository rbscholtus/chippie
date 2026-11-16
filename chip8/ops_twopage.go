package chip8

// The COSMAC VIP's 2 Page Display for CHIP-8 (Andy Modla and Jef Winsor,
// VIPER 1/3, September 1978), a 64×64 mode often called "Hires CHIP-8"
// (DESIGN §6.3). A ROM for it starts at 0x200 with 1260, a jump over a machine-code patch that
// switches the VIP interpreter to 64×64; the CHIP-8 program starts at 0x2C0.
// Chippie doesn't run the patch: it applies its effect. See
// https://github.com/kripod/chip8-roms/blob/master/hires/%21hires_information.txt.

const (
	// twoPageStartOp is the jump over the patch, as the first instruction.
	twoPageStartOp = 0x1260
	// twoPageEntry is where the CHIP-8 program starts, after the patch.
	twoPageEntry = 0x2C0
	// twoPageClearOp calls the patch's routine that clears the 64×64 screen.
	twoPageClearOp = 0x0230
)

// startsTwoPage reports whether the instruction at pc is a 2 Page Display
// ROM's start.
func (m *Machine) startsTwoPage(pc, op uint16) bool {
	return m.ops&opsTwoPage != 0 && pc == ProgramStart && op == twoPageStartOp
}

// startTwoPage does what the patch does: the screen becomes 64×64 (and
// clear), and the program continues at 0x2C0.
func (m *Machine) startTwoPage() {
	m.disp.setResolution(twoPageSize, twoPageSize)
	m.pc = twoPageEntry
}

// executeTwoPage0 runs 0230, which clears the whole 64×64 screen, and reports
// whether op was it.
func (m *Machine) executeTwoPage0(op uint16) bool {
	if m.ops&opsTwoPage == 0 || op != twoPageClearOp {
		return false
	}

	m.op00E0()

	return true
}
