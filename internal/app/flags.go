package app

import (
	"encoding/hex"

	"github.com/rbscholtus/chippie/internal/chip8"
)

// Saved flag registers (DESIGN §13): programs keep data such as a high score
// in the flag registers with FX75, and Chippie saves them per ROM in the
// preferences, so they come back the next time the ROM runs.

// flagsPrefPrefix starts the preference key for a ROM's flags; the SHA-1
// follows. The value is the 16 bytes in hex; none means all zero.
const flagsPrefPrefix = "flags/"

// flagSaveFrames is the least time between two writes of one ROM's flags.
// A ROM that uses the flags as scratch space can run FX75 every frame.
const flagSaveFrames = 60

// flagMachine is what the saver reads; *chip8.Machine implements it.
type flagMachine interface {
	Flags() [chip8.NumFlags]byte
	FlagSaves() uint32
}

// prefSetter stores a preference; *platform.Prefs implements it.
type prefSetter interface {
	Set(key, value string)
}

// flagSaver saves the running ROM's flag registers when the program changes
// them: only when the bytes differ from what is saved, and at most once
// every flagSaveFrames.
type flagSaver struct {
	key     string // preference key; "" when no ROM runs
	seen    uint32 // the machine's FlagSaves when last checked
	saved   [chip8.NumFlags]byte
	pending [chip8.NumFlags]byte
	dirty   bool // pending differs from saved
	wait    int  // frames until the next write may happen
}

// flagsKey is the preference key for a ROM's flags.
func flagsKey(sha1 string) string { return flagsPrefPrefix + sha1 }

// prefGetter reads a preference; *platform.Prefs implements it.
type prefGetter interface {
	Get(key string) (string, bool)
}

// loadFlags returns the flags saved for a ROM; all zero if none were saved
// or the value is unreadable.
func loadFlags(prefs prefGetter, sha1 string) [chip8.NumFlags]byte {
	var f [chip8.NumFlags]byte

	s, ok := prefs.Get(flagsKey(sha1))
	if !ok {
		return f
	}

	b, err := hex.DecodeString(s)
	if err != nil || len(b) != len(f) {
		return f
	}

	copy(f[:], b)

	return f
}

// start begins watching a newly loaded ROM, whose flags are f (restored
// from the preferences). The machine has just been reset.
func (s *flagSaver) start(sha1 string, f [chip8.NumFlags]byte) {
	*s = flagSaver{key: flagsKey(sha1), saved: f}
}

// frame runs after each emulated frame: it notes a save by the program and
// writes it when allowed.
func (s *flagSaver) frame(m flagMachine, prefs prefSetter) {
	if s.key == "" {
		return
	}

	if n := m.FlagSaves(); n != s.seen {
		s.seen = n
		s.pending = m.Flags()
		s.dirty = s.pending != s.saved
	}

	if s.wait > 0 {
		s.wait--

		return
	}

	if s.dirty {
		s.write(prefs)
	}
}

// flush writes a pending save now, e.g. when the game pauses or another
// ROM is about to start.
func (s *flagSaver) flush(prefs prefSetter) {
	if s.key != "" && s.dirty {
		s.write(prefs)
	}
}

func (s *flagSaver) write(prefs prefSetter) {
	value := hex.EncodeToString(s.pending[:])
	if s.pending == ([chip8.NumFlags]byte{}) {
		value = "" // all zero: delete the preference
	}

	prefs.Set(s.key, value)
	s.saved, s.dirty, s.wait = s.pending, false, flagSaveFrames
}
