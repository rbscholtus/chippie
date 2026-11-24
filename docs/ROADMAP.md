# Chippie: roadmap

What's been built, what's next, and what's still open. The design behind it
is in [DESIGN.md](DESIGN.md).

## Milestones

Detailed plans for M1–M4: [M1-M4.md](plans/M1-M4.md). For M5
and M7: [M5-M7.md](plans/M5-M7.md). For M6 and M8:
[M6-M8.md](plans/M6-M8.md). For M10: [M10-hires.md](plans/M10-hires.md). The order is M5, M7, M6, M8;
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
| M10 | Hires CHIP-8 | The 64×64 COSMAC VIP mode implemented and tested; its 8 known ROMs in the library and playable ([plan](plans/M10-hires.md), [below](#hires-chip-8)) |

M1–M8 and M10 are done; M1–M8 are deployed.

## Platforms

### Hires CHIP-8

Done in M10: Chippie plays the COSMAC VIP's two-page 64×64 mode and its 8
known ROMs (kripod/chip8-roms' `hires/` folder), all added through the
overlay with the platform. All 8 play with the two rules (the `1260` start
and `0230`); none needed anything else.

**Upstream: done.** Renamed to `twoPageChip8`, Andy Modla and Jef Winsor's
"2 Page Display for CHIP-8" (VIPER, September 1978), and detected by its
patch (DESIGN §6.3). chip-8-database merged the platform and the 8 programs
in [PR #34](https://github.com/chip-8/chip-8-database/pull/34) (drafts
[07](upstream/07-hires-platform.md) and [08](upstream/08-hires-roms.md)),
and the weekly refresh deleted the overlay entries.

### Not planned

- **CHIP-8X** (RCA, 1980): colour, sound and a second keypad, through VIP
  expansion boards. chip-8-database lists no ROMs for it, and only two
  programs are known.
- **MEGA-CHIP** (Revival Studios, 2007): 256×192 with 256 colours, sampled
  sound, large ROMs. chip-8-database lists no ROMs for it, only a few demos
  and one game are known, and its specification is loose.

Either can be reconsidered if ROMs turn up in the database.

- **A CDP1802 core for the COSMAC VIP** (idea from the
  [extensions review](DESIGN.md#63-platforms), 2026-11-16): run the real VIP
  interpreter instead of reimplementing it, as Cadmium does. That gives
  `hybridVIP` (machine-code `0NNN` calls) and would replace the Hires
  special case, and most VIPER-era variants (four-page 64×128, CHIP-8E,
  CHIP-10, CHIP-8X) would run by loading their interpreter. A milestone-sized
  job for few ROMs: chip-8-database has one `hybridVIP` ROM (Clock Program)
  and none for the variants. More ROMs come from finding downloads (310 of
  585 have one) than from new platforms.
- **Octo's `0000` halt** on XO-CHIP: stop cleanly instead of reporting an
  unknown opcode when a program runs into empty memory. Not more ROMs, a
  nicer message.

## Open questions

To be settled while building, not before:

- **Licensing of kripod ROMs.** The repo has no licence file. We only link to
  it, never copy from it, so this matters only if we ever mirror files.
- **Fonts for the UI and OSD.** Settled in M6 for now: the Go fonts already
  in the project (Go Medium for the channel dial, Go Mono Bold for the OSD).
  A period face (OFL only) could still replace them.

**Ideas for the end of the project:**

- **Follow up on the upstream submissions** ([upstream/](upstream/README.md);
  `assets/catalogue/overlay.json`). Answer the maintainers' questions on
  chip-8-database #29–#34 and #36 (keys hints and notes), and
  chip8-test-suite #38. When chip-8-database merges a PR, the weekly
  refresh lists its overlay entries as done: delete them from
  `assets/catalogue/overlay.json`. If the test suite's pictures are fixed,
  drop `cmd/golden-import`'s four pixel corrections.
- **Upstream proposals, deferred until the maintainers react** (M8 items
  4–6): a standard "where to download" field in chip-8-database, with
  Chippie's `sources.json` as an example; asking where its ~230 unsourced
  ROM hashes came from. (The 64×64 platform: drafts 07 and 08.)
- **chip8Archive's platform fields:** for 5–6 of its 7 conflicts with
  chip-8-database the archive's platform is too basic (`docs/coverage.md`).
  An issue there, after play-testing Chipcross and Piper.
- **XO-CHIP scrolling reference pictures** for the test suite, which has
  none (Chippie generates its own).
- **One entry per program in the library.** Programs with several ROMs
  (e.g. Guess the Number) are listed once per ROM, which adds near-identical
  rows. List the database's first ROM only, and offer the others in Settings.

- **Download sources for the games the library leaves out** (checked
  2026-11-23). Of chip-8-database's 585 ROMs, Chippie could download 310.
  The library left out the other 275, which had no known source: 172
  CHIP-8, 53 SUPER-CHIP, 42 XO-CHIP and 7 modern CHIP-8 ROMs. Other
  emulators' built-in ROMs and the Octojams have 23 of them, maybe 24; the
  first 19 are done (329 downloadable now), the rest are open:
  - [00SChip8](https://github.com/00laboratories/00SChip8/tree/master/games)'s
    `games/` is Revival Studios' whole Chip-8 Program Pack (RS-C8000,
    2011); kripod and dmatlack carry only its CHIP-8 half. It adds the
    pack's 17 SuperChip games and demos (Revival's Climax Slideshow, Super
    Trip8 Demo and Super Astro Dodge, Hap's Sokoban and Loopz, SuperWorm
    V4, Laser, Matches…) and the 00Schip8 Life demo. **Done 2026-11-23**:
    a `catalogue-gen` archive that only links, as with kripod (no licence
    for the ROMs).
  - [Timendus/alien-inv8sion](https://github.com/Timendus/alien-inv8sion)
    (GPL-3.0, Octojam 8): Alien-Inv8sion (XO-CHIP), as
    `bin/alien-inv8sion.ch8`. **Done 2026-11-23**: an archive like 3D
    Viper Maze's.
  - Octojams 6–10 on itch.io ([Octojam 10](https://itch.io/jam/octojam-10/entries),
    86 entries in all): every other entry is playable already. Dump Trump
    (Micheal Wales, [Octojam 6](https://itch.io/jam/octojam-6/entries)) is
    only a "name your own price" download. Fetch it with an itch.io API key
    ([itch-dl](https://github.com/DragoonAethis/itch-dl)). Browsers can't
    download from itch.io, so it needs another home: ask the author, e.g.
    to add it to chip8Archive.
  - [Octo](https://johnearnest.github.io/Octo/)'s Examples
    ([examples/](https://github.com/JohnEarnest/Octo/tree/gh-pages/examples),
    MIT): Deep8, F8Z and Snow Daze, as source only. Assemble them with the
    pinned Octo (`scripts/build-games.sh`); if the SHA-1s match
    chip-8-database's, host them like `web/hosted-roms/`.
  - [binji/raw-wasm](https://github.com/binji/raw-wasm/tree/main/chip8)
    (MIT): chip-8-database's "Black Lives Matter demo" (Ben Smith, 2020);
    the 365 bytes in its `index.html` match. Host them.

  Nothing new in [taniarascia/chip8](https://github.com/taniarascia/chip8),
  [dmatlack/chip8](https://github.com/dmatlack/chip8),
  [dustinbowers/rust-chip8](https://github.com/dustinbowers/rust-chip8)
  (its Alien-Inv8sion is Timendus' file),
  [David Winter's CHIP8 2.2.0](https://www.pong-story.com/chip8/chp8_220.zip)
  (its SUPER-CHIP games are those we host),
  [ticalc's chiplib](https://www.ticalc.org/archives/files/fileinfo/74/7492.html)
  (the same games for the TI-92), or
  [Cadmium](https://chip8.gulrak.net/cadmium/)'s Library (metadata for the
  user's own files, without ROMs; it knows the same 275, and 543 ROMs
  chip-8-database doesn't, mostly ETI-660, VIP machine code, MEGA-CHIP and
  CHIP-8X). The original Revival pack (`chip8.com/downloads/Chip-8
  Pack.zip`, "freely distributed in its original form") survives in the
  [Internet Archive](http://web.archive.org/web/20160514030144/http://chip8.com/downloads/Chip-8%20Pack.zip).

- **Arrow keys for games with a keys hint.** For ROMs whose chip-8-database
  entry has a `keys` hint (27 of the 204 downloadable ones), also map the
  arrow keys to its up/down/left/right and Space/Shift to `a`/`b`, next to the
  usual 1234/QWER/ASDF/ZXCV layout. Hints with 0x10 added mean a second
  keypad; show those properly instead of "?".
- **A realistic switch-off** (2026-11-22). Real CRTs don't squash to a line
  and then a dot over half a second: the sweep dies within a few frames, and
  on sets without a spot killer (most before the late 1970s) the still-hot
  cathode leaves a soft spot that fades over seconds (repair FAQs, the
  spot-killer patents). Switching on, a cold set shows nothing until the
  cathode heats, then the picture fades in (30 s or more; "instant-on" sets
  of the 1970s about a second). Prototype on branch `crt-switch-off`: a
  6-update collapse and a 3-second fading spot; on, a 2.5-second fade-in at
  full size; no dot, no white flash ([crt-shader.md](crt-shader.md)). Still
  open: tuning against videos and on phones.
- **Favourite games.** Mark a game as a favourite (e.g. a star in the
  details), stored per device like the ROM cache, and list favourites first
  in the library drawer (or behind a "Favourites" filter) so they start in
  one tap.
- **Record the keys games use (desktop only).** Only worth it if the data
  turns out to be useful upstream. While a ROM runs, record per SHA-1 the keys
  it reads (`EX9E`/`EXA1`), the keys actually held when read (filters out
  keypad scans) and the key that ends an `FX0A` wait. Merge across sessions
  into a JSON file in the user config folder, saved off the `Update` loop via
  `internal/platform/` (no-op on WASM). Roles (up, fire…) can't be inferred:
  fill the database-format `keys` block by hand, and export filled entries
  with a small `cmd/` tool for a chip-8-database PR.

