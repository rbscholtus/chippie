// Package testrig runs ROMs headless for tests and compares the display with
// golden frames. It is used only by tests and the golden-import tool.
package testrig

import (
	"fmt"

	"github.com/rbscholtus/chippie/chip8"
)

// Poke writes one byte of memory before the first frame, for example 0x1FF
// to skip a test ROM's menu.
type Poke struct {
	Addr  uint16
	Value byte
}

// KeyEvent holds a key down from the start of frame Press until the start of
// frame Release. Frames are numbered from 0. Release <= Press means the key
// stays down until the end of the run.
type KeyEvent struct {
	Key     uint8
	Press   int
	Release int
}

// Script describes a headless run.
type Script struct {
	Pokes  []Poke
	Keys   []KeyEvent
	Frames int // frames to run
}

// Run loads the ROM with the profile, applies the script and returns the
// machine after the last frame. It fails if the ROM can't be loaded or the
// machine halts.
func Run(rom []byte, p chip8.Profile, s Script) (*chip8.Machine, error) {
	m := chip8.New(p)
	if err := m.LoadROM(rom); err != nil {
		return nil, fmt.Errorf("load ROM: %w", err)
	}

	for _, pk := range s.Pokes {
		m.Poke(pk.Addr, pk.Value)
	}

	for f := range s.Frames {
		applyKeys(m, s.Keys, f)
		m.RunFrame()

		if err := m.State().Err; err != nil {
			return m, fmt.Errorf("frame %d: %w", f, err)
		}
	}

	return m, nil
}

// applyKeys sets the keys that change at the start of frame f.
func applyKeys(m *chip8.Machine, events []KeyEvent, f int) {
	for _, e := range events {
		switch {
		case f == e.Press:
			m.SetKey(e.Key, true)
		case f == e.Release && e.Release > e.Press:
			m.SetKey(e.Key, false)
		}
	}
}
