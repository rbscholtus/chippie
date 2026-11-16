package input

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// extraKeys are the keys beyond the keypad, each standing in for one name in
// chip-8-database's keys hint: the arrow keys for the directions, Space for
// "a" and Shift for "b".
var extraKeys = [...]struct { //nolint:gochecknoglobals // read-only key table
	hint string
	key  ebiten.Key
}{
	{hintUp, ebiten.KeyArrowUp},
	{hintDown, ebiten.KeyArrowDown},
	{hintLeft, ebiten.KeyArrowLeft},
	{hintRight, ebiten.KeyArrowRight},
	{hintA, ebiten.KeySpace},
	{hintB, ebiten.KeyShift},
}

// numHexKeys is the size of one keypad. A hint of 0x10 or more names a key
// on a second keypad, which Chippie doesn't have.
const numHexKeys = 16

// Extras says which hex key each extra key presses for the running ROM,
// from its keys hint. The zero value presses nothing.
type Extras struct {
	hex [len(extraKeys)]uint8
	set uint8 // bit i: extraKeys[i] has a hex key
}

// NewExtras maps the extra keys for a ROM's keys hint (e.g. {"up": 5}).
// Names the hint lacks, and second-keypad keys, leave their extra key
// unused.
func NewExtras(hints map[string]int) Extras {
	var x Extras

	for i, e := range extraKeys {
		if hex, ok := hints[e.hint]; ok && hex >= 0 && hex < numHexKeys {
			x.hex[i] = uint8(hex)
			x.set |= 1 << i
		}
	}

	return x
}

// Held returns the hex keys the extra keys hold down (bit n for hex key n),
// asking pressed which keys are down.
func (x Extras) Held(pressed func(ebiten.Key) bool) uint16 {
	var mask uint16

	for i, e := range extraKeys {
		if x.set&(1<<i) != 0 && pressed(e.key) {
			mask |= 1 << x.hex[i]
		}
	}

	return mask
}

// Describe names the extra keys in use, for the keys line: e.g. "arrows,
// Space". It is empty when there are none.
func (x Extras) Describe() string {
	const arrows = 0b1111 // the first four extraKeys

	parts := make([]string, 0, 3) //nolint:mnd // arrows, Space, Shift
	if x.set&arrows != 0 {
		parts = append(parts, "arrows")
	}

	for i, name := range []string{"Space", "Shift"} {
		if x.set&(1<<(4+i)) != 0 {
			parts = append(parts, name)
		}
	}

	return strings.Join(parts, ", ")
}
