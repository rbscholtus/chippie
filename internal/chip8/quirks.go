package chip8

// Quirks are the behaviour differences between CHIP-8 platforms. Field names
// follow chip-8-database's quirks.json. Instruction code checks these flags;
// it never checks which platform is running, except to decide whether an
// opcode exists at all.
type Quirks struct {
	Shift                 bool // 8XY6/8XYE shift vX in place (ignore vY)
	MemoryIncrementByX    bool // FX55/FX65 increment I by X (not X+1)
	MemoryLeaveIUnchanged bool // FX55/FX65 leave I unchanged
	Wrap                  bool // sprites wrap at the screen edge instead of clipping
	Jump                  bool // BNNN behaves as BXNN (jump to XNN + vX)
	VBlank                bool // DXYN waits for vertical blank
	Logic                 bool // 8XY1/2/3 reset vF to 0

	// LegacySuperChip selects how SUPER-CHIP really worked on the HP48. It
	// is Chippie's own: quirks.json has no such quirk, and its superchip
	// entry describes the modern behaviour almost all ROMs expect. With it
	// on, on SUPER-CHIP platforms:
	//   - lo-res DXYN waits for vertical blank;
	//   - a lo-res pixel is 2×2 screen pixels and scrolling moves screen
	//     pixels, so half of a lo-res pixel at a time;
	//   - lo-res DXY0 draws 8×16 instead of 16×16;
	//   - hi-res DXYN sets VF to the rows that collided or were clipped at
	//     the bottom, instead of 0 or 1;
	//   - 00FE and 00FF keep the screen instead of clearing it.
	//
	// See third_party/chip8-test-suite/legacy-superchip.md.
	LegacySuperChip bool
}

// DefaultQuirks returns the platform's quirks from platforms.json.
func DefaultQuirks(p Platform) Quirks { return p.info().quirks }
