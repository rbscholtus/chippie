# Draft 06: four alternate ROMs of programs already in chip-8-database

- **Where:** PR to [chip-8/chip-8-database](https://github.com/chip-8/chip-8-database),
  from branch `add-alternate-roms` of the owner's fork, based on `master`.
  It adds no programs, so it's independent of #30 and #31.
- **Status:** draft, waiting for review.
- **Overlay entries:** the four "add ROMs to" entries and the Delay Timer
  Test "change" in `catalogue/overlay.json`.

## What and why

Each file is a different build of a program the database already has
(most bytes differ from the database's version), found by hashing the ROM
sources Chippie uses:

| Program (database #) | New ROM | SHA-1 | Source | Platforms |
|---|---|---|---|---|
| Guess the Number (#26) | `Guess [David Winter] (alt).ch8`, 148 bytes; also `GUESS` in David Winter's CHIP8 2.2.0 package and Zophar's pack | `5260f8931e0e9f41e555b382a14a88368e3ed886` | [kripod/chip8-roms](https://github.com/kripod/chip8-roms) | originalChip8, modernChip8 |
| Rush Hour (#55) | `Rush Hour [Hap, 2006] (alt).ch8`, the other of its two versions | `29a41ab4d0aa3bc0d6a9d2fa71d533fe463344b3` | kripod/chip8-roms | originalChip8 |
| Delay Timer Test (#19) | `delay_timer_test.ch8`, 2016 | `ba603bde1d8596c575e81096fff3cea40173d7e3` | [mattmikolay/chip-8](https://github.com/mattmikolay/chip-8/tree/master/delaytimer) (MIT) | originalChip8 |
| Random Number Test (#49) | `random_number_test.ch8`, 2016 | `b7b46ad49871e54302496c95c41be842e4a4abdf` | [mattmikolay/chip-8](https://github.com/mattmikolay/chip-8/tree/master/randomnumber) (MIT) | originalChip8 |

Also: Delay Timer Test (#19) has no `authors` or `release`; kripod's file
name for it says "Matthew Mikolay, 2010", as for Random Number Test (#49).

- **Platforms:** the same as the database's existing ROM of each program;
  all four also run headless on both `originalChip8` and `modernChip8`.
- **Rush Hour's versions:** hap's notes describe 1.0 (2006-12-17) and 1.1
  (2008-02-08). Which file is which isn't stated anywhere I found, so the
  entry doesn't claim a version number.
- **Keys:** Guess's notes say "press 5 if so"; Rush Hour's give 5/8/7/9 to
  move and A to slide.
- **Left out for now:** kripod's 8 "hires" ROMs, which need a 64×64
  platform the database doesn't have (a separate question).

## Review checklist

- [x] SHA-1s computed from the files at Chippie's pinned commits
  (kripod `c723a9e`, mattmikolay `fb6f446`).
- [x] All four run headless on `originalChip8` and `modernChip8`
  (2026-11-15).
- [ ] Played in Chippie by the owner: the library lists them as "Guess the
  Number (Guess [David Winter] (alt))", "Rush Hour (Rush Hour [Hap, 2006]
  (alt))", "Delay Timer Test (delay_timer_test)" and "Random Number Test
  (random_number_test)".
- [ ] Text below approved.

## The PR

**Title:** Add four alternate ROMs: Guess, Rush Hour, Delay Timer Test, Random Number Test

**Body:**

> These four ROMs are different builds of programs the database already
> has, so each is added to its existing program, with the same platforms
> as the ROM already there:
>
> - **Guess the Number:** `Guess [David Winter] (alt).ch8` from the
>   Revival Studios pack (kripod/chip8-roms), the same file as `GUESS` in
>   David Winter's CHIP8 2.2.0 package and Zophar's Chip-8 Games Pack.
> - **Rush Hour:** `Rush Hour [Hap, 2006] (alt).ch8`, also from that pack;
>   hap released a 1.0 and a 1.1, so this is presumably the other one.
> - **Delay Timer Test** and **Random Number Test:** the 2016 builds in
>   Matthew Mikolay's own repository (mattmikolay/chip-8, MIT), with a
>   ROM-level `release`.
>
> It also gives Delay Timer Test the `authors` and `release` that Random
> Number Test has (Matthew Mikolay, 2010), as kripod's file name for it
> says. `sha1-hashes.json` is extended to match.
>
> Checked by playing each ROM in Chippie (https://rbscholtus.github.io/chippie/).
