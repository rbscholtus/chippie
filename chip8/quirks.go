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
}

// DefaultQuirks returns the platform's quirks from platforms.json.
func DefaultQuirks(p Platform) Quirks { return p.info().quirks }
