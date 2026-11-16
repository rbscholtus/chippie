# Draft 08: add the 8 Hires CHIP-8 programs

- **Where:** PR to [chip-8/chip-8-database](https://github.com/chip-8/chip-8-database),
  from a branch of the owner's fork, after the platform issue (draft 07)
  is agreed. It can add the platform too, if the maintainers prefer one PR.
- **Status:** posted 2026-11-16 as
  [#34](https://github.com/chip-8/chip-8-database/pull/34), open. It also
  adds the platform (`platforms.json`, the schema's platform list and the
  README's), because the schema only accepts known platform ids; one
  sentence saying so was added to the body, as agreed. Branch
  `add-hires-chip8` of the owner's fork.
- **Overlay entries:** the 8 `add` entries on `hiresChip8` in
  `catalogue/overlay.json`.

## What and why

kripod/chip8-roms' `hires/` folder holds 8 ROMs for the COSMAC VIP's
64×64 mode. None is in the database. Each is added as a program of its
own, named as in kripod's file names, because the database already lists
lo-res and SUPER-CHIP versions of several of them as separate programs
(e.g. Super Trip8 Demo, Super Particle Demo).

| Program | File (kripod `hires/`) | Bytes | SHA-1 |
|---|---|---|---|
| Astro Dodge Hires | `Astro Dodge Hires [Revival Studios, 2008].ch8` | 1531 | `066e7a84efde433e4d937d8aa41518666955086c` |
| Hires Maze | `Hires Maze [David Winter, 199x].ch8` | 230 | `70aa0e7f25f0f0fd6ec7c59e427bf1d03ee95617` |
| Hires Particle Demo | `Hires Particle Demo [zeroZshadow, 2008].ch8` | 545 | `1ebcb2ec0be2ec9fa209d5c73be19b2d408399bf` |
| Hires Sierpinski | `Hires Sierpinski [Sergey Naydenov, 2010].ch8` | 725 | `200b313e4d4c1970641142cc7ff578d7956b93da` |
| Hires Stars | `Hires Stars [Sergey Naydenov, 2010].ch8` | 1162 | `af98ee11adae28a6153cae8e4c16afa00f861907` |
| Hires Test | `Hires Test [Tom Swan, 1979].ch8` | 214 | `8d56a781bf16acccb307177b80ff326f62aabbdc` |
| Hires Worm V4 | `Hires Worm V4 [RB-Revival Studios, 2007].ch8` | 881 | `71d06da9e605804d2099b808c02548ab2b3511b2` |
| Trip8 Hires Demo | `Trip8 Hires Demo (2008) [Revival Studios].ch8` | 3156 | `b2c55b6aba3e2910036d5b5bc3956cf7493e0221` |

## Review checklist

- [x] SHA-1s computed from the files at Chippie's pinned commit (kripod
  `c723a9e`).
- [x] All 8 play in Chippie on `hiresChip8` (2026-11-15).
- [x] **One program each** (owner's decision, 2026-11-15), not extra ROMs
  on the lo-res programs (Astro Dodge, Maze, Particle Demo, Sierpinski,
  Stars, SuperWorm V4, Trip8 Demo). They are different programs for a
  different platform, as the database already does for Super Trip8 Demo
  and Super Particle Demo, and it matches Chippie's overlay.
- [x] Titles, authors and descriptions checked by the owner (with the text, 2026-11-15). Authors follow
  the database's entries for the lo-res versions.
- [x] Text below approved (owner, 2026-11-15). All 8 also play-tested by the owner.

## The PR

**Title:** Add the 8 Hires CHIP-8 (64×64) programs from kripod/chip8-roms

**Body:**

> Following #ISSUE, this adds the `hiresChip8` platform's 8 known programs,
> from kripod/chip8-roms' `hires/` folder. Each is its own program, named
> as in kripod's file names, like the existing Super Trip8 Demo and Super
> Particle Demo, which are separate from their lo-res versions too.
>
> Titles, authors and dates come from the file names and the folder's
> `.txt` files; authors match the database's entries for the lo-res
> versions. Astro Dodge Hires has a `keys` hint from its notes (2/4/6/8 to
> move, 5 to start). They're appended at the end of `programs.json`, and
> `sha1-hashes.json` is extended to match.
>
> Checked by playing each ROM in Chippie (https://rbscholtus.github.io/chippie/).

## The change

Appended at the end of `database/programs.json`:

```json
[
  {
    "title": "Astro Dodge Hires",
    "description": "The 64x64 version of Astro Dodge: make your way through the asteroid field, scoring a point for each asteroid you dodge. Keys 2, 4, 6 and 8 move the ship; 5 starts the game.",
    "release": "2008",
    "authors": [
      "Revival Studios",
      "Martijn Wenting"
    ],
    "roms": {
      "066e7a84efde433e4d937d8aa41518666955086c": {
        "file": "Astro Dodge Hires [Revival Studios, 2008].ch8",
        "platforms": [
          "hiresChip8"
        ],
        "keys": {
          "up": 2,
          "down": 8,
          "left": 4,
          "right": 6,
          "a": 5
        }
      }
    }
  },
  {
    "title": "Hires Maze",
    "description": "The 64x64 version of Maze: draws a random maze of diagonal lines across the whole screen.",
    "release": "199x",
    "authors": [
      "David Winter"
    ],
    "roms": {
      "70aa0e7f25f0f0fd6ec7c59e427bf1d03ee95617": {
        "file": "Hires Maze [David Winter, 199x].ch8",
        "platforms": [
          "hiresChip8"
        ]
      }
    }
  },
  {
    "title": "Hires Particle Demo",
    "description": "The 64x64 version of zeroZshadow's particle demo. Works on real hardware as well as emulators.",
    "release": "2008",
    "authors": [
      "zeroZshadow"
    ],
    "roms": {
      "1ebcb2ec0be2ec9fa209d5c73be19b2d408399bf": {
        "file": "Hires Particle Demo [zeroZshadow, 2008].ch8",
        "platforms": [
          "hiresChip8"
        ]
      }
    }
  },
  {
    "title": "Hires Sierpinski",
    "description": "The 64x64 version of the Sierpinski demo.",
    "release": "2010",
    "authors": [
      "Sergey Naydenov"
    ],
    "roms": {
      "200b313e4d4c1970641142cc7ff578d7956b93da": {
        "file": "Hires Sierpinski [Sergey Naydenov, 2010].ch8",
        "platforms": [
          "hiresChip8"
        ]
      }
    }
  },
  {
    "title": "Hires Stars",
    "description": "The 64x64 version of the Stars demo.",
    "release": "2010",
    "authors": [
      "Sergey Naydenov"
    ],
    "roms": {
      "af98ee11adae28a6153cae8e4c16afa00f861907": {
        "file": "Hires Stars [Sergey Naydenov, 2010].ch8",
        "platforms": [
          "hiresChip8"
        ]
      }
    }
  },
  {
    "title": "Hires Test",
    "description": "Draws a diagonal pattern across the whole 64x64 screen, to show that the mode works.",
    "release": "1979",
    "authors": [
      "Tom Swan"
    ],
    "roms": {
      "8d56a781bf16acccb307177b80ff326f62aabbdc": {
        "file": "Hires Test [Tom Swan, 1979].ch8",
        "platforms": [
          "hiresChip8"
        ]
      }
    }
  },
  {
    "title": "Hires Worm V4",
    "description": "The 64x64 port of SuperWorm V4, Martijn Wenting's update of RB's SUPER-CHIP game Worm3.",
    "release": "2007",
    "authors": [
      "RB-Revival Studios",
      "Martijn Wenting"
    ],
    "roms": {
      "71d06da9e605804d2099b808c02548ab2b3511b2": {
        "file": "Hires Worm V4 [RB-Revival Studios, 2007].ch8",
        "platforms": [
          "hiresChip8"
        ]
      }
    }
  },
  {
    "title": "Trip8 Hires Demo",
    "description": "The 64x64 version of the Trip8 demo: an intro, 3D vector balls and four randomised dot effects.",
    "release": "2008",
    "authors": [
      "Revival Studios"
    ],
    "roms": {
      "b2c55b6aba3e2910036d5b5bc3956cf7493e0221": {
        "file": "Trip8 Hires Demo (2008) [Revival Studios].ch8",
        "platforms": [
          "hiresChip8"
        ]
      }
    }
  }
]
```

## Follow-up: update for the 2 Page Display (2026-11-17)

After tobiasvl's comment on chip-8-database issue #33 (see draft 07), one
more commit on branch `add-hires-chip8`, synced from Chippie's overlay
(Chippie PR #20):

- `hiresChip8` → `twoPageChip8` in `platforms.json`, the schema and the
  README; name "2 Page Display for CHIP-8", `release` 1978-09, and the new
  description (how programs start, how to detect the patch).
- Hires Test: authors Andy Modla and Jef Winsor, release 1978-09, and a
  description saying where it comes from.
- Hires Sierpinski and Hires Stars: `"platforms": []` and
  `quirkyPlatforms` `twoPageChip8` with `memoryLeaveIUnchanged: true`.

`npm test` (schema and formatting) passes.

- [ ] Commit approved for pushing.
- [ ] Title and body below approved.

**Title:** Add the 2 Page Display for CHIP-8 platform and its 8 programs
from kripod/chip8-roms

**Body:**

> Following #33, this adds the `twoPageChip8` platform, Andy Modla and Jef
> Winsor's 2 Page Display for CHIP-8 (VIPER vol. 1 issue 3, September 1978),
> and the 8 programs for it in kripod/chip8-roms' `hires/` folder. Each is
> its own program, named as in kripod's file names, like the existing Super
> Trip8 Demo and Super Particle Demo, which are separate from their lo-res
> versions too.
>
> The platform is in `platforms.json`, the schema's platform list and the
> README, so the database stays valid. Happy to split it if you'd rather
> settle the platform first.
>
> Titles, authors and dates come from the file names and the folder's
> `.txt` files; authors match the database's entries for the lo-res
> versions. Two exceptions, explained in #33: Hires Test is credited to
> Modla and Winsor, because it is the article's own test program, and
> Hires Sierpinski and Hires Stars are under `quirkyPlatforms` with
> `memoryLeaveIUnchanged`. Astro Dodge Hires has a `keys` hint from its
> notes (2/4/6/8 to move, 5 to start). The programs are appended at the
> end of `programs.json`, and `sha1-hashes.json` is extended to match.
>
> Checked by playing each ROM in Chippie (https://rbscholtus.github.io/chippie/).
