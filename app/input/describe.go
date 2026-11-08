package input

import (
	"sort"
	"strings"
)

// keyLabels are the keyboard keys of the hex keys, as printed on the
// keyboard; the index is the hex key (see Keypad).
var keyLabels = [16]string{ //nolint:gochecknoglobals // read-only table
	0x0: "X",
	0x1: "1", 0x2: "2", 0x3: "3", 0xC: "4",
	0x4: "Q", 0x5: "W", 0x6: "E", 0xD: "R",
	0x7: "A", 0x8: "S", 0x9: "D", 0xE: "F",
	0xA: "Z", 0xB: "C", 0xF: "V",
}

// padOrder lists the hex keys in the order of the keypad, row by row.
var padOrder = [16]int{0x1, 0x2, 0x3, 0xC, 0x4, 0x5, 0x6, 0xD, 0x7, 0x8, 0x9, 0xE, 0xA, 0x0, 0xB, 0xF} //nolint:gochecknoglobals,lll // read-only table

// KeyLabel returns the keyboard key for a hex key, e.g. "W" for 5.
func KeyLabel(hex int) string {
	if hex < 0 || hex >= len(keyLabels) {
		return "?"
	}

	return keyLabels[hex]
}

// DescribeMask lists the keyboard keys for a set of hex keys (bit k for key
// k), in keypad order: e.g. "W A S D".
func DescribeMask(mask uint16) string {
	labels := make([]string, 0, len(padOrder))

	for _, hex := range padOrder {
		if mask&(1<<hex) != 0 {
			labels = append(labels, keyLabels[hex])
		}
	}

	return strings.Join(labels, " ")
}

// hintNames gives chip-8-database's key names a short label.
var hintNames = map[string]string{ //nolint:gochecknoglobals // read-only table
	"up": "↑", "down": "↓", "left": "←", "right": "→", "a": "A", "b": "B",
	"player2Up": "P2 ↑", "player2Down": "P2 ↓", "player2Left": "P2 ←", "player2Right": "P2 →",
	"player2A": "P2 A", "player2B": "P2 B",
}

// hintOrder sorts hints: directions first, then buttons, then player 2.
var hintOrder = []string{ //nolint:gochecknoglobals // read-only table
	"up", "down", "left", "right", "a", "b",
	"player2Up", "player2Down", "player2Left", "player2Right", "player2A", "player2B",
}

// DescribeHints turns the database's keys hint (e.g. {"up": 5}) into
// keyboard keys: "↑ W · ↓ S · A E".
func DescribeHints(keys map[string]int) string {
	names := make([]string, 0, len(keys))
	for name := range keys {
		names = append(names, name)
	}

	rank := func(n string) int {
		for i, o := range hintOrder {
			if o == n {
				return i
			}
		}

		return len(hintOrder)
	}

	sort.Slice(names, func(i, j int) bool {
		if rank(names[i]) != rank(names[j]) {
			return rank(names[i]) < rank(names[j])
		}

		return names[i] < names[j]
	})

	parts := make([]string, 0, len(names))

	for _, n := range names {
		label, ok := hintNames[n]
		if !ok {
			label = n
		}

		parts = append(parts, label+" "+KeyLabel(keys[n]))
	}

	return strings.Join(parts, " · ")
}
