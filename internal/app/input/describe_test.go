package input

import (
	"strings"
	"testing"
)

// TestKeyLabelsMatchKeypad checks every label against the key Ebitengine
// reads for it ("Digit1" for 1, "W" for W).
func TestKeyLabelsMatchKeypad(t *testing.T) {
	t.Parallel()

	for hex, k := range Keypad {
		name := strings.TrimPrefix(k.String(), "Digit")
		if name != keyLabels[hex] {
			t.Errorf("key %X: keypad uses %s, label says %q", hex, k, keyLabels[hex])
		}
	}

	if KeyLabel(16) != "?" || KeyLabel(-1) != "?" {
		t.Error("out-of-range keys should be labelled ?")
	}
}

func TestDescribeMask(t *testing.T) {
	t.Parallel()

	wasd := uint16(1<<5 | 1<<7 | 1<<8 | 1<<9)
	if got := DescribeMask(wasd); got != "W A S D" {
		t.Errorf("DescribeMask = %q, want W A S D", got)
	}

	if DescribeMask(0) != "" {
		t.Error("no keys should describe as empty")
	}
}

func TestDescribeHints(t *testing.T) {
	t.Parallel()

	got := DescribeHints(map[string]int{"a": 6, "right": 9, "up": 5, "odd": 0xF})
	if want := "↑ W · → D · A E · odd V"; got != want {
		t.Errorf("DescribeHints = %q, want %q", got, want)
	}
}

func TestDescribeControls(t *testing.T) {
	t.Parallel()

	got := DescribeControls(map[string]int{"up": 5, "down": 8, "a": 6})
	if want := "↑ W · ↓ S · A E (or arrows, Space)"; got != want {
		t.Errorf("DescribeControls = %q, want %q", got, want)
	}

	if got, want := DescribeControls(map[string]int{"player2A": 1}), "P2 A 1"; got != want {
		t.Errorf("DescribeControls without extras = %q, want %q", got, want)
	}
}
