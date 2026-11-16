package input

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// down returns a pressed func that reports the given keys as down.
func down(keys ...ebiten.Key) func(ebiten.Key) bool {
	return func(k ebiten.Key) bool { return slices.Contains(keys, k) }
}

func TestExtrasHeld(t *testing.T) {
	t.Parallel()

	depot := map[string]int{"up": 5, "left": 7, "down": 8, "right": 9, "a": 6}

	tests := []struct {
		name  string
		hints map[string]int
		keys  []ebiten.Key
		want  uint16
	}{
		{"up arrow", depot, []ebiten.Key{ebiten.KeyArrowUp}, 1 << 5},
		{"all arrows", depot, []ebiten.Key{
			ebiten.KeyArrowUp, ebiten.KeyArrowDown, ebiten.KeyArrowLeft, ebiten.KeyArrowRight,
		}, 1<<5 | 1<<8 | 1<<7 | 1<<9},
		{"space is a", depot, []ebiten.Key{ebiten.KeySpace}, 1 << 6},
		{"shift without b", depot, []ebiten.Key{ebiten.KeyShift}, 0},
		{"shift is b", map[string]int{"b": 0xF}, []ebiten.Key{ebiten.KeyShift}, 1 << 0xF},
		{"key 0", map[string]int{"up": 0}, []ebiten.Key{ebiten.KeyArrowUp}, 1 << 0},
		{"no hint", nil, []ebiten.Key{ebiten.KeyArrowUp, ebiten.KeySpace}, 0},
		{"second keypad ignored", map[string]int{"up": 0x15}, []ebiten.Key{ebiten.KeyArrowUp}, 0},
		{"player 2 ignored", map[string]int{"player2Up": 2}, []ebiten.Key{ebiten.KeyArrowUp}, 0},
		{"keypad keys aren't extras", depot, []ebiten.Key{ebiten.KeyW}, 0},
	}

	for _, tt := range tests {
		if got := NewExtras(tt.hints).Held(down(tt.keys...)); got != tt.want {
			t.Errorf("%s: held %016b, want %016b", tt.name, got, tt.want)
		}
	}
}

func TestExtrasZeroValue(t *testing.T) {
	t.Parallel()

	var x Extras
	if got := x.Held(down(ebiten.KeyArrowUp, ebiten.KeySpace, ebiten.KeyShift)); got != 0 {
		t.Errorf("zero Extras held %016b, want nothing", got)
	}

	if got := x.Describe(); got != "" {
		t.Errorf("zero Extras describe %q, want empty", got)
	}
}

func TestExtrasDescribe(t *testing.T) {
	t.Parallel()

	tests := []struct {
		hints map[string]int
		want  string
	}{
		{map[string]int{"up": 5, "down": 8, "a": 6}, "arrows, Space"},
		{map[string]int{"left": 4, "right": 6}, "arrows"},
		{map[string]int{"a": 6, "b": 4}, "Space, Shift"},
		{map[string]int{"up": 5, "a": 6, "b": 4}, "arrows, Space, Shift"},
		{map[string]int{"player2A": 1}, ""},
	}

	for _, tt := range tests {
		if got := NewExtras(tt.hints).Describe(); got != tt.want {
			t.Errorf("%v: %q, want %q", tt.hints, got, tt.want)
		}
	}
}
