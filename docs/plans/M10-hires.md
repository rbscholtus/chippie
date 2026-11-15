# Chippie: implementation plan for M10 (Hires CHIP-8)

## Context

chip-8-database has neither the COSMAC VIP's 64×64 "Hires" platform nor
the 8 ROMs in kripod/chip8-roms' `hires/` folder: Astro Dodge Hires, Hires
Maze, Hires Particle Demo, Hires Sierpinski, Hires Stars, Hires Test, Hires
Worm V4 and Trip8 Hires Demo.

**How the mode works** (kripod's `hires/!hires_information.txt`): each ROM
starts at 0x200 with `1260`, a jump over a machine-code patch that switches
the VIP interpreter to 64×64. An emulator skips the patch: when the
instruction at 0x200 is `1260`, it switches to 64×64 and jumps to 0x2C0.
`0230` clears the 64×64 screen.

**What exploring the code found:**

- `chip8/` copes with any size up to a platform's largest resolution
  (`display.go` resets, switches, clears, views and draws at the current
  size). Only the start size, 64×32, is fixed in `Machine.Reset`.
- `step()` in `cpu.go` knows PC and opcode before it dispatches: the place
  for the `1260` rule. `0230` now halts as an unknown opcode.
- The overlay can't add a platform: `checkROM` (`catalogue/overlay.go`)
  rejects a ROM on a platform that isn't in `platforms.json`, which fails the
  whole catalogue load. `Resolve` (`catalogue/profile.go`) takes quirks and
  tick rate only from the database's platforms.
- `catalogue-gen` already hashes kripod's `hires/*.ch8`. Once their SHA-1s
  are in the overlay, a run gives them pinned kripod URLs. It applies the
  overlay to the freshly fetched database, so adding platforms belongs in
  `ParseOverlay`/`ApplyOverlay`.
- The picture is fixed at 2:1 in `app/tv.go`; the renderer stretches any
  frame size.
- Several tests pin the list of platforms and need the new one: the
  platform table, the suite cases, the benchmarks, the disassembler goldens,
  the per-platform opcode map, the snapshot and overlay tests, the library
  families and the settings steps.

## Steps

1. **Check the 8 ROMs** (no code). Download them from kripod at the pinned
   commit. Record SHA-1 and size, whether each starts with the same `1260`
   and patch, and which other `0NNN` calls follow 0x2C0. A ROM that needs
   more than the two rules is left out, with a roadmap note on what it
   needs.
2. **Core (`chip8/`).** A new `HiresChip8` platform, appended after
   `XOChip`: id `hiresChip8`, name "Hires CHIP-8 (COSMAC VIP)", 15 ticks,
   4 KB, up to 64×64, one plane, VIP quirks, and a new op flag `opsHires`.
   - In `step()`, with `opsHires`: if PC is 0x200 and the opcode is `1260`,
     switch to 64×64 and jump to 0x2C0. Without `1260` the ROM runs as VIP
     CHIP-8 at 64×32.
   - `0230` clears the screen under `opsHires`. The disassembler shows it
     as `CLS`; `1260` stays `JP 0x260`, because the disassembler doesn't
     know the PC.
   - Tests: unit tests for the `1260` start (and `1260` elsewhere as a plain
     jump), `0230` clearing all of 64×64 (and still halting on other
     platforms), sprites at row 64; a synthetic test ROM built in the test,
     with golden frames; the Timendus suite on the new platform, with the
     VIP CHIP-8 goldens; disassembler, per-platform opcode and benchmark
     cases.
3. **Catalogue.** `overlay.json` gets a `platforms` list, entries in
   `platforms.json`'s shape, each with an `upstream` link. The new entry:
   id `hiresChip8`, release 1978, resolutions 64×32 and 64×64, tick rate
   15, VIP quirks. `ApplyOverlay` adds platforms first; an entry is done
   once upstream has its id, reported like programs (`docs/coverage.md`,
   the refresh summary). Overlay platforms are checked like the database's,
   unknown quirk ids are rejected, and the platform must exist in `chip8`.
   One `add` per ROM that passed step 1. Run `catalogue-gen`, so
   `sources.json` gains their kripod URLs and the coverage report's
   "Not in chip-8-database" loses them.
4. **App.** The library counts the platform in the CHIP-8 family and finds
   it by the word "hires". Settings needs no change: check that switching
   platform works through `Resolve`. **The picture stays 2:1** (the owner's
   decision): a 64×64 frame fills the usual picture, with tall pixels.
5. **Upstream drafts** (not posted): `docs/upstream/07-hires-platform.md`,
   an issue proposing a `hiresChip8` platform, and
   `docs/upstream/08-hires-roms.md`, a PR adding the programs after it. The
   issue must name the variant precisely: the VIP's two-page 64×64 mode,
   not Tom Swan's 1980 HI-RES CHIP-8 (64×128).
6. **Docs.** README: the platform's row playable, search words, roadmap
   summary. ROADMAP: M10 done, the platform section trimmed to what's left.
   DESIGN: the platform table (§6.3), the overlay adding platforms (§7,
   §10).

## Verification

- `golangci-lint run` (also with `GOOS=js GOARCH=wasm`), `go test ./...`,
  `go build ./cmd/chippie`, `./scripts/build.sh` and the Playwright tests.
- Play-check in the browser build: load each hires ROM from the library,
  run it for some seconds and look at a screenshot.
- `docs/coverage.md` lists no hires files under "Not in chip-8-database".
