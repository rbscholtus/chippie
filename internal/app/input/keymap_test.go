package input

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestKeypadCoversAllKeysOnce(t *testing.T) {
	t.Parallel()

	seen := map[ebiten.Key]int{}

	for hex, k := range Keypad {
		if prev, dup := seen[k]; dup {
			t.Errorf("%v is mapped to both %X and %X", k, prev, hex)
		}

		seen[k] = hex
	}

	// Spot-check the layout corners.
	corners := map[int]ebiten.Key{
		0x1: ebiten.Key1, 0xC: ebiten.Key4, 0xA: ebiten.KeyZ, 0xF: ebiten.KeyV, 0x0: ebiten.KeyX,
	}
	for hex, want := range corners {
		if Keypad[hex] != want {
			t.Errorf("key %X = %v, want %v", hex, Keypad[hex], want)
		}
	}
}

type recorder [16]bool

func (r *recorder) SetKey(k uint8, down bool) { r[k] = down }

func TestRelease(t *testing.T) {
	t.Parallel()

	var r recorder
	for i := range r {
		r[i] = true
	}

	Release(&r)

	if r != (recorder{}) {
		t.Errorf("keys still down: %v", r)
	}
}
