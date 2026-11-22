package app

import "testing"

func TestSwitchOff(t *testing.T) {
	t.Parallel()

	if c, s := switchOff(0); c != 0 || s != 1 {
		t.Errorf("switchOff(0) = %v, %v; want 0, 1 (full picture)", c, s)
	}

	if c, _ := switchOff(collapseTicks); c != 1 {
		t.Errorf("switchOff(%d): collapse %v, want 1: the sweep is gone", collapseTicks, c)
	}

	if c, s := switchOff(switchOffTicks); c != 1 || s != 0 {
		t.Errorf("switchOff(%d) = %v, %v; want 1, 0 (dark)", switchOffTicks, c, s)
	}

	prev := float32(2)

	for ticks := collapseTicks; ticks <= switchOffTicks; ticks++ {
		_, s := switchOff(ticks)
		if s > prev || s < 0 {
			t.Fatalf("switchOff(%d): spot %v after %v; want it to fade", ticks, s, prev)
		}

		prev = s
	}
}
