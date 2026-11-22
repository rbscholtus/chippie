// Package input maps the keyboard to the CHIP-8 hex keypad.
package input

import "github.com/hajimehoshi/ebiten/v2"

// Keypad is the usual layout: the left block of a QWERTY keyboard stands in
// for the COSMAC VIP's 4×4 hex keypad.
//
//	1 2 3 4        1 2 3 C
//	Q W E R   →    4 5 6 D
//	A S D F        7 8 9 E
//	Z X C V        A 0 B F
//
// The index is the hex key.
var Keypad = [16]ebiten.Key{ //nolint:gochecknoglobals // read-only key table
	0x0: ebiten.KeyX,
	0x1: ebiten.Key1, 0x2: ebiten.Key2, 0x3: ebiten.Key3, 0xC: ebiten.Key4,
	0x4: ebiten.KeyQ, 0x5: ebiten.KeyW, 0x6: ebiten.KeyE, 0xD: ebiten.KeyR,
	0x7: ebiten.KeyA, 0x8: ebiten.KeyS, 0x9: ebiten.KeyD, 0xE: ebiten.KeyF,
	0xA: ebiten.KeyZ, 0xB: ebiten.KeyC, 0xF: ebiten.KeyV,
}

// KeySetter receives keypad state; *chip8.Machine implements it.
type KeySetter interface {
	SetKey(key uint8, down bool)
}

// Poll reads the keyboard and sets every hex key's state. Keys in also
// (bit n for hex key n, e.g. from the touch remote) count as held too.
func Poll(m KeySetter, also uint16) {
	for hex, k := range Keypad {
		m.SetKey(uint8(hex), ebiten.IsKeyPressed(k) || also&(1<<hex) != 0)
	}
}

// Release sets every hex key up, e.g. when the game pauses.
func Release(m KeySetter) {
	for hex := range Keypad {
		m.SetKey(uint8(hex), false)
	}
}
