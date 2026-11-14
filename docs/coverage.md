# ROM coverage

Generated 2026-11-14T19:26:40Z by `cmd/catalogue-gen`.

chip-8-database: 484 programs, 543 ROMs.

| Source | Unique ROMs | In database | Missing |
|---|---|---|---|
| chip8Archive | 104 | 104 | 0 |
| kripod/chip8-roms | 106 | 96 | 10 |
| Timendus CHIP-8 test suite | 8 | 8 | 0 |
| Matthew Mikolay's CHIP-8 programs | 6 | 4 | 2 |
| 3D Viper Maze (Timendus) | 1 | 1 | 0 |
| 3D VIP'r Maze (Timendus) | 2 | 2 | 0 |
| Octojam games from itch.io (via maxisimonazzi/chip8-emulator) | 68 | 68 | 0 |
| Hosted by Chippie (from Zophar and David Winter) | 18 | 18 | 0 |
| **All combined** | **310** | **298** | **12** |

Downloadable ROMs in the manifest: 298 of 573 in the database and Chippie's overlay.

## Chippie's overlay (pending upstream)

Additions and corrections in `catalogue/overlay.json`; "done" ones can be deleted.

- change Mr Worm: title: waiting for https://github.com/chip-8/chip-8-database/pull/28
- change Computer Simulator: screenRotation: not yet proposed upstream
- change Infini Br8kr: screenRotation: not yet proposed upstream
- change Octo Space Program: screenRotation: not yet proposed upstream
- add Dino Run: waiting for https://github.com/chip-8/chip-8-database/pull/30
- add Mato8: waiting for https://github.com/chip-8/chip-8-database/pull/30
- add Tick: waiting for https://github.com/chip-8/chip-8-database/pull/30
- add ROMs to Mr Worm: waiting for https://github.com/chip-8/chip-8-database/pull/28
- add Bad Apple!!: not yet proposed upstream
- add Bad Apple: not yet proposed upstream
- add Beware of Sn8k!: not yet proposed upstream
- add BIM Logo Animation: not yet proposed upstream
- add Blobo: not yet proposed upstream
- add C8SALT logo: not yet proposed upstream
- add Chipception OS: not yet proposed upstream
- add Collision Course: not yet proposed upstream
- add Destroy The Line: not yet proposed upstream
- add Dodge Catch Shoot: not yet proposed upstream
- add Duel Of The F8s: Special Edition: not yet proposed upstream
- add Enter the mine: not yet proposed upstream
- add F-Chip8 Racing: not yet proposed upstream
- add Falling Ghosts: not yet proposed upstream
- add Fencing: not yet proposed upstream
- add H8 2: Carry Flag: not yet proposed upstream
- add Halloween-Blaster: not yet proposed upstream
- add Invisible Man: not yet proposed upstream
- add Kanji Calculator (WIP): not yet proposed upstream
- add Man Hang: not yet proposed upstream
- add Octogon Trail: not yet proposed upstream
- add Octris: not yet proposed upstream
- add Termlib demo: not yet proposed upstream
- add RPN Calculator: not yet proposed upstream
- add XO-NYANCAT!!: not yet proposed upstream
- add XO-TRACKER DEMO: not yet proposed upstream

## Not in chip-8-database

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

## chip8Archive vs chip-8-database

The database wins; these are candidates for upstream fixes.

- Chipcross (83300ff710acdd8417376b88adf40f68171f7ec7): platform: chip-8-database "xochip", chip8Archive "schip"
- Eaty The Alien (fcaa793332a83c93f4ed79f5ffbc8403c8b8aea0): platform: chip-8-database "xochip", chip8Archive "schip"
- Grad School Simulator 2014 (9797a7eaf1e80ec19c085c60bb37991420f54678): platform: chip-8-database "xochip", chip8Archive "schip"
- Octojam 1 Title (f26993a4afd5cda2fea19935773fd3db54866623): platform: chip-8-database "superchip,xochip", chip8Archive "chip8"
- Octovore (74936ffb0db233c722f5a39932d75240b6437a72): platform: chip-8-database "xochip", chip8Archive "schip"
- Piper (7da3eba52a8d8025ddf14ee40d28f151585529a0): platform: chip-8-database "superchip,xochip", chip8Archive "chip8"
- Sub-Terr8nia (64536d549c986e9edf25de9fa89db60d2ade85c0): platform: chip-8-database "xochip", chip8Archive "schip"
