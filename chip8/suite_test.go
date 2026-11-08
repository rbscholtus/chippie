package chip8_test

import (
	"crypto/sha1" //nolint:gosec // G505: SHA-1 identifies ROMs, as in chip-8-database; not for security
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/rbscholtus/chippie/chip8"
	"github.com/rbscholtus/chippie/internal/testrig"
)

// Paths from the chip8/ package directory.
const (
	suiteBin  = "../third_party/chip8-test-suite/bin"
	goldenDir = "../testdata/golden"
)

// Hex key values used by the test ROMs.
const (
	key1 = 0x1
	key6 = 0x6
	keyB = 0xB
)

// testSeed is the CXNN seed for every suite run.
const testSeed = 0xC0FFEE

// suiteROMs pins the test suite (tag v4.2): a submodule bump that changes a
// ROM fails here, loudly.
var suiteROMs = map[string]string{
	"1-chip8-logo.ch8": "30f27e5cee5b325fd1681ee98a14de60bfbe951f",
	"2-ibm-logo.ch8":   "b9bbc12cee3f7b9d3b1f69161f7d7a2d86953379",
	"3-corax+.ch8":     "b2dacf6d85785d6c2315ce449912c8a8a5954e2e",
	"4-flags.ch8":      "55a6716dacc2f93dce3d39fb8d231083016a1cc0",
	"6-keypad.ch8":     "455b9fc69cc06e2b5b72f7d1ac5f6c86ac349e77",
	"7-beep.ch8":       "b119651b5aa08557a85ca2ad5de3d1a86796b66b",
}

// suitePlatforms are the platforms the screenshot tests run on.
var suitePlatforms = []chip8.Platform{chip8.ModernChip8, chip8.OriginalChip8}

// loadSuiteROM reads a test ROM and checks its pinned hash.
func loadSuiteROM(t *testing.T, name string) []byte {
	t.Helper()

	rom, err := os.ReadFile(filepath.Join(suiteBin, name))
	if err != nil {
		t.Fatalf("read %s (is the submodule checked out? git submodule update --init): %v", name, err)
	}

	sum := sha1.Sum(rom) //nolint:gosec // G401: see import
	if got, want := hex.EncodeToString(sum[:]), suiteROMs[name]; got != want {
		t.Fatalf("%s: SHA-1 %s, want %s (test suite changed?)", name, got, want)
	}

	return rom
}

func suiteProfile(p chip8.Platform) chip8.Profile {
	prof := chip8.DefaultProfile(p)
	prof.Seed = testSeed

	return prof
}

func TestSuiteScreens(t *testing.T) {
	t.Parallel()

	held := func(k uint8, from int) testrig.KeyEvent { return testrig.KeyEvent{Key: k, Press: from} }

	tests := []struct {
		name   string
		rom    string
		script testrig.Script
		golden string
	}{
		{"chip8-logo", "1-chip8-logo.ch8", testrig.Script{Frames: 60}, "1-chip8-logo"},
		{"ibm-logo", "2-ibm-logo.ch8", testrig.Script{Frames: 60}, "2-ibm-logo"},
		{"corax+", "3-corax+.ch8", testrig.Script{Frames: 300}, "3-corax+"},
		{"flags", "4-flags.ch8", testrig.Script{Frames: 300}, "4-flags"},
		{
			"keypad-down", "6-keypad.ch8",
			testrig.Script{
				Pokes:  []testrig.Poke{{Addr: 0x1FF, Value: 1}},
				Keys:   []testrig.KeyEvent{held(key1, 30), held(key6, 30)},
				Frames: 120,
			},
			"6-keypad-down",
		},
		{
			"keypad-up", "6-keypad.ch8",
			testrig.Script{
				Pokes:  []testrig.Poke{{Addr: 0x1FF, Value: 2}},
				Keys:   []testrig.KeyEvent{held(key1, 30), held(key6, 30)},
				Frames: 120,
			},
			"6-keypad-up",
		},
		{
			// FX0A must halt (the delay timer runs out first) and wait for
			// the release before reporting "ALL GOOD".
			"keypad-getkey", "6-keypad.ch8",
			testrig.Script{
				Pokes:  []testrig.Poke{{Addr: 0x1FF, Value: 3}},
				Keys:   []testrig.KeyEvent{{Key: key1, Press: 30, Release: 40}},
				Frames: 90,
			},
			"6-keypad-getkey",
		},
	}

	for _, tt := range tests {
		for _, p := range suitePlatforms {
			t.Run(tt.name+"/"+p.String(), func(t *testing.T) {
				t.Parallel()

				rom := loadSuiteROM(t, tt.rom)

				m, err := testrig.Run(rom, suiteProfile(p), tt.script)
				if err != nil {
					t.Fatal(err)
				}

				want, err := testrig.LoadGolden(filepath.Join(goldenDir, tt.golden+".png"))
				if err != nil {
					t.Fatal(err)
				}

				diff, err := testrig.Diff(want, testrig.FromDisplay(m.Display()))
				if err != nil {
					t.Fatal(err)
				}

				if diff != "" {
					t.Errorf("display differs from %s:\n%s", tt.golden, diff)
				}
			})
		}
	}
}

// TestSuiteBeep checks test 7 through the buzzer state: SOS starts by
// itself, and holding B gives manual control.
func TestSuiteBeep(t *testing.T) {
	t.Parallel()

	for _, p := range suitePlatforms {
		t.Run(p.String(), func(t *testing.T) {
			t.Parallel()

			rom := loadSuiteROM(t, "7-beep.ch8")

			m, err := testrig.Run(rom, suiteProfile(p), testrig.Script{Frames: 3})
			if err != nil {
				t.Fatal(err)
			}

			if !m.Sound().Buzzer {
				t.Error("SOS: buzzer off at frame 3, want on")
			}

			// Hold B from frame 10: manual mode keeps the buzzer on while held.
			m, err = testrig.Run(rom, suiteProfile(p), testrig.Script{
				Keys:   []testrig.KeyEvent{{Key: keyB, Press: 10}},
				Frames: 120,
			})
			if err != nil {
				t.Fatal(err)
			}

			if !m.Sound().Buzzer {
				t.Error("manual: buzzer off while B is held, want on")
			}

			// Release B: the buzzer goes off within a frame or two.
			m, err = testrig.Run(rom, suiteProfile(p), testrig.Script{
				Keys:   []testrig.KeyEvent{{Key: keyB, Press: 10, Release: 100}},
				Frames: 105,
			})
			if err != nil {
				t.Fatal(err)
			}

			if m.Sound().Buzzer {
				t.Error("manual: buzzer on after B was released, want off")
			}
		})
	}
}
