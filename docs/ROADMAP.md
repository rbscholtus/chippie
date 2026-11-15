# Chippie: roadmap

What's been built, what's next, and what's still open. The design behind it
is in [DESIGN.md](DESIGN.md).

## Milestones

Detailed plans for M1–M4: [M1-M4.md](plans/M1-M4.md). For M5
and M7: [M5-M7.md](plans/M5-M7.md). For M6 and M8:
[M6-M8.md](plans/M6-M8.md). The order is M5, M7, M6, M8;
M7 does not depend on M6, and M8 does not depend on M6.

| # | Milestone | Done when |
|---|---|---|
| M1 | Core: `modernChip8` + `originalChip8`, rig basics | Tests 1–4 and 6–7 pass headless (minimal test rig + golden import); benchmarks in place |
| M2 | Complete test rig + golden import | All 64×32 golden frames imported; failure diffs and `-update` work; test 5 on CHIP-8 |
| M3 | Ebitengine shell | Native and browser builds play a bundled ROM: 60 TPS, keyboard, beep, CRT shader (from the prototype), loader page, Pages deploy |
| M4 | Catalogue + library | Database snapshot + `sources.json` + `romstore` (download, SHA-1, IndexedDB cache) + library drawer that closes on pick + file picker + drag-and-drop |
| M5 | SCHIP | All SCHIP quirks + scrolling tests pass; the 12 Zophar ROMs hosted |
| M6 | The 1970s TV | Generated art in, `skin.json`, knobs (click/drag/wheel), power animation, OSD, remote on touch devices, HiDPI layout, CRT at 60 FPS on iPhone |
| M7 | XO-CHIP | Quirks + scrolling tests for XO-CHIP; pattern audio |
| M8 | Upstream | PRs/issues to chip-8-database and the test suite submitted ([upstream/](upstream/README.md)) |
| M9 | Our own game (in progress) | Depot (XO-CHIP, Octo) bundled and tested; arrow keys for ROMs with a `keys` hint; flag registers saved between sessions |

M1–M8 are done and deployed.

## Open questions

To be settled while building, not before:

- **Licensing of kripod ROMs.** The repo has no licence file. We only link to
  it, never copy from it, so this matters only if we ever mirror files.
- **Hires CHIP-8 (64×64).** Add it as a platform in Chippie before the
  database has it? See "Hires CHIP-8 ROMs" below.
- **Fonts for the UI and OSD.** Settled in M6 for now: the Go fonts already
  in the project (Go Medium for the channel dial, Go Mono Bold for the OSD).
  A period face (OFL only) could still replace them.

**Ideas for the end of the project:**

- **Follow up on the upstream submissions** ([upstream/](upstream/README.md);
  `catalogue/overlay.json`). Answer the maintainers'
  questions on chip-8-database #29–#32 and chip8-test-suite #38. When
  chip-8-database merges a PR, the weekly refresh lists its overlay entries
  as done: delete them from `catalogue/overlay.json`. If the test suite's
  pictures are fixed, drop `cmd/golden-import`'s four pixel corrections.
- **Upstream proposals, deferred until the maintainers react** (M8 items
  4–6): a standard "where to download" field in chip-8-database, with
  Chippie's `sources.json` as an example; asking where its ~230 unsourced
  ROM hashes came from; a 64×64 "Hires CHIP-8" platform, which kripod's 8
  hires ROMs need.
- **chip8Archive's platform fields:** for 5–6 of its 7 conflicts with
  chip-8-database the archive's platform is too basic (`docs/coverage.md`).
  An issue there, after play-testing Chipcross and Piper.
- **XO-CHIP scrolling reference pictures** for the test suite, which has
  none (Chippie generates its own).
- **Hires CHIP-8 ROMs.** Eight ROMs in kripod/chip8-roms' `hires/` folder
  are in no chip-8-database entry and can't be played yet
  (`docs/coverage.md`, "Not in chip-8-database"):
  Astro Dodge Hires, Hires Maze, Hires Particle Demo, Hires Sierpinski,
  Hires Stars, Hires Test, Hires Worm V4 and Trip8 Hires Demo. Most need
  the 64×64 "Hires CHIP-8" variant of the COSMAC VIP: each ROM starts with
  a `1260` jump over a short machine-code patch (the same bytes in every
  one checked) that switches the interpreter to a 64×64 screen; the program
  itself starts at 0x260. Steps:
  1. Add the platform to `chip8/` (64×64 display, VIP quirks, recognise
     the patch rather than run it) with tests, and to Chippie's overlay,
     so the ROMs get download locations and a platform.
  2. Play-test each ROM to confirm its platform; note any that need
     something else.
  3. Upstream: the deferred issue proposing the platform to chip-8-database
     ([upstream/](upstream/README.md)), then a PR adding the ROMs, with
     Chippie as the evidence.
- **One entry per program in the library.** Programs with several ROMs
  (e.g. Guess the Number) are listed once per ROM, which adds near-identical
  rows. List the database's first ROM only, and offer the others in Settings.

- **Arrow keys for games with a keys hint.** For ROMs whose chip-8-database
  entry has a `keys` hint (27 of the 204 downloadable ones), also map the
  arrow keys to its up/down/left/right and Space/Shift to `a`/`b`, next to the
  usual 1234/QWER/ASDF/ZXCV layout. Hints with 0x10 added mean a second
  keypad; show those properly instead of "?".
- **Favourite games.** Mark a game as a favourite (e.g. a star in the
  details), stored per device like the ROM cache, and list favourites first
  in the library drawer (or behind a "Favourites" filter) so they start in
  one tap.
- **Record the keys games use (desktop only).** Only worth it if the data
  turns out to be useful upstream. While a ROM runs, record per SHA-1 the keys
  it reads (`EX9E`/`EXA1`), the keys actually held when read (filters out
  keypad scans) and the key that ends an `FX0A` wait. Merge across sessions
  into a JSON file in the user config folder, saved off the `Update` loop via
  `platform/` (no-op on WASM). Roles (up, fire…) can't be inferred: fill the
  database-format `keys` block by hand, and export filled entries with a small
  `cmd/` tool for a chip-8-database PR.

