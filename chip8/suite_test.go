package chip8_test

import (
	"crypto/sha1" //nolint:gosec // G505: SHA-1 identifies ROMs, as in chip-8-database; not for security
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rbscholtus/chippie/chip8"
	"github.com/rbscholtus/chippie/internal/testrig"
)

// Paths from the chip8/ package directory.
const (
	suiteBin   = "../third_party/chip8-test-suite/bin"
	goldenDir  = "../testdata/golden"
	failureDir = "../testdata/failures"
)

// Hex key values used by the test ROMs.
const (
	key1 = 0x1
	key6 = 0x6
	keyB = 0xB
)

// testSeed is the CXNN seed for every suite run.
const testSeed = 0xC0FFEE

// TestMain clears old failure files, so testdata/failures only holds this
// run's.
func TestMain(m *testing.M) {
	if err := testrig.ClearFailures(failureDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// suiteROMs pins the test suite (tag v4.2): a submodule bump that changes a
// ROM fails here, loudly.
var suiteROMs = map[string]string{
	"1-chip8-logo.ch8": "30f27e5cee5b325fd1681ee98a14de60bfbe951f",
	"2-ibm-logo.ch8":   "b9bbc12cee3f7b9d3b1f69161f7d7a2d86953379",
	"3-corax+.ch8":     "b2dacf6d85785d6c2315ce449912c8a8a5954e2e",
	"4-flags.ch8":      "55a6716dacc2f93dce3d39fb8d231083016a1cc0",
	"5-quirks.ch8":     "e2149cb836131a142ca7e2dc2f2283381ae5faaa",
	"6-keypad.ch8":     "455b9fc69cc06e2b5b72f7d1ac5f6c86ac349e77",
	"7-beep.ch8":       "b119651b5aa08557a85ca2ad5de3d1a86796b66b",
	"8-scrolling.ch8":  "477b3e09c43839ea5478b4f0e24536edab594f89",
}

// The platforms the CHIP-8 tests run on.
var (
	chip8Platforms = []chip8.Platform{chip8.ModernChip8, chip8.OriginalChip8}
	modernOnly     = []chip8.Platform{chip8.ModernChip8}
	originalOnly   = []chip8.Platform{chip8.OriginalChip8}
)

// suiteCase is one row of the screenshot table. M5 and M7 add rows for
// SUPER-CHIP and XO-CHIP.
type suiteCase struct {
	name      string
	rom       string
	platforms []chip8.Platform
	script    testrig.Script
	golden    string
	// imported: the golden comes from the suite's pictures. Otherwise it
	// is generated with -update and checked by eye; a "%s" in golden is
	// replaced by the platform.
	imported bool
}

func poke1FF(v byte) []testrig.Poke { return []testrig.Poke{{Addr: 0x1FF, Value: v}} }

// keysFrom is when the keypad tests start holding keys: after the test's
// screen is drawn.
const keysFrom = 30

// held holds a key from keysFrom to the end of the run.
func held(k uint8) testrig.KeyEvent { return testrig.KeyEvent{Key: k, Press: keysFrom} }

var suiteCases = []suiteCase{
	{"chip8-logo", "1-chip8-logo.ch8", chip8Platforms, testrig.Script{Frames: 60}, "1-chip8-logo", true},
	{"ibm-logo", "2-ibm-logo.ch8", chip8Platforms, testrig.Script{Frames: 60}, "2-ibm-logo", true},
	{"corax+", "3-corax+.ch8", chip8Platforms, testrig.Script{Frames: 300}, "3-corax+", true},
	{"flags", "4-flags.ch8", chip8Platforms, testrig.Script{Frames: 300}, "4-flags", true},
	// The menus' cursor blinks: frame 56 shows it on both platforms, as in
	// the reference pictures.
	{"quirks-menu", "5-quirks.ch8", chip8Platforms, testrig.Script{Frames: 56}, "5-quirks-menu", true},
	{
		// Quirks test with the CHIP-8 target on the COSMAC VIP: every
		// quirk matches (quirks.png).
		"quirks-chip8", "5-quirks.ch8", originalOnly,
		testrig.Script{Pokes: poke1FF(1), Frames: 600},
		"5-quirks-chip8", true,
	},
	{
		// The same target on modern CHIP-8: vF reset and display wait are
		// off, so they show a cross. No reference picture exists.
		"quirks-chip8", "5-quirks.ch8", modernOnly,
		testrig.Script{Pokes: poke1FF(1), Frames: 600},
		"5-quirks-chip8-on-%s", false,
	},
	{"keypad-menu", "6-keypad.ch8", chip8Platforms, testrig.Script{Frames: 56}, "6-keypad-menu", true},
	{
		"keypad-down", "6-keypad.ch8", chip8Platforms,
		testrig.Script{Pokes: poke1FF(1), Keys: []testrig.KeyEvent{held(key1), held(key6)}, Frames: 120},
		"6-keypad-down", true,
	},
	{
		"keypad-up", "6-keypad.ch8", chip8Platforms,
		testrig.Script{Pokes: poke1FF(2), Keys: []testrig.KeyEvent{held(key1), held(key6)}, Frames: 120},
		"6-keypad-up", true,
	},
	{
		// FX0A must halt (the delay timer runs out first) and wait for the
		// release before reporting "ALL GOOD".
		"keypad-getkey", "6-keypad.ch8", chip8Platforms,
		testrig.Script{Pokes: poke1FF(3), Keys: []testrig.KeyEvent{{Key: key1, Press: 30, Release: 40}}, Frames: 90},
		"6-keypad-getkey", true,
	},
	// 8-scrolling (menu and lo-res goldens are imported) needs SUPER-CHIP's
	// 00FE even for its menu: its rows arrive with M5.
}

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

	checker := testrig.Checker{GoldenDir: goldenDir, FailureDir: failureDir, Update: chip8.Update()}

	for _, tc := range suiteCases {
		for _, p := range tc.platforms {
			name := tc.golden
			if !tc.imported {
				name = fmt.Sprintf(tc.golden, p)
			}

			t.Run(tc.name+"/"+p.String(), func(t *testing.T) {
				t.Parallel()

				m, err := testrig.Run(loadSuiteROM(t, tc.rom), suiteProfile(p), tc.script)
				if err != nil {
					t.Fatal(err)
				}

				// Failure files go in a per-platform folder, so platforms
				// sharing a golden don't overwrite each other's.
				c := checker
				c.FailureDir = filepath.Join(failureDir, p.String())

				if err := c.Check(testrig.Golden{Name: name, Imported: tc.imported}, testrig.FromDisplay(m.Display())); err != nil {
					t.Error(err)
				}
			})
		}
	}
}

// TestSuiteBeep checks test 7 through the buzzer state: SOS starts by
// itself, and holding B gives manual control.
func TestSuiteBeep(t *testing.T) {
	t.Parallel()

	for _, p := range chip8Platforms {
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
