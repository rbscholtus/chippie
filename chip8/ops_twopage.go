package chip8

import "bytes"

// The COSMAC VIP's 2 Page Display for CHIP-8 (Andy Modla and Jef Winsor,
// VIPER volume 1 issue 3, September 1978, pp. 10–12), a 64×64 mode often
// called "Hires CHIP-8" (DESIGN §6.3).
//
// The article patches the VIP interpreter (at 0x003–0x1FB) and puts a
// machine-code patch at 0x200–0x249, which every program must include:
// 1260 at 0x200 jumps over it, and "your CHIP-8 programs start at 0260".
// 0230 calls the patch's routine that clears both display pages.
//
// The ROMs in circulation (kripod/chip8-roms' hires/ folder, by Revival
// Studios and others) also patch the interpreter themselves, so they run on
// an unmodified VIP: their CHIP-8 code at 0x260 copies the article's
// interpreter changes (data at 0x24A–0x25F) into 0x003–0x1FB, calls more
// machine code (02AC), clears the screen (0230 at 0x2BE) and falls through
// to the program at 0x2C0.
//
// Chippie runs none of the 1802 code. On twoPageChip8, a ROM whose first
// instruction is 1260 and that carries the article's patch switches to
// 64×64 and continues where its CHIP-8 program starts: at 0x2C0 after the
// self-patching code, else at 0x260. 0230 then clears the screen. A ROM
// without the patch runs as on originalChip8, at 64×32.

const (
	// twoPageStartOp is the jump over the patch, the first instruction.
	twoPageStartOp = 0x1260
	// twoPageProgram is where the article's programs start.
	twoPageProgram = 0x260
	// twoPageSelfPatchedProgram is where a self-patching ROM's program
	// starts, after the code that patches the interpreter.
	twoPageSelfPatchedProgram = 0x2C0
	// twoPageClearOp calls the patch's routine that clears both pages.
	twoPageClearOp = 0x0230
)

// twoPagePatch is the article's machine code at 0x202–0x249 (VIPER 1/3,
// p. 12): the new interrupt routine, the two-page clear at 0x230 and the
// display-page setup at 0x245. No ordinary CHIP-8 program contains it.
var twoPagePatch = []byte{ //nolint:gochecknoglobals // read-only
	0x01, 0x7A, 0x42, 0x70, 0x22, 0x78, 0x22, 0x52, 0xC4, 0x19, 0xF8, 0x00,
	0xA0, 0x9B, 0xFA, 0x0E, 0xB0, 0xE2, 0xE2, 0x80, 0xE2, 0x20, 0xA0, 0xE2,
	0x3C, 0x15, 0x80, 0xE2, 0x20, 0xA0, 0x34, 0x1C, 0x98, 0x32, 0x29, 0xAB,
	0x2B, 0x8B, 0xB8, 0x88, 0x32, 0x03, 0x7B, 0x28, 0x30, 0x04, 0xF8, 0x02,
	0xAE, 0x9B, 0xBF, 0xF8, 0x00, 0xAF, 0xF8, 0x00, 0x5F, 0x1F, 0x8F, 0x3A,
	0x38, 0x2E, 0x8E, 0x3A, 0x38, 0xD4, 0x01, 0x9B, 0xFF, 0x01, 0xBB, 0xD4,
}

// twoPageSelfPatch is what self-patching ROMs have at 0x24A–0x2B2: the
// article's interpreter changes as data (for a 4 KB VIP), then the CHIP-8
// code that copies them into the interpreter, and the machine code that
// 02AC calls. Then come 11 bytes of padding (0x2B3–0x2BD, "REVIVAL2011"
// or zeros) and 0230.
var twoPageSelfPatch = []byte{ //nolint:gochecknoglobals // read-only
	0x9B, 0x7C, 0x00, 0xBC, 0x30, 0x86, 0x9C, 0x7C, 0x00, 0xBC, 0xFB, 0x10,
	0x30, 0xFC, 0x3A, 0xB3, 0x30, 0xD9, 0x02, 0x45, 0x02, 0x30, 0x60, 0x12,
	0x61, 0xBE, 0xA2, 0x00, 0xF1, 0x55, 0xA2, 0x4A, 0xFD, 0x65, 0xA0, 0xE0,
	0xFD, 0x55, 0xA2, 0x58, 0xF3, 0x65, 0xA0, 0xFC, 0xF3, 0x55, 0xA0, 0x7E,
	0x60, 0x3F, 0xF0, 0x55, 0xA0, 0x84, 0x60, 0x30, 0x61, 0xE0, 0xF1, 0x55,
	0xA0, 0xD7, 0x61, 0xE6, 0xF1, 0x55, 0xA2, 0x5A, 0xF3, 0x65, 0xA1, 0xFA,
	0xF3, 0x55, 0x60, 0xFA, 0xA0, 0x18, 0xF0, 0x55, 0x60, 0x06, 0xA0, 0x0D,
	0xF0, 0x55, 0x60, 0x02, 0xA0, 0x0A, 0xF0, 0x55, 0xA0, 0x03, 0xF0, 0x55,
	0x02, 0xAC, 0xF8, 0x00, 0xB0, 0xA0, 0x9B, 0xB1, 0xD0,
}

// Where the patches are in memory.
const (
	twoPagePatchAddr     = ProgramStart + 2
	twoPageSelfPatchAddr = twoPagePatchAddr + 72 // len(twoPagePatch)
	twoPageSelfClearAddr = 0x2BE                 // 0230, before the program
)

// startsTwoPage reports whether the instruction at pc starts a 2 Page
// Display program: 1260 at 0x200, followed by the article's patch.
func (m *Machine) startsTwoPage(pc, op uint16) bool {
	return m.ops&opsTwoPage != 0 && pc == ProgramStart && op == twoPageStartOp &&
		m.memAt(twoPagePatchAddr, twoPagePatch)
}

// startTwoPage does what the patch does: the screen becomes 64×64 (and
// clear), 0230 clears it from now on, and the program continues where its
// CHIP-8 code starts.
func (m *Machine) startTwoPage() {
	m.twoPage = true
	m.disp.setResolution(twoPageSize, twoPageSize)

	m.pc = twoPageProgram
	if m.memAt(twoPageSelfPatchAddr, twoPageSelfPatch) &&
		uint16(m.mem[twoPageSelfClearAddr])<<8|uint16(m.mem[twoPageSelfClearAddr+1]) == twoPageClearOp {
		m.pc = twoPageSelfPatchedProgram
	}
}

// memAt reports whether memory at addr holds want.
func (m *Machine) memAt(addr int, want []byte) bool {
	return addr+len(want) <= len(m.mem) && bytes.Equal(m.mem[addr:addr+len(want)], want)
}

// executeTwoPage0 runs 0230, which clears the whole 64×64 screen once the
// 2 Page Display has started, and reports whether op was it. Before that,
// 0230 is an unsupported machine-code call, as on originalChip8.
func (m *Machine) executeTwoPage0(op uint16) bool {
	if !m.twoPage || op != twoPageClearOp {
		return false
	}

	m.op00E0()

	return true
}
