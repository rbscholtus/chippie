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

func TestHintMask(t *testing.T) {
	t.Parallel()

	got := HintMask(map[string]int{"up": 5, "a": 0xA, "player2Up": 0x15, "odd": -1})
	if want := uint16(1<<5 | 1<<0xA); got != want {
		t.Errorf("HintMask = %016b, want %016b (second-keypad and bad keys left out)", got, want)
	}
}

func TestAlsoRead(t *testing.T) {
	t.Parallel()

	const (
		wasd = 1<<5 | 1<<7 | 1<<8 | 1<<9
		fire = 1 << 6
	)

	tests := []struct {
		name             string
		hint, read, want uint16
	}{
		{"the keys read beyond the hint", wasd, wasd | fire, fire},
		{"nothing beyond the hint", wasd, wasd, 0},
		{"no hint: every key read", 0, wasd, wasd},
		{"a keypad scan says nothing", wasd, everyKey, 0},
		{"a keypad scan without a hint is still listed", 0, everyKey, everyKey},
	}

	for _, tt := range tests {
		if got := AlsoRead(tt.hint, tt.read); got != tt.want {
			t.Errorf("%s: %016b, want %016b", tt.name, got, tt.want)
		}
	}
}

func TestDescribeHints(t *testing.T) {
	t.Parallel()

	got := DescribeHints(map[string]int{"a": 6, "right": 9, "up": 5, "odd": 0xF})
	if want := "↑ W · → D · A E · odd V"; got != want {
		t.Errorf("DescribeHints = %q, want %q", got, want)
	}
}
