# Draft 07: propose a Hires CHIP-8 (64×64) platform

- **Where:** issue on [chip-8/chip-8-database](https://github.com/chip-8/chip-8-database).
- **Status:** posted 2026-11-16 as
  [#33](https://github.com/chip-8/chip-8-database/issues/33), open. The
  owner chose to post it, and 08, without waiting for #29–#32. Its last
  line was changed to point to the PR (#34), as agreed.
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

## Follow-up: reply to tobiasvl (2026-11-17)

- **Status:** posted 2026-11-17 as
  [a comment](https://github.com/chip-8/chip-8-database/issues/33#issuecomment-6925757552),
  after Chippie PR #20 was merged; the issue retitled as below.

tobiasvl [commented](https://github.com/chip-8/chip-8-database/issues/33#issuecomment-6899378929)
on 2026-11-15: call it "2 Page Display for CHIP-8" (Modla and Winsor, VIPER
1/3, September 1978); `1260` jumps to 0x260, not 0x2C0; detect the mode by
the patch's bytes, not `1260` alone. All three checked against the
[VIPER scan](https://github.com/mattmikolay/viper/blob/master/volume1/issue3.pdf)
(pp. 10–12) and implemented in Chippie PR #20; chip-8-database PR #34 is
updated to match (draft 08).

- [x] Facts checked: the patch at 0x200–0x249 matches the article's listing
  byte for byte, in all 8 ROMs; Hires Test is the article's test program,
  moved by 0x60; the two Naydenov demos need `memoryLeaveIUnchanged`.
- [x] "PIPS for VIPs": from tobiasvl's extensions page, not checked in the
  book itself; kept (owner, 2026-11-17).
- [x] Text below approved (owner, 2026-11-17).

**Comment on chip-8-database issue #33:**

> Thanks, that was very helpful: you were right on all three points. I read
> the article in the [VIPER scan](https://github.com/mattmikolay/viper/blob/master/volume1/issue3.pdf)
> (pp. 10–12) and have updated #34 to match.
>
> **Name and date.** The platform is now `twoPageChip8`, "2 Page Display for
> CHIP-8", with `"release": "1978-09"`, crediting Andy Modla and Jef Winsor.
>
> **0x260.** I had this wrong. The article's programs start at 0x260, after
> the patch at 0x200–0x249 that every program must include. The ROMs in
> kripod's `hires/` folder go one step further so that they run on an
> unmodified VIP: their CHIP-8 code at 0x260 copies the article's
> interpreter changes into 0x003–0x1FB (`Fx55` stores of data kept at
> 0x24A), calls `02AC`, clears the screen with `0230` at 0x2BE, and only
> then reaches the program proper at 0x2C0. The description now explains
> both.
>
> **Detection.** Agreed. The suggested rule is now: if 0x200 holds `1260`
> and 0x202–0x249 hold the article's patch byte for byte, switch to 64×64
> and continue at 0x2C0 after the self-patching code, else at 0x260; then
> treat `0230` as clearing the screen. All 8 ROMs have exactly the
> article's bytes there. Chippie now works this way
> ([`chip8/ops_twopage.go`](https://github.com/rbscholtus/chippie/blob/main/chip8/ops_twopage.go)).
>
> Two more things came up while checking:
>
> - **Hires Test** is the article's own test program, byte for byte, moved
>   from 0x260 to 0x2C0. I've credited it to Modla and Winsor (1978-09) and
>   kept kripod's attribution to Tom Swan in its description.
> - **Hires Sierpinski and Hires Stars** got stuck on the VIP's quirks:
>   Sergey Naydenov's compiler stores back to the address it just loaded
>   from (`A684 F065 7001 F055`), which assumes `Fx65` leaves `I` alone. The
>   lo-res Stars is already listed as `superchip`, probably for the same
>   reason. Both are now under `quirkyPlatforms`, `twoPageChip8` with
>   `memoryLeaveIUnchanged: true`.

**Also:** retitle the issue to "Add a platform for the COSMAC VIP's 2 Page
Display for CHIP-8 (64×64)".
