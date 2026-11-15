# Draft 07: propose a Hires CHIP-8 (64×64) platform

- **Where:** issue on [chip-8/chip-8-database](https://github.com/chip-8/chip-8-database).
- **Status:** approved, not posted. Deferred until the maintainers react to
  #29–#32 (owner's decision, 2026-11-15).
- **Overlay entry:** the `hiresChip8` entry under `platforms` in
  `catalogue/overlay.json`. Its ROMs follow in draft 08.

## What and why

kripod/chip8-roms has a `hires/` folder of 8 ROMs that the database has
neither a platform for nor any entry. They use the COSMAC VIP's two-page
64×64 mode, which runs on an unmodified VIP (and the Telmac 1800). Chippie
implements it (M10), and all 8 play.

"Hires" names more than one CHIP-8 variant, so the issue says exactly which
one: the VIP's two-page 64×64 mode, not Tom Swan's HI-RES CHIP-8 (1980),
which is 64×128 and needs a modified interpreter.

## Review checklist

- [x] All 8 ROMs start with the same `1260` and the same machine-code patch
  (0x200–0x2BF). Only 11 bytes of padding at 0x2B3 differ: zeros in Hires
  Test, "REVIVAL2011" in the others. Checked 2026-11-15 at kripod `c723a9e`.
- [x] No machine-code call other than `0230` found: none in a scan of each
  program, and none reached while playing them.
- [x] All 8 play in Chippie with these two rules and VIP quirks
  (2026-11-15).
- [x] **Release year: left out** (owner's decision, 2026-11-15). No source
  dates the two-page 64×64 mode: the earlier "1978" was a guess from the
  VIP CHIP-8 era, and the oldest dated ROM, Tom Swan's Hires Test, is "1979"
  only in kripod's file name. Swan's known hires work is the different
  64×128 HI-RES CHIP-8 (VIPER, January 1980), so even that date may not
  belong to this mode. `modernChip8` has no `release` either. The issue
  asks the maintainers for a date if they know one; the VIPER scans
  ([mattmikolay/viper](https://github.com/mattmikolay/viper)) are where to
  look.
- [x] **Resolutions: 64×64 only** (owner's decision, 2026-11-15). A Hires
  program switches to 64×64 with its first instruction and has no way back,
  so it never shows anything at 64×32. That Chippie runs a ROM without the
  `1260` start at 64×32 on this platform is an emulator fallback (it lets
  the Timendus suite run on it), not a resolution of the platform.
- [x] Text below approved (owner, 2026-11-15).

## The issue

**Title:** Add a platform for the COSMAC VIP's two-page 64×64 "Hires CHIP-8" mode

**Body:**

> The database has no platform for the COSMAC VIP's two-page 64×64 mode,
> often called "Hires CHIP-8", and so none of its ROMs. kripod/chip8-roms
> has 8 of them in its
> [`hires/`](https://github.com/kripod/chip8-roms/tree/master/hires) folder,
> by Revival Studios, David Winter, zeroZshadow, Sergey Naydenov, RB and
> Tom Swan. I'd like to add the platform, then the ROMs in a follow-up PR.
>
> **Which mode.** This is the VIP's two-page 64×64 mode, which runs on an
> unmodified VIP or Telmac 1800. It's not Tom Swan's HI-RES CHIP-8 (1980),
> a 64×128 interpreter.
>
> **How it works** (from the folder's
> [`!hires_information.txt`](https://github.com/kripod/chip8-roms/blob/master/hires/%21hires_information.txt)):
> each ROM starts at 0x200 with `1260`, a jump over a short machine-code
> patch that switches the VIP interpreter to 64×64. The CHIP-8 program
> starts at 0x2C0. `0230` calls the patch's routine that clears the 64×64
> screen. An emulator doesn't need to run the patch:
>
> - when the instruction at 0x200 is `1260`, switch to 64×64 and continue at
>   0x2C0;
> - treat `0230` as a clear screen.
>
> All 8 ROMs have the same patch (apart from 11 bytes of padding), and
> none calls other machine code. Everything else is VIP CHIP-8, with the VIP's quirks.
>
> **Proposed entry** for `platforms.json`:
>
> ```json
> {
>   "id": "hiresChip8",
>   "name": "Cosmac VIP Hires CHIP-8",
>   "description": "A 64x64 mode for CHIP-8 on the Cosmac VIP, using two pages of display memory. It runs on an unmodified VIP: each program starts with a jump (1260) over a short machine-code patch that switches the interpreter to 64x64, and the CHIP-8 program starts at 0x2C0. 0230 clears the 64x64 screen. Emulators can skip the patch: when the first instruction at 0x200 is 1260, switch to 64x64 and continue at 0x2C0. Not to be confused with Tom Swan's 64x128 HI-RES CHIP-8 (1980).",
>   "displayResolutions": ["64x64"],
>   "defaultTickrate": 15,
>   "quirks": {
>     "shift": false,
>     "memoryIncrementByX": false,
>     "memoryLeaveIUnchanged": false,
>     "wrap": false,
>     "jump": false,
>     "vblank": true,
>     "logic": true
>   }
> }
> ```
>
> The quirks and tick rate are `originalChip8`'s. The id follows the
> existing ones; happy to change it, or the name, to whatever fits best.
>
> I've left out `release`: I couldn't find when this mode was first
> published. If you know the VIPER issue or another source that introduced
> it, I'll add the date.
>
> **Reference implementation:** Chippie
> (https://rbscholtus.github.io/chippie/), which plays all 8 ROMs this way
> (`chip8/ops_hires.go` in https://github.com/rbscholtus/chippie). Search
> its library for "hires".
>
> If you agree, I'll open a PR adding the platform and the 8 programs.
