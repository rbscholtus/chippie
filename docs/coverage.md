# ROM coverage

Generated 2026-11-13T18:17:16Z by `cmd/catalogue-gen`.

chip-8-database: 484 programs, 543 ROMs.

| Source | Unique ROMs | In database | Missing |
|---|---|---|---|
| chip8Archive | 104 | 100 | 4 |
| kripod/chip8-roms | 106 | 96 | 10 |
| Timendus CHIP-8 test suite | 8 | 8 | 0 |
| Matthew Mikolay's CHIP-8 programs | 6 | 4 | 2 |
| 3D Viper Maze (Timendus) | 1 | 1 | 0 |
| 3D VIP'r Maze (Timendus) | 2 | 2 | 0 |
| Octojam games from itch.io (via maxisimonazzi/chip8-emulator) | 68 | 42 | 26 |
| Hosted by Chippie (from Zophar and David Winter) | 18 | 18 | 0 |
| **All combined** | **310** | **268** | **42** |

Downloadable ROMs in the manifest: 268 of 543 in the database.

## Not in chip-8-database

- chip8Archive: `roms/dinorun.ch8`
- chip8Archive: `roms/mato8.ch8`
- chip8Archive: `roms/mrworm.ch8`
- chip8Archive: `roms/tick.ch8`
- kripod/chip8-roms: `games/Guess [David Winter] (alt).ch8`
- kripod/chip8-roms: `games/Rush Hour [Hap, 2006] (alt).ch8`
- kripod/chip8-roms: `hires/Astro Dodge Hires [Revival Studios, 2008].ch8`
- kripod/chip8-roms: `hires/Hires Maze [David Winter, 199x].ch8`
- kripod/chip8-roms: `hires/Hires Particle Demo [zeroZshadow, 2008].ch8`
- kripod/chip8-roms: `hires/Hires Sierpinski [Sergey Naydenov, 2010].ch8`
- kripod/chip8-roms: `hires/Hires Stars [Sergey Naydenov, 2010].ch8`
- kripod/chip8-roms: `hires/Hires Test [Tom Swan, 1979].ch8`
- kripod/chip8-roms: `hires/Hires Worm V4 [RB-Revival Studios, 2007].ch8`
- kripod/chip8-roms: `hires/Trip8 Hires Demo (2008) [Revival Studios].ch8`
- Matthew Mikolay's CHIP-8 programs: `delaytimer/delay_timer_test.ch8`
- Matthew Mikolay's CHIP-8 programs: `randomnumber/random_number_test.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/bad-apple-koorogi.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/bad-apple-timendus.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/beware-of-sn8k.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/bim-logo-animation.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/blobo.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/c8salt-logo.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/chipception-os.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/collision-course.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/destroy-the-line.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/dodge-catch-shoot.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/duel-of-the-f8s-special-edition.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/enter-the-mine.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/f-chip8-racing.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/falling-ghosts.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/fencing.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/h8-2-carry-flag.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/halloween-blaster.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/invisible-man.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/kanji-calculator.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/man-hang.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/octogon-trail.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/octris.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/termlib-demo.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/upn-calculator.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/xo-nyancat.ch8`
- Octojam games from itch.io (via maxisimonazzi/chip8-emulator): `roms/octojam/xo-tracker-demo.ch8`

## chip8Archive vs chip-8-database

The database wins; these are candidates for upstream fixes.

- Chipcross (83300ff710acdd8417376b88adf40f68171f7ec7): platform: chip-8-database "xochip", chip8Archive "schip"
- Eaty The Alien (fcaa793332a83c93f4ed79f5ffbc8403c8b8aea0): platform: chip-8-database "xochip", chip8Archive "schip"
- Grad School Simulator 2014 (9797a7eaf1e80ec19c085c60bb37991420f54678): platform: chip-8-database "xochip", chip8Archive "schip"
- Octojam 1 Title (f26993a4afd5cda2fea19935773fd3db54866623): platform: chip-8-database "superchip,xochip", chip8Archive "chip8"
- Octovore (74936ffb0db233c722f5a39932d75240b6437a72): platform: chip-8-database "xochip", chip8Archive "schip"
- Piper (7da3eba52a8d8025ddf14ee40d28f151585529a0): platform: chip-8-database "superchip,xochip", chip8Archive "chip8"
- Sub-Terr8nia (64536d549c986e9edf25de9fa89db60d2ade85c0): platform: chip-8-database "xochip", chip8Archive "schip"
