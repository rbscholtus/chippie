// Package chip8 is the Chippie emulator core: CPU, memory, display, timers,
// quirks and the instruction trace.
//
// The core imports only the standard library, never reads the clock and does
// no I/O. It advances only when the caller calls [Machine.RunFrame], so every
// run with the same profile, seed and inputs is identical.
package chip8

// Platform identifies a CHIP-8 platform (a historical interpreter or a
// modern variant). The values follow the ids in chip-8-database's
// platforms.json.
type Platform uint8

// Platforms known to Chippie. Only some are implemented; see
// [Platform.Implemented].
const (
	OriginalChip8 Platform = iota // COSMAC VIP CHIP-8
	HybridVIP                     // COSMAC VIP CHIP-8 with machine-code routines
	ModernChip8                   // the behaviour most modern ROMs expect
	Chip8X                        // VIP colour extension
	Chip48                        // HP48 CHIP-48
	SuperChip1                    // SUPER-CHIP 1.0
	SuperChip                     // SUPER-CHIP 1.1 (modern behaviour unless Quirks.LegacySuperChip)
	MegaChip8                     // MEGA-CHIP
	XOChip                        // XO-CHIP
	numPlatforms
)

// Memory sizes in bytes. They must be powers of two, because addresses are
// masked with size-1.
const (
	memSize4K  = 4 << 10  // CHIP-8, CHIP-48 and SUPER-CHIP
	memSize64K = 64 << 10 // XO-CHIP
	memSize16M = 16 << 20 // MEGA-CHIP
)

// Display resolutions used by the platforms.
const (
	loresWidth   = 64
	loresHeight  = 32
	hiresWidth   = 128
	hiresHeight  = 64
	megaWidth    = 256
	megaHeight   = 192
	singlePlane  = 1
	xoChipPlanes = 2
)

// opSet says which optional instruction groups a platform has.
type opSet uint8

const (
	// opsSuper is SUPER-CHIP 1.0: 00FD 00FE 00FF DXY0 FX30 FX75 FX85.
	opsSuper opSet = 1 << iota
	// opsScroll is SUPER-CHIP 1.1's scrolling: 00CN 00FB 00FC.
	opsScroll
)

// platformInfo holds the defaults for one platform. Tick rates, resolutions
// and quirks come from chip-8-database's platforms.json.
type platformInfo struct {
	id          string // platforms.json id
	name        string // human-readable name
	ticks       int    // defaultTickrate: instructions per frame
	memSize     int    // bytes of addressable memory
	maxWidth    int    // largest display resolution
	maxHeight   int
	planes      int // display bit-planes
	quirks      Quirks
	ops         opSet // optional instruction groups
	implemented bool
}

// vipQuirks are the COSMAC VIP's quirks (originalChip8, hybridVIP, chip8x).
var vipQuirks = Quirks{VBlank: true, Logic: true} //nolint:gochecknoglobals // read-only defaults

// platforms is indexed by Platform.
var platforms = [numPlatforms]platformInfo{ //nolint:gochecknoglobals // read-only defaults table
	OriginalChip8: {
		id: "originalChip8", name: "CHIP-8 (COSMAC VIP)", ticks: 15, memSize: memSize4K,
		maxWidth: loresWidth, maxHeight: loresHeight, planes: singlePlane,
		quirks: vipQuirks, implemented: true,
	},
	HybridVIP: {
		id: "hybridVIP", name: "CHIP-8 with machine code (COSMAC VIP)", ticks: 15, memSize: memSize4K,
		maxWidth: loresWidth, maxHeight: loresHeight, planes: singlePlane,
		quirks: vipQuirks,
	},
	ModernChip8: {
		id: "modernChip8", name: "Modern CHIP-8", ticks: 12, memSize: memSize4K,
		maxWidth: loresWidth, maxHeight: loresHeight, planes: singlePlane,
		implemented: true,
	},
	Chip8X: {
		id: "chip8x", name: "CHIP-8X", ticks: 15, memSize: memSize4K,
		maxWidth: loresWidth, maxHeight: loresHeight, planes: singlePlane,
		quirks: vipQuirks,
	},
	Chip48: {
		id: "chip48", name: "CHIP-48", ticks: 30, memSize: memSize4K,
		maxWidth: loresWidth, maxHeight: loresHeight, planes: singlePlane,
		quirks: Quirks{Shift: true, MemoryIncrementByX: true, Jump: true}, implemented: true,
	},
	SuperChip1: {
		id: "superchip1", name: "SUPER-CHIP 1.0", ticks: 30, memSize: memSize4K,
		maxWidth: hiresWidth, maxHeight: hiresHeight, planes: singlePlane,
		quirks: Quirks{Shift: true, MemoryIncrementByX: true, Jump: true},
		ops:    opsSuper, implemented: true,
	},
	SuperChip: {
		id: "superchip", name: "SUPER-CHIP 1.1", ticks: 30, memSize: memSize4K,
		maxWidth: hiresWidth, maxHeight: hiresHeight, planes: singlePlane,
		quirks: Quirks{Shift: true, MemoryLeaveIUnchanged: true, Jump: true},
		ops:    opsSuper | opsScroll, implemented: true,
	},
	MegaChip8: {
		id: "megachip8", name: "MEGA-CHIP", ticks: 1000, memSize: memSize16M,
		maxWidth: megaWidth, maxHeight: megaHeight, planes: singlePlane,
		quirks: Quirks{Shift: true, MemoryLeaveIUnchanged: true, Jump: true},
	},
	XOChip: {
		id: "xochip", name: "XO-CHIP", ticks: 100, memSize: memSize64K,
		maxWidth: hiresWidth, maxHeight: hiresHeight, planes: xoChipPlanes,
		quirks: Quirks{Wrap: true},
	},
}

// String returns the platforms.json id, e.g. "modernChip8".
func (p Platform) String() string {
	if p >= numPlatforms {
		return "unknown"
	}

	return platforms[p].id
}

// Name returns a human-readable name, e.g. "SUPER-CHIP 1.1".
func (p Platform) Name() string { return p.info().name }

// Implemented reports whether Chippie can run ROMs for this platform yet.
func (p Platform) Implemented() bool { return p < numPlatforms && platforms[p].implemented }

// HasLegacyMode reports whether [Quirks.LegacySuperChip] has any effect on
// this platform.
func (p Platform) HasLegacyMode() bool { return p == SuperChip1 || p == SuperChip }

// DefaultTicks returns the platform's default instructions per frame.
func (p Platform) DefaultTicks() int { return p.info().ticks }

// MemorySize returns the platform's addressable memory in bytes.
func (p Platform) MemorySize() int { return p.info().memSize }

// info returns the platform's defaults, falling back to modernChip8 for an
// out-of-range value so that a bad Platform never panics.
func (p Platform) info() *platformInfo {
	if p >= numPlatforms {
		return &platforms[ModernChip8]
	}

	return &platforms[p]
}

// Platforms returns every known platform, in platforms.json order.
func Platforms() []Platform {
	out := make([]Platform, 0, numPlatforms)
	for p := range numPlatforms {
		out = append(out, p)
	}

	return out
}

// ParsePlatform returns the Platform for a platforms.json id.
func ParsePlatform(id string) (Platform, bool) {
	for p := range numPlatforms {
		if platforms[p].id == id {
			return p, true
		}
	}

	return ModernChip8, false
}

// Profile is everything needed to start a ROM: platform, quirks and speed.
type Profile struct {
	Platform      Platform
	Quirks        Quirks
	TicksPerFrame int    // instructions per frame (the speed knob)
	Seed          uint64 // seed for CXNN; tests use a fixed seed
}

// DefaultProfile returns the platform's default quirks and tick rate, with
// seed 0.
func DefaultProfile(p Platform) Profile {
	return Profile{
		Platform:      p,
		Quirks:        DefaultQuirks(p),
		TicksPerFrame: p.DefaultTicks(),
	}
}
