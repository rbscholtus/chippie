// Package depot_test runs Depot (games/depot/depot.8o, built into
// assets/roms/depot.ch8) headless and checks its screens and state
// (docs/plans/M9-game.md §4).
package depot_test

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/rbscholtus/chippie/assets"
	"github.com/rbscholtus/chippie/chip8"
	"github.com/rbscholtus/chippie/internal/testrig"
)

var update = flag.Bool("update", false, "rewrite generated golden files")

// Paths from this package's directory.
const (
	goldenDir  = "testdata/golden"
	failureDir = "../../testdata/failures/depot"
)

// The game runs at 500 ticks per frame (design §2.4).
const ticksPerFrame = 500

// testSeed is the CXNN seed for every run.
const testSeed = 0xDE90

// Addresses in the state block (depot.8o, "State block").
const (
	addrFrame = 0x202 // 16 bits, low byte first
	addrMode  = 0x204
	addrRound = 0x205
	addrLives = 0x206
	addrScore = 0x207 // four digits, most significant first
	addrX     = 0x20B
	addrRow   = 0x20C
	addrGhost = 0x211 // not 0: nothing hits the courier
)

// Modes (s-mode).
const (
	modePlay  = 1
	modeDying = 2
	modeOver  = 3
)

// ghost makes the courier un-hittable, for tests about movement.
var ghost = []testrig.Poke{{Addr: addrGhost, Value: 1}}

// Keys (depot.8o, "Keys").
const (
	keyUp    = 0x5
	keyLeft  = 0x7
	keyDown  = 0x8
	keyRight = 0x9
)

// The courier's start: x 60 on the dock (row 8). Hops are 4 px or one row.
const (
	startX   = 60
	startRow = 8
	maxX     = 120
	topRow   = 1 // the hatch row (0) can't be entered yet
)

// begin is the first frame the tests press keys in, after start-up.
const begin = 10

// tap presses key at frame f for two frames.
func tap(key uint8, f int) testrig.KeyEvent {
	return testrig.KeyEvent{Key: key, Press: f, Release: f + 2}
}

// hold holds key from frame begin for n frames.
func hold(key uint8, n int) testrig.KeyEvent {
	return testrig.KeyEvent{Key: key, Press: begin, Release: begin + n}
}

func TestMain(m *testing.M) {
	flag.Parse()

	if err := testrig.ClearFailures(failureDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func loadROM(t *testing.T) []byte {
	t.Helper()

	rom, err := assets.ROMs.ReadFile("roms/depot.ch8")
	if err != nil {
		t.Fatalf("read ROM (run scripts/build-games.sh): %v", err)
	}

	return rom
}

func profile() chip8.Profile {
	p := chip8.DefaultProfile(chip8.XOChip)
	p.TicksPerFrame = ticksPerFrame
	p.Seed = testSeed

	return p
}

func run(t *testing.T, s testrig.Script) *chip8.Machine {
	t.Helper()

	m, err := testrig.Run(loadROM(t), profile(), s)
	if err != nil {
		t.Fatal(err)
	}

	return m
}

func check(t *testing.T, m *chip8.Machine, golden string) {
	t.Helper()

	c := testrig.Checker{GoldenDir: goldenDir, FailureDir: failureDir, Update: *update}
	if err := c.Check(testrig.Golden{Name: golden}, testrig.FromDisplay(m.Display())); err != nil {
		t.Error(err)
	}
}

func frames(m *chip8.Machine) int {
	return int(m.Peek(addrFrame)) | int(m.Peek(addrFrame+1))<<8
}

// score reads the four score digits as a number.
func score(m *chip8.Machine) int {
	n := 0
	for i := range uint16(4) {
		n = n*10 + int(m.Peek(addrScore+i))
	}

	return n
}

// courier returns the courier's x and row.
func courier(m *chip8.Machine) (int, int) {
	return int(m.Peek(addrX)), int(m.Peek(addrRow))
}

func TestPlayfield(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Frames: 2})
	check(t, m, "playfield")

	for _, s := range []struct {
		name string
		addr uint16
		want byte
	}{
		{"mode", addrMode, modePlay}, // the title screen comes in step 6
		{"round", addrRound, 1},
		{"lives", addrLives, 3},
		{"x", addrX, 60},
		{"row", addrRow, 8},
	} {
		if got := m.Peek(s.addr); got != s.want {
			t.Errorf("%s = %d, want %d", s.name, got, s.want)
		}
	}
}

// TestFrameBudget checks that the game never drops a frame: after start-up
// (which may take more than a frame), its frame counter must go up by one
// every frame (design §2.1). Runs are deterministic, so a short and a long
// run share their first frames.
func TestFrameBudget(t *testing.T) {
	t.Parallel()

	const (
		warmUp = 60
		n      = 3600 // one minute
	)

	start := frames(run(t, testrig.Script{Frames: warmUp}))
	end := frames(run(t, testrig.Script{Frames: warmUp + n}))

	if got := end - start; got != n {
		t.Errorf("game counted %d frames in %d: it dropped %d", got, n, n-got)
	}
}

func TestMoves(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name         string
		keys         []testrig.KeyEvent
		x, row, scor int
	}{
		{"nothing pressed", nil, startX, startRow, 0},
		{"up scores a row", []testrig.KeyEvent{tap(keyUp, begin)}, startX, startRow - 1, 10},
		{"left", []testrig.KeyEvent{tap(keyLeft, begin)}, startX - 4, startRow, 0},
		{"right", []testrig.KeyEvent{tap(keyRight, begin)}, startX + 4, startRow, 0},
		{"down stops at the dock", []testrig.KeyEvent{tap(keyDown, begin)}, startX, startRow, 0},
		{
			"three taps, three hops",
			[]testrig.KeyEvent{tap(keyUp, begin), tap(keyUp, begin+4), tap(keyUp, begin+8)},
			startX, startRow - 3, 30,
		},
		{
			// Hops at frames 0, 12, 18, 24 and 30 of the hold.
			"held key repeats",
			[]testrig.KeyEvent{hold(keyLeft, 31)},
			startX - 5*4, startRow, 0,
		},
		{
			"a row scores once per crossing",
			[]testrig.KeyEvent{tap(keyUp, begin), tap(keyDown, begin+4), tap(keyUp, begin+8)},
			startX, startRow - 1, 10,
		},
		{
			"up stops below the hatches",
			[]testrig.KeyEvent{hold(keyUp, 300)},
			startX, topRow, 10 * (startRow - topRow),
		},
		{"left stops at the edge", []testrig.KeyEvent{hold(keyLeft, 300)}, 0, startRow, 0},
		{"right stops at the edge", []testrig.KeyEvent{hold(keyRight, 300)}, maxX, startRow, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := run(t, testrig.Script{Pokes: ghost, Keys: tc.keys, Frames: begin + 320})

			if x, row := courier(m); x != tc.x || row != tc.row {
				t.Errorf("courier at x %d, row %d; want x %d, row %d", x, row, tc.x, tc.row)
			}

			if got := score(m); got != tc.scor {
				t.Errorf("score %d, want %d", got, tc.scor)
			}
		})
	}
}

// TestMovedScreen checks the courier and the score are drawn where they
// should be, and erased where they were.
func TestMovedScreen(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{
		Pokes: ghost,
		Keys: []testrig.KeyEvent{
			hold(keyUp, 13), // two hops: rows 7 and 6
			tap(keyRight, begin+20),
			tap(keyRight, begin+24),
		},
		Frames: begin + 40,
	})
	check(t, m, "moved")
}

// TestFrameBudgetWhileMoving is the frame budget with the courier hopping
// around all minute.
func TestFrameBudgetWhileMoving(t *testing.T) {
	t.Parallel()

	const (
		warmUp = 60
		n      = 3600
	)

	var keys []testrig.KeyEvent
	for f, k := warmUp, 0; f < warmUp+n; f, k = f+3, k+1 {
		keys = append(keys, tap([]uint8{keyUp, keyRight, keyDown, keyLeft, keyUp}[k%5], f))
	}

	start := frames(run(t, testrig.Script{Keys: keys, Frames: warmUp}))
	end := frames(run(t, testrig.Script{Keys: keys, Frames: warmUp + n}))

	if got := end - start; got != n {
		t.Errorf("game counted %d frames in %d: it dropped %d", got, n, n-got)
	}
}

// until runs the machine a frame at a time, keeping its keys, until done
// reports true, for at most limit frames.
func until(t *testing.T, m *chip8.Machine, limit int, done func() bool) {
	t.Helper()

	for range limit {
		if done() {
			return
		}

		m.RunFrame()

		if err := m.State().Err; err != nil {
			t.Fatal(err)
		}
	}

	t.Fatalf("not done after %d frames", limit)
}

// TestHitInTheYard hops into yard 3 and waits for a forklift.
func TestHitInTheYard(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Keys: []testrig.KeyEvent{tap(keyUp, begin)}, Frames: begin + 3})
	if _, row := courier(m); row != startRow-1 {
		t.Fatalf("courier in row %d, want %d", row, startRow-1)
	}

	until(t, m, 600, func() bool { return m.Peek(addrMode) == modeDying })
	until(t, m, 60, func() bool { return m.Peek(addrMode) == modePlay })

	if got := m.Peek(addrLives); got != 2 {
		t.Errorf("lives %d, want 2", got)
	}

	if x, row := courier(m); x != startX || row != startRow {
		t.Errorf("courier at x %d, row %d after a hit; want back on the dock at x %d", x, row, startX)
	}

	if got := score(m); got != 10 {
		t.Errorf("score %d, want the 10 from before the hit", got)
	}
}

// TestDockIsSafe waits a minute on the dock.
func TestDockIsSafe(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Frames: 3600})
	if got := m.Peek(addrLives); got != 3 {
		t.Errorf("lives %d, want 3", got)
	}
}

// TestGameOver hops into yard 3 with every life and waits there, until
// the lives run out; then a new game starts.
func TestGameOver(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Frames: begin})

	for lives := byte(3); lives > 0; lives-- {
		until(t, m, 120, func() bool { return m.Peek(addrMode) == modePlay })
		m.SetKey(keyUp, true)
		m.RunFrame()
		m.RunFrame()
		m.SetKey(keyUp, false)

		until(t, m, 600, func() bool { return m.Peek(addrMode) == modeDying })

		if got := m.Peek(addrLives); got != lives {
			t.Fatalf("lives %d while dying, want %d", got, lives)
		}
	}

	until(t, m, 60, func() bool { return m.Peek(addrMode) == modeOver })

	if got := m.Peek(addrLives); got != 0 {
		t.Errorf("lives %d at game over, want 0", got)
	}

	check(t, m, "game-over")

	until(t, m, 200, func() bool { return m.Peek(addrMode) == modePlay })

	if got := m.Peek(addrLives); got != 3 {
		t.Errorf("lives %d in the new game, want 3", got)
	}

	if got := score(m); got != 0 {
		t.Errorf("score %d in the new game, want 0", got)
	}
}

// TestTraffic checks the lanes have moved after a second (golden).
func TestTraffic(t *testing.T) {
	t.Parallel()

	check(t, run(t, testrig.Script{Frames: 60}), "traffic")
}
