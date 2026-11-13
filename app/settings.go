package app

import (
	"slices"
	"strconv"

	"github.com/rbscholtus/chippie/chip8"
)

// settingKind is what a settings row controls.
type settingKind uint8

const (
	settingPlatform settingKind = iota
	settingSpeed
	settingQuirk
	settingReset
	settingTrace
	settingCopyTrace
	settingBack
)

// settingRow is one line of the settings panel.
type settingRow struct {
	kind  settingKind
	quirk int // index into quirkSettings, for settingQuirk
}

// quirkSettings are the quirks in the panel, with plain names.
var quirkSettings = []struct { //nolint:gochecknoglobals // read-only table
	id, label string
}{
	{"logic", "VF reset after AND/OR/XOR"},
	{"memoryIncrementByX", "Load/store: I += X"},
	{"memoryLeaveIUnchanged", "Load/store: I unchanged"},
	{"vblank", "Wait for vblank to draw"},
	{"wrap", "Sprites wrap at the edges"},
	{"shift", "Shift uses VX only"},
	{"jump", "BNNN jumps to XNN + VX"},
	{legacyQuirkID, "Legacy SUPER-CHIP (HP48)"},
}

// legacyQuirkID is the row for chip8.Quirks.LegacySuperChip. It is shown only
// on platforms it affects, and changes the display layout, so it restarts
// the ROM.
const legacyQuirkID = "legacySuperChip"

// speedSteps are the speed knob's settings, in instructions per frame.
var speedSteps = []int{1, 2, 3, 5, 7, 10, 12, 15, 20, 30, 50, 100, 200, 300, 500, 1000, 2000, 5000, 10000} //nolint:gochecknoglobals,lll // read-only table

// traceLength is how many instructions the trace keeps.
const traceLength = 4096

// settingEffect tells the game what a change needs.
type settingEffect uint8

const (
	effectNone    settingEffect = iota
	effectLive                  // apply speed and quirks to the running machine
	effectRestart               // the platform changed: restart the ROM
	effectTrace                 // tracing was switched on or off
	effectCopy                  // copy the trace to the clipboard
	effectClose                 // close the panel
)

// settings is the panel's state for the running ROM. It holds no
// Ebitengine state, so it can be tested directly.
type settings struct {
	rows        []settingRow
	selected    int
	profile     chip8.Profile                      // what the ROM runs with now
	own         chip8.Profile                      // the ROM's recommended profile
	forPlatform func(chip8.Platform) chip8.Profile // recommended on another platform
	trace       bool
}

// newSettings opens the panel. own is the ROM's recommended profile;
// forPlatform gives its recommended profile on another platform
// (catalogue.Resolve with that platform).
func newSettings(current, own chip8.Profile, forPlatform func(chip8.Platform) chip8.Profile, trace bool) *settings {
	s := &settings{profile: current, own: own, forPlatform: forPlatform, trace: trace}
	s.buildRows()

	return s
}

func (s *settings) buildRows() {
	s.rows = s.rows[:0]
	s.rows = append(s.rows, settingRow{kind: settingPlatform}, settingRow{kind: settingSpeed})

	for i, q := range quirkSettings {
		if q.id == legacyQuirkID && !s.profile.Platform.HasLegacyMode() {
			continue
		}

		s.rows = append(s.rows, settingRow{kind: settingQuirk, quirk: i})
	}

	s.rows = append(s.rows, settingRow{kind: settingReset}, settingRow{kind: settingTrace})
	if s.trace {
		s.rows = append(s.rows, settingRow{kind: settingCopyTrace})
	}

	s.rows = append(s.rows, settingRow{kind: settingBack})
	s.selected = min(s.selected, len(s.rows)-1)
}

// move changes the selected row, wrapping at the ends.
func (s *settings) move(delta int) {
	n := len(s.rows)
	s.selected = ((s.selected+delta)%n + n) % n
}

// change acts on row i: dir -1 or +1 steps a value down or up, 0 toggles
// or activates.
func (s *settings) change(i, dir int) settingEffect {
	if i < 0 || i >= len(s.rows) {
		return effectNone
	}

	row := s.rows[i]

	switch row.kind {
	case settingPlatform:
		next := stepPlatform(s.profile.Platform, dir)
		if next == s.profile.Platform {
			return effectNone
		}

		s.profile = s.forPlatform(next)
		s.buildRows()

		return effectRestart
	case settingSpeed:
		s.profile.TicksPerFrame = stepSpeed(s.profile.TicksPerFrame, dir)

		return effectLive
	case settingQuirk:
		f := quirkPointer(&s.profile.Quirks, quirkSettings[row.quirk].id)
		*f = !*f

		if quirkSettings[row.quirk].id == legacyQuirkID {
			return effectRestart
		}

		return effectLive
	case settingReset:
		old := s.profile
		s.profile = s.own
		s.buildRows()

		if s.profile.Platform != old.Platform || s.profile.Quirks.LegacySuperChip != old.Quirks.LegacySuperChip {
			return effectRestart
		}

		return effectLive
	case settingTrace:
		s.trace = !s.trace
		s.buildRows()

		return effectTrace
	case settingCopyTrace:
		return effectCopy
	case settingBack:
		return effectClose
	default:
		return effectNone
	}
}

// settingLabel is how a row is shown: its name and value, and whether the
// value differs from the recommendation.
type settingLabel struct {
	name, value string
	changed     bool
}

// label returns how row i is shown.
func (s *settings) label(i int) settingLabel {
	row := s.rows[i]

	rec := s.own
	if s.profile.Platform != s.own.Platform {
		rec = s.forPlatform(s.profile.Platform)
	}

	switch row.kind {
	case settingPlatform:
		return settingLabel{"Platform", s.profile.Platform.Name(), s.profile.Platform != s.own.Platform}
	case settingSpeed:
		speed := strconv.Itoa(s.profile.TicksPerFrame) + " per frame"

		return settingLabel{"Speed", speed, s.profile.TicksPerFrame != rec.TicksPerFrame}
	case settingQuirk:
		q := quirkSettings[row.quirk]
		on := *quirkPointer(&s.profile.Quirks, q.id)

		return settingLabel{q.label, onOff(on), on != *quirkPointer(&rec.Quirks, q.id)}
	case settingReset:
		return settingLabel{"Reset to recommended", "", false}
	case settingTrace:
		return settingLabel{"Instruction trace", onOff(s.trace), false}
	case settingCopyTrace:
		return settingLabel{"Copy trace", "", false}
	case settingBack:
		return settingLabel{"Back", "", false}
	default:
		return settingLabel{}
	}
}

// steps reports whether a row takes left/right steps (platform, speed).
func (s *settings) steps(i int) bool {
	return i >= 0 && i < len(s.rows) && (s.rows[i].kind == settingPlatform || s.rows[i].kind == settingSpeed)
}

func onOff(b bool) string {
	if b {
		return "on"
	}

	return "off"
}

// stepPlatform returns the next (dir > 0) or previous implemented platform,
// wrapping; dir 0 steps forward.
func stepPlatform(p chip8.Platform, dir int) chip8.Platform {
	var impl []chip8.Platform

	for _, q := range chip8.Platforms() {
		if q.Implemented() {
			impl = append(impl, q)
		}
	}

	i := 0

	for j, q := range impl {
		if q == p {
			i = j
		}
	}

	if dir >= 0 {
		return impl[(i+1)%len(impl)]
	}

	return impl[(i-1+len(impl))%len(impl)]
}

// stepSpeed returns the next speed step above (dir > 0) or below n.
func stepSpeed(n, dir int) int {
	if dir < 0 {
		for _, speedStep := range slices.Backward(speedSteps) {
			if speedStep < n {
				return speedStep
			}
		}

		return speedSteps[0]
	}

	for _, v := range speedSteps {
		if v > n {
			return v
		}
	}

	return speedSteps[len(speedSteps)-1]
}

// quirkPointer returns the field for a quirks.json id.
func quirkPointer(q *chip8.Quirks, id string) *bool {
	switch id {
	case "logic":
		return &q.Logic
	case "memoryIncrementByX":
		return &q.MemoryIncrementByX
	case "memoryLeaveIUnchanged":
		return &q.MemoryLeaveIUnchanged
	case "vblank":
		return &q.VBlank
	case "wrap":
		return &q.Wrap
	case "shift":
		return &q.Shift
	case legacyQuirkID:
		return &q.LegacySuperChip
	default: // "jump"
		return &q.Jump
	}
}
