package app

import "testing"

func TestBarLit(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		value float64
		lit   int
	}{{0, 0}, {0.3, 3}, {0.7, 7}, {1, 10}, {0.05, 1}} {
		n := 0

		for i := range osdBarSteps {
			if barLit(i, tc.value) {
				n++
			}
		}

		if n != tc.lit {
			t.Errorf("value %.2f: %d bars lit, want %d", tc.value, n, tc.lit)
		}
	}
}

// TestDimIsPremultiplied checks the bug that lit every bar: an unlit colour
// must have no channel brighter than its opacity.
func TestDimIsPremultiplied(t *testing.T) {
	t.Parallel()

	c := dim(osdColour, osdUnlit)

	if c.A != osdUnlit || c.R > c.A || c.G > c.A || c.B > c.A {
		t.Errorf("dim(osdColour) = %v: not a valid pre-multiplied colour", c)
	}

	if c.G == 0 {
		t.Errorf("dim(osdColour) = %v: should still be a faint green", c)
	}

	for _, p := range []struct {
		name       string
		r, g, b, a uint8
	}{{"osdPanel", osdPanel.R, osdPanel.G, osdPanel.B, osdPanel.A}} {
		if p.r > p.a || p.g > p.a || p.b > p.a {
			t.Errorf("%s is not pre-multiplied", p.name)
		}
	}
}

// TestOSDHold checks that held lines (PAUSED) stay, come back after
// something shown over them, and go when released.
func TestOSDHold(t *testing.T) {
	t.Parallel()

	o := &osd{bar: -1}
	o.hold("PAUSED")

	if len(o.lines) != 1 || o.lines[0] != "PAUSED" {
		t.Fatalf("held: %v", o.lines)
	}

	o.showBar("VOLUME", 0.5)

	for range osdTicks {
		o.update()
	}

	if len(o.lines) != 1 || o.lines[0] != "PAUSED" || o.bar >= 0 {
		t.Errorf("after the volume: %v, bar %v; want PAUSED again", o.lines, o.bar)
	}

	o.hold()

	if len(o.lines) != 0 {
		t.Errorf("released: %v", o.lines)
	}
}
