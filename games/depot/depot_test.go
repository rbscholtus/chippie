// Package depot_test runs Depot (games/depot/depot.8o, built into
// assets/roms/depot.ch8) headless and checks its screens and state
// (docs/plans/M9-game.md §4).
package depot_test

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"

	"github.com/rbscholtus/chippie/assets"
	"github.com/rbscholtus/chippie/internal/chip8"
	"github.com/rbscholtus/chippie/internal/testrig"
)

var update = flag.Bool("update", false, "rewrite generated golden files")

// Paths from this package's directory.
const (
	goldenDir  = "testdata/golden"
	failureDir = "testdata/failures"
)

// The game runs at 500 ticks per frame (design §2.4).
const ticksPerFrame = 500

// testSeed is the CXNN seed for every run.
const testSeed = 0xDE90

// Addresses in the state block (depot.8o, "State block").
const (
	addrFrame  = 0x202 // 16 bits, low byte first
	addrMode   = 0x204
	addrRound  = 0x205
	addrLives  = 0x206
	addrScore  = 0x207 // four digits, most significant first
	addrX      = 0x20B
	addrRow    = 0x20C
	addrLit    = 0x20D // the lit hatch, 0-3
	addrFilled = 0x20E // filled hatches, one bit each
	addrTimer  = 0x210 // frames left in a pause
	addrGhost  = 0x211 // not 0: nothing hits the courier
	addrLate   = 0x212 // frames whose work ran over into the next
	addrTime   = 0x213 // time-bar pixels left
	addrLanes  = 0x218 // the lane table: laneSize bytes per row, from row 1
	addrHi     = 0x25E // best score, four digits like addrScore
	addrNewBst = 0x262 // not 0: the game just over set a new best
)

// Lane records (depot.8o, "Lanes"): y, flags, speed, acc, sprite, count,
// x0-x3.
const (
	laneSize   = 10
	laneCount  = 5
	laneX      = 6
	rowChute3  = 3
	rowWalkway = 4
)

// laneIndex is each row's record in the lane table, which keeps the
// walkway (row 4) last.
var laneIndex = map[int]uint16{1: 0, 2: 1, 3: 2, 4: 6, 5: 3, 6: 4, 7: 5}

// laneAddr returns the address of field off in row's lane record.
func laneAddr(row int, off uint16) uint16 {
	return addrLanes + laneIndex[row]*laneSize + off
}

// quietYard empties the three yard lanes, so the courier can walk to the
// chute; more pokes can follow.
func quietYard(more ...testrig.Poke) []testrig.Poke {
	p := make([]testrig.Poke, 0, 3+len(more))
	for row := 5; row <= 7; row++ {
		p = append(p, testrig.Poke{Addr: laneAddr(row, laneCount), Value: 0})
	}

	return append(p, more...)
}

// Modes (s-mode).
const (
	modeTitle   = 0
	modePlay    = 1
	modeDying   = 2
	modeOver    = 3
	modeDeliver = 4
	modeStart   = 5
)

// ghost makes the courier un-hittable, for tests about movement.
var ghost = []testrig.Poke{{Addr: addrGhost, Value: 1}}

// Keys (depot.8o, "Keys").
const (
	keyUp    = 0x5
	keyLeft  = 0x7
	keyDown  = 0x8
	keyRight = 0x9
	keyStart = 0x6
)

// The courier's start: x 60 on the dock (row 8). Hops are 4 px or one row.
const (
	startX   = 60
	startRow = 8
	maxX     = 120
	topRow   = 1 // the hatch row (0) can't be entered yet
)

// The title takes frames 0-2 to draw and takes the start key from frame
// 3; the board then takes 4 frames. run presses start at startPress, so
// the game is playing by frame begin, the first frame the tests press keys
// in.
const (
	startPress = 3
	begin      = 10
)

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

// run starts a game (it presses start on the title screen) and runs s.
func run(t *testing.T, s testrig.Script) *chip8.Machine {
	t.Helper()

	s.Keys = append([]testrig.KeyEvent{tap(keyStart, startPress)}, s.Keys...)

	return runTitle(t, s)
}

// runTitle runs s from power-on, without pressing start.
func runTitle(t *testing.T, s testrig.Script) *chip8.Machine {
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

// best reads the four best-score digits as a number.
func best(m *chip8.Machine) int {
	n := 0
	for i := range uint16(4) {
		n = n*10 + int(m.Peek(addrHi+i))
	}

	return n
}

// courier returns the courier's x and row.
func courier(m *chip8.Machine) (int, int) {
	return int(m.Peek(addrX)), int(m.Peek(addrRow))
}

func TestPlayfield(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Frames: begin})
	check(t, m, "playfield")

	for _, s := range []struct {
		name string
		addr uint16
		want byte
	}{
		{"mode", addrMode, modePlay},
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

	budget(t, testrig.Script{}, warmUp, n)
}

// budget runs s for warmUp frames of start-up and then n more, and checks
// the game took none of them late or dropped any: its frame counter went up
// by n and no frame's work ran over into the next.
func budget(t *testing.T, s testrig.Script, warmUp, n int) *chip8.Machine {
	t.Helper()

	s.Frames = warmUp
	before := run(t, s)
	s.Frames = warmUp + n
	after := run(t, s)

	// The game's frame counter is 16 bits: it wraps every 65536 frames.
	const counterMask = 0xFFFF
	if got := (frames(after) - frames(before)) & counterMask; got != n&counterMask {
		t.Errorf("game counted %d frames in %d (mod 65536): it dropped %d", got, n&counterMask, n&counterMask-got)
	}

	if got := after.Peek(addrLate) - before.Peek(addrLate); got != 0 {
		t.Errorf("%d frames ran late", got)
	}

	return after
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

	budget(t, testrig.Script{Keys: keys}, warmUp, n)
}

// noneLate notes the game's late-frame count; the function it returns
// checks no more frames ran late since (the frame budget, for paths the
// budget tests don't take).
func noneLate(t *testing.T, m *chip8.Machine) func() {
	t.Helper()

	before := m.Peek(addrLate)

	return func() {
		t.Helper()

		if got := m.Peek(addrLate) - before; got != 0 {
			t.Errorf("%d frames ran late", got)
		}
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

// press presses key for two frames and lets go for two, so the next press
// is a new hop.
func press(t *testing.T, m *chip8.Machine, key uint8) {
	t.Helper()

	for i := range 4 {
		m.SetKey(key, i < 2)
		m.RunFrame()
	}

	if err := m.State().Err; err != nil {
		t.Fatal(err)
	}
}

// TestHitInTheYard hops into yard 3 and waits for a forklift.
func TestHitInTheYard(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Keys: []testrig.KeyEvent{tap(keyUp, begin)}, Frames: begin + 3})
	if _, row := courier(m); row != startRow-1 {
		t.Fatalf("courier in row %d, want %d", row, startRow-1)
	}

	until(t, m, 600, func() bool { return m.Peek(addrMode) == modeDying })
	checkSound(t, m, "hit", patternNoise, pitchHit)
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

// TestDockIsSafe waits 15 s on the dock: no traffic comes, and the time
// bar (16 s) hasn't run out.
func TestDockIsSafe(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Frames: 900})
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

	for range 4 { // the text takes three frames
		m.RunFrame()
	}

	check(t, m, "game-over") // 30 points is a new best: NEW BEST shows

	startGame(t, m)

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

// TestLaneTable checks the tests' idea of where the lane table is.
func TestLaneTable(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Frames: 1})
	for row := 1; row <= 7; row++ {
		if got, want := m.Peek(laneAddr(row, 0)), byte(6*row+6); got != want {
			t.Errorf("row %d lane y = %d, want %d", row, got, want)
		}
	}
}

// TestFallInTheChute walks over a quiet yard into chute 3 with no pallets.
func TestFallInTheChute(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{
		Pokes:  quietYard(testrig.Poke{Addr: laneAddr(rowChute3, laneCount), Value: 0}),
		Frames: begin,
	})

	for range startRow - rowChute3 {
		press(t, m, keyUp)
	}

	until(t, m, 10, func() bool { return m.Peek(addrMode) == modeDying })
	checkSound(t, m, "fall", patternNoise, pitchFall)
	until(t, m, 60, func() bool { return m.Peek(addrMode) == modePlay })

	if got := m.Peek(addrLives); got != 2 {
		t.Errorf("lives %d, want 2", got)
	}
}

// onPallet walks over a quiet yard to the walkway, waits for a pallet in
// chute 3 to come under the courier, and hops onto it. It returns which
// pallet.
func onPallet(t *testing.T) (*chip8.Machine, uint16) {
	t.Helper()

	m := run(t, testrig.Script{Pokes: quietYard(), Frames: begin})
	for range startRow - rowWalkway {
		press(t, m, keyUp)
	}

	// A pallet's deck is x+1 to x+14 and the courier is 5 px wide; the
	// pallet moves left while the hop is pressed, so leave it a margin.
	var pallet uint16

	until(t, m, 600, func() bool {
		cx := int(m.Peek(addrX))
		for i := range uint16(3) {
			x := int(m.Peek(laneAddr(rowChute3, laneX+i)))
			if x+3 <= cx && cx+5 <= x+12 {
				pallet = i

				return true
			}
		}

		return false
	})

	m.SetKey(keyUp, true)
	m.RunFrame()
	m.SetKey(keyUp, false)
	m.RunFrame()

	if _, row := courier(m); row != rowChute3 {
		t.Fatalf("courier in row %d, want %d", row, rowChute3)
	}

	return m, pallet
}

// TestRidePallet rides a pallet for a second: the courier moves with it.
func TestRidePallet(t *testing.T) {
	t.Parallel()

	m, pallet := onPallet(t)
	palletX := func() int { return int(m.Peek(laneAddr(rowChute3, laneX+pallet))) }

	x0, _ := courier(m)
	offset := x0 - palletX()

	for range 60 {
		m.RunFrame()
	}

	if got := m.Peek(addrMode); got != modePlay {
		t.Fatalf("mode %d after riding, want playing (%d)", got, modePlay)
	}

	x, _ := courier(m)
	if x >= x0 {
		t.Errorf("courier at x %d after riding a pallet left from %d", x, x0)
	}

	if got := x - palletX(); got != offset {
		t.Errorf("courier %d px from its pallet after riding, was %d", got, offset)
	}

	check(t, m, "riding")
}

// TestCarriedOffTheEdge rides a pallet until it reaches the left edge.
func TestCarriedOffTheEdge(t *testing.T) {
	t.Parallel()

	m, _ := onPallet(t)
	until(t, m, 600, func() bool { return m.Peek(addrMode) == modeDying })

	if x, _ := courier(m); x != 0 {
		t.Errorf("courier fell at x %d, want the edge (0)", x)
	}

	until(t, m, 60, func() bool { return m.Peek(addrMode) == modePlay })

	if got := m.Peek(addrLives); got != 2 {
		t.Errorf("lives %d, want 2", got)
	}
}

// Hatches (depot.8o, hatch-xs). The courier delivers from x = hatch x + 4,
// which a hop from the start (x 60) can reach.
var hatchX = [4]int{8, 40, 72, 104}

// Time bar (depot.8o, "Time bar").
const (
	timeFull   = 32
	tickFrames = 30
)

// walkTo hops the courier sideways to x, a multiple of 4 away.
func walkTo(t *testing.T, m *chip8.Machine, x int) {
	t.Helper()

	for {
		cx, _ := courier(m)

		switch {
		case cx < x:
			press(t, m, keyRight)
		case cx > x:
			press(t, m, keyLeft)
		default:
			return
		}
	}
}

// crossTo walks a ghost courier from the dock to x and hops up to the
// hatch wall, ending with the hop into it: it returns whether that hop
// delivered, and the time left when it did.
func crossTo(t *testing.T, m *chip8.Machine, x int) (bool, int) {
	t.Helper()

	walkTo(t, m, x)

	for range startRow - topRow {
		press(t, m, keyUp)
	}

	m.SetKey(keyUp, true)
	m.RunFrame()

	delivered, left := m.Peek(addrMode) == modeDeliver, int(m.Peek(addrTime))

	m.SetKey(keyUp, false)
	m.RunFrame()

	return delivered, left
}

// deliver crosses a ghost courier to the lit hatch and waits for the next
// one on the dock. It returns the hatch and the time left at delivery.
func deliver(t *testing.T, m *chip8.Machine) (int, int) {
	t.Helper()

	lit := int(m.Peek(addrLit))

	delivered, left := crossTo(t, m, hatchX[lit]+4)
	if !delivered {
		t.Fatalf("hop into lit hatch %d didn't deliver", lit)
	}

	checkSound(t, m, "delivery", patternSquare, pitchChime)

	until(t, m, 60, func() bool { return m.Peek(addrMode) == modePlay })

	return lit, left
}

// TestTimeBar checks a pixel goes every half second.
func TestTimeBar(t *testing.T) {
	t.Parallel()

	before := run(t, testrig.Script{Frames: begin})
	after := run(t, testrig.Script{Frames: begin + 10*tickFrames})

	if got, want := after.Peek(addrTime), before.Peek(addrTime)-10; got != want {
		t.Errorf("time bar %d px after 5 s, want %d", got, want)
	}
}

// TestTimeOut waits on the dock until the time bar runs out.
func TestTimeOut(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Frames: begin})
	start := frames(m)

	until(t, m, timeFull*tickFrames, func() bool { return m.Peek(addrTime) == tickFrom-1 })
	checkSound(t, m, "time bar tick", patternNoise, pitchTick)

	until(t, m, timeFull*tickFrames+tickFrames, func() bool { return m.Peek(addrMode) == modeDying })

	if got, want := frames(m)-start, timeFull*tickFrames-begin; got < want {
		t.Errorf("time ran out after %d frames, want at least %d", got, want)
	}

	until(t, m, 60, func() bool { return m.Peek(addrMode) == modePlay })

	if got := m.Peek(addrLives); got != 2 {
		t.Errorf("lives %d, want 2", got)
	}

	if got := m.Peek(addrTime); got != timeFull {
		t.Errorf("time bar %d px for the next courier, want it refilled to %d", got, timeFull)
	}
}

// TestDelivery crosses to the lit hatch: it's filled, scored and a new one
// lit (golden).
func TestDelivery(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Pokes: ghost, Frames: begin})
	lit, left := deliver(t, m)

	if got, want := m.Peek(addrFilled), byte(1)<<lit; got != want {
		t.Errorf("filled hatches %04b, want %04b", got, want)
	}

	if got := int(m.Peek(addrLit)); got == lit {
		t.Errorf("hatch %d still lit after its delivery", got)
	}

	if want := 10*(startRow-topRow) + 50 + left; score(m) != want {
		t.Errorf("score %d, want %d (rows, delivery, %d px of time)", score(m), want, left)
	}

	if x, row := courier(m); x != startX || row != startRow {
		t.Errorf("courier at x %d, row %d; want the next one on the dock", x, row)
	}

	if got := m.Peek(addrTime); got != timeFull {
		t.Errorf("time bar %d px, want it refilled to %d", got, timeFull)
	}

	check(t, m, "delivered")
}

// TestUnlitHatchBlocks hops up into a hatch that isn't lit.
func TestUnlitHatchBlocks(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Pokes: ghost, Frames: begin})
	other := (int(m.Peek(addrLit)) + 1) % 4

	if delivered, _ := crossTo(t, m, hatchX[other]+4); delivered {
		t.Fatalf("unlit hatch %d took a delivery", other)
	}

	if _, row := courier(m); row != topRow {
		t.Errorf("courier in row %d, want stopped in row %d", row, topRow)
	}
}

// TestRoundClear fills all four hatches: round 2 starts, faster.
func TestRoundClear(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Pokes: ghost, Frames: begin})
	speed := m.Peek(laneAddr(1, 2))
	want := 200

	defer noneLate(t, m)()

	for range 4 {
		_, left := deliver(t, m)
		want += 10*(startRow-topRow) + 50 + left
	}

	if got := m.Peek(addrRound); got != 2 {
		t.Errorf("round %d, want 2", got)
	}

	if got := m.Peek(addrFilled); got != 0 {
		t.Errorf("filled hatches %04b in the new round, want none", got)
	}

	if got := score(m); got != want {
		t.Errorf("score %d, want %d", got, want)
	}

	if got := m.Peek(laneAddr(1, 2)); got != speed+2 {
		t.Errorf("chute 1 speed %d in round 2, want %d", got, speed+2)
	}
}

// TestLitHatchMoves waits half the time bar in round 3: the light moves.
func TestLitHatchMoves(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Pokes: ghost, Frames: begin})
	m.Poke(addrRound, 3)

	lit := m.Peek(addrLit)
	until(t, m, timeFull*tickFrames, func() bool { return m.Peek(addrTime) == timeFull/2 })

	if got := m.Peek(addrLit); got == lit {
		t.Errorf("hatch %d still lit half way through round 3", got)
	}
}

// TestNewGameClearsTheBoard delivers a parcel, loses every life, and checks
// the new game's HUD and hatches look like a fresh start's: no parcels,
// one light, score 0, three lives, round 1, a full time bar.
func TestNewGameClearsTheBoard(t *testing.T) {
	t.Parallel()

	fresh := run(t, testrig.Script{Frames: begin}).Display()

	m := run(t, testrig.Script{Pokes: ghost, Frames: begin})
	defer noneLate(t, m)()

	deliver(t, m)
	m.Poke(addrGhost, 0)

	for range 3 {
		until(t, m, 120, func() bool { return m.Peek(addrMode) == modePlay })
		press(t, m, keyUp) // into yard 3, and wait for a forklift
		until(t, m, 600, func() bool { return m.Peek(addrMode) == modeDying })
	}

	until(t, m, 120, func() bool { return m.Peek(addrMode) == modeOver })
	startGame(t, m)

	const hatchRows = 12 // the HUD and the hatch wall

	d := m.Display()
	lit := hatchX[m.Peek(addrLit)]
	light := 0

	for y := range hatchRows {
		for x := range d.Width() {
			if got, want := d.Pixel(x, y)&1, fresh.Pixel(x, y)&1; got != want {
				t.Fatalf("plane 1 at (%d, %d) is %d in the new game, %d at a fresh start", x, y, got, want)
			}

			if d.Pixel(x, y)&2 != 0 {
				light++

				if x <= lit || x >= lit+7 {
					t.Errorf("plane 2 at (%d, %d), outside lit hatch at x %d", x, y, lit)
				}
			}
		}
	}

	if light != 6*5 {
		t.Errorf("%d px of light in the hatch wall, want one hatch's (30)", light)
	}
}

// startGame waits for the title screen after a game, presses start and
// waits for the new game.
func startGame(t *testing.T, m *chip8.Machine) {
	t.Helper()

	until(t, m, 300, func() bool { return m.Peek(addrMode) == modeTitle && m.Peek(addrTimer) == 0 })
	press(t, m, keyStart)
	until(t, m, 10, func() bool { return m.Peek(addrMode) == modePlay })
}

// TestTitle checks the title screen at power-on: it waits for the start
// key, with no best score yet, and is silent (golden).
func TestTitle(t *testing.T) {
	t.Parallel()

	m := runTitle(t, testrig.Script{Frames: 20}) // before the prompt first blinks off

	if got := m.Peek(addrMode); got != modeTitle {
		t.Errorf("mode %d, want the title (%d)", got, modeTitle)
	}

	if got := best(m); got != 0 {
		t.Errorf("best score %d with nothing saved, want 0", got)
	}

	if m.Sound().Buzzer {
		t.Error("buzzer on at the title")
	}

	check(t, m, "title")

	m = runTitle(t, testrig.Script{Keys: []testrig.KeyEvent{tap(keyStart, startPress)}, Frames: begin})
	if got := m.Peek(addrMode); got != modePlay {
		t.Errorf("mode %d after start, want playing (%d)", got, modePlay)
	}
}

// loseAll loses every life, by hopping into yard 3 and waiting, and waits
// for game over to show its text.
func loseAll(t *testing.T, m *chip8.Machine) {
	t.Helper()

	m.Poke(addrGhost, 0)

	for m.Peek(addrLives) > 0 {
		until(t, m, 120, func() bool { return m.Peek(addrMode) == modePlay })
		press(t, m, keyUp)
		until(t, m, 600, func() bool { return m.Peek(addrMode) == modeDying })
		until(t, m, 120, func() bool { return m.Peek(addrMode) != modeDying })
	}

	until(t, m, 10, func() bool { return m.Peek(addrMode) == modeOver })

	for range 4 {
		m.RunFrame()
	}
}

// TestBestScore plays two games: the first sets the best score, shown on
// the title; the second scores less and leaves it.
func TestBestScore(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Pokes: ghost, Frames: begin})
	defer noneLate(t, m)()

	deliver(t, m)
	loseAll(t, m)

	first := score(m)
	if got := best(m); got != first {
		t.Errorf("best score %d after the first game, want its score %d", got, first)
	}

	if m.Peek(addrNewBst) == 0 {
		t.Error("first game's score isn't marked a new best")
	}

	until(t, m, 300, func() bool { return m.Peek(addrMode) == modeTitle && m.Peek(addrTimer) == 0 })
	check(t, m, "title-best")

	press(t, m, keyStart)
	until(t, m, 10, func() bool { return m.Peek(addrMode) == modePlay })
	loseAll(t, m)

	if got := score(m); got >= first {
		t.Fatalf("second game scored %d, want less than %d", got, first)
	}

	if got := best(m); got != first {
		t.Errorf("best score %d after a lower score, want %d kept", got, first)
	}

	if m.Peek(addrNewBst) != 0 {
		t.Error("lower score marked a new best")
	}
}

// Sounds (depot.8o, "Sounds"): the pattern each starts with, by its first
// byte, and its pitch.
const (
	patternSquare = 0xFF
	patternNoise  = 0xA5
	pitchHop      = 200
	pitchTick     = 200
	pitchHit      = 120
	pitchFall     = 150
	pitchChime    = 143
	tickFrom      = 7 // the time bar ticks from 6 px left
)

// checkSound checks the sound that started this frame.
func checkSound(t *testing.T, m *chip8.Machine, what string, pattern, pitch byte) {
	t.Helper()

	s := m.Sound()
	if !s.Buzzer || !s.HasPattern || s.Pattern[0] != pattern || s.Pitch != pitch {
		t.Errorf("%s: buzzer %v, pattern %v from %02X, pitch %d; want on, %02X, %d",
			what, s.Buzzer, s.HasPattern, s.Pattern[0], s.Pitch, pattern, pitch)
	}
}

// TestHopSound hops once: a click, which stops after two frames.
func TestHopSound(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Pokes: ghost, Keys: []testrig.KeyEvent{tap(keyUp, begin)}, Frames: begin + 1})
	checkSound(t, m, "hop", patternSquare, pitchHop)

	for range 3 {
		m.RunFrame()
	}

	if m.Sound().Buzzer {
		t.Error("hop still sounding after 3 frames")
	}
}

// TestBestScoreSurvivesAReload sets a best score, then starts the ROM again
// on a new machine with the flag registers the app saved (as Chippie does
// after a reload, internal/app/flags.go): the title shows the same best score.
func TestBestScoreSurvivesAReload(t *testing.T) {
	t.Parallel()

	m := run(t, testrig.Script{Pokes: ghost, Frames: begin})
	deliver(t, m)
	loseAll(t, m)

	want := best(m)
	if want == 0 {
		t.Fatal("no best score to save")
	}

	flags := m.Flags()
	if flags[0] != 0xD3 || flags[1] != 0x90 {
		t.Fatalf("flags % X, want the D3 90 marker first", flags)
	}

	again := chip8.New(profile())
	if err := again.LoadROM(loadROM(t)); err != nil {
		t.Fatal(err)
	}

	again.SetFlags(flags)

	for range begin {
		again.RunFrame()
	}

	if got := best(again); got != want {
		t.Errorf("best score %d after a reload, want %d", got, want)
	}

	if got := again.Peek(addrMode); got != modeTitle {
		t.Errorf("mode %d after a reload, want the title", got)
	}
}

// randomKeys returns n frames of random play from frame from: a tap of a
// random key (start included, so games end and start again) every 2 to 20
// frames, from a fixed seed.
func randomKeys(from, n int) []testrig.KeyEvent {
	r := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // test input, not security
	keys := []uint8{keyUp, keyUp, keyDown, keyLeft, keyRight, keyStart}

	var out []testrig.KeyEvent
	for f := from; f < from+n; f += 2 + r.IntN(19) {
		out = append(out, tap(keys[r.IntN(len(keys))], f))
	}

	return out
}

// TestFrameBudgetForAnHour plays randomly for an hour of game time (design
// §4): no frame may run late or be dropped, through deaths, game overs,
// titles and new games. It takes ~5 s; with -race (CI), whose emulator is
// ~15 times slower and adds nothing for one goroutine, it plays 10
// minutes. Skipped with -short.
func TestFrameBudgetForAnHour(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("an hour of game time")
	}

	const warmUp = 60

	n := 60 * 60 * 60
	if raceDetector {
		n = 10 * 60 * 60
	}

	m := budget(t, testrig.Script{Keys: randomKeys(warmUp, n)}, warmUp, n)

	// The random player dies a lot: games ended and a best score was set.
	if best(m) == 0 {
		t.Error("no best score after an hour: the run never reached game over")
	}

	if err := m.State().Err; err != nil {
		t.Errorf("halted: %v", err)
	}
}

// TestSameGameAt1000Ticks runs the same random minute at 500 and at 1000
// instructions a frame: the game is locked to the 60 Hz timer, so it plays
// exactly the same (design §4).
func TestSameGameAt1000Ticks(t *testing.T) {
	t.Parallel()

	s := testrig.Script{Keys: randomKeys(begin, 3600), Frames: begin + 3600}
	s.Keys = append([]testrig.KeyEvent{tap(keyStart, startPress)}, s.Keys...)

	at := func(ticks int) *chip8.Machine {
		p := profile()
		p.TicksPerFrame = ticks

		m, err := testrig.Run(loadROM(t), p, s)
		if err != nil {
			t.Fatal(err)
		}

		return m
	}

	normal, fast := at(ticksPerFrame), at(2*ticksPerFrame)

	// The state block and the lane table, up to the sound state.
	for addr := uint16(addrFrame); addr < addrHi+8; addr++ {
		if addr == addrLate {
			continue // the fast run can't be late; the normal one isn't either
		}

		if a, b := normal.Peek(addr), fast.Peek(addr); a != b {
			t.Errorf("%03X: %d at %d ticks, %d at %d", addr, a, ticksPerFrame, b, 2*ticksPerFrame)
		}
	}

	// The same golden for both compares the whole screen.
	check(t, fast, "random-minute")
	check(t, normal, "random-minute")
}
