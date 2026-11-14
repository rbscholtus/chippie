# ROM coverage

Generated 2026-11-14T21:02:53Z by `cmd/catalogue-gen`.

chip-8-database: 484 programs, 543 ROMs.

| Source | Unique ROMs | In database | Missing |
|---|---|---|---|
| chip8Archive | 104 | 104 | 0 |
| kripod/chip8-roms | 106 | 98 | 8 |
| Timendus CHIP-8 test suite | 8 | 8 | 0 |
| Matthew Mikolay's CHIP-8 programs | 6 | 6 | 0 |
| 3D Viper Maze (Timendus) | 1 | 1 | 0 |
| 3D VIP'r Maze (Timendus) | 2 | 2 | 0 |
| Octojam games from itch.io (via maxisimonazzi/chip8-emulator) | 68 | 68 | 0 |
| Hosted by Chippie (from Zophar and David Winter) | 18 | 18 | 0 |
| **All combined** | **310** | **302** | **8** |

Downloadable ROMs in the manifest: 302 of 577 in the database and Chippie's overlay.

## Chippie's overlay (pending upstream)

Additions and corrections in `catalogue/overlay.json`; "done" ones can be deleted.

- change Mr Worm: title: waiting for https://github.com/chip-8/chip-8-database/pull/28
- change Computer Simulator: screenRotation: waiting for https://github.com/chip-8/chip-8-database/pull/31
- change Infini Br8kr: screenRotation: waiting for https://github.com/chip-8/chip-8-database/pull/31
- change Octo Space Program: screenRotation: waiting for https://github.com/chip-8/chip-8-database/pull/31
- change Delay Timer Test: authors, release: waiting for https://github.com/chip-8/chip-8-database/pull/32
- add Dino Run: waiting for https://github.com/chip-8/chip-8-database/pull/30
- add Mato8: waiting for https://github.com/chip-8/chip-8-database/pull/30
- add Tick: waiting for https://github.com/chip-8/chip-8-database/pull/30
- add ROMs to Mr Worm: waiting for https://github.com/chip-8/chip-8-database/pull/28
- add Bad Apple!!: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Bad Apple: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Beware of Sn8k!: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add BIM Logo Animation: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Blobo: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add C8SALT logo: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Chipception OS: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Collision Course: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Destroy The Line: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Dodge Catch Shoot: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Duel Of The F8s: Special Edition: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Enter the mine: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add F-Chip8 Racing: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Falling Ghosts: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Fencing: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add H8 2: Carry Flag: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Halloween-Blaster: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Invisible Man: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Kanji Calculator (WIP): waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Man Hang: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Octogon Trail: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Octris: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add Termlib demo: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add RPN Calculator: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add XO-NYANCAT!!: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add XO-TRACKER DEMO: waiting for https://github.com/chip-8/chip-8-database/pull/31
- add ROMs to Guess the Number: waiting for https://github.com/chip-8/chip-8-database/pull/32
- add ROMs to Rush Hour: waiting for https://github.com/chip-8/chip-8-database/pull/32
- add ROMs to Delay Timer Test: waiting for https://github.com/chip-8/chip-8-database/pull/32
- add ROMs to Random Number Test: waiting for https://github.com/chip-8/chip-8-database/pull/32

## Not in chip-8-database

- kripod/chip8-roms: `hires/Astro Dodge Hires [Revival Studios, 2008].ch8`
- kripod/chip8-roms: `hires/Hires Maze [David Winter, 199x].ch8`
- kripod/chip8-roms: `hires/Hires Particle Demo [zeroZshadow, 2008].ch8`
- kripod/chip8-roms: `hires/Hires Sierpinski [Sergey Naydenov, 2010].ch8`
- kripod/chip8-roms: `hires/Hires Stars [Sergey Naydenov, 2010].ch8`
- kripod/chip8-roms: `hires/Hires Test [Tom Swan, 1979].ch8`
- kripod/chip8-roms: `hires/Hires Worm V4 [RB-Revival Studios, 2007].ch8`
- kripod/chip8-roms: `hires/Trip8 Hires Demo (2008) [Revival Studios].ch8`

## chip8Archive vs chip-8-database

The database wins; these are candidates for upstream fixes.

- Chipcross (83300ff710acdd8417376b88adf40f68171f7ec7): platform: chip-8-database "xochip", chip8Archive "schip"
- Eaty The Alien (fcaa793332a83c93f4ed79f5ffbc8403c8b8aea0): platform: chip-8-database "xochip", chip8Archive "schip"
- Grad School Simulator 2014 (9797a7eaf1e80ec19c085c60bb37991420f54678): platform: chip-8-database "xochip", chip8Archive "schip"
- Octojam 1 Title (f26993a4afd5cda2fea19935773fd3db54866623): platform: chip-8-database "superchip,xochip", chip8Archive "chip8"
- Octovore (74936ffb0db233c722f5a39932d75240b6437a72): platform: chip-8-database "xochip", chip8Archive "schip"
- Piper (7da3eba52a8d8025ddf14ee40d28f151585529a0): platform: chip-8-database "superchip,xochip", chip8Archive "chip8"
- Sub-Terr8nia (64536d549c986e9edf25de9fa89db60d2ade85c0): platform: chip-8-database "xochip", chip8Archive "schip"
