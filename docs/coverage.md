# ROM coverage

Generated 2026-11-15T01:26:02Z by `cmd/catalogue-gen`.

chip-8-database: 521 programs, 585 ROMs.

| Source | Unique ROMs | In database | Missing |
|---|---|---|---|
| chip8Archive | 104 | 104 | 0 |
| kripod/chip8-roms | 106 | 106 | 0 |
| Timendus CHIP-8 test suite | 8 | 8 | 0 |
| Matthew Mikolay's CHIP-8 programs | 6 | 6 | 0 |
| 3D Viper Maze (Timendus) | 1 | 1 | 0 |
| 3D VIP'r Maze (Timendus) | 2 | 2 | 0 |
| Octojam games from itch.io (via maxisimonazzi/chip8-emulator) | 68 | 68 | 0 |
| Hosted by Chippie (from Zophar and David Winter) | 18 | 18 | 0 |
| **All combined** | **310** | **310** | **0** |

Downloadable ROMs in the manifest: 310 of 585 in the database and Chippie's overlay.

## Chippie's overlay (pending upstream)

Additions and corrections in `assets/catalogue/overlay.json`. `catalogue-gen` deletes the "done" ones (upstream has them) from it as it writes this report.

Empty.

## Not in chip-8-database


## chip8Archive vs chip-8-database

The database wins; these are candidates for upstream fixes.

- Chipcross (83300ff710acdd8417376b88adf40f68171f7ec7): platform: chip-8-database "xochip", chip8Archive "schip"
- Eaty The Alien (fcaa793332a83c93f4ed79f5ffbc8403c8b8aea0): platform: chip-8-database "xochip", chip8Archive "schip"
- Grad School Simulator 2014 (9797a7eaf1e80ec19c085c60bb37991420f54678): platform: chip-8-database "xochip", chip8Archive "schip"
- Octojam 1 Title (f26993a4afd5cda2fea19935773fd3db54866623): platform: chip-8-database "superchip,xochip", chip8Archive "chip8"
- Octovore (74936ffb0db233c722f5a39932d75240b6437a72): platform: chip-8-database "xochip", chip8Archive "schip"
- Piper (7da3eba52a8d8025ddf14ee40d28f151585529a0): platform: chip-8-database "superchip,xochip", chip8Archive "chip8"
- Sub-Terr8nia (64536d549c986e9edf25de9fa89db60d2ade85c0): platform: chip-8-database "xochip", chip8Archive "schip"
