package catalogue

import "github.com/rbscholtus/chippie/chip8"

// quirkIDs are the quirks.json ids Chippie understands, in chip8.Quirks
// field order.
var quirkIDs = []string{ //nolint:gochecknoglobals // read-only table
	"shift", "memoryIncrementByX", "memoryLeaveIUnchanged", "wrap", "jump", "vblank", "logic",
}

// quirkField returns a pointer to the chip8.Quirks field for a quirk id.
func quirkField(q *chip8.Quirks, id string) *bool {
	switch id {
	case "shift":
		return &q.Shift
	case "memoryIncrementByX":
		return &q.MemoryIncrementByX
	case "memoryLeaveIUnchanged":
		return &q.MemoryLeaveIUnchanged
	case "wrap":
		return &q.Wrap
	case "jump":
		return &q.Jump
	case "vblank":
		return &q.VBlank
	case "logic":
		return &q.Logic
	default:
		return nil
	}
}

// applyQuirks sets the quirks named in m.
func applyQuirks(q *chip8.Quirks, m map[string]bool) {
	for id, v := range m {
		if f := quirkField(q, id); f != nil {
			*f = v
		}
	}
}

// QuirkMap returns the quirks as a quirks.json-style map.
func QuirkMap(q chip8.Quirks) map[string]bool {
	m := make(map[string]bool, len(quirkIDs))
	for _, id := range quirkIDs {
		m[id] = *quirkField(&q, id)
	}

	return m
}

// Overrides are the user's settings for one ROM (saved per ROM hash).
type Overrides struct {
	Platform string          `json:"platform,omitempty"` // platforms.json id; empty: recommended
	Quirks   map[string]bool `json:"quirks,omitempty"`   // quirks that differ from the platform's
	Tickrate int             `json:"tickrate,omitempty"` // 0: recommended
}

// Empty reports whether there are no overrides.
func (o Overrides) Empty() bool { return o.Platform == "" && len(o.Quirks) == 0 && o.Tickrate == 0 }

// Resolved is the profile a ROM runs with, and the hints the UI uses.
type Resolved struct {
	Profile    chip8.Profile
	PlatformID string
	ROM        *ROM // nil for a ROM that isn't in the database
}

// DefaultPlatform is used for ROMs that aren't in the database.
const DefaultPlatform = "modernChip8"

// Resolve works out the profile for a ROM (README §7):
//
//  1. platform defaults from platforms.json;
//  2. the ROM's entry in programs.json: its platform, quirkyPlatforms and
//     tickrate;
//  3. the user's overrides.
//
// The ROM's platform is the first one in its list that Chippie implements,
// or the first one if none is implemented yet. Unknown ROMs use modern
// CHIP-8.
func (db *DB) Resolve(sha1 string, o Overrides) Resolved {
	rom, _ := db.ROM(sha1)

	id := DefaultPlatform
	if rom != nil && len(rom.Platforms) > 0 {
		id = rom.Platforms[0]

		for _, p := range rom.Platforms {
			if cp, ok := chip8.ParsePlatform(p); ok && cp.Implemented() {
				id = p

				break
			}
		}
	}

	if o.Platform != "" && db.Platforms[o.Platform] != nil {
		id = o.Platform
	}

	cp, _ := chip8.ParsePlatform(id)
	prof := chip8.Profile{Platform: cp, TicksPerFrame: cp.DefaultTicks()}

	if p := db.Platforms[id]; p != nil {
		applyQuirks(&prof.Quirks, p.Quirks)
		prof.TicksPerFrame = p.Tickrate
	}

	if rom != nil {
		applyQuirks(&prof.Quirks, rom.QuirkyPlatforms[id])

		if rom.Tickrate > 0 {
			prof.TicksPerFrame = rom.Tickrate
		}
	}

	applyQuirks(&prof.Quirks, o.Quirks)

	if o.Tickrate > 0 {
		prof.TicksPerFrame = o.Tickrate
	}

	return Resolved{Profile: prof, PlatformID: id, ROM: rom}
}
