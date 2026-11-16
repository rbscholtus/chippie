# Chippie: design for M9, our own game

## Context

Chippie plays other people's ROMs. M9 adds one of our own: an original
XO-CHIP game written in Octo, built from source in this repository, and
bundled with the emulator like the hosted ROMs (DESIGN §9).

Decisions made so far (2026-11-15):

- **Genre: lane crossing** (the mechanics of Frogger: traffic lanes, rideable
  platforms, goal slots at the top). Original theme, art, name and rules; no
  copied characters, levels or look.
- **Platform: XO-CHIP**, 128×64, two planes, pattern audio.
- **Language: Octo**, assembled with the `octo-assembler` npm package (the
  same tool the Timendus suite uses, `third_party/chip8-test-suite/package.json`).
- **Same repository.** Source under `games/`, built ROM bundled with the app.
- **Claude writes the game**; the user play-tests and decides.
- **Name: Depot.** **Licence: CC0** for everything under `games/`, so the
  game can also go into chip8Archive.
- **Speed: 500 ticks per frame.**
- **Two emulator features come with it:** arrow keys for ROMs with a `keys`
  hint (an idea in ROADMAP), and XO-CHIP flag registers saved between sessions, so
  the best score survives a reload.

Why this genre: simple to play (four keys, no aiming), hard to master
(reading gaps several lanes ahead, riding platforms), and the difficulty is
one number (lane speed), so there are no levels to design.

---

## 1. The game

**Depot.** You are a courier in a night-time
parcel depot. Cross the loading yard, ride the pallets over the sorting chute,
and deliver the parcel to the lit hatch at the top.

### 1.1 Screen layout (128×64)

Rows are 6 px high; sprites are 5 px high, so neighbouring lanes never touch.

| y | Row | Contents |
|---|---|---|
| 0–5 | HUD | score (4 digits, 4×5 font), lives, round, time bar |
| 6–11 | Hatches | wall with 4 hatches; one is lit (the target) |
| 12–17 | Chute 1 | pallets moving left |
| 18–23 | Chute 2 | pallets moving right |
| 24–29 | Chute 3 | pallets moving left |
| 30–35 | Walkway | safe |
| 36–41 | Yard 1 | forklifts moving right |
| 42–47 | Yard 2 | trolleys moving left |
| 48–53 | Yard 3 | forklifts moving right, faster |
| 54–63 | Dock | start, safe (10 px, with a floor pattern) |

```
┌────────────────────────────────────────────────────────────────┐
│ 0420  ♦♦  R2                               ████████████░░░░░░  │ HUD
│▓▓▓▓[  ]▓▓▓▓▓▓▓▓[##]▓▓▓▓▓▓▓▓[**]▓▓▓▓▓▓▓▓[  ]▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓│ hatches (** = lit)
│     ▭▭▭▭▭        ▭▭▭▭▭▭▭           ▭▭▭▭▭          ◄           │ chute 1
│  ►         ▭▭▭▭▭▭▭▭        ▭▭▭▭▭▭▭▭        ▭▭▭▭▭▭▭▭           │ chute 2
│      ▭▭▭▭▭▭        ▭▭▭▭▭▭        ▭▭▭▭▭▭          ◄            │ chute 3
│................................................................│ walkway
│   ►  ▣▣▷           ▣▣▷               ▣▣▷                       │ yard 1
│ ◁▢▢        ◁▢▢          ◁▢▢           ◁▢▢             ◄        │ yard 2
│   ►       ▣▣▷                 ▣▣▷                              │ yard 3
│                              ☺                                 │ dock
└────────────────────────────────────────────────────────────────┘
```

Lanes use the full width, because XO-CHIP sprites wrap at the screen edge:
an object leaving on the right comes back on the left for free.

### 1.2 Rules

- **Move** with four keys, one hop per press: one row up or down, or 4 px
  sideways. Held keys repeat after 12 frames, then every 6.
- **Yard lanes:** touching a forklift or trolley costs a life.
- **Chute lanes:** you must stand on a pallet, which carries you along.
  Standing on nothing, or being carried against the screen edge, costs a life.
- **Hatches:** the hatch row is a wall. Hopping up into the **lit** hatch
  delivers the parcel. (As built: the courier's middle must be over the
  hatch's 8 px, so two hop positions reach each one.) Every other spot, including the unlit hatches and
  filled ones, blocks the hop.
- **The twist:** only one hatch is lit at a time, picked at random from the
  empty ones after each delivery. You can't fill them left to right; you
  have to plan a route to wherever it is. From round 3, the lit hatch moves
  to another empty one if you take longer than 8 s.
- **Time bar:** 32 px, one pixel per 0.5 s (16 s per delivery). Running out
  costs a life. It refills after a delivery or a lost life.
- **Rounds:** fill all four hatches to clear the round. Each round, every
  lane's speed goes up by a fixed step, up to a cap.
- **Lives:** 3. Game over returns to the title screen, which shows the best
  score. The best score is kept in the flag registers (`saveflags` /
  `loadflags`, §2.5), so it survives a reload in Octo and in Chippie.

**Scoring:** 10 for each row reached for the first time on a crossing,
50 for a delivery, 1 for every pixel left on the time bar, 200 for clearing
a round.

### 1.3 Controls

Octo's usual keys, so the game works the same in Octo and other emulators:

| Action | CHIP-8 key | Chippie keyboard (default layout) |
|---|---|---|
| Up | 5 | W |
| Left | 7 | A |
| Down | 8 | S |
| Right | 9 | D |
| Start | 6 | E |

The overlay entry gets a `keys` hint (`up 5, left 7, down 8, right 9, a 6`),
so the remote highlights these keys on touch screens, and Chippie also maps
the arrow keys and Space to them (§3.1).

### 1.4 Colours and sound

- **Plane 1:** scenery, lanes and the HUD. **Plane 2:** the courier.
  Where the courier stands on a pallet the pixels overlap and show the third
  colour, so "I'm on it" is visible at a glance.
- **Palette:** in the overlay's `colors` field, e.g. dark blue-grey
  background, amber scenery, white courier, orange overlap.
- **Sound (XO-CHIP audio patterns and pitch):**

| Event | Sound |
|---|---|
| Hop | short click, high pitch, 2 frames |
| Fall into the chute | falling-pitch noise |
| Hit by a forklift | noise burst |
| Delivery | two-note chime |
| Round cleared | rising arpeggio |
| Last 3 s of the time bar | a tick every 0.5 s |

---

## 2. Engine design

### 2.1 Main loop

One game frame per 60 Hz frame, locked to the delay timer:

```
loop
  wait until delay = 0, then delay := 1
  frame += 1                ; counter the tests use to spot dropped frames
  read input → hop
  move lanes (carry the courier if it's on a moving pallet)
  check collisions → lose life / deliver
  update time bar, HUD digits that changed, sound
again
```

If one frame's work takes longer than a frame, the game slows down instead
of breaking. The budget test (§4) makes sure that doesn't happen at the
target speed.

### 2.2 Lanes are data

Each lane is a record in a table, so tuning means editing numbers:

| Field | Bytes | Meaning |
|---|---|---|
| `y` | 1 | top pixel of the lane |
| `kind` | 1 | hazard (yard) or platform (chute) |
| `dir` | 1 | +1 right, −1 left |
| `speed` | 1 | 1/16 px per frame at round 1 |
| `acc` | 1 | sub-pixel accumulator |
| `sprite` | 1 | index into the sprite table |
| `tiles` | 1 | object width in 8 px tiles (1–3) |
| `count` | 1 | objects in the lane (1–4) |
| `x[4]` | 4 | object positions, 0–127 |

Each update: `acc += speed + round × step`; every 16 makes a whole pixel,
and the lane moves that far in one go: erase every object with XOR, change
the `x` values, draw again.

As built (step 4), a record is 10 bytes, `y, flags, speed, acc, sprite,
count, x0–x3`: the direction and kind are bits in `flags`, and every
object is one 16 px sprite (a long pallet is two side by side). A lane
updates every other frame (§2.4), so `speed` is per two frames. The table
sits in the state block, so tests can poke a quiet yard and read where the
pallets are.

### 2.3 Collisions are worked out, not read from VF

Earlier I said the collision flag gives you collisions for free. That's not
true here: the courier is on plane 2 and the lanes on plane 1, so drawing one
never sets VF for the other, and the erase-and-redraw of a lane sets VF every
time. Instead the game compares the courier's x range with the objects in
its row (at most 4 range checks), which also answers "am I standing on a
pallet?" and is deterministic for the tests.

### 2.4 Speed budget

- **Target: 500 ticks per frame**, set in the overlay entry's `tickrate`
  (Chippie's XO-CHIP default is 100; Octo offers 500 as a preset).
- **Worst case:** 6 lanes moving in the same frame × up to 6 tiles × 2 draws
  ≈ 72 draws, plus ~15 instructions of bookkeeping per tile ≈ 400
  instructions. Slow lanes move on fewer frames, so the average is far lower.
- **If it gets tight:** draw a solid object's move as two 1-column sprites
  (clear the trailing column, set the leading one) instead of erasing and
  redrawing all its tiles. Only if the budget test asks for it.
- **As built (step 4):** the first chute made the worst frame ~650
  instructions. Three changes brought it to ~450, measured over a minute
  of hopping: a lane loads its whole record into v0–v9 once and moves each
  object in four instructions; each frame updates every other lane record,
  and the table keeps the empty walkway last so both halves have three busy
  lanes; and the hit test is one range check per object.
- **As built (step 5):** the heavy moments are spread over pauses, a
  piece of work per frame: a delivery is a 40-frame pause (score, parcel,
  round clear, speed-up, new light on frames of their own), the time bar
  refills a pixel a frame during it and while dying, and the end of a game
  clears the board over its last three frames. The worst frame is ~480 of
  500, so step 6 must keep the play frame lean.
- **As built (step 6):** the title screen and a new game's board are drawn
  a piece a frame (3 and 4 frames) with the lanes stopped, and game over
  writes its text over three frames. A sound costs a few instructions a
  frame (a step list played by `sound-frame`). Measured by lowering the
  tick rate until the budget tests see late frames: play stays under
  ~380 ticks a frame, and the worst frame overall, on the title and
  game-over paths, is ~445 of 500.
- **Late frames:** a frame that overruns doesn't drop (the counter still
  goes up), it starts late, and the game reads input a frame late. So
  `wait-frame` counts the frames where the timer had already run out
  (`s-late`), and the budget tests check that stays 0.

### 2.5 Memory map

| Address | Contents |
|---|---|
| `0x200` | `jump main` |
| `0x202–0x21F` | **state block** at fixed addresses: frame counter, game state, round, lives, score digits, courier x/row, lit hatch, filled hatches. Tests read it with `Machine.Peek`. |
| `0x218–0x25D` | the lane table (step 4) |
| `0x25E–0x264` | best score, new-best flag, sound step and wait (step 6) |
| `0x220–` | code, lane table, sprites, sound patterns |
| `0x1000–` | spare (XO-CHIP), reached with `i := long` if assets grow |

**Flag registers:** the best score is 2 bytes (BCD-style digits packed in
nibbles) plus a 2-byte marker (`0xD3 0x90`) so a never-saved slot reads as
"no best score". `saveflags v3` after a new best, `loadflags v3` at start-up. As built
(step 6) the flags are the marker then the digits: `D3 90 d1d2 d3d4`.

The whole game is expected to fit in about 2.5 KB, well under the ~3.5 KB
that jumps can reach.

### 2.6 Source structure

One `depot.8o` with clear sections: constants (`:const`, `:calc`), state
block, main loop, input, lanes, collisions, HUD, sound, title and game-over
screens, data. Octopus (`:include`) only if the file passes ~1500 lines.

---

## 3. In the repository

```
games/
├── README.md            what's here, how to build, licence
├── LICENSE              CC0-1.0
├── package.json         @chipcode/octo-assembler pinned; "build" script
├── package-lock.json
└── depot/
    ├── depot.8o         the source
    ├── README.md        how to play, design notes, tuning table
    ├── depot_test.go    headless tests (§4)
    └── testdata/golden/
scripts/build-games.sh   assembles games/*/*.8o into assets/roms/*.ch8
assets/roms/depot.ch8    the built ROM, committed and compiled into the app
```

- **Bundled like the chip8Archive games already in `assets/roms/`:** the
  ROM is compiled into the app and listed in `assets/roms/bundled.json`, so
  it's playable offline and needs no download source. `catalogue-gen` and
  `sources.json` don't change.
- **The built ROM is committed**, so `go build` and `go test` never need
  Node. CI runs `scripts/build-games.sh` and fails if `git diff` shows the
  committed ROM is stale.
- **Assembler:** `@chipcode/octo-assembler`, the maintained successor of the
  `octo-assembler` package the Timendus suite uses.
- **Metadata:** an `add` entry in `catalogue/overlay.json` with
  `"upstream": "pending"`, because the library lists programs from the
  database: title, author, release date, description, platform `xochip`,
  `tickrate` 500, `colors`, `keys`. When it's ready we can propose it to
  chip-8-database and chip8Archive, and the entry then goes.
- **Licences:** `assets/LICENSES.md` and the `assets` package comment say
  the bundled ROMs are CC0 from chip8Archive; they gain "and Chippie's own".

### 3.1 Emulator features

**Arrow keys (an idea from ROADMAP).** For ROMs whose `keys` hint names `up`,
`down`, `left`, `right`, `a` or `b`, the arrow keys press the hinted key for
that direction, Space presses `a` and Shift presses `b`, next to the usual
`1234/QWER/ASDF/ZXCV` layout. Hints with 0x10 added (a second keypad) are
ignored for now. Lives in `app/input`, with unit tests and a Playwright check.

**Saved flag registers.**

- `chip8`: `Machine.Flags() [16]byte` and `Machine.SetFlags`, plus a counter
  that `FX75` increments, so the app can see a save without polling the
  bytes. `chip8` stays clock-free and dependency-free.
- `app`: after `RunFrame`, if the counter changed, hand the 16 bytes to a
  goroutine that stores them, keyed by ROM SHA-1. Restore them right after
  `LoadROM`. Never blocks `Update`.
- `platform`: IndexedDB in the browser (next to the ROM cache), a JSON file
  in the user config folder natively.
- Applies to every ROM that uses `FX75`, not just Depot, as in Octo. A
  "forget saved data" button in the game's details is a possible follow-up.

## 4. Testing

Headless with `internal/testrig`, fixed RNG seed, at 500 ticks per frame:

| Test | How | Checks |
|---|---|---|
| Title screen | run 60 frames | golden frame |
| Playfield | press Start, run 30 frames | golden frame |
| Hit in the yard | press Start, hop up into yard 1 as a forklift arrives | lives 3 → 2 (`Peek`), game state "dying" |
| Fall in the chute | hop onto chute 1 where there's no pallet | lives 3 → 2 |
| Ride a pallet | hop onto a pallet, wait 60 frames | courier x moved with the lane |
| Delivery | a scripted route to the lit hatch (fixed seed) | hatch filled, score +50 + bonuses, golden frame |
| Time out | press Start, wait 16 s | lives 3 → 2 |
| Speed budget | 1 hour of random key presses (fixed seed) | frame counter = frames run (none dropped); never halts |
| Same speed at 1000 | the budget test at 1000 ticks | game plays at the same speed (it's locked to the timer) |

- Golden frames: generated with `go test ./games/... -update`, checked by
  eye, committed. Tuning lane speeds changes them on purpose; never change
  them just to make a test pass.
- **Playwright:** the game shows in the library and starts.
- **By hand:** play-test in Octo (the web IDE, with its debugger) and in
  Chippie on desktop and iPhone.

## 5. Build order

Each step ends with a playable or testable ROM.

| Step | Contents | Done when |
|---|---|---|
| 1 | Toolchain | `games/`, `package.json`, `build-games.sh`, CI stale check, a placeholder ROM that draws the playfield, bundled entry and overlay entry; the ROM appears in Chippie's library |
| 2 | Courier | hops, key repeat, row limits, HUD digits; state block; golden tests |
| 3 | Yard | lane table, moving forklifts and trolleys, hits, lives, game over |
| 4 | Chute | pallets, riding, falling, edge deaths |
| 5 | Hatches | lit hatch, deliveries, time bar, rounds, scoring |
| 6 | Presentation | title and game-over screens, best score, palette, sound |
| 7 | Arrow keys | §3.1, for every ROM with a `keys` hint |
| 8 | Saved flags | §3.1; Depot's best score survives a reload |
| 9 | Tuning and finish | speed table from play-testing, budget test, Playwright, DESIGN §9/§13, ROADMAP, the user README, games README |

## 6. Open questions

None at the moment. Settled on 2026-11-15: the name (Depot), the licence
(CC0), the speed (500), arrow keys (in this milestone) and saved flags
(between sessions).
